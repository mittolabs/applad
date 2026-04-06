// Package deploy implements Applad's deployment service:
// targets, pipelines, releases, and execution management.
package deploy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/uid"
)

// ── Models ──

// Target represents a deploy target (serverless function, web app, or container).
type Target struct {
	ID          string            `json:"$id"`
	ProjectID   string            `json:"projectId"`
	Name        string            `json:"name"`
	Type        string            `json:"type"` // serverless, web, container
	Runtime     string            `json:"runtime"`
	Entrypoint  string            `json:"entrypoint"`
	TimeoutMs   int               `json:"timeoutMs"`
	MemoryMB    int               `json:"memoryMb"`
	EnvVars     map[string]string `json:"envVars"`
	Permissions json.RawMessage   `json:"permissions"`
	Cron        string            `json:"cron,omitempty"`
	CreatedAt   time.Time         `json:"$createdAt"`
	UpdatedAt   time.Time         `json:"$updatedAt"`
}

// Pipeline represents a build/deploy pipeline tied to a target.
type Pipeline struct {
	ID         string            `json:"$id"`
	ProjectID  string            `json:"projectId"`
	TargetID   string            `json:"targetId"`
	Name       string            `json:"name"`
	SourceType string            `json:"sourceType"` // upload, git
	SourceURL  string            `json:"sourceUrl"`
	Branch     string            `json:"branch"`
	BuildCmd   string            `json:"buildCmd"`
	OutputDir  string            `json:"outputDir"`
	EnvVars    map[string]string `json:"envVars"`
	TriggerOn  json.RawMessage   `json:"triggerOn"`
	CacheDirs  json.RawMessage   `json:"cacheDirs"`
	TimeoutMs  int               `json:"timeoutMs"`
	CreatedAt  time.Time         `json:"$createdAt"`
	UpdatedAt  time.Time         `json:"$updatedAt"`
}

// Release represents a single build+deploy run of a pipeline.
type Release struct {
	ID           string     `json:"$id"`
	ProjectID    string     `json:"projectId"`
	PipelineID   string     `json:"pipelineId"`
	TargetID     string     `json:"targetId"`
	Status       string     `json:"status"` // pending, building, deploying, success, failed, rolled_back
	TriggerType  string     `json:"triggerType"`
	TriggerActor string     `json:"triggerActor"`
	CommitSHA    string     `json:"commitSha,omitempty"`
	BuildLog     string     `json:"buildLog,omitempty"`
	DeployLog    string     `json:"deployLog,omitempty"`
	ArtifactPath string     `json:"artifactPath,omitempty"`
	DurationMs   int64      `json:"durationMs"`
	Error        string     `json:"error,omitempty"`
	StartedAt    *time.Time `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt"`
	CreatedAt    time.Time  `json:"$createdAt"`
}

// Execution represents a single invocation of a serverless target.
type Execution struct {
	ID         string    `json:"$id"`
	ProjectID  string    `json:"projectId"`
	TargetID   string    `json:"targetId"`
	Status     string    `json:"status"` // pending, running, completed, failed
	StatusCode int       `json:"statusCode"`
	Request    string    `json:"request"`
	Response   string    `json:"response"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	DurationMs int64     `json:"durationMs"`
	Trigger    string    `json:"trigger"`
	CreatedAt  time.Time `json:"$createdAt"`
}

// ── Service ──

// Service handles deploy business logic.
type Service struct {
	db    *db.DB
	queue *queue.Queue
}

// NewService creates a new deploy Service.
func NewService(database *db.DB, q *queue.Queue) *Service {
	return &Service{db: database, queue: q}
}

// ── Target CRUD ──

// CreateTarget creates a new deploy target.
func (s *Service) CreateTarget(ctx context.Context, projectID, name, targetType, runtime, entrypoint string, timeoutMs, memoryMB int, envVars map[string]string, permissions json.RawMessage, cron string) (*Target, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	if envVars == nil {
		envVars = map[string]string{}
	}
	if permissions == nil {
		permissions = json.RawMessage("[]")
	}

	envJSON, _ := json.Marshal(envVars)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO deploy_targets (id, project_id, name, type, runtime, entrypoint, timeout_ms, memory_mb, env_vars, permissions, cron, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, name, targetType, runtime, entrypoint, timeoutMs, memoryMB, envJSON, permissions, cron, now, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: create target: %w", err)
	}
	return &Target{
		ID: id, ProjectID: projectID, Name: name, Type: targetType,
		Runtime: runtime, Entrypoint: entrypoint, TimeoutMs: timeoutMs, MemoryMB: memoryMB,
		EnvVars: envVars, Permissions: permissions, Cron: cron,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// GetTarget returns a target by ID.
func (s *Service) GetTarget(ctx context.Context, id, projectID string) (*Target, error) {
	var t Target
	var envJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, type, runtime, entrypoint, timeout_ms, memory_mb, env_vars, permissions, cron, created_at, updated_at
		 FROM deploy_targets WHERE id = ? AND project_id = ?`, id, projectID,
	).Scan(&t.ID, &t.ProjectID, &t.Name, &t.Type, &t.Runtime, &t.Entrypoint,
		&t.TimeoutMs, &t.MemoryMB, &envJSON, &t.Permissions, &t.Cron, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("target not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envJSON, &t.EnvVars)
	if t.EnvVars == nil {
		t.EnvVars = map[string]string{}
	}
	if t.Permissions == nil {
		t.Permissions = json.RawMessage("[]")
	}
	return &t, nil
}

// ListTargets returns all targets for a project.
func (s *Service) ListTargets(ctx context.Context, projectID string) ([]*Target, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, type, runtime, entrypoint, timeout_ms, memory_mb, env_vars, permissions, cron, created_at, updated_at
		 FROM deploy_targets WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var targets []*Target
	for rows.Next() {
		var t Target
		var envJSON []byte
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Name, &t.Type, &t.Runtime, &t.Entrypoint,
			&t.TimeoutMs, &t.MemoryMB, &envJSON, &t.Permissions, &t.Cron, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(envJSON, &t.EnvVars)
		if t.EnvVars == nil {
			t.EnvVars = map[string]string{}
		}
		if t.Permissions == nil {
			t.Permissions = json.RawMessage("[]")
		}
		targets = append(targets, &t)
	}
	return targets, len(targets), nil
}

// UpdateTarget updates a deploy target.
func (s *Service) UpdateTarget(ctx context.Context, id, projectID, name, targetType, runtime, entrypoint string, timeoutMs, memoryMB int, envVars map[string]string, permissions json.RawMessage, cron string) (*Target, error) {
	if envVars == nil {
		envVars = map[string]string{}
	}
	if permissions == nil {
		permissions = json.RawMessage("[]")
	}
	envJSON, _ := json.Marshal(envVars)

	_, err := s.db.ExecContext(ctx,
		`UPDATE deploy_targets SET name=?, type=?, runtime=?, entrypoint=?, timeout_ms=?, memory_mb=?, env_vars=?, permissions=?, cron=?, updated_at=?
		 WHERE id=? AND project_id=?`,
		name, targetType, runtime, entrypoint, timeoutMs, memoryMB, envJSON, permissions, cron, time.Now().UTC(), id, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetTarget(ctx, id, projectID)
}

// DeleteTarget removes a target and its related pipelines, releases, and executions.
func (s *Service) DeleteTarget(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM deploy_targets WHERE id = ? AND project_id = ?", id, projectID)
	return err
}

// ── Pipeline CRUD ──

// CreatePipeline creates a new pipeline.
func (s *Service) CreatePipeline(ctx context.Context, projectID, targetID, name, sourceType, sourceURL, branch, buildCmd, outputDir string, envVars map[string]string, triggerOn, cacheDirs json.RawMessage, timeoutMs int) (*Pipeline, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	if envVars == nil {
		envVars = map[string]string{}
	}
	if triggerOn == nil {
		triggerOn = json.RawMessage("[]")
	}
	if cacheDirs == nil {
		cacheDirs = json.RawMessage("[]")
	}

	envJSON, _ := json.Marshal(envVars)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO deploy_pipelines (id, project_id, target_id, name, source_type, source_url, branch, build_cmd, output_dir, env_vars, trigger_on, cache_dirs, timeout_ms, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, targetID, name, sourceType, sourceURL, branch, buildCmd, outputDir, envJSON, triggerOn, cacheDirs, timeoutMs, now, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: create pipeline: %w", err)
	}
	return &Pipeline{
		ID: id, ProjectID: projectID, TargetID: targetID, Name: name,
		SourceType: sourceType, SourceURL: sourceURL, Branch: branch,
		BuildCmd: buildCmd, OutputDir: outputDir, EnvVars: envVars,
		TriggerOn: triggerOn, CacheDirs: cacheDirs, TimeoutMs: timeoutMs,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// GetPipeline returns a pipeline by ID.
func (s *Service) GetPipeline(ctx context.Context, id, projectID string) (*Pipeline, error) {
	var p Pipeline
	var envJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, target_id, name, source_type, source_url, branch, build_cmd, output_dir, env_vars, trigger_on, cache_dirs, timeout_ms, created_at, updated_at
		 FROM deploy_pipelines WHERE id = ? AND project_id = ?`, id, projectID,
	).Scan(&p.ID, &p.ProjectID, &p.TargetID, &p.Name, &p.SourceType, &p.SourceURL,
		&p.Branch, &p.BuildCmd, &p.OutputDir, &envJSON, &p.TriggerOn, &p.CacheDirs,
		&p.TimeoutMs, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envJSON, &p.EnvVars)
	if p.EnvVars == nil {
		p.EnvVars = map[string]string{}
	}
	if p.TriggerOn == nil {
		p.TriggerOn = json.RawMessage("[]")
	}
	if p.CacheDirs == nil {
		p.CacheDirs = json.RawMessage("[]")
	}
	return &p, nil
}

// ListPipelines returns all pipelines for a project.
func (s *Service) ListPipelines(ctx context.Context, projectID string) ([]*Pipeline, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, target_id, name, source_type, source_url, branch, build_cmd, output_dir, env_vars, trigger_on, cache_dirs, timeout_ms, created_at, updated_at
		 FROM deploy_pipelines WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pipelines []*Pipeline
	for rows.Next() {
		var p Pipeline
		var envJSON []byte
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.TargetID, &p.Name, &p.SourceType, &p.SourceURL,
			&p.Branch, &p.BuildCmd, &p.OutputDir, &envJSON, &p.TriggerOn, &p.CacheDirs,
			&p.TimeoutMs, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(envJSON, &p.EnvVars)
		if p.EnvVars == nil {
			p.EnvVars = map[string]string{}
		}
		if p.TriggerOn == nil {
			p.TriggerOn = json.RawMessage("[]")
		}
		if p.CacheDirs == nil {
			p.CacheDirs = json.RawMessage("[]")
		}
		pipelines = append(pipelines, &p)
	}
	return pipelines, len(pipelines), nil
}

// UpdatePipeline updates a pipeline.
func (s *Service) UpdatePipeline(ctx context.Context, id, projectID, targetID, name, sourceType, sourceURL, branch, buildCmd, outputDir string, envVars map[string]string, triggerOn, cacheDirs json.RawMessage, timeoutMs int) (*Pipeline, error) {
	if envVars == nil {
		envVars = map[string]string{}
	}
	if triggerOn == nil {
		triggerOn = json.RawMessage("[]")
	}
	if cacheDirs == nil {
		cacheDirs = json.RawMessage("[]")
	}
	envJSON, _ := json.Marshal(envVars)

	_, err := s.db.ExecContext(ctx,
		`UPDATE deploy_pipelines SET target_id=?, name=?, source_type=?, source_url=?, branch=?, build_cmd=?, output_dir=?, env_vars=?, trigger_on=?, cache_dirs=?, timeout_ms=?, updated_at=?
		 WHERE id=? AND project_id=?`,
		targetID, name, sourceType, sourceURL, branch, buildCmd, outputDir, envJSON, triggerOn, cacheDirs, timeoutMs, time.Now().UTC(), id, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetPipeline(ctx, id, projectID)
}

// DeletePipeline removes a pipeline.
func (s *Service) DeletePipeline(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM deploy_pipelines WHERE id = ? AND project_id = ?", id, projectID)
	return err
}

// ── Release operations ──

// TriggerPipeline creates a new release for a pipeline and enqueues the build job.
func (s *Service) TriggerPipeline(ctx context.Context, pipelineID, projectID, triggerType, actor, commitSHA string) (*Release, error) {
	p, err := s.GetPipeline(ctx, pipelineID, projectID)
	if err != nil {
		return nil, err
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO deploy_releases (id, project_id, pipeline_id, target_id, status, trigger_type, trigger_actor, commit_sha, started_at, created_at)
		 VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, ?, ?)`,
		id, projectID, p.ID, p.TargetID, triggerType, actor, commitSHA, now, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: trigger pipeline: %w", err)
	}

	rel := &Release{
		ID: id, ProjectID: projectID, PipelineID: p.ID, TargetID: p.TargetID,
		Status: "pending", TriggerType: triggerType, TriggerActor: actor,
		CommitSHA: commitSHA, StartedAt: &now, CreatedAt: now,
	}

	if s.queue != nil {
		s.queue.Push(ctx, "builds", queue.Job{
			ID:   id,
			Type: "deploy_release",
			Payload: map[string]interface{}{
				"releaseId":  id,
				"pipelineId": p.ID,
				"targetId":   p.TargetID,
				"projectId":  projectID,
			},
			CreatedAt: now,
		})
	}

	return rel, nil
}

// GetRelease returns a release by ID.
func (s *Service) GetRelease(ctx context.Context, id, projectID string) (*Release, error) {
	var r Release
	var errStr, commitSHA, buildLog, deployLog, artifactPath sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, pipeline_id, target_id, status, trigger_type, trigger_actor,
		        commit_sha, build_log, deploy_log, artifact_path, duration_ms, error,
		        started_at, completed_at, created_at
		 FROM deploy_releases WHERE id = ? AND project_id = ?`, id, projectID,
	).Scan(&r.ID, &r.ProjectID, &r.PipelineID, &r.TargetID, &r.Status,
		&r.TriggerType, &r.TriggerActor, &commitSHA, &buildLog, &deployLog,
		&artifactPath, &r.DurationMs, &errStr, &startedAt, &completedAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("release not found")
	}
	if err != nil {
		return nil, err
	}
	r.CommitSHA = commitSHA.String
	r.BuildLog = buildLog.String
	r.DeployLog = deployLog.String
	r.ArtifactPath = artifactPath.String
	r.Error = errStr.String
	if startedAt.Valid {
		r.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	return &r, nil
}

// ListReleases returns releases for a project, optionally filtered by pipeline, target, or status.
func (s *Service) ListReleases(ctx context.Context, projectID, pipelineID, targetID, status string) ([]*Release, int, error) {
	query := `SELECT id, project_id, pipeline_id, target_id, status, trigger_type, trigger_actor,
	                 commit_sha, build_log, deploy_log, artifact_path, duration_ms, error,
	                 started_at, completed_at, created_at
	          FROM deploy_releases WHERE project_id = ?`
	args := []interface{}{projectID}

	if pipelineID != "" {
		query += " AND pipeline_id = ?"
		args = append(args, pipelineID)
	}
	if targetID != "" {
		query += " AND target_id = ?"
		args = append(args, targetID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var releases []*Release
	for rows.Next() {
		var r Release
		var errStr, commitSHA, buildLog, deployLog, artifactPath sql.NullString
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.PipelineID, &r.TargetID, &r.Status,
			&r.TriggerType, &r.TriggerActor, &commitSHA, &buildLog, &deployLog,
			&artifactPath, &r.DurationMs, &errStr, &startedAt, &completedAt, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		r.CommitSHA = commitSHA.String
		r.BuildLog = buildLog.String
		r.DeployLog = deployLog.String
		r.ArtifactPath = artifactPath.String
		r.Error = errStr.String
		if startedAt.Valid {
			r.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		releases = append(releases, &r)
	}
	return releases, len(releases), nil
}

// UpdateRelease updates the status, logs, and timing of a release.
func (s *Service) UpdateRelease(ctx context.Context, id string, status string, buildLog, deployLog, artifactPath, releaseErr string, startedAt, completedAt *time.Time, durationMs int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE deploy_releases SET status=?, build_log=?, deploy_log=?, artifact_path=?, error=?, started_at=?, completed_at=?, duration_ms=?
		 WHERE id=?`,
		status, buildLog, deployLog, artifactPath, releaseErr, startedAt, completedAt, durationMs, id)
	return err
}

// RollbackRelease re-deploys a previous successful release by creating a new release that copies its artifact.
func (s *Service) RollbackRelease(ctx context.Context, releaseID, projectID, actor string) (*Release, error) {
	orig, err := s.GetRelease(ctx, releaseID, projectID)
	if err != nil {
		return nil, err
	}
	if orig.Status != "success" {
		return nil, fmt.Errorf("can only rollback a successful release")
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO deploy_releases (id, project_id, pipeline_id, target_id, status, trigger_type, trigger_actor, commit_sha, artifact_path, started_at, created_at)
		 VALUES (?, ?, ?, ?, 'pending', 'rollback', ?, ?, ?, ?, ?)`,
		id, projectID, orig.PipelineID, orig.TargetID, actor, orig.CommitSHA, orig.ArtifactPath, now, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: rollback: %w", err)
	}

	rel := &Release{
		ID: id, ProjectID: projectID, PipelineID: orig.PipelineID, TargetID: orig.TargetID,
		Status: "pending", TriggerType: "rollback", TriggerActor: actor,
		CommitSHA: orig.CommitSHA, ArtifactPath: orig.ArtifactPath,
		StartedAt: &now, CreatedAt: now,
	}

	if s.queue != nil {
		s.queue.Push(ctx, "builds", queue.Job{
			ID:   id,
			Type: "deploy_rollback",
			Payload: map[string]interface{}{
				"releaseId":        id,
				"originalReleaseId": releaseID,
				"pipelineId":       orig.PipelineID,
				"targetId":         orig.TargetID,
				"projectId":        projectID,
				"artifactPath":     orig.ArtifactPath,
			},
			CreatedAt: now,
		})
	}

	return rel, nil
}

// ── Execution operations (serverless invocations) ──

// InvokeTarget creates an execution for a serverless target and enqueues it.
func (s *Service) InvokeTarget(ctx context.Context, targetID, projectID, request, trigger string) (*Execution, error) {
	t, err := s.GetTarget(ctx, targetID, projectID)
	if err != nil {
		return nil, err
	}
	if t.Type != "serverless" {
		return nil, fmt.Errorf("only serverless targets can be invoked")
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO deploy_executions (id, project_id, target_id, status, request, trigger, created_at)
		 VALUES (?, ?, ?, 'pending', ?, ?, ?)`,
		id, projectID, targetID, request, trigger, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: invoke target: %w", err)
	}

	exec := &Execution{
		ID: id, ProjectID: projectID, TargetID: targetID,
		Status: "pending", Request: request, Trigger: trigger,
		CreatedAt: now,
	}

	if s.queue != nil {
		s.queue.Push(ctx, "executions", queue.Job{
			ID:   id,
			Type: "deploy_execution",
			Payload: map[string]interface{}{
				"executionId": id,
				"targetId":    targetID,
				"projectId":   projectID,
				"request":     request,
			},
			CreatedAt: now,
		})
	}

	return exec, nil
}

// GetExecution returns an execution by ID.
func (s *Service) GetExecution(ctx context.Context, id, targetID, projectID string) (*Execution, error) {
	var e Execution
	var resp, stdout, stderr, errTrigger sql.NullString
	var statusCode sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, target_id, status, status_code, request, response, stdout, stderr, duration_ms, trigger, created_at
		 FROM deploy_executions WHERE id = ? AND target_id = ? AND project_id = ?`,
		id, targetID, projectID,
	).Scan(&e.ID, &e.ProjectID, &e.TargetID, &e.Status, &statusCode,
		&e.Request, &resp, &stdout, &stderr, &e.DurationMs, &errTrigger, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	if err != nil {
		return nil, err
	}
	e.StatusCode = int(statusCode.Int64)
	e.Response = resp.String
	e.Stdout = stdout.String
	e.Stderr = stderr.String
	e.Trigger = errTrigger.String
	return &e, nil
}

// ListExecutions returns all executions for a target.
func (s *Service) ListExecutions(ctx context.Context, targetID, projectID string) ([]*Execution, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, target_id, status, status_code, request, response, stdout, stderr, duration_ms, trigger, created_at
		 FROM deploy_executions WHERE target_id = ? AND project_id = ? ORDER BY created_at DESC`,
		targetID, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var execs []*Execution
	for rows.Next() {
		var e Execution
		var resp, stdout, stderr, trig sql.NullString
		var statusCode sql.NullInt64
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.TargetID, &e.Status, &statusCode,
			&e.Request, &resp, &stdout, &stderr, &e.DurationMs, &trig, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.StatusCode = int(statusCode.Int64)
		e.Response = resp.String
		e.Stdout = stdout.String
		e.Stderr = stderr.String
		e.Trigger = trig.String
		execs = append(execs, &e)
	}
	return execs, len(execs), nil
}

// UpdateExecution updates the status and output of an execution.
func (s *Service) UpdateExecution(ctx context.Context, id string, status string, statusCode int, response, stdout, stderr string, durationMs int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE deploy_executions SET status=?, status_code=?, response=?, stdout=?, stderr=?, duration_ms=?
		 WHERE id=?`,
		status, statusCode, response, stdout, stderr, durationMs, id)
	return err
}

// ── Stats ──

// TargetStats holds statistics for a single target.
type TargetStats struct {
	TargetID        string  `json:"targetId"`
	TotalReleases   int     `json:"totalReleases"`
	SuccessReleases int     `json:"successReleases"`
	FailedReleases  int     `json:"failedReleases"`
	TotalExecutions int     `json:"totalExecutions"`
	AvgDurationMs   float64 `json:"avgDurationMs"`
}

// GetTargetStats returns statistics for a specific target.
func (s *Service) GetTargetStats(ctx context.Context, targetID, projectID string) (*TargetStats, error) {
	stats := &TargetStats{TargetID: targetID}

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status='success' THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END),0)
		 FROM deploy_releases WHERE target_id = ? AND project_id = ?`, targetID, projectID,
	).Scan(&stats.TotalReleases, &stats.SuccessReleases, &stats.FailedReleases)

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(AVG(duration_ms),0)
		 FROM deploy_executions WHERE target_id = ? AND project_id = ?`, targetID, projectID,
	).Scan(&stats.TotalExecutions, &stats.AvgDurationMs)

	return stats, nil
}

// AggregateStats holds project-wide deploy statistics.
type AggregateStats struct {
	TotalTargets    int     `json:"totalTargets"`
	TotalPipelines  int     `json:"totalPipelines"`
	TotalReleases   int     `json:"totalReleases"`
	SuccessReleases int     `json:"successReleases"`
	FailedReleases  int     `json:"failedReleases"`
	TotalExecutions int     `json:"totalExecutions"`
	AvgBuildMs      float64 `json:"avgBuildMs"`
}

// DetailedStatsBucket represents a single time bucket in the detailed stats time series.
type DetailedStatsBucket struct {
	Timestamp      time.Time `json:"timestamp"`
	ColdStartCount int       `json:"coldStartCount"`
	AvgDurationMs  float64   `json:"avgDurationMs"`
	ErrorCount     int       `json:"errorCount"`
	TotalCount     int       `json:"totalCount"`
}

// DetailedStats holds detailed time-series statistics for a target.
type DetailedStats struct {
	TargetID   string                `json:"targetId"`
	Range      string                `json:"range"`
	Granularity string               `json:"granularity"`
	Buckets    []DetailedStatsBucket `json:"buckets"`
}

// GetTargetDetailedStats returns time-series execution statistics for a target,
// grouped by hour or day depending on the requested range.
func (s *Service) GetTargetDetailedStats(ctx context.Context, targetID, projectID, timeRange string) (*DetailedStats, error) {
	// Determine time window and granularity
	var since time.Time
	var granularity, dateFmt string
	now := time.Now().UTC()

	switch timeRange {
	case "7d":
		since = now.AddDate(0, 0, -7)
		granularity = "day"
		dateFmt = "%Y-%m-%d"
	case "30d":
		since = now.AddDate(0, 0, -30)
		granularity = "day"
		dateFmt = "%Y-%m-%d"
	default: // "24h"
		since = now.Add(-24 * time.Hour)
		granularity = "hour"
		dateFmt = "%Y-%m-%d %H:00:00"
		timeRange = "24h"
	}

	query := fmt.Sprintf(
		`SELECT DATE_FORMAT(created_at, '%s') AS bucket,
		        COUNT(*) AS total_count,
		        COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS error_count,
		        COALESCE(SUM(CASE WHEN duration_ms > 0 AND duration_ms > 1000 THEN 1 ELSE 0 END), 0) AS cold_start_count,
		        COALESCE(AVG(CASE WHEN duration_ms > 0 THEN duration_ms END), 0) AS avg_duration_ms
		 FROM deploy_executions
		 WHERE target_id = ? AND project_id = ? AND created_at >= ?
		 GROUP BY bucket
		 ORDER BY bucket ASC`, dateFmt)

	rows, err := s.db.QueryContext(ctx, query, targetID, projectID, since)
	if err != nil {
		return nil, fmt.Errorf("deploy: detailed stats: %w", err)
	}
	defer rows.Close()

	var buckets []DetailedStatsBucket
	for rows.Next() {
		var b DetailedStatsBucket
		var bucketStr string
		if err := rows.Scan(&bucketStr, &b.TotalCount, &b.ErrorCount, &b.ColdStartCount, &b.AvgDurationMs); err != nil {
			return nil, err
		}
		if granularity == "hour" {
			b.Timestamp, _ = time.Parse("2006-01-02 15:04:05", bucketStr)
		} else {
			b.Timestamp, _ = time.Parse("2006-01-02", bucketStr)
		}
		buckets = append(buckets, b)
	}

	if buckets == nil {
		buckets = []DetailedStatsBucket{}
	}

	return &DetailedStats{
		TargetID:    targetID,
		Range:       timeRange,
		Granularity: granularity,
		Buckets:     buckets,
	}, nil
}

// GetAggregateStats returns project-wide deploy statistics.
func (s *Service) GetAggregateStats(ctx context.Context, projectID string) (*AggregateStats, error) {
	stats := &AggregateStats{}

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM deploy_targets WHERE project_id = ?`, projectID,
	).Scan(&stats.TotalTargets)

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM deploy_pipelines WHERE project_id = ?`, projectID,
	).Scan(&stats.TotalPipelines)

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status='success' THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END),0), COALESCE(AVG(duration_ms),0)
		 FROM deploy_releases WHERE project_id = ?`, projectID,
	).Scan(&stats.TotalReleases, &stats.SuccessReleases, &stats.FailedReleases, &stats.AvgBuildMs)

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM deploy_executions WHERE project_id = ?`, projectID,
	).Scan(&stats.TotalExecutions)

	return stats, nil
}
