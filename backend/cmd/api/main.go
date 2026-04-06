package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/router"
)

func main() {
	cfg := config.Load()

	// Refuse to start with default JWT secret in production
	if cfg.AppEnv == "production" && cfg.JWTSecret == "change-me-in-production" {
		log.Fatal("FATAL: JWT_SECRET must be changed from the default value in production. Set the JWT_SECRET environment variable to a strong random secret.")
	}
	if cfg.JWTSecret == "change-me-in-production" {
		log.Println("WARNING: Using default JWT_SECRET — set JWT_SECRET env var before going to production")
	}

	database, err := db.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	cacheClient, err := cache.Connect(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("cache connect: %v", err)
	}

	r := router.New(cfg, database, cacheClient)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("applad api listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("server stopped")
}
