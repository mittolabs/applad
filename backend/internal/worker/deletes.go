package worker

import (
	"context"
	"log"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

type Deletes struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
	db    *db.DB
}

func NewDeletes(cfg *config.Config) *Deletes {
	return &Deletes{cfg: cfg, stop: make(chan struct{})}
}

func (w *Deletes) Start() error {
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	// Connect to DB
	database, err := db.Connect(w.cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	w.db = database

	log.Println("deletes worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "deletes")
			if err != nil {
				log.Printf("deletes worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue // timeout, loop again
			}
			w.process(ctx, job)
		}
	}
}

func (w *Deletes) process(ctx context.Context, job *queue.Job) {
	log.Printf("deletes worker: processing job %s type=%s", job.ID, job.Type)

	resourceType, _ := job.Payload["resourceType"].(string)
	resourceID, _ := job.Payload["resourceId"].(string)
	projectID, _ := job.Payload["projectId"].(string)

	var err error
	switch resourceType {
	case "user":
		// Delete user sessions first, then user
		w.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ? AND project_id = ?", resourceID, projectID)
		w.db.ExecContext(ctx, "DELETE FROM memberships WHERE user_id = ?", resourceID)
		_, err = w.db.ExecContext(ctx, "DELETE FROM users WHERE id = ? AND project_id = ?", resourceID, projectID)
	case "project":
		// Cascade delete all project data
		w.db.ExecContext(ctx, "DELETE FROM workflow_executions WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM workflows WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM sessions WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM users WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM documents WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM collections WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM _databases WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM files WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM buckets WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM teams WHERE project_id = ?", projectID)
		w.db.ExecContext(ctx, "DELETE FROM api_keys WHERE project_id = ?", projectID)
		_, err = w.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", projectID)
	case "database":
		dbID := resourceID
		w.db.ExecContext(ctx, "DELETE FROM documents WHERE database_id = ? AND project_id = ?", dbID, projectID)
		w.db.ExecContext(ctx, "DELETE FROM collections WHERE database_id = ? AND project_id = ?", dbID, projectID)
		_, err = w.db.ExecContext(ctx, "DELETE FROM _databases WHERE id = ? AND project_id = ?", dbID, projectID)
	case "collection":
		collID := resourceID
		w.db.ExecContext(ctx, "DELETE FROM documents WHERE collection_id = ?", collID)
		w.db.ExecContext(ctx, "DELETE FROM attributes WHERE collection_id = ?", collID)
		w.db.ExecContext(ctx, "DELETE FROM _indexes WHERE collection_id = ?", collID)
		_, err = w.db.ExecContext(ctx, "DELETE FROM collections WHERE id = ?", collID)
	case "bucket":
		bucketID := resourceID
		w.db.ExecContext(ctx, "DELETE FROM files WHERE bucket_id = ? AND project_id = ?", bucketID, projectID)
		_, err = w.db.ExecContext(ctx, "DELETE FROM buckets WHERE id = ? AND project_id = ?", bucketID, projectID)
	default:
		log.Printf("deletes worker: unknown resource type %q", resourceType)
		return
	}

	if err != nil {
		log.Printf("deletes worker: error processing job %s: %v", job.ID, err)
	} else {
		log.Printf("deletes worker: completed job %s", job.ID)
	}
}

func (w *Deletes) Stop() { close(w.stop) }
