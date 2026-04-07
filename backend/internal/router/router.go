package router

import (
	"context"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/mittolabs/applad/internal/analytics"
	"github.com/mittolabs/applad/internal/appcache"
	"github.com/mittolabs/applad/internal/audit"
	"github.com/mittolabs/applad/internal/auth"
	"github.com/mittolabs/applad/internal/billing"
	"github.com/mittolabs/applad/internal/avatars"
	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/console"
	"github.com/mittolabs/applad/internal/content"
	"github.com/mittolabs/applad/internal/credentials"
	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/edge"
	"github.com/mittolabs/applad/internal/flags"
	"github.com/mittolabs/applad/internal/functions"
	"github.com/mittolabs/applad/internal/health"
	"github.com/mittolabs/applad/internal/jobs"
	"github.com/mittolabs/applad/internal/locale"
	"github.com/mittolabs/applad/internal/messaging"
	"github.com/mittolabs/applad/internal/migrations"
	mw "github.com/mittolabs/applad/internal/middleware"
	oauthpkg "github.com/mittolabs/applad/internal/oauth"
	"github.com/mittolabs/applad/internal/organizations"
	"github.com/mittolabs/applad/internal/projects"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/regions"
	"github.com/mittolabs/applad/internal/search"
	"github.com/mittolabs/applad/internal/storage"
	"github.com/mittolabs/applad/internal/teams"
	"github.com/mittolabs/applad/internal/usage"
	"github.com/mittolabs/applad/internal/vectors"
	"github.com/mittolabs/applad/internal/webhooks"
	"github.com/mittolabs/applad/internal/workflows"
)

// New builds and returns the application router.
func New(cfg *config.Config, database *db.DB, cacheClient *cache.Cache) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.CORS)
	r.Use(mw.SecurityHeaders)
	r.Use(mw.RateLimitRedis(100, cacheClient.Client()))
	r.Use(mw.MaxBodySize(10 << 20))

	// Audit log middleware — records all authenticated API calls
	auditSvc := audit.NewService(database)
	r.Use(audit.Middleware(auditSvc))

	projectSvc := projects.NewService(database)
	authSvc := auth.NewService(database, cfg.JWTSecret)
	dbSvc := databases.NewService(database)
	storageSvc := storage.NewService(database, cfg.StoragePath)
	teamSvc := teams.NewService(database)
	deployQueue := queue.New(cacheClient.Client())
	deploySvc := deploy.NewService(database, deployQueue)
	healthHandler := health.NewHandler(database, cacheClient)
	messagingSvc := messaging.NewService(messaging.Config{
		Host:         cfg.SMTPHost,
		Port:         cfg.SMTPPort,
		Username:     cfg.SMTPUser,
		Password:     cfg.SMTPPass,
		From:         cfg.SMTPFrom,
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

	// Realtime hub — wire into services for auto-publishing
	hub := realtime.NewHub()
	realtimeHandler := realtime.NewHandler(hub)
	dbSvc.SetEventPublisher(hub)
	storageSvc.SetEventPublisher(hub)

	// Workflows
	workflowQueue := queue.New(cacheClient.Client())
	workflowSvc := workflows.NewService(database, workflowQueue)
	workflowHandler := workflows.NewHandler(workflowSvc)

	// Console auth
	consoleSvc := console.NewService(database, cfg.JWTSecret)
	consoleHandler := console.NewHandler(consoleSvc, cfg.ConsoleSignupEnabled)

	r.Route("/v1", func(r chi.Router) {
		// Health — no auth required
		r.Mount("/health", health.Routes(healthHandler))

		// Console auth — system-level admin signup/login (no project header)
		r.Mount("/console", console.Routes(consoleHandler))

		// Organizations — console-level (no project header needed)
		orgSvc := organizations.NewService(database)
		r.Mount("/organizations", organizations.Routes(organizations.NewHandler(orgSvc)))

		// Projects — no project header needed (these manage projects)
		r.Mount("/projects", projects.Routes(projects.NewHandler(projectSvc)))

		// Locale — no auth required
		r.Mount("/locale", locale.Routes(locale.NewHandler()))

		// Avatars — no auth required
		r.Mount("/avatars", avatars.Routes(avatars.NewHandler()))

		// Regions — public catalog (no auth)
		r.Mount("/regions", regions.PublicRoutes(regions.NewHandler(regions.NewService(database))))

		// All service routes require X-Applad-Project header + optional auth
		r.Group(func(r chi.Router) {
			r.Use(mw.ProjectContext)
			r.Use(mw.Authenticate(cfg.JWTSecret, projectSvc))

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

			// Server-side routes — require auth
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireAuth)
				r.Mount("/users", auth.UserRoutes(auth.NewHandler(authSvc)))
				r.Mount("/teams", teams.Routes(teams.NewHandler(teamSvc)))
				r.Mount("/databases", databases.Routes(databases.NewHandler(dbSvc)))
				r.Mount("/storage", storage.Routes(storage.NewHandler(storageSvc)))
				r.Mount("/messaging", messaging.Routes(messaging.NewHandler(messagingSvc)))
				r.Mount("/deploy", deploy.Routes(deploy.NewHandler(deploySvc)))
				r.Mount("/functions", functions.Routes(functions.NewHandler(functionSvc)))
				r.Mount("/workflows", workflows.Routes(workflowHandler))

				// Migrations
				migrationQueue := queue.New(cacheClient.Client())
				migrationSvc := migrations.NewService(database, migrationQueue)
				r.Mount("/migrations", migrations.Routes(migrations.NewHandler(migrationSvc)))
				r.Mount("/credentials", credentials.Routes(credentials.NewHandler(credentials.NewService(database))))
				r.Mount("/flags", flags.Routes(flags.NewHandler(flags.NewService(database))))

				// Webhooks
				webhookSvc := webhooks.NewService(database)
				r.Mount("/webhooks", webhooks.Routes(webhooks.NewHandler(webhookSvc)))

				// Usage analytics
				usageSvc := usage.NewService(database)
				r.Mount("/usage", usage.Routes(usage.NewHandler(usageSvc)))

				// Future services (experimental)
				r.Mount("/analytics", analytics.Routes(analytics.NewHandler(analytics.NewService(database))))
				r.Mount("/cache", appcache.Routes(appcache.NewHandler(appcache.NewService(cacheClient.Client()))))
				r.Mount("/billing", billing.Routes(billing.NewHandler(billing.NewService(database))))
				r.Mount("/content", content.Routes(content.NewHandler(content.NewService(database))))
				r.Mount("/edge", edge.Routes(edge.NewHandler(edge.NewService(database))))
				r.Mount("/jobs", jobs.Routes(jobs.NewHandler(jobs.NewService(database))))
				r.Mount("/search", search.Routes(search.NewHandler(search.NewService(database))))
				r.Mount("/vectors", vectors.Routes(vectors.NewHandler(vectors.NewService(database))))
				r.Mount("/project-regions", regions.ProjectRoutes(regions.NewHandler(regions.NewService(database))))
			})
		})
	})

	return r
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
