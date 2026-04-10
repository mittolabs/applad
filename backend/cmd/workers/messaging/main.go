package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/logger"
	"github.com/mittolabs/applad/internal/worker"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(logger.New(cfg.AppEnv))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var w worker.Worker
	switch "messaging" {
	case "builds":
		w = worker.NewBuilds(cfg)
	case "certificates":
		w = worker.NewCertificates(cfg)
	case "cron":
		w = worker.NewCron(cfg)
	case "databases":
		w = worker.NewDatabases(cfg)
	case "deletes":
		w = worker.NewDeletes(cfg)
	case "executions":
		w = worker.NewExecutions(cfg)
	case "mails":
		w = worker.NewMails(cfg)
	case "messaging":
		w = worker.NewMessaging(cfg)
	case "migrations":
		w = worker.NewMigrations(cfg)
	case "usage":
		w = worker.NewUsage(cfg)
	case "webhooks":
		w = worker.NewWebhooks(cfg)
	default:
		slog.Error("unknown worker type", "type", "messaging")
		os.Exit(1)
	}

	slog.Info("messaging worker starting")
	if err := w.Start(ctx); err != nil {
		slog.Error("messaging worker error", "error", err)
		os.Exit(1)
	}
	slog.Info("messaging worker stopped")
}
