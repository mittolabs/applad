package worker

import (
	"context"
	"log/slog"

	"github.com/mittolabs/applad/internal/auth"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/projects"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/storage"
	"github.com/mittolabs/applad/internal/transfer"
	"github.com/redis/go-redis/v9"
)

// Transfer runs data-migration jobs: importing a project's data from another
// platform into Applad. Kept separate from the maintenance "migrations" worker.
type Transfer struct {
	cfg   *config.Config
	queue *queue.Queue
	svc   *transfer.Service
}

func NewTransfer(cfg *config.Config) *Transfer {
	return &Transfer{cfg: cfg}
}

func (w *Transfer) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	authSvc := auth.NewService(database, w.cfg.JWTSecret)
	dbSvc := databases.NewService(database)
	stgSvc := storage.NewService(database, w.cfg.StoragePath, w.cfg.JWTSecret)
	projectSvc := projects.NewService(database, w.cfg.APIKeySecret, w.cfg.JWTSecret)
	w.svc = transfer.NewService(database, authSvc, dbSvc, stgSvc, projectSvc, w.queue)

	StartRedisHeartbeat(ctx, rdb, transfer.QueueName)
	w.queue.StartReaper(ctx, transfer.QueueName)

	// Touch the file heartbeat before the first tick so the compose
	// healthcheck does not see the worker as hung during start-up.
	Heartbeat()

	slog.Info("transfer worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, transfer.QueueName)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("transfer worker: shutting down")
				return nil
			}
			slog.Error("transfer worker: pop error", "error", err)
			continue
		}
		Heartbeat()

		if receipt == nil {
			continue
		}
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc(transfer.QueueName, "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc(transfer.QueueName, "completed")
			receipt.Ack()
		}
	}
}

func (w *Transfer) process(ctx context.Context, job *queue.Job) error {
	if job.Type != "data_import" {
		slog.Warn("transfer worker: unknown job type", "type", job.Type)
		return nil
	}
	migrationID, _ := job.Payload["migrationId"].(string)
	if migrationID == "" {
		slog.Warn("transfer worker: job missing migrationId", "job_id", job.ID)
		return nil
	}
	slog.Info("transfer worker: running migration", "migration_id", migrationID)
	if err := w.svc.RunJob(ctx, migrationID); err != nil {
		slog.Error("transfer worker: migration failed", "migration_id", migrationID, "error", err)
		return err
	}
	return nil
}
