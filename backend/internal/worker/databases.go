package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

type Databases struct {
	cfg   *config.Config
	queue *queue.Queue
	db    *db.DB
}

func NewDatabases(cfg *config.Config) *Databases {
	return &Databases{cfg: cfg}
}

func (w *Databases) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database
	w.queue.StartReaper(ctx, "databases")

	slog.Info("databases worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "databases")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("databases worker: shutting down")
				return nil
			}
			slog.Error("databases worker: pop error", "error", err)
			continue
		}
		if receipt == nil {
			continue
		}
		Heartbeat()
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("databases", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("databases", "completed")
			receipt.Ack()
		}
	}
}

func (w *Databases) process(ctx context.Context, job *queue.Job) error {
	slog.Info("databases worker: processing job", "job_id", job.ID, "type", job.Type)
	switch job.Type {
	case "attribute_create":
		return w.processAttributeCreate(ctx, job)
	case "index_create":
		return w.processIndexCreate(ctx, job)
	case "collection_stats":
		return w.processCollectionStats(ctx, job)
	default:
		slog.Warn("databases worker: unknown job type", "type", job.Type)
		return nil
	}
}

func (w *Databases) processAttributeCreate(ctx context.Context, job *queue.Job) error {
	attributeID, _ := job.Payload["attributeId"].(string)
	if attributeID == "" {
		return nil
	}
	w.db.ExecContext(ctx, "UPDATE columns SET status = 'processing' WHERE id = ?", attributeID) //nolint:errcheck
	w.db.ExecContext(ctx, "UPDATE columns SET status = 'available' WHERE id = ?", attributeID) //nolint:errcheck
	slog.Info("databases worker: column available", "column_id", attributeID)
	return nil
}

func (w *Databases) processIndexCreate(ctx context.Context, job *queue.Job) error {
	indexID, _ := job.Payload["indexId"].(string)
	if indexID == "" {
		return nil
	}
	w.db.ExecContext(ctx, "UPDATE indexes SET status = 'building' WHERE id = ?", indexID) //nolint:errcheck
	time.Sleep(500 * time.Millisecond)
	w.db.ExecContext(ctx, "UPDATE indexes SET status = 'available' WHERE id = ?", indexID) //nolint:errcheck
	slog.Info("databases worker: index available", "index_id", indexID)
	return nil
}

func (w *Databases) processCollectionStats(ctx context.Context, job *queue.Job) error {
	tableID, _ := job.Payload["tableId"].(string)
	if tableID == "" {
		return nil
	}
	var count int
	if err := w.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tables WHERE id = ?", tableID).Scan(&count); err != nil {
		metrics.DBErrors.Inc()
		return err
	}
	return nil
}
