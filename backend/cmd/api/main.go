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
	"github.com/mittolabs/applad/internal/logger"
	"github.com/mittolabs/applad/internal/router"
	"github.com/mittolabs/applad/internal/status"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(logger.New(cfg.AppEnv))

	if cfg.JWTSecret == "change-me-in-production" {
		if cfg.AppEnv == "production" {
			slog.Error("JWT_SECRET must be changed from default in production")
			panic("unsafe JWT_SECRET in production")
		}
		slog.Warn("using default JWT_SECRET — set JWT_SECRET before going to production")
	}
	if len(cfg.JWTSecret) < 32 {
		if cfg.AppEnv == "production" {
			slog.Error("JWT_SECRET must be at least 32 characters in production")
			panic("JWT_SECRET too short for production")
		}
		slog.Warn("JWT_SECRET is shorter than 32 characters — use a longer secret in production")
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
