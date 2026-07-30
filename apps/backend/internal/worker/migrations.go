package worker

import (
	"context"
	"log/slog"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Migrations processes async migration and maintenance jobs.
type Migrations struct {
	cfg   *config.Config
	queue *queue.Queue
	db    *db.DB
}

func NewMigrations(cfg *config.Config) *Migrations {
	return &Migrations{cfg: cfg}
}

func (w *Migrations) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database
	StartRedisHeartbeat(ctx, rdb, "migrations")
	w.queue.StartReaper(ctx, "migrations")

	slog.Info("migrations worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "migrations")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("migrations worker: shutting down")
				return nil
			}
			slog.Error("migrations worker: pop error", "error", err)
			continue
		}
		if receipt == nil {
			continue
		}
		Heartbeat()
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("migrations", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("migrations", "completed")
			receipt.Ack()
		}
	}
}

func (w *Migrations) process(ctx context.Context, job *queue.Job) error {
	slog.Info("migrations worker: processing job", "job_id", job.ID, "type", job.Type)
	switch job.Type {
	case "run_migrations":
		if err := w.db.Migrate(); err != nil {
			slog.Error("migrations worker: migration failed", "error", err)
			metrics.DBErrors.Inc()
			return err
		}
		slog.Info("migrations worker: migrations completed")

	case "optimize_tables":
		for _, table := range []string{"files", "sessions", "users"} {
			if _, err := w.db.ExecContext(ctx, "VACUUM ANALYZE "+table); err != nil {
				slog.Warn("migrations worker: vacuum failed", "table", table, "error", err)
			}
		}

	case "cleanup_sessions":
		result, err := w.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < NOW()")
		if err != nil {
			slog.Error("migrations worker: session cleanup failed", "error", err)
			metrics.DBErrors.Inc()
			return err
		}
		affected, _ := result.RowsAffected()
		slog.Info("migrations worker: cleaned expired sessions", "count", affected)

	default:
		slog.Warn("migrations worker: unknown job type", "type", job.Type)
	}
	return nil
}
