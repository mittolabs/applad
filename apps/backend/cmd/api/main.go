package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/extensions"
	"github.com/mittolabs/applad/internal/logger"
	"github.com/mittolabs/applad/internal/router"
	"github.com/mittolabs/applad/internal/status"
	"github.com/mittolabs/applad/internal/usage"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(logger.New(cfg.AppEnv))

	// JWT_SECRET also derives the API-key pepper and the credentials-vault
	// key, so a known or short value forfeits every credential on the
	// instance. Refused in every environment: APP_ENV defaults to
	// "development", which is exactly what an unconfigured production box
	// runs as.
	if cfg.JWTSecret == "" || cfg.JWTSecret == "change-me-in-production" {
		slog.Error("JWT_SECRET is unset or the known default — generate one: openssl rand -hex 32")
		panic("refusing to start with default JWT_SECRET")
	}
	if len(cfg.JWTSecret) < 32 {
		slog.Error("JWT_SECRET must be at least 32 characters — generate one: openssl rand -hex 32")
		panic("refusing to start with short JWT_SECRET")
	}

	database, err := db.Connect(cfg.DatabaseDSN, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		panic(err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		slog.Error("db migrate failed", "error", err)
		panic(err)
	}

	// Modules compiled into this build own their own schema and read usage
	// through core rather than querying its tables. A default build registers
	// no modules, so both of these are no-ops.
	extensions.SetUsageReporter(usage.NewReporter(database))
	var extraMigrations []db.ExtraMigration
	for _, m := range extensions.Migrations() {
		extraMigrations = append(extraMigrations, db.ExtraMigration{Version: m.Version, SQL: m.SQL})
	}
	if err := database.MigrateExtras(extraMigrations); err != nil {
		slog.Error("extension migrate failed", "error", err)
		panic(err)
	}

	cacheClient, err := cache.Connect(cfg.RedisAddr)
	if err != nil {
		slog.Error("cache connect failed", "error", err)
		panic(err)
	}

	r := router.New(cfg, database, cacheClient)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("applad api listening", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
		}
	}()

	// Self-monitoring: probe our own components on a schedule so the public
	// status page (status.applad.io) reflects real health.
	go status.NewService(database, cacheClient, cfg).Run(ctx)

	<-ctx.Done()
	stop()

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
