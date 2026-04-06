package worker

import (
	"context"
	"log"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/workflows"
	"github.com/redis/go-redis/v9"
)

type Executions struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
	db    *db.DB
}

func NewExecutions(cfg *config.Config) *Executions {
	return &Executions{cfg: cfg, stop: make(chan struct{})}
}

func (w *Executions) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	w.db = database

	log.Println("executions worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "executions")
			if err != nil {
				log.Printf("executions worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Executions) process(ctx context.Context, job *queue.Job) {
	log.Printf("executions worker: processing job %s", job.ID)

	executionID, _ := job.Payload["executionId"].(string)
	workflowID, _ := job.Payload["workflowId"].(string)

	if executionID == "" || workflowID == "" {
		log.Printf("executions worker: job %s missing executionId or workflowId", job.ID)
		return
	}

	triggerData, _ := job.Payload["triggerData"].(map[string]interface{})
	if triggerData == nil {
		triggerData = map[string]interface{}{}
	}

	svc := workflows.NewService(w.db, nil)

	// Load workflow
	wf, err := svc.GetByID(ctx, workflowID)
	if err != nil {
		log.Printf("executions worker: workflow %s not found: %v", workflowID, err)
		now := time.Now().UTC()
		svc.UpdateExecution(ctx, executionID, "failed", &now, &now, 0, "workflow not found", nil)
		return
	}

	// Mark as running
	startedAt := time.Now().UTC()
	svc.UpdateExecution(ctx, executionID, "running", &startedAt, nil, 0, "", nil)

	// Execute the workflow
	logs, execErr := workflows.RunWorkflow(ctx, wf, triggerData)

	completedAt := time.Now().UTC()
	durationMs := completedAt.Sub(startedAt).Milliseconds()

	status := "completed"
	errMsg := ""
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
	}

	if err := svc.UpdateExecution(ctx, executionID, status, &startedAt, &completedAt, durationMs, errMsg, logs); err != nil {
		log.Printf("executions worker: failed to update execution %s: %v", executionID, err)
		return
	}

	log.Printf("executions worker: completed job %s (status=%s, duration=%dms)", job.ID, status, durationMs)
}

func (w *Executions) Stop() { close(w.stop) }
