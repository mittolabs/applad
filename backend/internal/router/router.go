package router

import (
	"context"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/mittolabs/applad/internal/auth"
	"github.com/mittolabs/applad/internal/avatars"
	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/console"
	oauthpkg "github.com/mittolabs/applad/internal/oauth"
	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/functions"
	"github.com/mittolabs/applad/internal/health"
	"github.com/mittolabs/applad/internal/locale"
	"github.com/mittolabs/applad/internal/messaging"
	mw "github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/projects"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/storage"
	"github.com/mittolabs/applad/internal/teams"
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
	r.Use(mw.RateLimit(100))
	r.Use(mw.MaxBodySize(10 << 20))

	projectSvc := projects.NewService(database)
	authSvc := auth.NewService(database, cfg.JWTSecret)
	dbSvc := databases.NewService(database)
	storageSvc := storage.NewService(database, cfg.StoragePath)
	teamSvc := teams.NewService(database)
	deploySvc := deploy.NewService(database)
	healthHandler := health.NewHandler(database, cacheClient)
	messagingSvc := messaging.NewService(messaging.Config{
		Host:         cfg.SMTPHost,
		Port:         cfg.SMTPPort,
		Username:     cfg.SMTPUser,
		Password:     cfg.SMTPPass,
		From:         cfg.SMTPFrom,
		TwilioSID:    cfg.TwilioSID,
		TwilioToken:  cfg.TwilioToken,
		TwilioFrom:   cfg.TwilioFrom,
		FCMServerKey: cfg.FCMServerKey,
	})

	// Functions
	functionQueue := queue.New(cacheClient.Client())
	functionSvc := functions.NewService(database, functionQueue)

	// Realtime hub
	hub := realtime.NewHub()
	realtimeHandler := realtime.NewHandler(hub)

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

		// Projects — no project header needed (these manage projects)
		r.Mount("/projects", projects.Routes(projects.NewHandler(projectSvc)))

		// Locale — no auth required
		r.Mount("/locale", locale.Routes(locale.NewHandler()))

		// Avatars — no auth required
		r.Mount("/avatars", avatars.Routes(avatars.NewHandler()))

		// All service routes require X-Applad-Project header + optional auth
		r.Group(func(r chi.Router) {
			r.Use(mw.ProjectContext)
			r.Use(mw.Authenticate(cfg.JWTSecret, projectSvc))

			// Account (client-side) — some public, some require auth
			authHandler := auth.NewHandler(authSvc)

			// Wire OAuth2 providers
			oauthProviders := oauthpkg.Providers(
				oauthpkg.ProviderConfig{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret},
				oauthpkg.ProviderConfig{ClientID: cfg.GitHubClientID, ClientSecret: cfg.GitHubClientSecret},
				oauthpkg.ProviderConfig{ClientID: cfg.AppleClientID, ClientSecret: cfg.AppleClientSecret},
			)
			oauthAdapters := make(map[string]auth.OAuthProvider, len(oauthProviders))
			for name, p := range oauthProviders {
				oauthAdapters[name] = &oauthAdapter{p: p}
			}
			authHandler.SetOAuthProviders(oauthAdapters)

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
