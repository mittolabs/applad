package worker

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mittolabs/applad/internal/browsershot"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/githubapp"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/runtime"
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
	docker         *runtime.Client
	// Consulted only to mint GitHub tokens for a clone; the worker does its
	// own querying for everything else.
	deploySvc *deploy.Service
	github    *githubapp.App
}

func NewBuilds(cfg *config.Config) *Builds {
	return &Builds{cfg: cfg}
}

func (w *Builds) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.rdb = rdb
	w.queue = queue.New(rdb)
	StartRedisHeartbeat(ctx, rdb, "builds")

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database

	w.executor = runtime.NewExecutor()
	w.deployExecutor = runtime.NewDeployExecutor()
	w.docker = runtime.NewClient()

	// The worker clones repositories, so it needs to be able to mint a token
	// for a private one. An instance with no GitHub App configured carries on
	// with public repositories only.
	w.deploySvc = deploy.NewService(database, w.queue)
	w.deploySvc.SetRedis(rdb)
	if app, err := githubapp.FromConfig(w.cfg); err == nil {
		w.github = app
		w.deploySvc.SetGitHubApp(app)
		slog.Info("builds worker: github app configured", "slug", app.Slug())
	} else if !errors.Is(err, githubapp.ErrNotConfigured) {
		slog.Error("builds worker: github app misconfigured", "error", err)
	}

	w.queue.StartReaper(ctx, "builds")
	w.startBrowserReaper(ctx)

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
		cloned, err := w.cloneToSource(ctx, projectID, repository, branch)
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
		EnvVars: envVarsFromPayload(job.Payload["envVars"]),
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
	projectID, _ := job.Payload["projectId"].(string)
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
		cloned, err := w.cloneToSource(ctx, projectID, repository, branch)
		if err != nil {
			w.db.ExecContext(ctx, "UPDATE functions SET status = 'failed' WHERE id = ?", functionID) //nolint:errcheck
			return err
		}
		sourceDir = cloned
		defer os.RemoveAll(sourceDir)
	}

	env := envVarsFromPayload(job.Payload["envVars"])
	req := runtime.ExecRequest{FunctionID: functionID, Runtime: runtimeName,
		Entrypoint: entrypoint, Source: source, SourceDir: sourceDir, Dockerfile: dockerfile, EnvVars: env}
	if _, err := w.executor.Build(ctx, req); err != nil {
		w.db.ExecContext(ctx, "UPDATE functions SET status = 'failed' WHERE id = ?", functionID) //nolint:errcheck
		return err
	}
	// Pre-warm with the same env the container will run with, so a reused warm
	// container already carries the function's configured variables.
	warmReq := runtime.ExecRequest{FunctionID: functionID, Runtime: runtimeName,
		Entrypoint: entrypoint, Source: source, SourceDir: sourceDir, Timeout: 30, EnvVars: env}
	if _, err := w.executor.Execute(ctx, warmReq); err != nil {
		slog.Warn("builds worker: pre-warm failed (non-fatal)", "function_id", functionID, "error", err)
	}
	w.db.ExecContext(ctx, "UPDATE functions SET status = 'active' WHERE id = ?", functionID) //nolint:errcheck
	slog.Info("builds worker: function ready", "function_id", functionID)
	return nil
}

// envVarsFromPayload reads a function's configured environment variables out of
// a queue job payload. The payload is JSON on the wire, so a map[string]string
// arrives back as map[string]interface{}; values are coerced to strings and
// non-string entries are skipped. Only the function's own vars are returned, so
// nothing else in the worker's environment leaks into the container.
func envVarsFromPayload(v interface{}) map[string]string {
	raw, ok := v.(map[string]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	env := make(map[string]string, len(raw))
	for k, val := range raw {
		if s, ok := val.(string); ok {
			env[k] = s
		}
	}
	return env
}

// cloneToSource shallow-clones a git repository and returns the cloned directory path.
// The caller is responsible for reading files from it; the directory is left on disk
// so the executor can tar it up. A temp dir is created under /tmp/applad-git-*.
//
// A private repository is reached with a token minted for the project's own
// GitHub App installation, which is why the project has to be named: the token
// is what authorises the fetch, and whose it is decides what may be fetched.
func (w *Builds) cloneToSource(ctx context.Context, projectID, repository, branch string) (string, error) {
	dir, err := os.MkdirTemp("", "applad-git-*")
	if err != nil {
		return "", fmt.Errorf("git clone: mktemp: %w", err)
	}

	authURL := ""
	if w.deploySvc != nil && projectID != "" {
		token, err := w.deploySvc.CloneTokenForRepo(ctx, projectID, repository)
		if err != nil {
			os.RemoveAll(dir)
			return "", err
		}
		if token != "" {
			authURL = githubapp.CloneURL(repository, token)
		}
	}

	if err := runtime.CloneRepoAs(ctx, repository, authURL, branch, dir); err != nil {
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

// deployNarrative describes a deploy in terms of what happened to the
// project, with the project's own build output in the middle.
//
// The raw Docker stream narrates a Dockerfile that Applad wrote, so on its own
// it tells somebody about plumbing they never asked for.
func deployNarrative(cfg *pipelineConfig, rawBuildLog string, durationMs, sizeBytes int64, served bool) string {
	var b strings.Builder

	switch {
	case cfg.sourceType == "git" && cfg.sourceURL != "":
		fmt.Fprintf(&b, "Cloned %s", cfg.sourceURL)
		if cfg.branch != "" {
			fmt.Fprintf(&b, " (%s)", cfg.branch)
		}
		b.WriteByte('\n')
	default:
		b.WriteString("Using the uploaded source\n")
	}

	// Two builders, two log formats: a generated Dockerfile narrates steps,
	// railpack emits BuildKit progress. Picking the wrong renderer produced an
	// empty result, which the line below then reported as "no build step" for
	// a build that had just installed and compiled the whole project.
	if output := runtime.RenderBuildLog(rawBuildLog); output != "" {
		b.WriteString("\n")
		b.WriteString(output)
		b.WriteString("\n\n")
	} else if served {
		// Only a deploy that worked can claim there was nothing to build. A
		// failed one produced no output because it never got that far, and
		// saying "the files are served as they are" of a site that is not
		// served at all is the sort of confident falsehood this log exists to
		// avoid.
		b.WriteString("No build step — the files are served as they are\n")
	}

	if sizeBytes > 0 {
		fmt.Fprintf(&b, "Built image (%.1f MB) in %s\n",
			float64(sizeBytes)/(1024*1024), time.Duration(durationMs)*time.Millisecond)
	}
	// Only when it is actually serving: a failed build used to sign off with
	// an address that answers nothing.
	if served && cfg.subdomain != "" {
		// The full address, since "the-range" alone is not something anyone
		// can open.
		domain := os.Getenv("APPLAD_DEPLOY_DOMAIN")
		if domain == "" {
			domain = "applad.dev.localhost"
		}
		scheme := "https"
		if strings.HasSuffix(domain, ".localhost") {
			scheme = "http"
		}
		fmt.Fprintf(&b, "Serving at %s://%s.%s\n", scheme, cfg.subdomain, domain)
	}
	return b.String()
}

// captureSitePreview photographs a site once it is serving.
//
// Taken at deploy time rather than on demand: the console asking for a
// screenshot would mean starting a browser while somebody waits, and a site
// looks the way it looked when it shipped.
func (w *Builds) captureSitePreview(ctx context.Context, targetID, subdomain string) {
	url := fmt.Sprintf("http://applad-site-%s", subdomain)

	sessionID := "preview-" + targetID
	containerID, wsURL, err := w.deployExecutor.StartBrowser(ctx, sessionID, runtime.BrowserImage())
	if err != nil {
		slog.Warn("builds worker: preview browser failed to start", "target_id", targetID, "error", err)
		return
	}
	defer w.deployExecutor.StopBrowser(context.Background(), containerID) //nolint:errcheck

	png, err := browsershot.Screenshot(ctx, wsURL, url, 1280, 800)
	if err != nil {
		slog.Warn("builds worker: preview capture failed", "target_id", targetID, "error", err)
		return
	}

	path := deploy.PreviewPath(targetID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if err := os.WriteFile(path, png, 0o644); err != nil {
		slog.Warn("builds worker: preview write failed", "target_id", targetID, "error", err)
		return
	}
	slog.Info("builds worker: captured site preview", "target_id", targetID, "bytes", len(png))
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
	if targetID, _ := job.Payload["targetId"].(string); targetID != "" {
		os.Remove(deploy.PreviewPath(targetID)) //nolint:errcheck
	}

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
	installCmd string
	buildCmd   string
	startCmd   string
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

	var sourceURL, branch, installCmd, buildCmd, startCmd, outputDir sql.NullString
	var runtimeName, entrypoint, domain, storedSub sql.NullString
	err := w.db.QueryRowContext(ctx,
		`SELECT dp.source_type, dp.source_url, dp.branch,
		        dp.install_cmd, dp.build_cmd, dp.start_cmd, dp.output_dir,
		        dt.type, dt.runtime, dt.entrypoint, dp.timeout_ms, dt.domain, dt.name, dt.subdomain
		 FROM deploy_pipelines dp
		 JOIN deploy_targets dt ON dt.id = dp.target_id
		 WHERE dp.id = $1 AND dp.project_id = $2`, pipelineID, projectID,
	).Scan(&cfg.sourceType, &sourceURL, &branch,
		&installCmd, &buildCmd, &startCmd, &outputDir,
		&cfg.targetType, &runtimeName, &entrypoint, &cfg.timeoutMs, &domain, &cfg.targetName, &storedSub)
	if err != nil {
		return nil, fmt.Errorf("load pipeline config: %w", err)
	}
	cfg.sourceURL, cfg.branch = sourceURL.String, branch.String
	cfg.installCmd, cfg.buildCmd = installCmd.String, buildCmd.String
	cfg.startCmd, cfg.outputDir = startCmd.String, outputDir.String
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

// logStreamer surfaces a release's build output as it happens. Each line is
// pushed to subscribers immediately over realtime (pg_notify, which the API's
// hub forwards to WebSocket clients) and buffered for a periodic append to
// deploy_releases.build_log, so a browser that opens or refreshes mid-build sees
// the progress so far. The final status write replaces build_log with the
// canonical full log.
type logStreamer struct {
	db        *db.DB
	releaseID string
	mu        sync.Mutex
	seq       int64
	pending   strings.Builder
}

func newLogStreamer(database *db.DB, releaseID string) *logStreamer {
	return &logStreamer{db: database, releaseID: releaseID}
}

// line is the sink handed to the build. It must not block the build goroutine,
// so the realtime notify is best-effort and the DB append is deferred to flush.
func (ls *logStreamer) line(s string) {
	ls.mu.Lock()
	ls.seq++
	seq := ls.seq
	ls.pending.WriteString(s)
	ls.pending.WriteByte('\n')
	ls.mu.Unlock()

	payload, err := json.Marshal(map[string]interface{}{
		"kind": "deploy_log", "release_id": ls.releaseID, "seq": seq, "line": s,
	})
	if err != nil {
		return
	}
	// Best-effort live push; a dropped notify only costs this one line's
	// immediacy, and the periodic flush persists it regardless.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ls.db.ExecContext(ctx, "SELECT pg_notify('applad_changes', ?)", string(payload)) //nolint:errcheck
}

// flush appends everything buffered since the last flush to build_log.
func (ls *logStreamer) flush(ctx context.Context) {
	ls.mu.Lock()
	chunk := ls.pending.String()
	ls.pending.Reset()
	ls.mu.Unlock()
	if chunk == "" {
		return
	}
	ls.db.ExecContext(ctx, //nolint:errcheck
		"UPDATE deploy_releases SET build_log = COALESCE(build_log, '') || ? WHERE id = ?",
		chunk, ls.releaseID)
}

// startFlushing persists buffered lines on a ticker until the returned stop is
// called (which also does a final flush).
func (ls *logStreamer) startFlushing() func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(800 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				ls.flush(ctx)
				cancel()
			}
		}
	}()
	return func() {
		close(stop)
		<-done
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ls.flush(ctx)
		cancel()
	}
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
		cloned, err := w.cloneToSource(ctx, projectID, cfg.sourceURL, cfg.branch)
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

	// Each phase is settled on its own: what the pipeline states wins, and
	// anything it leaves blank is inferred from the source.
	//
	// This used to be all-or-nothing — detection ran only when the build
	// command was empty — so naming a build command silently discarded the
	// install that has to precede it, and the build died with `next: not
	// found`. Filling in the form was enough to break it.
	installCmd, buildCmd := cfg.installCmd, cfg.buildCmd
	startCmd, outputDir := cfg.startCmd, cfg.outputDir
	serveMode, nodeVersion := "", ""
	packageManagerPin := ""
	railpackConfig, platform := "", ""

	if sourceDir != "" {
		d := deploy.DetectDir(sourceDir)
		installCmd = firstNonEmpty(installCmd, d.InstallCommand)
		buildCmd = firstNonEmpty(buildCmd, d.BuildCommand)
		startCmd = firstNonEmpty(startCmd, d.StartCommand)
		if outputDir == "" || outputDir == "." {
			outputDir = firstNonEmpty(d.OutputDir, outputDir)
		}
		serveMode, nodeVersion = d.ServeMode, d.NodeVersion
		packageManagerPin = d.PackageManagerPin
		railpackConfig = railpackConfigFor(d)
		platform = buildPlatformFor(d)
		slog.Info("builds worker: build plan", "release_id", releaseID,
			"framework", d.Framework, "reason", d.Reason,
			"install", installCmd, "build", buildCmd, "start", startCmd, "output", outputDir)
		// Recorded on the target so the console can name what this is.
		w.db.ExecContext(ctx, //nolint:errcheck
			"UPDATE deploy_targets SET framework = $1 WHERE id = $2", d.Framework, targetID)
	}

	deployConfig := runtime.ParseDeployConfig(map[string]interface{}{
		"installCmd":        installCmd,
		"buildCmd":          buildCmd,
		"startCmd":          startCmd,
		"packageManagerPin": packageManagerPin,
		"railpackConfig":    railpackConfig,
		"platform":          platform,
		"outputDir":         outputDir,
		"serveMode":         serveMode,
		"nodeVersion":       nodeVersion,
		"runtime":           cfg.runtime,
		"entrypoint":        cfg.entrypoint,
		"sourceDir":         sourceDir,
		"subdomain":         cfg.subdomain,
	})

	// Stream the build's output live and persist it as it arrives, so the
	// console shows progress during the build instead of a blank until it ends.
	streamer := newLogStreamer(w.db, releaseID)
	deployConfig.LogSink = streamer.line
	stopFlushing := streamer.startFlushing()

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
	stopFlushing()

	durationMs := time.Since(start).Milliseconds()
	var imageSize int64
	if deployErr == nil {
		// Recorded so the console can show what the deploy produced instead of
		// a pair of dashes.
		if size, err := w.deployExecutor.ImageSize(ctx, "applad-deploy-"+releaseID); err == nil && size > 0 {
			imageSize = size
			w.db.ExecContext(ctx, //nolint:errcheck
				"UPDATE deploy_releases SET size_bytes = $1 WHERE id = $2", size, releaseID)
		}
	}
	if deployErr == nil && cfg.subdomain != "" {
		// A picture of what shipped, taken now rather than promised later.
		w.captureSitePreview(ctx, targetID, cfg.subdomain)
	}
	if deployErr != nil {
		// The log matters most when it failed, so it is stored alongside the
		// error rather than folded into it.
		w.updateReleaseStatus(ctx, releaseID, "failed",
			deployNarrative(cfg, buildLog, durationMs, 0, false), deployErr.Error(), durationMs)
		w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "failure", "Deploy failed")
		return deployErr
	}

	// Kept whether or not it succeeded: the log of a deploy that worked is
	// what the next one gets compared against.
	w.updateReleaseStatus(ctx, releaseID, "success",
		deployNarrative(cfg, buildLog, durationMs, imageSize, true), "", durationMs)
	w.postReleaseCommitStatus(ctx, projectID, pipelineID, commitSHA, "success", "Deployed successfully")
	w.reapReleaseImages(ctx, targetID, releaseID, cfg.targetType)
	slog.Info("builds worker: release complete", "release_id", releaseID, "duration_ms", durationMs)
	return nil
}

// reapReleaseImages removes the images earlier releases of a target left
// behind. Every release tags applad-deploy-<id> and nothing ever deleted
// them — 17GB accumulated in a day on one host. Only settled releases are
// touched: keepID is the one whose image the serving container runs, and a
// row still building belongs to another worker.
func (w *Builds) reapReleaseImages(ctx context.Context, targetID, keepID, targetType string) {
	if targetType == "serverless" || targetType == "function" {
		return // functions build applad-fn-<id>; the executor reaps those
	}
	rows, err := w.db.QueryContext(ctx,
		`SELECT id FROM deploy_releases
		  WHERE target_id = $1 AND id != $2
		    AND status IN ('success', 'failed', 'rolled_back')`,
		targetID, keepID)
	if err != nil {
		slog.Warn("builds worker: release image reap query failed", "target_id", targetID, "error", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			continue
		}
		// A missing image (most failed builds never tagged one) 404s, which
		// RemoveImage ignores.
		if err := w.docker.RemoveImage(ctx, "applad-deploy-"+id); err != nil {
			slog.Warn("builds worker: remove release image failed", "release_id", id, "error", err)
		}
	}
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
	// A rollback serves its own freshly built image, so older ones are as
	// dead here as after a normal release.
	w.reapReleaseImages(ctx, targetID, releaseID, cfg.targetType)
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

// firstNonEmpty returns the first value that was actually set.
//
// The order is the policy: what somebody wrote down beats what was inferred,
// per phase rather than for the build as a whole.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// railpackConfigFor describes a build the builder has no provider for.
//
// The finished app is a directory of files, so it is served by Caddy — taken
// from mise rather than from a caddy base image, because every Caddy tag is
// Alpine and railpack's start wrapper needs a shell those images do not have.
// The default runtime image has one.
//
// Railpack covers Node, Python, Go, PHP, Java and Ruby. Flutter is not among
// them, so a Flutter app failed with "no start command detected" — accurately,
// since without the SDK there is nothing to detect. mise can install the SDK,
// and the result is a directory of files, so the finished app is served by
// Caddy rather than by carrying a 1GB toolchain into production.
func railpackConfigFor(d deploy.Detection) string {
	if d.Framework != "flutter_web" {
		return ""
	}
	out := d.OutputDir
	if out == "" {
		out = "build/web"
	}
	// The first input of a step must be an image or another step, carrying no
	// filters; the source comes in after it. Without that first layer the
	// build has no toolchain, and with the source missing it has nothing to
	// build — the two ways this failed before.
	return fmt.Sprintf(`{
  "$schema": "https://schema.railpack.com",
  "packages": { "flutter": "latest", "caddy": "latest" },
  "steps": {
    "build": {
      "inputs": [
        { "step": "packages:mise" },
        { "local": true, "include": ["."] }
      ],
      "commands": [%q, %q]
    }
  },
  "deploy": {
    "inputs": [
      { "step": "packages:mise", "include": ["/mise/shims", "/mise/installs", "/usr/local/bin/mise", "/etc/mise/config.toml", "/root/.local/state/mise"] },
      { "step": "build", "include": ["/app/%s"] }
    ],
    "startCommand": "caddy file-server --listen :80 --root /app/%s"
  }
}`, d.InstallCommand, d.BuildCommand, out, out)
}

// buildPlatformFor pins an architecture when the toolchain ships for only one.
//
// Flutter publishes linux-x64 and no linux-arm64, so on an arm64 machine mise
// fails outright with "No URL for platform linux-arm64". Targeting amd64 is
// native on the x86 servers these apps are deployed to, and emulated on an
// Apple Silicon laptop.
func buildPlatformFor(d deploy.Detection) string {
	if d.Framework == "flutter_web" {
		return "linux/amd64"
	}
	return ""
}

// browserMaxAge is the hard ceiling on a preview browser's life. Capturing a
// site preview stops its own browser, so this only covers the ones that outlive
// their capture — a worker killed mid-deploy, say.
const browserMaxAge = 2 * time.Hour

// startBrowserReaper sweeps abandoned browsers.
//
// Only this worker holds the Docker socket, so it is the only thing that can
// see a container nothing is tracking any more. Without it a crashed capture
// left a Chromium running indefinitely.
func (w *Builds) startBrowserReaper(ctx context.Context) {
	if w.deployExecutor == nil {
		return
	}
	go func() {
		t := time.NewTicker(15 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n, err := w.deployExecutor.ReapStaleBrowsers(ctx, browserMaxAge)
				if err != nil {
					slog.Warn("builds worker: browser reap failed", "error", err)
					continue
				}
				if n > 0 {
					slog.Info("builds worker: reaped stale browsers", "count", n)
				}
			}
		}
	}()
}
