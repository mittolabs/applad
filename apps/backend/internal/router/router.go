package router

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/mittolabs/applad/internal/aichat"
	"github.com/mittolabs/applad/internal/analytics"
	"github.com/mittolabs/applad/internal/appcache"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/audit"
	"github.com/mittolabs/applad/internal/auth"
	"github.com/mittolabs/applad/internal/avatars"
	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/console"
	"github.com/mittolabs/applad/internal/credentials"
	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/edge"
	"github.com/mittolabs/applad/internal/entitlements"
	"github.com/mittolabs/applad/internal/extensions"
	"github.com/mittolabs/applad/internal/flags"
	"github.com/mittolabs/applad/internal/functions"
	"github.com/mittolabs/applad/internal/githubapp"
	"github.com/mittolabs/applad/internal/health"
	"github.com/mittolabs/applad/internal/jobs"
	"github.com/mittolabs/applad/internal/locale"
	"github.com/mittolabs/applad/internal/messaging"
	"github.com/mittolabs/applad/internal/metrics"
	mw "github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/migrations"
	oauthpkg "github.com/mittolabs/applad/internal/oauth"
	"github.com/mittolabs/applad/internal/observe"
	"github.com/mittolabs/applad/internal/organizations"
	"github.com/mittolabs/applad/internal/plan"
	"github.com/mittolabs/applad/internal/projects"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/regions"
	"github.com/mittolabs/applad/internal/search"
	"github.com/mittolabs/applad/internal/status"
	"github.com/mittolabs/applad/internal/storage"
	"github.com/mittolabs/applad/internal/teams"
	"github.com/mittolabs/applad/internal/testlab"
	"github.com/mittolabs/applad/internal/trace"
	"github.com/mittolabs/applad/internal/usage"
	"github.com/mittolabs/applad/internal/vectors"
	"github.com/mittolabs/applad/internal/webhooks"
	"github.com/mittolabs/applad/internal/workflows"
)

// New builds and returns the application router.
func New(cfg *config.Config, database *db.DB, cacheClient *cache.Cache) *chi.Mux {
	r := chi.NewRouter()

	// Audit log middleware — records all authenticated API calls
	auditSvc := audit.NewService(database)

	// Perf collector — buffers per-request latencies and flushes percentiles
	// to observe_perf_snapshots every 60 s.
	observeSvcEarly := observe.NewService(database)
	perfCollector := observe.NewPerfCollector(observeSvcEarly, 60*time.Second)
	perfCollector.Start(context.Background())

	// JSON 404 / 405 — override chi's plain-text defaults.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		apperr.Write(w, http.StatusNotFound, "general_route_not_found",
			"Route "+r.Method+" "+r.URL.Path+" not found.")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		apperr.Write(w, http.StatusMethodNotAllowed, "general_method_not_allowed",
			"Method "+r.Method+" is not allowed on this route.")
	})

	r.Use(chimw.RequestID)
	// Forwarding headers are believed only from TRUSTED_PROXY_CIDRS peers;
	// chi's RealIP believed anyone, letting a client choose its own address.
	r.Use(mw.RealIP)
	r.Use(trace.Middleware)
	r.Use(observabilityMiddleware(perfCollector))
	r.Use(mw.Recover) // JSON panic recovery (replaces chimw.Recoverer)
	r.Use(mw.CORS)
	r.Use(mw.SecurityHeaders)
	// The secret lets the limiter verify a token before granting the larger
	// authed bucket, instead of taking any Authorization header's word for it.
	r.Use(mw.RateLimitRedisTiered(cfg.RateLimitAnonPerMinute, cfg.RateLimitAuthedPerMinute, cacheClient.Client(), cfg.JWTSecret))
	// Credential attempts, before anything knows who the caller is. Keyed by
	// address and by the account being attempted, because an attacker rotates
	// the first and cannot rotate the second.
	r.Use(mw.RateLimitRules(cacheClient.Client(), mw.AuthRules()))
	r.Use(mw.MaxBodySize(10 << 20))
	r.Use(audit.Middleware(auditSvc))

	// Metrics endpoint — expose on a separate path so it can be firewall-restricted.
	// In Kubernetes, restrict access with a NetworkPolicy to allow only Prometheus pods.
	r.Handle("/metrics", metrics.Default.Handler())

	projectSvc := projects.NewService(database, cfg.APIKeySecret, cfg.JWTSecret)
	authSvc := auth.NewService(database, cfg.JWTSecret)
	dbSvc := databases.NewService(database)
	storageSvc := storage.NewService(database, cfg.StoragePath, cfg.JWTSecret)
	if cfg.StorageDriver == "s3" {
		storageSvc.SetDriver(storage.NewS3Driver(
			cfg.S3Endpoint, cfg.S3Bucket, cfg.S3Region, cfg.S3AccessKey, cfg.S3SecretKey,
		))
	}
	teamSvc := teams.NewService(database)
	deployQueue := queue.New(cacheClient.Client())
	deploySvc := deploy.NewService(database, deployQueue)
	// Git deploys go through the Applad GitHub App where one is configured.
	// Without it the console offers no "Connect GitHub" button rather than
	// offering one that cannot work.
	if ghApp, err := githubapp.FromConfig(cfg); err == nil {
		deploySvc.SetGitHubApp(ghApp)
		slog.Info("github app configured", "slug", ghApp.Slug())
	} else if !errors.Is(err, githubapp.ErrNotConfigured) {
		slog.Error("github app misconfigured — git deploys will be unavailable", "error", err)
	}
	healthHandler := health.NewHandler(database, cacheClient)
	messagingSvc := messaging.NewService(database, messaging.Config{
		Host:            cfg.SMTPHost,
		Port:            cfg.SMTPPort,
		Username:        cfg.SMTPUser,
		Password:        cfg.SMTPPass,
		From:            cfg.SMTPFrom,
		TwilioSID:       cfg.TwilioSID,
		TwilioToken:     cfg.TwilioToken,
		TwilioFrom:      cfg.TwilioFrom,
		FCMServerKey:    cfg.FCMServerKey,
		MailgunAPIKey:   cfg.MailgunAPIKey,
		MailgunDomain:   cfg.MailgunDomain,
		ResendAPIKey:    cfg.ResendAPIKey,
		VonageAPIKey:    cfg.VonageAPIKey,
		VonageAPISecret: cfg.VonageAPISecret,
		VonageFrom:      cfg.VonageFrom,
		MSG91AuthKey:    cfg.MSG91AuthKey,
		MSG91SenderID:   cfg.MSG91SenderID,
		APNSKeyID:       cfg.APNSKeyID,
		APNSTeamID:      cfg.APNSTeamID,
		APNSKeyPath:     cfg.APNSKeyPath,
		APNSBundleID:    cfg.APNSBundleID,
	})

	// Functions
	functionQueue := queue.New(cacheClient.Client())
	functionSvc := functions.NewService(database, functionQueue)

	// Realtime hub — wire into services for auto-publishing.
	// Pass RedisAddr so events are fanned through Redis when scaling horizontally.
	hub := realtime.NewHub(cfg.DatabaseDSN, cfg.RedisAddr)
	realtimeHandler := realtime.NewHandler(hub)
	dbSvc.SetEventPublisher(hub)
	storageSvc.SetEventPublisher(hub)

	// Workflows
	workflowQueue := queue.New(cacheClient.Client())
	workflowSvc := workflows.NewService(database, workflowQueue)
	workflowHandler := workflows.NewHandler(workflowSvc)

	// Deploy handler — created here so it can be shared between the public webhook
	// route (no auth) and the authenticated /deploy mount below.
	deploySvc.SetDeployDomain(cfg.DeployDomain)
	deploySvc.SetRedis(cacheClient.Client())
	deployHandler := deploy.NewHandler(deploySvc)
	testlabHandler := testlab.NewHandler(testlab.NewService(database, deployQueue), deployQueue, cacheClient.Client())

	// Console auth
	consoleSvc := console.NewService(database, cfg.JWTSecret)
	consoleHandler := console.NewHandler(consoleSvc, cfg.ConsoleSignupEnabled, console.SMTPConfig{
		Host: cfg.SMTPHost,
		Port: cfg.SMTPPort,
		User: cfg.SMTPUser,
		Pass: cfg.SMTPPass,
		From: cfg.SMTPFrom,
	}, console.CookieConfig{
		Domain: cfg.SessionCookieDomain,
	})

	// Console OAuth providers (Google, GitHub, SSO) — for admin console login only.
	// Per-project OAuth is configured through the console UI.
	consoleOAuthConfigs := map[string]oauthpkg.ProviderConfig{
		"github": {ClientID: cfg.ConsoleGitHubClientID, ClientSecret: cfg.ConsoleGitHubClientSecret},
		"google": {ClientID: cfg.ConsoleGoogleClientID, ClientSecret: cfg.ConsoleGoogleClientSecret},
	}
	consoleProviders := oauthpkg.Providers(consoleOAuthConfigs)
	if cfg.ConsoleSSOClientID != "" && cfg.ConsoleSSOAuthURL != "" {
		ssoProviders := oauthpkg.ProvidersWithDomain(
			map[string]oauthpkg.ProviderConfig{
				"sso": {ClientID: cfg.ConsoleSSOClientID, ClientSecret: cfg.ConsoleSSOClientSecret},
			},
			"", "", cfg.ConsoleSSOAuthURL, cfg.ConsoleSSOTokenURL, cfg.ConsoleSSOUserInfoURL,
		)
		for k, v := range ssoProviders {
			consoleProviders[k] = v
		}
	}
	consoleHandler.SetProviders(consoleProviders)

	r.Route("/v1", func(r chi.Router) {
		// Health — no auth required
		r.Mount("/health", health.Routes(healthHandler))

		// Status — public self-monitoring snapshot powering status.applad.io
		r.Mount("/status", status.Routes(status.NewHandler(status.NewService(database, cacheClient, cfg))))

		// Console auth — system-level admin signup/login (no project header)
		r.Mount("/console", console.Routes(consoleHandler))

		// Console-level management — no project header, but never anonymous:
		// a valid console session is required, and each handler checks org
		// membership through consoleSvc. These were mounted open, which let an
		// unauthenticated curl list every project and mint "*"-scoped keys.
		r.Group(func(r chi.Router) {
			r.Use(mw.RequireConsoleAuth(consoleSvc))

			orgSvc := organizations.NewService(database)
			r.Mount("/organizations", organizations.Routes(organizations.NewHandler(orgSvc, consoleSvc)))

			projectsHandler := projects.NewHandler(projectSvc)
			projectsHandler.SetAccess(consoleSvc)
			r.Mount("/projects", projects.Routes(projectsHandler))
		})

		// AI chat — console JWT required, no project header needed
		aiSvc := aichat.NewService(cfg.AIProvider, cfg.AIAPIKey, cfg.AIModel, cfg.AIBaseURL)
		r.Mount("/ai", aichat.Routes(aichat.NewHandler(aiSvc, consoleSvc, cfg.Port)))

		// Locale — no auth required
		r.Mount("/locale", locale.Routes(locale.NewHandler()))

		// Avatars — no auth required
		r.Mount("/avatars", avatars.Routes(avatars.NewHandler()))

		// Regions — public catalog (no auth)
		r.Mount("/regions", regions.PublicRoutes(regions.NewHandler(regions.NewService(database))))

		// Entitlements — what this subject may use, and anything to tell them
		// about it. Unlimited with no notices unless a provider is registered.
		r.Mount("/entitlements", entitlements.Routes(entitlements.NewHandler()))

		// Modules compiled into this build mount their own surface here. A
		// default build registers none and this loop does nothing.
		for _, m := range extensions.All() {
			if m.Routes == nil {
				continue
			}
			slog.Info("extensions: mounting routes", "module", m.Name)
			mod := m
			r.Group(func(gr chi.Router) { mod.Routes(gr, extensions.Deps{DB: database.DB}) })
		}

		// All service routes require X-Applad-Project header + optional auth
		r.Group(func(r chi.Router) {
			r.Use(mw.ProjectContext)
			// consoleSvc gates console JWTs by org membership: a console token
			// is an administrator's identity, not a skeleton key to every project.
			r.Use(mw.Authenticate(cfg.JWTSecret, projectSvc, consoleSvc))

			// Account (client-side) — some public, some require auth
			authHandler := auth.NewHandler(authSvc)

			// Wire OAuth2 providers
			oauthConfigs := map[string]oauthpkg.ProviderConfig{
				"google":      {ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret},
				"github":      {ClientID: cfg.GitHubClientID, ClientSecret: cfg.GitHubClientSecret},
				"apple":       {ClientID: cfg.AppleClientID, ClientSecret: cfg.AppleClientSecret},
				"amazon":      {ClientID: cfg.AmazonClientID, ClientSecret: cfg.AmazonClientSecret},
				"auth0":       {ClientID: cfg.Auth0ClientID, ClientSecret: cfg.Auth0ClientSecret},
				"autodesk":    {ClientID: cfg.AutodeskClientID, ClientSecret: cfg.AutodeskClientSecret},
				"bitly":       {ClientID: cfg.BitlyClientID, ClientSecret: cfg.BitlyClientSecret},
				"box":         {ClientID: cfg.BoxClientID, ClientSecret: cfg.BoxClientSecret},
				"dailymotion": {ClientID: cfg.DailymotionClientID, ClientSecret: cfg.DailymotionClientSecret},
				"disqus":      {ClientID: cfg.DisqusClientID, ClientSecret: cfg.DisqusClientSecret},
				"dropbox":     {ClientID: cfg.DropboxClientID, ClientSecret: cfg.DropboxClientSecret},
				"etsy":        {ClientID: cfg.EtsyClientID, ClientSecret: cfg.EtsyClientSecret},
				"figma":       {ClientID: cfg.FigmaClientID, ClientSecret: cfg.FigmaClientSecret},
				"hubspot":     {ClientID: cfg.HubspotClientID, ClientSecret: cfg.HubspotClientSecret},
				"kakao":       {ClientID: cfg.KakaoClientID, ClientSecret: cfg.KakaoClientSecret},
				"line":        {ClientID: cfg.LineClientID, ClientSecret: cfg.LineClientSecret},
				"mailchimp":   {ClientID: cfg.MailchimpClientID, ClientSecret: cfg.MailchimpClientSecret},
				"notion":      {ClientID: cfg.NotionClientID, ClientSecret: cfg.NotionClientSecret},
				"okta":        {ClientID: cfg.OktaClientID, ClientSecret: cfg.OktaClientSecret},
				"oidc":        {ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret},
				"patreon":     {ClientID: cfg.PatreonClientID, ClientSecret: cfg.PatreonClientSecret},
				"paypal":      {ClientID: cfg.PayPalClientID, ClientSecret: cfg.PayPalClientSecret},
				"podio":       {ClientID: cfg.PodioClientID, ClientSecret: cfg.PodioClientSecret},
				"reddit":      {ClientID: cfg.RedditClientID, ClientSecret: cfg.RedditClientSecret},
				"salesforce":  {ClientID: cfg.SalesforceClientID, ClientSecret: cfg.SalesforceClientSecret},
				"tradeshift":  {ClientID: cfg.TradeshiftClientID, ClientSecret: cfg.TradeshiftClientSecret},
				"wordpress":   {ClientID: cfg.WordPressClientID, ClientSecret: cfg.WordPressClientSecret},
				"yahoo":       {ClientID: cfg.YahooClientID, ClientSecret: cfg.YahooClientSecret},
				"yammer":      {ClientID: cfg.YammerClientID, ClientSecret: cfg.YammerClientSecret},
				"yandex":      {ClientID: cfg.YandexClientID, ClientSecret: cfg.YandexClientSecret},
				"zoho":        {ClientID: cfg.ZohoClientID, ClientSecret: cfg.ZohoClientSecret},
				"zoom":        {ClientID: cfg.ZoomClientID, ClientSecret: cfg.ZoomClientSecret},
			}
			oauthProviders := oauthpkg.ProvidersWithDomain(
				oauthConfigs,
				cfg.Auth0Domain, cfg.OktaDomain,
				cfg.OIDCAuthURL, cfg.OIDCTokenURL, cfg.OIDCUserInfoURL,
			)
			oauthAdapters := make(map[string]auth.OAuthProvider, len(oauthProviders))
			for name, p := range oauthProviders {
				oauthAdapters[name] = &oauthAdapter{p: p}
			}
			authHandler.SetOAuthProviders(oauthAdapters)
			authHandler.SetMailer(messagingSvc)

			r.Mount("/account", auth.AccountRoutes(authHandler))

			// Realtime WebSocket — auth optional
			r.Mount("/realtime", realtime.Routes(realtimeHandler))

			// Workflow webhook trigger — no auth required, resolves project from workflow ID
			r.Mount("/workflows/webhooks", workflows.WebhookRoutes(workflowHandler))

			// Git push/PR webhook — no project auth; HMAC-verified by the handler.
			r.Mount("/deploy/git/webhook", deploy.WebhookRoutes(deployHandler))

			// Server-side routes — require auth
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireAuth)
				r.Use(mw.RateLimitUser(300, cacheClient.Client()))
				// Work a project causes — builds, messages, executions — as
				// distinct from requests it makes. Applied here because the
				// project is known by now, and it is the project that pays.
				r.Use(mw.RateLimitRules(cacheClient.Client(), mw.ProjectWorkRules()))
				r.Mount("/users", auth.UserRoutes(auth.NewHandler(authSvc)))
				r.Mount("/teams", teams.Routes(teams.NewHandler(teamSvc)))
				r.Mount("/databases", databases.Routes(databases.NewHandler(dbSvc)))
				r.Mount("/storage", storage.Routes(storage.NewHandler(storageSvc)))
				r.Mount("/messaging", messaging.Routes(messaging.NewHandler(messagingSvc)))
				r.Mount("/deploy", deploy.Routes(deployHandler))
				r.Mount("/functions", functions.Routes(functions.NewHandler(functionSvc)))
				r.Mount("/workflows", workflows.Routes(workflowHandler))
				r.Mount("/tests", testlab.Routes(testlabHandler))
				r.Mount("/plan", plan.Routes(plan.NewHandler(plan.NewService(database))))
				r.Mount("/studio", testlab.StudioRoutes(testlabHandler))

				// Migrations
				migrationQueue := queue.New(cacheClient.Client())
				migrationSvc := migrations.NewService(database, migrationQueue)
				r.Mount("/migrations", migrations.Routes(migrations.NewHandler(migrationSvc)))
				r.Mount("/credentials", credentials.Routes(credentials.NewHandler(credentials.NewService(database))))
				r.Mount("/flags", flags.Routes(flags.NewHandler(flags.NewService(database))))

				// Webhooks
				webhookSvc := webhooks.NewService(database)
				r.Mount("/webhooks", webhooks.Routes(webhooks.NewHandler(webhookSvc)))

				// Audit logs
				r.Mount("/audit", audit.Routes(audit.NewHandler(auditSvc)))

				// Usage analytics
				usageSvc := usage.NewService(database)
				r.Mount("/usage", usage.Routes(usage.NewHandler(usageSvc)))

				// Future services (experimental)
				r.Mount("/analytics", analytics.Routes(analytics.NewHandler(analytics.NewService(database))))
				r.Mount("/cache", appcache.Routes(appcache.NewHandler(appcache.NewService(cacheClient.Client()))))
				r.Mount("/edge", edge.Routes(edge.NewHandler(edge.NewService(database))))
				r.Mount("/jobs", jobs.Routes(jobs.NewHandler(jobs.NewService(database))))
				r.Mount("/search", search.Routes(search.NewHandler(search.NewService(database))))
				r.Mount("/vectors", vectors.Routes(vectors.NewHandler(vectors.NewService(database))))
				r.Mount("/project-regions", regions.ProjectRoutes(regions.NewHandler(regions.NewService(database))))

				// Observe (errors, logs, performance, releases, replays, uptime, crons, alerts)
				observeSvc := observeSvcEarly
				observeSvc.StartUptimeWorker(context.Background())
				r.Mount("/observe", observe.Routes(observe.NewHandler(observeSvc)))
			})
		})
	})

	return r
}

// observabilityMiddleware records HTTP request metrics (count + latency) using
// the built-in metrics package, and also feeds per-project latency samples
// into the PerfCollector for DB-backed percentile snapshots.
func observabilityMiddleware(pc *observe.PerfCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)
			elapsed := time.Since(start)

			// Use chi's route pattern for low-cardinality labels.
			pattern := r.URL.Path
			if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
				pattern = rctx.RoutePattern()
			}

			metrics.ObserveRequest(r.Method, pattern, ww.status, start)

			// Record per-project sample for DB-backed performance snapshots.
			if projectID := mw.ProjectFromContext(r.Context()); projectID != "" {
				pc.Record(projectID, r.Method, pattern,
					float64(elapsed.Milliseconds()), ww.status >= 400)
			}
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Hijack passes the connection through, so wrapping a response does not stop
// a WebSocket from being upgraded. Without it every endpoint under this
// middleware — the studio's live session, realtime — fails the handshake.
func (sw *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := sw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("router: response does not support hijacking")
}

// Flush lets streaming responses through for the same reason.
func (sw *statusWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// oauthAdapter adapts oauth.Provider to auth.OAuthProvider interface.
type oauthAdapter struct {
	p *oauthpkg.Provider
}

func (a *oauthAdapter) GetAuthURL(redirectURI, state string) string {
	return a.p.GetAuthURL(redirectURI, state)
}

func (a *oauthAdapter) ExchangeCode(ctx context.Context, code, redirectURI string) (string, error) {
	return a.p.ExchangeCode(ctx, code, redirectURI)
}

func (a *oauthAdapter) GetUserInfo(ctx context.Context, accessToken string) (auth.OAuthUserInfo, error) {
	info, err := a.p.GetUserInfo(ctx, accessToken)
	if err != nil {
		return auth.OAuthUserInfo{}, err
	}
	return auth.OAuthUserInfo{
		ID: info.ID, Email: info.Email, Name: info.Name, Provider: info.Provider,
	}, nil
}
