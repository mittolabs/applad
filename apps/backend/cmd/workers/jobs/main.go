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

	w := worker.NewJobs(cfg)

	slog.Info("jobs worker starting")
	if err := w.Start(ctx); err != nil {
		slog.Error("jobs worker error", "error", err)
		os.Exit(1)
	}
	slog.Info("jobs worker stopped")
}
