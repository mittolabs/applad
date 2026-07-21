package worker

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/runtime"
	"github.com/mittolabs/applad/internal/testlab"
	"github.com/redis/go-redis/v9"
)

// Builds processes deployment build jobs and function execution jobs.
type Builds struct {
	cfg            *config.Config
	queue          *queue.Queue
	rdb            *redis.Client
	db             *db.DB
	executor       *runtime.Executor
	deployExecutor *runtime.DeployExecutor
}

func NewBuilds(cfg *config.Config) *Builds {
	return &Builds{cfg: cfg}
}

func (w *Builds) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.rdb = rdb
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
	case "deploy_release":
		return w.processRelease(ctx, job)
	case "deploy_rollback":
		return w.processRollback(ctx, job)
	case "deploy_teardown":
		return w.processTeardown(ctx, job)
	case "test_run":
		return w.processTestRun(ctx, job)
	case "studio_start":
		return w.processStudioStart(ctx, job)
	case "studio_stop":
		return w.processStudioStop(ctx, job)
	case "container_logs":
		return w.processContainerLogs(ctx, job)
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
	sourceType, _ := job.Payload["sourceType"].(string)
	source, _ := job.Payload["source"].(string)
	repository, _ := job.Payload["repository"].(string)
	branch, _ := job.Payload["branch"].(string)
	timeoutF, _ := job.Payload["timeout"].(float64)
	timeout := int(timeoutF)
	if timeout <= 0 {
		timeout = 15
	}

	var sourceDir string
	// For git source, clone the repo to a temp directory
	if sourceType == "git" && repository != "" {
		cloned, err := cloneToSource(ctx, repository, branch)
		if err != nil {
			w.updateExecution(ctx, executionID, "failed", "", err.Error(), 0)
			return err
		}
		sourceDir = cloned
		defer os.RemoveAll(sourceDir)
	}

	w.updateExecution(ctx, executionID, "processing", "", "", 0)

	req := runtime.ExecRequest{
		FunctionID: functionID, ProjectID: projectID,
		Runtime: runtimeName, Entrypoint: entrypoint, Source: source,
		SourceDir: sourceDir, Timeout: timeout,
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
	sourceType, _ := job.Payload["sourceType"].(string)
	source, _ := job.Payload["source"].(string)
	repository, _ := job.Payload["repository"].(string)
	branch, _ := job.Payload["branch"].(string)
	dockerfile, _ := job.Payload["dockerfile"].(string)

	var sourceDir string
	// For git source, clone the repo to a temp directory
	if sourceType == "git" && repository != "" {
		cloned, err := cloneToSource(ctx, repository, branch)
		if err != nil {
			w.db.ExecContext(ctx, "UPDATE functions SET status = 'failed' WHERE id = ?", functionID) //nolint:errcheck
			return err
		}
		sourceDir = cloned
		defer os.RemoveAll(sourceDir)
	}

	req := runtime.ExecRequest{FunctionID: functionID, Runtime: runtimeName,
		Entrypoint: entrypoint, Source: source, SourceDir: sourceDir, Dockerfile: dockerfile}
	if _, err := w.executor.Build(ctx, req); err != nil {
		w.db.ExecContext(ctx, "UPDATE functions SET status = 'failed' WHERE id = ?", functionID) //nolint:errcheck
		return err
	}
	warmReq := runtime.ExecRequest{FunctionID: functionID, Runtime: runtimeName,
		Entrypoint: entrypoint, Source: source, SourceDir: sourceDir, Timeout: 30}
	if _, err := w.executor.Execute(ctx, warmReq); err != nil {
		slog.Warn("builds worker: pre-warm failed (non-fatal)", "function_id", functionID, "error", err)
	}
	w.db.ExecContext(ctx, "UPDATE functions SET status = 'active' WHERE id = ?", functionID) //nolint:errcheck
	slog.Info("builds worker: function ready", "function_id", functionID)
	return nil
}

// cloneToSource shallow-clones a git repository and returns the cloned directory path.
// The caller is responsible for reading files from it; the directory is left on disk
// so the executor can tar it up. A temp dir is created under /tmp/applad-git-*.
func cloneToSource(ctx context.Context, repository, branch string) (string, error) {
	dir, err := os.MkdirTemp("", "applad-git-*")
	if err != nil {
		return "", fmt.Errorf("git clone: mktemp: %w", err)
	}
	if err := runtime.CloneRepo(ctx, repository, branch, dir); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// extractZip unpacks a .zip source archive into dir, refusing entries whose
// paths escape it.
func extractZip(archive, dir string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("uploaded source is not a readable zip: %w", err)
	}
	defer zr.Close()

	for _, entry := range zr.File {
		target := filepath.Join(dir, filepath.Clean("/"+entry.Name))
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(os.PathSeparator)) {
			continue
		}
		if entry.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755) //nolint:errcheck
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := entry.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc) //nolint:gosec
		out.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// processTestRun executes a project's own test suite and records what it
// found. A suite that fails is a normal outcome recorded as results; only the
// run itself failing to execute is an error.
func (w *Builds) processTestRun(ctx context.Context, job *queue.Job) error {
	runID, _ := job.Payload["runId"].(string)
	runnerID, _ := job.Payload["runnerId"].(string)
	projectID, _ := job.Payload["projectId"].(string)
	target, _ := job.Payload["target"].(string)
	grep, _ := job.Payload["grep"].(string)
	start := time.Now()

	svc := testlab.NewService(w.db, w.queue)
	runner, err := svc.GetRunner(ctx, runnerID, projectID)
	if err != nil {
		svc.RecordResults(ctx, runID, projectID, runnerID, nil, "", "runner no longer exists", 0) //nolint:errcheck
		return err
	}
	svc.MarkRunning(ctx, runID)

	sourceDir, err := w.testSource(ctx, runner)
	if err != nil {
		svc.RecordResults(ctx, runID, projectID, runnerID, nil, "", err.Error(), time.Since(start).Milliseconds()) //nolint:errcheck
		return err
	}
	defer os.RemoveAll(sourceDir)

	// The target belongs to the run, not the runner, so testing a branch and
	// testing main is the same configuration pointed somewhere else.
	env := map[string]string{}
	for k, v := range runner.EnvVars {
		env[k] = v
	}
	if target != "" {
		env["BASE_URL"] = target
	}

	command := runner.Command
	if grep != "" {
		// A selection is passed to the runner as a name filter rather than by
		// rewriting the project, which keeps the runner ignorant of selections.
		command += " --grep " + shellQuote(grep)
	}

	// Published as it happens, so the console can show a run unfolding rather
	// than a spinner. Redis is the carrier because the API, which serves the
	// console, cannot see this container.
	onLine := func(line string) {
		w.rdb.Publish(ctx, "applad:testrun:"+runID, line) //nolint:errcheck
	}

	result, runErr := w.deployExecutor.RunTests(ctx, runID, runtime.TestRunConfig{
		OnLine:        onLine,
		SourceDir:     sourceDir,
		Image:         runner.Image,
		SetupCmd:      runner.SetupCmd,
		Command:       command,
		ReportPath:    runner.ReportPath,
		ArtifactsPath: runner.ArtifactsPath,
		Env:           env,
		TimeoutMs:     runner.TimeoutMs,
	})
	durationMs := time.Since(start).Milliseconds()

	if runErr != nil {
		logOut := ""
		if result != nil {
			logOut = result.Log
		}
		svc.RecordResults(ctx, runID, projectID, runnerID, nil, logOut, runErr.Error(), durationMs) //nolint:errcheck
		return runErr
	}

	cases, parseErr := testlab.ParseJUnit(bytes.NewReader(result.Report))
	if parseErr != nil {
		msg := fmt.Sprintf("no test report at %s: %v", runner.ReportPath, parseErr)
		if result.ExitCode != 0 {
			msg = fmt.Sprintf("tests exited %d and left no report at %s", result.ExitCode, runner.ReportPath)
		}
		svc.RecordResults(ctx, runID, projectID, runnerID, nil, result.Log, msg, durationMs) //nolint:errcheck
		slog.Warn("builds worker: test run produced no report", "run_id", runID, "error", parseErr)
		return nil
	}

	// Attempts at the same test are one result, and one that failed then
	// passed is flaky rather than failing.
	cases = testlab.MergeRetries(cases)

	if err := svc.RecordResults(ctx, runID, projectID, runnerID, cases, result.Log, "", durationMs); err != nil {
		return err
	}
	if err := svc.StoreArtifacts(ctx, runID, projectID, result.Artifacts); err != nil {
		slog.Warn("builds worker: storing test artifacts failed", "run_id", runID, "error", err)
	}

	summary := testlab.Summarise(cases)
	slog.Info("builds worker: test run complete", "run_id", runID,
		"total", summary.Total, "failed", summary.Failed, "flaky", summary.Flaky,
		"duration_ms", durationMs)
	return nil
}

// shellQuote makes a value safe inside the single-quoted command the runner
// executes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// testSource materialises the project under test, from git or from an upload.
func (w *Builds) testSource(ctx context.Context, runner *testlab.Runner) (string, error) {
	if runner.SourceType == "git" && runner.SourceURL != "" {
		return cloneToSource(ctx, runner.SourceURL, runner.Branch)
	}
	return extractUploadedSource(runner.ID)
}

// processStudioStart opens a browser for a recording session.
//
// The API asks for it rather than doing it, because only this worker holds the
// Docker socket. Once it is up the API talks to it directly over the shared
// network, so the endpoint is all that needs handing back.
func (w *Builds) processStudioStart(ctx context.Context, job *queue.Job) error {
	sessionID, _ := job.Payload["sessionId"].(string)
	image, _ := job.Payload["image"].(string)
	if image == "" {
		image = runtime.StudioBrowserImage()
	}

	key := "applad:studio:" + sessionID
	_, wsURL, err := w.deployExecutor.StartBrowser(ctx, sessionID, image)
	if err != nil {
		// Reported rather than dropped, so the console says what went wrong
		// instead of waiting out the timeout.
		w.rdb.Set(ctx, key, "error:"+err.Error(), 5*time.Minute) //nolint:errcheck
		return err
	}

	// The session holds the browser open, so the key outlives a long recording
	// but not a forgotten one.
	w.rdb.Set(ctx, key, wsURL, 4*time.Hour) //nolint:errcheck
	slog.Info("builds worker: studio browser ready", "session_id", sessionID)
	return nil
}

func (w *Builds) processStudioStop(ctx context.Context, job *queue.Job) error {
	sessionID, _ := job.Payload["sessionId"].(string)
	name := "applad-studio-" + sessionID
	if err := w.deployExecutor.StopBrowserByName(ctx, name); err != nil {
		slog.Warn("builds worker: stopping studio browser", "session_id", sessionID, "error", err)
	}
	w.rdb.Del(ctx, "applad:studio:"+sessionID) //nolint:errcheck
	return nil
}

// processContainerLogs answers a request for a running container's output.
// The API cannot see containers, so it asks and reads the reply from Redis.
func (w *Builds) processContainerLogs(ctx context.Context, job *queue.Job) error {
	name, _ := job.Payload["container"].(string)
	replyKey, _ := job.Payload["replyKey"].(string)
	if name == "" || replyKey == "" {
		return nil
	}

	lines := []string{}
	if out, err := w.deployExecutor.ContainerLogsByName(ctx, name); err == nil {
		for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	// Tail rather than everything: a busy site would otherwise return a
	// megabyte the console cannot use.
	if len(lines) > 500 {
		lines = lines[len(lines)-500:]
	}

	data, _ := json.Marshal(lines)
	w.rdb.Set(ctx, replyKey, data, 30*time.Second) //nolint:errcheck
	return nil
}

// processTeardown removes everything a deleted deploy target left behind.
// Deleting the database row alone used to leave the app's container running
// and still served on its subdomain.
func (w *Builds) processTeardown(ctx context.Context, job *queue.Job) error {
	targetName, _ := job.Payload["targetName"].(string)
	domain, _ := job.Payload["domain"].(string)

	sub, _ := job.Payload["subdomain"].(string)
	if sub == "" {
		sub = deploy.Subdomain(domain)
	}
	if sub == "" {
		sub = deploy.Subdomain(targetName)
	}
	if sub != "" && w.deployExecutor != nil {
		if err := w.deployExecutor.StopByName(ctx, "applad-site-"+sub); err != nil {
			slog.Warn("teardown: stop site container", "subdomain", sub, "error", err)
		}
	}

	// Uploaded source archives live on the shared storage volume and are keyed
	// by pipeline; the cascaded row delete does not touch them.
	if ids, ok := job.Payload["pipelineIds"].([]interface{}); ok {
		for _, v := range ids {
			pid, _ := v.(string)
			if pid == "" {
				continue
			}
			os.Remove(deploy.SourceArchivePath(pid)) //nolint:errcheck
		}
	}
	return nil
}

// extractUploadedSource unpacks the tarball uploaded for a pipeline
// (POST /deploy/pipelines/{id}/source) into a temp dir the builder can use.
// The API and this worker share the storage volume.
func extractUploadedSource(pipelineID string) (string, error) {
	archive := deploy.SourceArchivePath(pipelineID)
	f, err := os.Open(archive)
	if err != nil {
		return "", fmt.Errorf("no uploaded source for this pipeline — upload one to /deploy/pipelines/%s/source", pipelineID)
	}
	defer f.Close()

	// The console can send either a gzipped tar it built from a dropped folder
	// or a .zip the user picked. Detect by magic bytes rather than trusting a
	// filename we control anyway.
	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return "", fmt.Errorf("uploaded source is empty or truncated")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "applad-upload-*")
	if err != nil {
		return "", err
	}

	if bytes.HasPrefix(magic, []byte("PK\x03\x04")) {
		if err := extractZip(archive, dir); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
		return dir, nil
	}

	gz, err := gzip.NewReader(f)
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("uploaded source is not a gzipped tar or zip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(dir)
			return "", fmt.Errorf("read source archive: %w", err)
		}
		// Refuse paths that escape the extraction root (zip-slip).
		target := filepath.Join(dir, filepath.Clean("/"+hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(os.PathSeparator)) {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0o755) //nolint:errcheck
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0o755) //nolint:errcheck
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				os.RemoveAll(dir)
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil { //nolint:gosec
				out.Close()
				os.RemoveAll(dir)
				return "", err
			}
			out.Close()
		}
	}
	return dir, nil
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
		_, deployErr = w.deployExecutor.DeployWeb(ctx, deploymentID, projectID, cfg)
	case "container":
		w.updateDeployStatus(ctx, deploymentID, projectID, "deploying")
		_, deployErr = w.deployExecutor.DeployContainer(ctx, deploymentID, projectID, cfg)
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
		_, deployErr = w.deployExecutor.DeployWeb(ctx, deploymentID, projectID, cfg)
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

// ── deploy_release / deploy_rollback processors ───────────────────────────────

// pipelineConfig holds the fields we need from deploy_pipelines + deploy_targets.
type pipelineConfig struct {
	pipelineID string
	targetID   string
	projectID  string
	sourceType string // "git" | "upload"
	sourceURL  string
	branch     string
	buildCmd   string
	outputDir  string
	targetType string // "serverless" | "web" | "container"
	runtime    string
	entrypoint string
	timeoutMs  int
	targetName string
	subdomain  string // <sub>.applad.dev this app is served on
}

func (w *Builds) loadPipelineConfig(ctx context.Context, pipelineID, targetID, projectID string) (*pipelineConfig, error) {
	var cfg pipelineConfig
	cfg.pipelineID = pipelineID
	cfg.targetID = targetID
	cfg.projectID = projectID

	var sourceURL, branch, buildCmd, outputDir, runtimeName, entrypoint, domain, storedSub sql.NullString
	err := w.db.QueryRowContext(ctx,
		`SELECT dp.source_type, dp.source_url, dp.branch, dp.build_cmd, dp.output_dir,
		        dt.type, dt.runtime, dt.entrypoint, dp.timeout_ms, dt.domain, dt.name, dt.subdomain
		 FROM deploy_pipelines dp
		 JOIN deploy_targets dt ON dt.id = dp.target_id
		 WHERE dp.id = $1 AND dp.project_id = $2`, pipelineID, projectID,
	).Scan(&cfg.sourceType, &sourceURL, &branch, &buildCmd, &outputDir,
		&cfg.targetType, &runtimeName, &entrypoint, &cfg.timeoutMs, &domain, &cfg.targetName, &storedSub)
	if err != nil {
		return nil, fmt.Errorf("load pipeline config: %w", err)
	}
	cfg.sourceURL, cfg.branch = sourceURL.String, branch.String
	cfg.buildCmd, cfg.outputDir = buildCmd.String, outputDir.String
	cfg.runtime, cfg.entrypoint = runtimeName.String, entrypoint.String

	// The subdomain a deployed app is served on: <sub>.applad.dev.
	// The target's claimed subdomain is authoritative; a custom domain or the
	// name are only fallbacks for targets created before it was stored.
	if storedSub.Valid && storedSub.String != "" {
		cfg.subdomain = storedSub.String
	} else if sub := deploy.Subdomain(domain.String); sub != "" {
		cfg.subdomain = sub
	} else {
		cfg.subdomain = deploy.Subdomain(cfg.targetName)
	}
	return &cfg, nil
}

func (w *Builds) updateReleaseStatus(ctx context.Context, releaseID, status, buildLog, releaseErr string, durationMs int64) {
	now := time.Now().UTC()
	var completedAt interface{}
	if status == "success" || status == "failed" {
		completedAt = now
	}
	if _, err := w.db.ExecContext(ctx,
		`UPDATE deploy_releases SET status=?, build_log=?, error=?, completed_at=?, duration_ms=? WHERE id=?`,
		status, buildLog, releaseErr, completedAt, durationMs, releaseID); err != nil {
		slog.Error("builds worker: update release status failed", "release_id", releaseID, "error", err)
		metrics.DBErrors.Inc()
	}
}

func (w *Builds) processRelease(ctx context.Context, job *queue.Job) error {
	releaseID, _ := job.Payload["releaseId"].(string)
	pipelineID, _ := job.Payload["pipelineId"].(string)
	targetID, _ := job.Payload["targetId"].(string)
	projectID, _ := job.Payload["projectId"].(string)
	commitSHA, _ := job.Payload["commitSha"].(string)

	if releaseID == "" || pipelineID == "" {
		slog.Warn("builds worker: deploy_release job missing ids", "job_id", job.ID)
		return nil
	}

	start := time.Now()
	w.updateReleaseStatus(ctx, releaseID, "building", "", "", 0)
	w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "pending", "Building…")

	cfg, err := w.loadPipelineConfig(ctx, pipelineID, targetID, projectID)
	if err != nil {
		w.updateReleaseStatus(ctx, releaseID, "failed", "", err.Error(), time.Since(start).Milliseconds())
		w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "failure", "Build configuration error")
		return err
	}

	var sourceDir string
	switch {
	case cfg.sourceType == "git" && cfg.sourceURL != "":
		cloned, err := cloneToSource(ctx, cfg.sourceURL, cfg.branch)
		if err != nil {
			w.updateReleaseStatus(ctx, releaseID, "failed", err.Error(), "", time.Since(start).Milliseconds())
			w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "failure", "Clone failed")
			return err
		}
		sourceDir = cloned
		defer os.RemoveAll(sourceDir)

	case cfg.sourceType == "upload":
		extracted, err := extractUploadedSource(pipelineID)
		if err != nil {
			w.updateReleaseStatus(ctx, releaseID, "failed", "", err.Error(), time.Since(start).Milliseconds())
			w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "failure", "Source unavailable")
			return err
		}
		sourceDir = extracted
		defer os.RemoveAll(sourceDir)
	}

	w.updateReleaseStatus(ctx, releaseID, "deploying", "", "", 0)

	// A pipeline with no build configuration gets it inferred from the source.
	// The console prefills the same values from the same detector, so this is
	// the fallback for API-created pipelines and for git sources, which have
	// nothing to inspect until the clone lands here.
	buildCmd, outputDir, serveMode, nodeVersion := cfg.buildCmd, cfg.outputDir, "", ""
	if sourceDir != "" && buildCmd == "" && (outputDir == "" || outputDir == ".") {
		d := deploy.DetectDir(sourceDir)
		buildCmd, outputDir, serveMode, nodeVersion = d.BuildCommand, d.OutputDir, d.ServeMode, d.NodeVersion
		if d.InstallCommand != "" && buildCmd != "" {
			buildCmd = d.InstallCommand + " && " + buildCmd
		}
		slog.Info("builds worker: detected framework", "release_id", releaseID,
			"framework", d.Framework, "reason", d.Reason, "build", buildCmd, "output", outputDir)
		// Recorded on the target so the console can name what this is.
		w.db.ExecContext(ctx, //nolint:errcheck
			"UPDATE deploy_targets SET framework = $1 WHERE id = $2", d.Framework, targetID)
	}

	deployConfig := runtime.ParseDeployConfig(map[string]interface{}{
		"buildCmd":    buildCmd,
		"outputDir":   outputDir,
		"serveMode":   serveMode,
		"nodeVersion": nodeVersion,
		"runtime":     cfg.runtime,
		"entrypoint":  cfg.entrypoint,
		"sourceDir":   sourceDir,
		"subdomain":   cfg.subdomain,
	})

	var deployErr error
	var buildLog string
	switch cfg.targetType {
	case "serverless", "function":
		req := runtime.ExecRequest{
			FunctionID: targetID, ProjectID: projectID,
			Runtime: cfg.runtime, Entrypoint: cfg.entrypoint, SourceDir: sourceDir,
			Timeout: cfg.timeoutMs / 1000,
		}
		_, deployErr = w.executor.Build(ctx, req)
	case "container":
		buildLog, deployErr = w.deployExecutor.DeployContainer(ctx, releaseID, projectID, deployConfig)
	default: // "web" and unknown
		buildLog, deployErr = w.deployExecutor.DeployWeb(ctx, releaseID, projectID, deployConfig)
	}

	durationMs := time.Since(start).Milliseconds()
	if deployErr == nil {
		// Recorded so the console can show what the deploy produced instead of
		// a pair of dashes.
		if size, err := w.deployExecutor.ImageSize(ctx, "applad-deploy-"+releaseID); err == nil && size > 0 {
			w.db.ExecContext(ctx, //nolint:errcheck
				"UPDATE deploy_releases SET size_bytes = $1 WHERE id = $2", size, releaseID)
		}
	}
	if deployErr == nil && cfg.subdomain != "" {
		// What just shipped is what gets checked.
		svc := testlab.NewService(w.db, w.queue)
		if n, err := svc.TriggerOnDeploy(ctx, projectID,
			"http://applad-site-"+cfg.subdomain, "deploy:"+releaseID); err == nil && n > 0 {
			slog.Info("builds worker: triggered tests after deploy", "release_id", releaseID, "suites", n)
		}
	}
	if deployErr != nil {
		// The log matters most when it failed, so it is stored alongside the
		// error rather than folded into it.
		w.updateReleaseStatus(ctx, releaseID, "failed", buildLog, deployErr.Error(), durationMs)
		w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "failure", "Deploy failed")
		return deployErr
	}

	// Kept whether or not it succeeded: the log of a deploy that worked is
	// what the next one gets compared against.
	w.updateReleaseStatus(ctx, releaseID, "success", buildLog, "", durationMs)
	w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "success", "Deployed successfully")
	slog.Info("builds worker: release complete", "release_id", releaseID, "duration_ms", durationMs)
	return nil
}

func (w *Builds) processRollback(ctx context.Context, job *queue.Job) error {
	releaseID, _ := job.Payload["releaseId"].(string)
	originalReleaseID, _ := job.Payload["originalReleaseId"].(string)
	pipelineID, _ := job.Payload["pipelineId"].(string)
	targetID, _ := job.Payload["targetId"].(string)
	projectID, _ := job.Payload["projectId"].(string)
	artifactPath, _ := job.Payload["artifactPath"].(string)

	if releaseID == "" {
		return nil
	}

	start := time.Now()
	w.updateReleaseStatus(ctx, releaseID, "deploying", "", "", 0)

	cfg, err := w.loadPipelineConfig(ctx, pipelineID, targetID, projectID)
	if err != nil {
		w.updateReleaseStatus(ctx, releaseID, "failed", "", err.Error(), time.Since(start).Milliseconds())
		return err
	}

	deployConfig := runtime.ParseDeployConfig(map[string]interface{}{
		"artifactPath": artifactPath,
		"runtime":      cfg.runtime,
		"entrypoint":   cfg.entrypoint,
	})

	var deployErr error
	switch cfg.targetType {
	case "container":
		_, deployErr = w.deployExecutor.DeployContainer(ctx, releaseID, projectID, deployConfig)
	default:
		_, deployErr = w.deployExecutor.DeployWeb(ctx, releaseID, projectID, deployConfig)
	}

	durationMs := time.Since(start).Milliseconds()
	if deployErr != nil {
		w.updateReleaseStatus(ctx, releaseID, "failed", "", deployErr.Error(), durationMs)
		return deployErr
	}

	// Mark the original release as rolled_back.
	if originalReleaseID != "" {
		w.db.ExecContext(ctx, //nolint:errcheck
			`UPDATE deploy_releases SET status='rolled_back' WHERE id=?`, originalReleaseID)
	}

	w.updateReleaseStatus(ctx, releaseID, "success", "", "", durationMs)
	slog.Info("builds worker: rollback complete", "release_id", releaseID)
	return nil
}

// postReleaseCommitStatus looks up the git connection for the pipeline's project
// and posts a commit status back to GitHub/GitLab.
func (w *Builds) postReleaseCommitStatus(ctx context.Context, projectID, pipelineID, commitSHA, state, description string) {
	if commitSHA == "" {
		return
	}

	var accessToken, provider, sourceURL string
	err := w.db.QueryRowContext(ctx,
		`SELECT gc.access_token, gc.provider, dp.source_url
		 FROM deploy_pipelines dp
		 JOIN git_connections gc ON gc.project_id = dp.project_id
		 WHERE dp.id = ? AND dp.project_id = ?
		 LIMIT 1`, pipelineID, projectID,
	).Scan(&accessToken, &provider, &sourceURL)
	if err == sql.ErrNoRows || err != nil {
		return // No git connection — nothing to post.
	}
	if accessToken == "" || sourceURL == "" {
		return
	}

	// Extract "owner/repo" from the source URL.
	repoFull := extractRepoName(sourceURL)
	if repoFull == "" {
		return
	}

	go func() {
		postCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		postCommitStatusHTTP(postCtx, provider, accessToken, repoFull, commitSHA, state, "", description)
	}()
}

func postCommitStatusHTTP(ctx context.Context, provider, accessToken, repoFull, sha, state, targetURL, description string) {
	var apiURL string
	var body []byte

	switch provider {
	case "github":
		ghState := map[string]string{"pending": "pending", "success": "success", "failure": "failure", "error": "error"}[state]
		if ghState == "" {
			ghState = "error"
		}
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", repoFull, sha)
		body, _ = json.Marshal(map[string]string{
			"state": ghState, "target_url": targetURL,
			"description": truncateStr(description, 140), "context": "applad/deploy",
		})
	case "gitlab":
		glState := map[string]string{"pending": "pending", "success": "success", "failure": "failed", "error": "failed"}[state]
		if glState == "" {
			glState = "failed"
		}
		encoded := strings.ReplaceAll(repoFull, "/", "%2F")
		apiURL = fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/statuses/%s", encoded, sha)
		body, _ = json.Marshal(map[string]string{
			"state": glState, "target_url": targetURL,
			"description": truncateStr(description, 140), "name": "applad/deploy",
		})
	default:
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if provider == "github" {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("builds worker: commit status post failed", "error", err)
		return
	}
	resp.Body.Close()
}

func extractRepoName(sourceURL string) string {
	// Handle https://github.com/owner/repo and https://github.com/owner/repo.git
	u := strings.TrimSuffix(sourceURL, ".git")
	parts := strings.Split(u, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return ""
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
