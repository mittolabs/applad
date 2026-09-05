package worker

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/workflows"
	"github.com/redis/go-redis/v9"
)

// defaultExecutionTimeout bounds a single workflow run. A workflow can chain
// external calls, waits and sub-workflows; without a ceiling one stuck run pins
// the worker indefinitely. Overridable via WORKFLOW_EXECUTION_TIMEOUT.
const defaultExecutionTimeout = 5 * time.Minute

func executionTimeout() time.Duration {
	if v := os.Getenv("WORKFLOW_EXECUTION_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return defaultExecutionTimeout
}

type Executions struct {
	cfg   *config.Config
	queue *queue.Queue
	db    *db.DB
}

func NewExecutions(cfg *config.Config) *Executions {
	return &Executions{cfg: cfg}
}

func (w *Executions) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database
	StartRedisHeartbeat(ctx, rdb, "executions")
	// Touch the file heartbeat before the first tick so the compose
	// healthcheck does not see the worker as hung during start-up.
	Heartbeat()
	w.queue.StartReaper(ctx, "executions")

	slog.Info("executions worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "executions")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("executions worker: shutting down")
				return nil
			}
			slog.Error("executions worker: pop error", "error", err)
			continue
		}
		Heartbeat()

		if receipt == nil {
			continue
		}
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("executions", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("executions", "completed")
			receipt.Ack()
		}
	}
}

func (w *Executions) process(ctx context.Context, job *queue.Job) error {
	executionID, _ := job.Payload["executionId"].(string)
	workflowID, _ := job.Payload["workflowId"].(string)

	if executionID == "" || workflowID == "" {
		slog.Warn("executions worker: job missing ids", "job_id", job.ID)
		return nil
	}

	triggerData, _ := job.Payload["triggerData"].(map[string]interface{})
	if triggerData == nil {
		triggerData = map[string]interface{}{}
	}

	svc := workflows.NewService(w.db, nil)

	wf, err := svc.GetByID(ctx, workflowID)
	if err != nil {
		slog.Error("executions worker: workflow not found", "workflow_id", workflowID, "error", err)
		now := time.Now().UTC()
		svc.UpdateExecution(ctx, executionID, "failed", &now, &now, 0, "workflow not found", nil) //nolint:errcheck
		return err
	}

	startedAt := time.Now().UTC()
	svc.UpdateExecution(ctx, executionID, "running", &startedAt, nil, 0, "", nil) //nolint:errcheck

	runCtx, cancel := context.WithTimeout(ctx, executionTimeout())
	defer cancel()
	logs, execErr := workflows.RunWorkflow(runCtx, wf, triggerData)

	completedAt := time.Now().UTC()
	durationMs := completedAt.Sub(startedAt).Milliseconds()

	status := "completed"
	errMsg := ""
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
	}

	if err := svc.UpdateExecution(ctx, executionID, status, &startedAt, &completedAt, durationMs, errMsg, logs); err != nil {
		slog.Error("executions worker: update execution failed", "execution_id", executionID, "error", err)
		metrics.DBErrors.Inc()
		return err
	}

	slog.Info("executions worker: completed", "job_id", job.ID, "status", status, "duration_ms", durationMs)
	return nil
}
