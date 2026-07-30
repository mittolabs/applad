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

type Deletes struct {
	cfg   *config.Config
	queue *queue.Queue
	db    *db.DB
}

func NewDeletes(cfg *config.Config) *Deletes {
	return &Deletes{cfg: cfg}
}

func (w *Deletes) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database
	StartRedisHeartbeat(ctx, rdb, "deletes")
	w.queue.StartReaper(ctx, "deletes")

	slog.Info("deletes worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "deletes")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("deletes worker: shutting down")
				return nil
			}
			slog.Error("deletes worker: pop error", "error", err)
			continue
		}
		if receipt == nil {
			continue
		}
		Heartbeat()
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("deletes", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("deletes", "completed")
			receipt.Ack()
		}
	}
}

func (w *Deletes) process(ctx context.Context, job *queue.Job) error {
	resourceType, _ := job.Payload["resourceType"].(string)
	resourceID, _ := job.Payload["resourceId"].(string)
	projectID, _ := job.Payload["projectId"].(string)

	var err error
	switch resourceType {
	case "user":
		w.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ? AND project_id = ?", resourceID, projectID) //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM memberships WHERE user_id = ?", resourceID)                            //nolint:errcheck
		_, err = w.db.ExecContext(ctx, "DELETE FROM users WHERE id = ? AND project_id = ?", resourceID, projectID)
	case "project":
		w.db.ExecContext(ctx, "DELETE FROM workflow_executions WHERE project_id = ?", projectID)                                                                                                                       //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM workflows WHERE project_id = ?", projectID)                                                                                                                                 //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM sessions WHERE project_id = ?", projectID)                                                                                                                                  //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM users WHERE project_id = ?", projectID)                                                                                                                                     //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM table_relationships WHERE table_id IN (SELECT id FROM tables WHERE project_id = ?) OR related_table IN (SELECT id FROM tables WHERE project_id = ?)", projectID, projectID) //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM indexes WHERE table_id IN (SELECT id FROM tables WHERE project_id = ?)", projectID)                                                                                         //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM columns WHERE table_id IN (SELECT id FROM tables WHERE project_id = ?)", projectID)                                                                                         //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM tables WHERE project_id = ?", projectID)                                                                                                                                    //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM databases WHERE project_id = ?", projectID)                                                                                                                                 //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM files WHERE project_id = ?", projectID)                                                                                                                                     //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM buckets WHERE project_id = ?", projectID)                                                                                                                                   //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM teams WHERE project_id = ?", projectID)                                                                                                                                     //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM api_keys WHERE project_id = ?", projectID)                                                                                                                                  //nolint:errcheck
		_, err = w.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", projectID)
	case "database":
		dbID := resourceID
		w.db.ExecContext(ctx, "DELETE FROM table_relationships WHERE table_id IN (SELECT id FROM tables WHERE database_id = ? AND project_id = ?) OR related_table IN (SELECT id FROM tables WHERE database_id = ? AND project_id = ?)", dbID, projectID, dbID, projectID) //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM indexes WHERE table_id IN (SELECT id FROM tables WHERE database_id = ? AND project_id = ?)", dbID, projectID)                                                                                                                   //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM columns WHERE table_id IN (SELECT id FROM tables WHERE database_id = ? AND project_id = ?)", dbID, projectID)                                                                                                                   //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM tables WHERE database_id = ? AND project_id = ?", dbID, projectID)                                                                                                                                                              //nolint:errcheck
		_, err = w.db.ExecContext(ctx, "DELETE FROM databases WHERE id = ? AND project_id = ?", dbID, projectID)
	case "table":
		tableID := resourceID
		w.db.ExecContext(ctx, "DELETE FROM table_relationships WHERE table_id = ? OR related_table = ?", tableID, tableID) //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM indexes WHERE table_id = ?", tableID)                                           //nolint:errcheck
		w.db.ExecContext(ctx, "DELETE FROM columns WHERE table_id = ?", tableID)                                           //nolint:errcheck
		_, err = w.db.ExecContext(ctx, "DELETE FROM tables WHERE id = ?", tableID)
	case "bucket":
		bucketID := resourceID
		w.db.ExecContext(ctx, "DELETE FROM files WHERE bucket_id = ? AND project_id = ?", bucketID, projectID) //nolint:errcheck
		_, err = w.db.ExecContext(ctx, "DELETE FROM buckets WHERE id = ? AND project_id = ?", bucketID, projectID)
	default:
		slog.Warn("deletes worker: unknown resource type", "type", resourceType)
		return nil
	}

	if err != nil {
		metrics.DBErrors.Inc()
		slog.Error("deletes worker: job failed", "job_id", job.ID, "error", err)
		return err
	}
	slog.Info("deletes worker: completed", "job_id", job.ID, "type", resourceType)
	return nil
}
