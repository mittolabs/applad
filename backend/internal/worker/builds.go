package worker

import (
	"context"
	"log"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Builds processes deployment build jobs. When a deployment is created or
// triggered, this worker picks up the job, transitions the deployment through
// building → deploying → active (or failed), and logs progress.
type Builds struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
	db    *db.DB
}

func NewBuilds(cfg *config.Config) *Builds {
	return &Builds{cfg: cfg, stop: make(chan struct{})}
}

func (w *Builds) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	w.db = database

	log.Println("builds worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "builds")
			if err != nil {
				log.Printf("builds worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Builds) process(ctx context.Context, job *queue.Job) {
	log.Printf("builds worker: processing job %s", job.ID)

	deploymentID, _ := job.Payload["deploymentId"].(string)
	projectID, _ := job.Payload["projectId"].(string)

	if deploymentID == "" || projectID == "" {
		log.Printf("builds worker: job %s missing deploymentId or projectId", job.ID)
		return
	}

	// Transition to building
	w.updateDeployStatus(ctx, deploymentID, projectID, "building")
	log.Printf("builds worker: building deployment %s", deploymentID)

	// Simulate build step — in production this would run actual build commands
	// (docker build, npm build, etc.) based on deployment type and config
	time.Sleep(2 * time.Second)

	// Transition to deploying
	w.updateDeployStatus(ctx, deploymentID, projectID, "deploying")
	log.Printf("builds worker: deploying %s", deploymentID)

	time.Sleep(1 * time.Second)

	// Transition to active
	w.updateDeployStatus(ctx, deploymentID, projectID, "active")
	log.Printf("builds worker: completed job %s — deployment %s is active", job.ID, deploymentID)
}

func (w *Builds) updateDeployStatus(ctx context.Context, id, projectID, status string) {
	_, err := w.db.ExecContext(ctx,
		"UPDATE deployments SET status = ?, updated_at = ? WHERE id = ? AND project_id = ?",
		status, time.Now().UTC(), id, projectID)
	if err != nil {
		log.Printf("builds worker: failed to update status for %s: %v", id, err)
	}
}

func (w *Builds) Stop() { close(w.stop) }
