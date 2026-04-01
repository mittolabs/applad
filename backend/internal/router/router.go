package router

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/mittolabs/applad/internal/auth"
	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/health"
	mw "github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/projects"
	"github.com/mittolabs/applad/internal/storage"
	"github.com/mittolabs/applad/internal/teams"
)

// New builds and returns the application router.
func New(cfg *config.Config, database *db.DB, cacheClient *cache.Cache) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.CORS)

	projectSvc := projects.NewService(database)
	authSvc := auth.NewService(database, cfg.JWTSecret)
	dbSvc := databases.NewService(database)
	storageSvc := storage.NewService(database, cfg.StoragePath)
	teamSvc := teams.NewService(database)
	healthHandler := health.NewHandler(database, cacheClient)

	r.Route("/v1", func(r chi.Router) {
		// Health — no auth required
		r.Mount("/health", health.Routes(healthHandler))

		// Projects — no project header needed (these manage projects)
		r.Mount("/projects", projects.Routes(projects.NewHandler(projectSvc)))

		// All service routes require X-Applad-Project header + optional auth
		r.Group(func(r chi.Router) {
			r.Use(mw.ProjectContext)
			r.Use(mw.Authenticate(cfg.JWTSecret, projectSvc))

			// Account (client-side) — some public, some require auth
			r.Mount("/account", auth.AccountRoutes(auth.NewHandler(authSvc)))

			// Server-side routes — require auth
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireAuth)
				r.Mount("/users", auth.UserRoutes(auth.NewHandler(authSvc)))
				r.Mount("/teams", teams.Routes(teams.NewHandler(teamSvc)))
				r.Mount("/databases", databases.Routes(databases.NewHandler(dbSvc)))
				r.Mount("/storage", storage.Routes(storage.NewHandler(storageSvc)))
			})
		})
	})

	return r
}
