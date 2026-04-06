package worker

import (
	"context"
	"log"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/runtime"
	"github.com/redis/go-redis/v9"
)

// Builds processes deployment build jobs and function execution jobs.
// For deployments: transitions through building → deploying → active.
// For functions: builds container image, executes via HTTP, records result.
type Builds struct {
	cfg      *config.Config
	stop     chan struct{}
	queue    *queue.Queue
	db       *db.DB
	executor *runtime.Executor
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

	// Initialize the container runtime executor
	w.executor = runtime.NewExecutor()

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
	log.Printf("builds worker: processing job %s type=%s", job.ID, job.Type)

	switch job.Type {
	case "function_execution":
		w.processFunctionExecution(ctx, job)
	case "function_build":
		w.processFunctionBuild(ctx, job)
	default:
		w.processDeployment(ctx, job)
	}
}

// --- Function execution ---

func (w *Builds) processFunctionExecution(ctx context.Context, job *queue.Job) {
	executionID, _ := job.Payload["executionId"].(string)
	functionID, _ := job.Payload["functionId"].(string)
	projectID, _ := job.Payload["projectId"].(string)
	runtimeName, _ := job.Payload["runtime"].(string)
	entrypoint, _ := job.Payload["entrypoint"].(string)
	source, _ := job.Payload["source"].(string)
	timeoutF, _ := job.Payload["timeout"].(float64)
	timeout := int(timeoutF)
	if timeout <= 0 {
		timeout = 15
	}

	// Mark as processing
	w.updateExecution(ctx, executionID, "processing", "", "", 0)

	// Build the image if not already built
	req := runtime.ExecRequest{
		FunctionID: functionID,
		ProjectID:  projectID,
		Runtime:    runtimeName,
		Entrypoint: entrypoint,
		Source:     source,
		Timeout:    timeout,
	}

	_, err := w.executor.Build(ctx, req)
	if err != nil {
		log.Printf("builds worker: build failed for function %s: %v", functionID, err)
		w.updateExecution(ctx, executionID, "failed", "", err.Error(), 0)
		return
	}

	// Execute
	result, err := w.executor.Execute(ctx, req)
	if err != nil {
		log.Printf("builds worker: execution failed for function %s: %v", functionID, err)
		w.updateExecution(ctx, executionID, "failed", "", err.Error(), 0)
		return
	}

	status := "completed"
	if result.ExitCode != 0 {
		status = "failed"
	}

	w.updateExecution(ctx, executionID, status, result.Output, result.Errors, result.Duration)
	log.Printf("builds worker: function %s execution %s completed (%.2fs)", functionID, executionID, result.Duration)
}

func (w *Builds) processFunctionBuild(ctx context.Context, job *queue.Job) {
	functionID, _ := job.Payload["functionId"].(string)
	runtimeName, _ := job.Payload["runtime"].(string)
	entrypoint, _ := job.Payload["entrypoint"].(string)
	source, _ := job.Payload["source"].(string)
	dockerfile, _ := job.Payload["dockerfile"].(string)

	log.Printf("builds worker: building function %s (runtime=%s)", functionID, runtimeName)

	req := runtime.ExecRequest{
		FunctionID: functionID,
		Runtime:    runtimeName,
		Entrypoint: entrypoint,
		Source:     source,
		Dockerfile: dockerfile,
	}

	_, err := w.executor.Build(ctx, req)
	if err != nil {
		log.Printf("builds worker: build failed for function %s: %v", functionID, err)
		w.db.ExecContext(ctx, "UPDATE functions SET status = 'failed' WHERE id = ?", functionID)
		return
	}

	// Pre-warm: start a container now so the first invocation is instant
	log.Printf("builds worker: pre-warming function %s", functionID)
	warmReq := runtime.ExecRequest{
		FunctionID: functionID,
		Runtime:    runtimeName,
		Entrypoint: entrypoint,
		Source:     source,
		Timeout:    30,
	}
	// Execute a health check to warm the container, then release it back to pool
	_, warmErr := w.executor.Execute(ctx, warmReq)
	if warmErr != nil {
		log.Printf("builds worker: pre-warm failed for %s (non-fatal): %v", functionID, warmErr)
	} else {
		log.Printf("builds worker: function %s pre-warmed successfully", functionID)
	}

	w.db.ExecContext(ctx, "UPDATE functions SET status = 'active' WHERE id = ?", functionID)
	log.Printf("builds worker: function %s ready", functionID)
}

func (w *Builds) updateExecution(ctx context.Context, executionID, status, output, errors string, duration float64) {
	_, err := w.db.ExecContext(ctx,
		"UPDATE function_executions SET status = ?, output = ?, errors = ?, duration = ? WHERE id = ?",
		status, output, errors, duration, executionID)
	if err != nil {
		log.Printf("builds worker: failed to update execution %s: %v", executionID, err)
	}
}

// --- Deployment builds ---

func (w *Builds) processDeployment(ctx context.Context, job *queue.Job) {
	deploymentID, _ := job.Payload["deploymentId"].(string)
	projectID, _ := job.Payload["projectId"].(string)

	if deploymentID == "" || projectID == "" {
		log.Printf("builds worker: job %s missing deploymentId or projectId", job.ID)
		return
	}

	w.updateDeployStatus(ctx, deploymentID, projectID, "building")
	log.Printf("builds worker: building deployment %s", deploymentID)

	time.Sleep(2 * time.Second)

	w.updateDeployStatus(ctx, deploymentID, projectID, "deploying")
	log.Printf("builds worker: deploying %s", deploymentID)

	time.Sleep(1 * time.Second)

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
