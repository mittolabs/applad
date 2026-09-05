package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Usage aggregates per-project usage statistics.
type Usage struct {
	cfg   *config.Config
	queue *queue.Queue
	db    *db.DB
	rdb   *redis.Client
}

func NewUsage(cfg *config.Config) *Usage {
	return &Usage{cfg: cfg}
}

func (w *Usage) Start(ctx context.Context) error {
	w.rdb = redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(w.rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database
	StartRedisHeartbeat(ctx, w.rdb, "usage")
	// Touch the file heartbeat before the first tick so the compose
	// healthcheck does not see the worker as hung during start-up.
	Heartbeat()
	w.queue.StartReaper(ctx, "usage")

	slog.Info("usage worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "usage")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("usage worker: shutting down")
				return nil
			}
			slog.Error("usage worker: pop error", "error", err)
			continue
		}
		Heartbeat()

		if receipt == nil {
			continue
		}
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("usage", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("usage", "completed")
			receipt.Ack()
		}
	}
}

func (w *Usage) process(ctx context.Context, job *queue.Job) error {
	switch job.Type {
	case "aggregate_project":
		return w.aggregateProject(ctx, job)
	case "aggregate_all":
		return w.aggregateAll(ctx)
	default:
		slog.Warn("usage worker: unknown job type", "type", job.Type)
		return nil
	}
}

type projectStats struct {
	ProjectID    string `json:"projectId"`
	Users        int    `json:"users"`
	Sessions     int    `json:"sessions"`
	Databases    int    `json:"databases"`
	Tables       int    `json:"tables"`
	Rows         int    `json:"rows"`
	Buckets      int    `json:"buckets"`
	Files        int    `json:"files"`
	StorageBytes int64  `json:"storageBytes"`
	Teams        int    `json:"teams"`
	Workflows    int    `json:"workflows"`
	Executions   int    `json:"executions"`
	Deployments  int    `json:"deployments"`
}

func (w *Usage) aggregateProject(ctx context.Context, job *queue.Job) error {
	projectID, _ := job.Payload["projectId"].(string)
	if projectID == "" {
		return nil
	}
	stats := w.collectStats(ctx, projectID)
	data, _ := json.Marshal(stats)
	w.rdb.Set(ctx, "applad:usage:"+projectID, data, 0) //nolint:errcheck
	slog.Info("usage worker: aggregated project", "project_id", projectID,
		"users", stats.Users, "rows", stats.Rows, "files", stats.Files)
	return nil
}

func (w *Usage) aggregateAll(ctx context.Context) error {
	rows, err := w.db.QueryContext(ctx, "SELECT id FROM projects")
	if err != nil {
		slog.Error("usage worker: query projects failed", "error", err)
		metrics.DBErrors.Inc()
		return err
	}
	defer rows.Close()
	var count int
	for rows.Next() {
		var projectID string
		rows.Scan(&projectID) //nolint:errcheck
		stats := w.collectStats(ctx, projectID)
		data, _ := json.Marshal(stats)
		w.rdb.Set(ctx, "applad:usage:"+projectID, data, 0) //nolint:errcheck
		count++
	}
	slog.Info("usage worker: aggregated all projects", "count", count)
	return nil
}

func (w *Usage) collectStats(ctx context.Context, projectID string) projectStats {
	s := projectStats{ProjectID: projectID}
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE project_id = ?", projectID).Scan(&s.Users)                      //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE project_id = ?", projectID).Scan(&s.Sessions)                //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM databases WHERE project_id = ?", projectID).Scan(&s.Databases)              //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tables WHERE project_id = ?", projectID).Scan(&s.Tables)                    //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM buckets WHERE project_id = ?", projectID).Scan(&s.Buckets)                  //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE project_id = ?", projectID).Scan(&s.Files)                      //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(size), 0) FROM files WHERE project_id = ?", projectID).Scan(&s.StorageBytes) //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams WHERE project_id = ?", projectID).Scan(&s.Teams)                      //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflows WHERE project_id = ?", projectID).Scan(&s.Workflows)              //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflow_executions WHERE project_id = ?", projectID).Scan(&s.Executions)   //nolint:errcheck
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments WHERE project_id = ?", projectID).Scan(&s.Deployments)          //nolint:errcheck
	return s
}
