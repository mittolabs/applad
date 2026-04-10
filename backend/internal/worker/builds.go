package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/runtime"
	"github.com/redis/go-redis/v9"
)

// Builds processes deployment build jobs and function execution jobs.
type Builds struct {
	cfg            *config.Config
	queue          *queue.Queue
	db             *db.DB
	executor       *runtime.Executor
	deployExecutor *runtime.DeployExecutor
}

func NewBuilds(cfg *config.Config) *Builds {
	return &Builds{cfg: cfg}
}

func (w *Builds) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database

	w.executor = runtime.NewExecutor()
	w.deployExecutor = runtime.NewDeployExecutor()

	w.queue.StartReaper(ctx, "builds")

	slog.Info("builds worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "builds")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("builds worker: shutting down")
				return nil
			}
			slog.Error("builds worker: pop error", "error", err)
			continue
		}
		if receipt == nil {
			continue
		}
		Heartbeat()
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			slog.Error("builds worker: job failed", "job_id", receipt.Job.ID, "error", processErr)
			metrics.QueueJobs.Inc("builds", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("builds", "completed")
			receipt.Ack()
		}
	}
}

func (w *Builds) process(ctx context.Context, job *queue.Job) error {
	slog.Info("builds worker: processing job", "job_id", job.ID, "type", job.Type)
	switch job.Type {
	case "function_execution":
		return w.processFunctionExecution(ctx, job)
	case "function_build":
		return w.processFunctionBuild(ctx, job)
	default:
		return w.processDeployment(ctx, job)
	}
}

func (w *Builds) processFunctionExecution(ctx context.Context, job *queue.Job) error {
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

	w.updateExecution(ctx, executionID, "processing", "", "", 0)

	req := runtime.ExecRequest{
		FunctionID: functionID, ProjectID: projectID,
		Runtime: runtimeName, Entrypoint: entrypoint, Source: source, Timeout: timeout,
	}
	if _, err := w.executor.Build(ctx, req); err != nil {
		w.updateExecution(ctx, executionID, "failed", "", err.Error(), 0)
		return err
	}
	result, err := w.executor.Execute(ctx, req)
	if err != nil {
		w.updateExecution(ctx, executionID, "failed", "", err.Error(), 0)
		return err
	}
	status := "completed"
	if result.ExitCode != 0 {
		status = "failed"
	}
	w.updateExecution(ctx, executionID, status, result.Output, result.Errors, result.Duration)
	slog.Info("builds worker: function execution done",
		"function_id", functionID, "status", status, "duration_s", result.Duration)
	return nil
}

func (w *Builds) processFunctionBuild(ctx context.Context, job *queue.Job) error {
	functionID, _ := job.Payload["functionId"].(string)
	runtimeName, _ := job.Payload["runtime"].(string)
	entrypoint, _ := job.Payload["entrypoint"].(string)
	source, _ := job.Payload["source"].(string)
	dockerfile, _ := job.Payload["dockerfile"].(string)

	req := runtime.ExecRequest{FunctionID: functionID, Runtime: runtimeName,
		Entrypoint: entrypoint, Source: source, Dockerfile: dockerfile}
	if _, err := w.executor.Build(ctx, req); err != nil {
		w.db.ExecContext(ctx, "UPDATE functions SET status = 'failed' WHERE id = ?", functionID) //nolint:errcheck
		return err
	}
	warmReq := runtime.ExecRequest{FunctionID: functionID, Runtime: runtimeName,
		Entrypoint: entrypoint, Source: source, Timeout: 30}
	if _, err := w.executor.Execute(ctx, warmReq); err != nil {
		slog.Warn("builds worker: pre-warm failed (non-fatal)", "function_id", functionID, "error", err)
	}
	w.db.ExecContext(ctx, "UPDATE functions SET status = 'active' WHERE id = ?", functionID) //nolint:errcheck
	slog.Info("builds worker: function ready", "function_id", functionID)
	return nil
}

func (w *Builds) updateExecution(ctx context.Context, id, status, output, errors string, duration float64) {
	if _, err := w.db.ExecContext(ctx,
		"UPDATE function_executions SET status = ?, output = ?, errors = ?, duration = ? WHERE id = ?",
		status, output, errors, duration, id); err != nil {
		slog.Error("builds worker: update execution failed", "execution_id", id, "error", err)
		metrics.DBErrors.Inc()
	}
}

func (w *Builds) processDeployment(ctx context.Context, job *queue.Job) error {
	deploymentID, _ := job.Payload["deploymentId"].(string)
	projectID, _ := job.Payload["projectId"].(string)
	if deploymentID == "" || projectID == "" {
		slog.Warn("builds worker: job missing ids", "job_id", job.ID)
		return nil
	}
	deployType, deployCfg := w.loadDeploymentConfig(ctx, deploymentID, projectID)
	w.updateDeployStatus(ctx, deploymentID, projectID, "building")
	cfg := runtime.ParseDeployConfig(deployCfg)
	var deployErr error
	switch deployType {
	case "web":
		w.updateDeployStatus(ctx, deploymentID, projectID, "deploying")
		deployErr = w.deployExecutor.DeployWeb(ctx, deploymentID, projectID, cfg)
	case "container":
		w.updateDeployStatus(ctx, deploymentID, projectID, "deploying")
		deployErr = w.deployExecutor.DeployContainer(ctx, deploymentID, projectID, cfg)
	case "function":
		runtimeName, _ := deployCfg["runtime"].(string)
		entrypoint, _ := deployCfg["entrypoint"].(string)
		source, _ := deployCfg["source"].(string)
		dockerfile, _ := deployCfg["dockerfile"].(string)
		req := runtime.ExecRequest{FunctionID: deploymentID, ProjectID: projectID,
			Runtime: runtimeName, Entrypoint: entrypoint, Source: source, Dockerfile: dockerfile}
		w.updateDeployStatus(ctx, deploymentID, projectID, "deploying")
		_, deployErr = w.executor.Build(ctx, req)
	default:
		w.updateDeployStatus(ctx, deploymentID, projectID, "deploying")
		deployErr = w.deployExecutor.DeployWeb(ctx, deploymentID, projectID, cfg)
	}
	if deployErr != nil {
		w.updateDeployStatusWithError(ctx, deploymentID, projectID, "failed", deployErr.Error())
		return deployErr
	}
	w.updateDeployStatus(ctx, deploymentID, projectID, "active")
	slog.Info("builds worker: deployment complete", "deployment_id", deploymentID)
	return nil
}

func (w *Builds) loadDeploymentConfig(ctx context.Context, deploymentID, projectID string) (string, map[string]interface{}) {
	var deployType string
	var cfgJSON []byte
	if err := w.db.QueryRowContext(ctx,
		"SELECT type, config FROM deployments WHERE id = ? AND project_id = ?",
		deploymentID, projectID).Scan(&deployType, &cfgJSON); err != nil {
		slog.Error("builds worker: load config failed", "deployment_id", deploymentID, "error", err)
		metrics.DBErrors.Inc()
		return "web", map[string]interface{}{}
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
		cfg = map[string]interface{}{}
	}
	return deployType, cfg
}

func (w *Builds) updateDeployStatus(ctx context.Context, id, projectID, status string) {
	if _, err := w.db.ExecContext(ctx,
		"UPDATE deployments SET status = ?, updated_at = ? WHERE id = ? AND project_id = ?",
		status, time.Now().UTC(), id, projectID); err != nil {
		slog.Error("builds worker: update status failed", "deployment_id", id, "error", err)
		metrics.DBErrors.Inc()
	}
}

func (w *Builds) updateDeployStatusWithError(ctx context.Context, id, projectID, status, errMsg string) {
	var cfgJSON []byte
	w.db.QueryRowContext(ctx, "SELECT config FROM deployments WHERE id = ? AND project_id = ?", id, projectID).Scan(&cfgJSON) //nolint:errcheck
	var cfg map[string]interface{}
	if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
		cfg = map[string]interface{}{}
	}
	cfg["error"] = errMsg
	updatedCfg, _ := json.Marshal(cfg)
	if _, err := w.db.ExecContext(ctx,
		"UPDATE deployments SET status = ?, config = ?, updated_at = ? WHERE id = ? AND project_id = ?",
		status, updatedCfg, time.Now().UTC(), id, projectID); err != nil {
		slog.Error("builds worker: update error status failed", "deployment_id", id, "error", err)
		metrics.DBErrors.Inc()
	}
}
