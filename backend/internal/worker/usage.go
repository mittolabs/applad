package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Usage aggregates per-project usage statistics: row counts,
// file counts, storage size, user counts, and workflow executions.
type Usage struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
	db    *db.DB
	rdb   *redis.Client
}

func NewUsage(cfg *config.Config) *Usage {
	return &Usage{cfg: cfg, stop: make(chan struct{})}
}

func (w *Usage) Start() error {
	w.rdb = redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(w.rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	w.db = database

	log.Println("usage worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "usage")
			if err != nil {
				log.Printf("usage worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Usage) process(ctx context.Context, job *queue.Job) {
	log.Printf("usage worker: processing job %s type=%s", job.ID, job.Type)

	switch job.Type {
	case "aggregate_project":
		w.aggregateProject(ctx, job)
	case "aggregate_all":
		w.aggregateAll(ctx)
	default:
		log.Printf("usage worker: unknown job type %q", job.Type)
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

func (w *Usage) aggregateProject(ctx context.Context, job *queue.Job) {
	projectID, _ := job.Payload["projectId"].(string)
	if projectID == "" {
		return
	}

	stats := w.collectStats(ctx, projectID)

	data, _ := json.Marshal(stats)
	w.rdb.Set(ctx, "applad:usage:"+projectID, data, 0)

	log.Printf("usage worker: aggregated stats for project %s — %d users, %d rows, %d files",
		projectID, stats.Users, stats.Rows, stats.Files)
}

func (w *Usage) aggregateAll(ctx context.Context) {
	rows, err := w.db.QueryContext(ctx, "SELECT id FROM projects")
	if err != nil {
		log.Printf("usage worker: query projects error: %v", err)
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var projectID string
		rows.Scan(&projectID)
		stats := w.collectStats(ctx, projectID)
		data, _ := json.Marshal(stats)
		w.rdb.Set(ctx, "applad:usage:"+projectID, data, 0)
		count++
	}
	log.Printf("usage worker: aggregated stats for %d projects", count)
}

func (w *Usage) collectStats(ctx context.Context, projectID string) projectStats {
	s := projectStats{ProjectID: projectID}
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE project_id = ?", projectID).Scan(&s.Users)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE project_id = ?", projectID).Scan(&s.Sessions)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM databases WHERE project_id = ?", projectID).Scan(&s.Databases)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tables WHERE project_id = ?", projectID).Scan(&s.Tables)
	// Row counts live in per-schema tables; set 0 until Phase 4 schema-per-database is implemented
	s.Rows = 0
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM buckets WHERE project_id = ?", projectID).Scan(&s.Buckets)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE project_id = ?", projectID).Scan(&s.Files)
	w.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(size), 0) FROM files WHERE project_id = ?", projectID).Scan(&s.StorageBytes)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams WHERE project_id = ?", projectID).Scan(&s.Teams)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflows WHERE project_id = ?", projectID).Scan(&s.Workflows)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflow_executions WHERE project_id = ?", projectID).Scan(&s.Executions)
	w.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments WHERE project_id = ?", projectID).Scan(&s.Deployments)
	return s
}

func (w *Usage) Stop() { close(w.stop) }
