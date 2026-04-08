// Package deploy implements Applad's deployment service:
// targets, pipelines, releases, and execution management.
package deploy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		`INSERT INTO deploy_executions (id, project_id, target_id, status, request, trigger_source, created_at)
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
		`SELECT id, project_id, target_id, status, status_code, request, response, stdout, stderr, duration_ms, trigger_source, created_at
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
		`SELECT id, project_id, target_id, status, status_code, request, response, stdout, stderr, duration_ms, trigger_source, created_at
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

// ── Custom Domain models + operations (Web Deploy Targets) ──

// CustomDomain represents a custom domain attached to a web deploy target.
type CustomDomain struct {
	ID           string     `json:"$id"`
	ProjectID    string     `json:"projectId"`
	TargetID     string     `json:"targetId"`
	Domain       string     `json:"domain"`
	Verification string     `json:"verification"`
	Verified     bool       `json:"verified"`
	SSLStatus    string     `json:"sslStatus"`
	SSLExpiresAt *time.Time `json:"sslExpiresAt"`
	CreatedAt    time.Time  `json:"$createdAt"`
}

// CreateCustomDomain adds a custom domain to a web deploy target.
func (s *Service) CreateCustomDomain(ctx context.Context, projectID, targetID, domain string) (*CustomDomain, error) {
	id := uid.New("unique()")
	verification := uid.RandomHex(16)
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO custom_domains (id, project_id, target_id, domain, verification, verified, ssl_status, created_at)
		 VALUES (?, ?, ?, ?, ?, FALSE, 'pending', ?)`,
		id, projectID, targetID, domain, verification, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: create custom domain: %w", err)
	}
	return &CustomDomain{
		ID: id, ProjectID: projectID, TargetID: targetID,
		Domain: domain, Verification: verification, Verified: false,
		SSLStatus: "pending", CreatedAt: now,
	}, nil
}

// VerifyDomain performs DNS verification and marks the domain as verified.
func (s *Service) VerifyDomain(ctx context.Context, domainID string) (*CustomDomain, error) {
	var domain, verification string
	err := s.db.QueryRowContext(ctx,
		`SELECT domain, verification FROM custom_domains WHERE id = ?`, domainID).Scan(&domain, &verification)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	// Check TXT records on the domain and _applad-verification subdomain
	verified := false
	for _, host := range []string{domain, "_applad-verification." + domain} {
		txts, _ := net.LookupTXT(host)
		for _, txt := range txts {
			if strings.Contains(txt, verification) {
				verified = true
				break
			}
		}
		if verified {
			break
		}
	}
	// Check CNAME as fallback
	if !verified {
		cname, _ := net.LookupCNAME(domain)
		if strings.Contains(cname, verification) {
			verified = true
		}
	}
	if !verified {
		return nil, fmt.Errorf("DNS verification failed: add TXT record for %s or _applad-verification.%s containing %s", domain, domain, verification)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE custom_domains SET verified = TRUE, ssl_status = 'active' WHERE id = ?`, domainID)
	if err != nil {
		return nil, fmt.Errorf("deploy: verify domain: %w", err)
	}

	var d CustomDomain
	var sslExpires sql.NullTime
	err = s.db.QueryRowContext(ctx,
		`SELECT id, project_id, target_id, domain, verification, verified, ssl_status, ssl_expires_at, created_at
		 FROM custom_domains WHERE id = ?`, domainID,
	).Scan(&d.ID, &d.ProjectID, &d.TargetID, &d.Domain, &d.Verification, &d.Verified,
		&d.SSLStatus, &sslExpires, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("domain not found")
	}
	if err != nil {
		return nil, err
	}
	if sslExpires.Valid {
		d.SSLExpiresAt = &sslExpires.Time
	}
	return &d, nil
}

// ListDomains returns all custom domains for a target.
func (s *Service) ListDomains(ctx context.Context, projectID, targetID string) ([]*CustomDomain, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, target_id, domain, verification, verified, ssl_status, ssl_expires_at, created_at
		 FROM custom_domains WHERE project_id = ? AND target_id = ? ORDER BY created_at DESC`,
		projectID, targetID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var domains []*CustomDomain
	for rows.Next() {
		var d CustomDomain
		var sslExpires sql.NullTime
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.TargetID, &d.Domain, &d.Verification,
			&d.Verified, &d.SSLStatus, &sslExpires, &d.CreatedAt); err != nil {
			return nil, 0, err
		}
		if sslExpires.Valid {
			d.SSLExpiresAt = &sslExpires.Time
		}
		domains = append(domains, &d)
	}
	return domains, len(domains), nil
}

// DeleteDomain removes a custom domain.
func (s *Service) DeleteDomain(ctx context.Context, domainID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM custom_domains WHERE id = ?", domainID)
	return err
}

// ServeStaticFile reads and returns a static file from the target's output directory.
func (s *Service) ServeStaticFile(ctx context.Context, targetID, filePath string) ([]byte, string, error) {
	var outputDir string
	err := s.db.QueryRowContext(ctx,
		`SELECT p.output_dir FROM deploy_pipelines p
		 JOIN deploy_targets t ON t.id = p.target_id
		 WHERE p.target_id = ? ORDER BY p.created_at DESC LIMIT 1`, targetID,
	).Scan(&outputDir)
	if err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("no pipeline found for target")
	}
	if err != nil {
		return nil, "", err
	}
	if outputDir == "" {
		outputDir = "dist"
	}

	// Sanitize path to prevent directory traversal
	cleanPath := filepath.Clean("/" + filePath)
	fullPath := filepath.Join(outputDir, cleanPath)

	// Ensure the resolved path is still within outputDir
	if !strings.HasPrefix(fullPath, filepath.Clean(outputDir)) {
		return nil, "", fmt.Errorf("invalid path")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		// Try index.html for SPA fallback
		indexPath := filepath.Join(outputDir, "index.html")
		data, err = os.ReadFile(indexPath)
		if err != nil {
			return nil, "", fmt.Errorf("file not found: %s", filePath)
		}
		return data, "text/html; charset=utf-8", nil
	}

	// Detect content type from extension
	contentType := "application/octet-stream"
	ext := strings.ToLower(filepath.Ext(fullPath))
	switch ext {
	case ".html", ".htm":
		contentType = "text/html; charset=utf-8"
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js", ".mjs":
		contentType = "application/javascript"
	case ".json":
		contentType = "application/json"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".svg":
		contentType = "image/svg+xml"
	case ".ico":
		contentType = "image/x-icon"
	case ".woff2":
		contentType = "font/woff2"
	case ".woff":
		contentType = "font/woff"
	case ".ttf":
		contentType = "font/ttf"
	case ".txt":
		contentType = "text/plain"
	case ".xml":
		contentType = "application/xml"
	case ".webp":
		contentType = "image/webp"
	case ".mp4":
		contentType = "video/mp4"
	case ".webm":
		contentType = "video/webm"
	case ".wasm":
		contentType = "application/wasm"
	}
	return data, contentType, nil
}

// ── Registry Image models + operations (Container Deploy Targets) ──

// RegistryImage represents a container image pushed to a target's registry.
type RegistryImage struct {
	ID         string    `json:"$id"`
	ProjectID  string    `json:"projectId"`
	TargetID   string    `json:"targetId"`
	Repository string    `json:"repository"`
	Tag        string    `json:"tag"`
	Digest     string    `json:"digest"`
	SizeBytes  int64     `json:"sizeBytes"`
	Platform   string    `json:"platform"`
	CreatedAt  time.Time `json:"$createdAt"`
}

// PushImage records a container image that was pushed to the registry.
func (s *Service) PushImage(ctx context.Context, targetID, projectID, repository, tag, digest string, sizeBytes int64, platform string) (*RegistryImage, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO registry_images (id, project_id, target_id, repository, tag, digest, size_bytes, platform, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, targetID, repository, tag, digest, sizeBytes, platform, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: push image: %w", err)
	}
	return &RegistryImage{
		ID: id, ProjectID: projectID, TargetID: targetID,
		Repository: repository, Tag: tag, Digest: digest,
		SizeBytes: sizeBytes, Platform: platform, CreatedAt: now,
	}, nil
}

// ListImages returns all registry images for a target.
func (s *Service) ListImages(ctx context.Context, targetID, projectID string) ([]*RegistryImage, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, target_id, repository, tag, digest, size_bytes, platform, created_at
		 FROM registry_images WHERE target_id = ? AND project_id = ? ORDER BY created_at DESC`,
		targetID, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var images []*RegistryImage
	for rows.Next() {
		var img RegistryImage
		if err := rows.Scan(&img.ID, &img.ProjectID, &img.TargetID, &img.Repository,
			&img.Tag, &img.Digest, &img.SizeBytes, &img.Platform, &img.CreatedAt); err != nil {
			return nil, 0, err
		}
		images = append(images, &img)
	}
	return images, len(images), nil
}

// DeleteImage removes a registry image record.
func (s *Service) DeleteImage(ctx context.Context, imageID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM registry_images WHERE id = ?", imageID)
	return err
}

// ── Build Agent models + operations ──

// Agent represents a build agent that can execute deploy jobs.
type Agent struct {
	ID            string     `json:"$id"`
	ProjectID     string     `json:"projectId"`
	Name          string     `json:"name"`
	Token         string     `json:"token,omitempty"`
	Labels        []string   `json:"labels"`
	Status        string     `json:"status"` // online, offline
	LastHeartbeat *time.Time `json:"lastHeartbeat"`
	CurrentJobID  string     `json:"currentJobId,omitempty"`
	OS            string     `json:"os"`
	Arch          string     `json:"arch"`
	CreatedAt     time.Time  `json:"$createdAt"`
}

// RegisterAgent creates a new build agent with a generated authentication token.
func (s *Service) RegisterAgent(ctx context.Context, projectID, name string, labels []string, os, arch string) (*Agent, error) {
	id := uid.New("unique()")
	token := uid.RandomHex(32)
	now := time.Now().UTC()

	if labels == nil {
		labels = []string{}
	}
	labelsJSON, _ := json.Marshal(labels)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO build_agents (id, project_id, name, token, labels, status, os, arch, created_at)
		 VALUES (?, ?, ?, ?, ?, 'offline', ?, ?, ?)`,
		id, projectID, name, token, labelsJSON, os, arch, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: register agent: %w", err)
	}
	return &Agent{
		ID: id, ProjectID: projectID, Name: name, Token: token,
		Labels: labels, Status: "offline", OS: os, Arch: arch,
		CreatedAt: now,
	}, nil
}

// ListAgents returns all build agents for a project.
func (s *Service) ListAgents(ctx context.Context, projectID string) ([]*Agent, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, labels, status, last_heartbeat, current_job_id, os, arch, created_at
		 FROM build_agents WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		var a Agent
		var labelsJSON []byte
		var lastHB sql.NullTime
		var currentJob sql.NullString
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &labelsJSON, &a.Status,
			&lastHB, &currentJob, &a.OS, &a.Arch, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(labelsJSON, &a.Labels)
		if a.Labels == nil {
			a.Labels = []string{}
		}
		if lastHB.Valid {
			a.LastHeartbeat = &lastHB.Time
		}
		a.CurrentJobID = currentJob.String
		agents = append(agents, &a)
	}
	return agents, len(agents), nil
}

// HeartbeatAgent updates an agent's heartbeat timestamp and sets its status to online.
func (s *Service) HeartbeatAgent(ctx context.Context, agentID string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE build_agents SET last_heartbeat = ?, status = 'online' WHERE id = ?`,
		now, agentID)
	return err
}

// DeleteAgent removes a build agent.
func (s *Service) DeleteAgent(ctx context.Context, agentID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM build_agents WHERE id = ?", agentID)
	return err
}

// AssignJob finds an available agent with a matching label and assigns it a job.
func (s *Service) AssignJob(ctx context.Context, agentLabel, jobID string) (*Agent, error) {
	// Find an online agent with the matching label that has no current job.
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, labels, status, last_heartbeat, current_job_id, os, arch, created_at
		 FROM build_agents WHERE status = 'online' AND (current_job_id IS NULL OR current_job_id = '')
		 ORDER BY last_heartbeat DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a Agent
		var labelsJSON []byte
		var lastHB sql.NullTime
		var currentJob sql.NullString
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &labelsJSON, &a.Status,
			&lastHB, &currentJob, &a.OS, &a.Arch, &a.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(labelsJSON, &a.Labels)
		if a.Labels == nil {
			a.Labels = []string{}
		}
		if lastHB.Valid {
			a.LastHeartbeat = &lastHB.Time
		}

		// Check if this agent has the requested label.
		for _, l := range a.Labels {
			if l == agentLabel {
				// Assign the job.
				_, err := s.db.ExecContext(ctx,
					`UPDATE build_agents SET current_job_id = ? WHERE id = ?`, jobID, a.ID)
				if err != nil {
					return nil, fmt.Errorf("deploy: assign job: %w", err)
				}
				a.CurrentJobID = jobID
				return &a, nil
			}
		}
	}

	return nil, fmt.Errorf("no available agent with label %q", agentLabel)
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
	Requests        int     `json:"requests"`
	BuildMinutes    float64 `json:"buildMinutes"`
	Bandwidth       int64   `json:"bandwidth"`
}

// GetTargetStats returns statistics for a specific target.
func (s *Service) GetTargetStats(ctx context.Context, targetID, projectID string) (*TargetStats, error) {
	stats := &TargetStats{TargetID: targetID}

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status='success' THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(duration_ms),0)/60000.0
		 FROM deploy_releases WHERE target_id = ? AND project_id = ?`, targetID, projectID,
	).Scan(&stats.TotalReleases, &stats.SuccessReleases, &stats.FailedReleases, &stats.BuildMinutes)

	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(AVG(duration_ms),0)
		 FROM deploy_executions WHERE target_id = ? AND project_id = ?`, targetID, projectID,
	).Scan(&stats.TotalExecutions, &stats.AvgDurationMs)

	stats.Requests = stats.TotalExecutions

	return stats, nil
}

// GetTargetLogs returns the latest build/deploy logs for a target as structured log entries.
func (s *Service) GetTargetLogs(ctx context.Context, targetID, projectID string) (map[string]interface{}, error) {
	var releaseID, buildLog, deployLog string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, COALESCE(build_log,''), COALESCE(deploy_log,'')
		 FROM deploy_releases WHERE target_id = ? AND project_id = ?
		 ORDER BY created_at DESC LIMIT 1`, targetID, projectID,
	).Scan(&releaseID, &buildLog, &deployLog)

	if err == sql.ErrNoRows {
		return map[string]interface{}{"logs": []interface{}{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("deploy: get target logs: %w", err)
	}

	var logs []map[string]interface{}
	combined := buildLog
	if deployLog != "" {
		if combined != "" {
			combined += "\n"
		}
		combined += deployLog
	}
	now := time.Now().UTC()
	for i, line := range strings.Split(combined, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"$id":        fmt.Sprintf("%s-%d", releaseID, i),
			"path":       "/",
			"method":     "LOG",
			"statusCode": 200,
			"duration":   0,
			"message":    line,
			"$createdAt": now,
		})
	}
	if logs == nil {
		logs = []map[string]interface{}{}
	}
	return map[string]interface{}{"logs": logs}, nil
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

// ── Deploy Template models + operations ──

// DeployTemplate represents a pre-built deploy template.
type DeployTemplate struct {
	ID          string            `json:"$id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"` // sites, containers, mobile, desktop
	Framework   string            `json:"framework"`
	UseCase     string            `json:"useCase"`
	RepoURL     string            `json:"repoUrl"`
	Branch      string            `json:"branch"`
	BuildCmd    string            `json:"buildCmd"`
	OutputDir   string            `json:"outputDir"`
	InstallCmd  string            `json:"installCmd"`
	EnvVars     map[string]string `json:"envVars"`
	Icon        string            `json:"icon"`
	Popularity  int               `json:"popularity"`
	CreatedAt   time.Time         `json:"$createdAt"`
}

// ListDeployTemplates returns templates filtered by category and optionally by framework.
func (s *Service) ListDeployTemplates(ctx context.Context, category, framework string) ([]*DeployTemplate, int, error) {
	query := `SELECT id, name, description, category, framework, use_case, repo_url, branch, build_cmd, output_dir, install_cmd, env_vars, icon, popularity, created_at
	          FROM deploy_templates WHERE 1=1`
	args := []interface{}{}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if framework != "" {
		query += " AND framework = ?"
		args = append(args, framework)
	}
	query += " ORDER BY popularity DESC, created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var templates []*DeployTemplate
	for rows.Next() {
		var t DeployTemplate
		var envJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.Framework,
			&t.UseCase, &t.RepoURL, &t.Branch, &t.BuildCmd, &t.OutputDir, &t.InstallCmd,
			&envJSON, &t.Icon, &t.Popularity, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(envJSON, &t.EnvVars)
		if t.EnvVars == nil {
			t.EnvVars = map[string]string{}
		}
		templates = append(templates, &t)
	}
	return templates, len(templates), nil
}

// GetDeployTemplate returns a single deploy template by ID.
func (s *Service) GetDeployTemplate(ctx context.Context, templateID string) (*DeployTemplate, error) {
	var t DeployTemplate
	var envJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, category, framework, use_case, repo_url, branch, build_cmd, output_dir, install_cmd, env_vars, icon, popularity, created_at
		 FROM deploy_templates WHERE id = ?`, templateID,
	).Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.Framework,
		&t.UseCase, &t.RepoURL, &t.Branch, &t.BuildCmd, &t.OutputDir, &t.InstallCmd,
		&envJSON, &t.Icon, &t.Popularity, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envJSON, &t.EnvVars)
	if t.EnvVars == nil {
		t.EnvVars = map[string]string{}
	}
	return &t, nil
}

// ── Git Connection models + operations ──

// GitConnection represents an OAuth-based connection to a Git provider.
type GitConnection struct {
	ID             string     `json:"$id"`
	ProjectID      string     `json:"projectId"`
	Provider       string     `json:"provider"` // github, gitlab, bitbucket
	InstallationID string     `json:"installationId"`
	AccessToken    string     `json:"accessToken,omitempty"`
	RefreshToken   string     `json:"refreshToken,omitempty"`
	AccountName    string     `json:"accountName"`
	AccountType    string     `json:"accountType"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	CreatedAt      time.Time  `json:"$createdAt"`
	UpdatedAt      time.Time  `json:"$updatedAt"`
}

// GitRepository represents a repository returned from the Git provider API.
type GitRepository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"fullName"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"defaultBranch"`
	CloneURL      string `json:"cloneUrl"`
	HTMLURL       string `json:"htmlUrl"`
	Language      string `json:"language"`
	UpdatedAt     string `json:"updatedAt"`
}

// CreateGitConnection creates a new git provider connection.
func (s *Service) CreateGitConnection(ctx context.Context, projectID, provider, installationID, accessToken, refreshToken, accountName, accountType string) (*GitConnection, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO git_connections (id, project_id, provider, installation_id, access_token, refresh_token, account_name, account_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, provider, installationID, accessToken, refreshToken, accountName, accountType, now, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: create git connection: %w", err)
	}
	return &GitConnection{
		ID: id, ProjectID: projectID, Provider: provider,
		InstallationID: installationID, AccessToken: accessToken,
		RefreshToken: refreshToken, AccountName: accountName,
		AccountType: accountType, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ListGitConnections returns all git connections for a project.
func (s *Service) ListGitConnections(ctx context.Context, projectID string) ([]*GitConnection, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, provider, installation_id, access_token, refresh_token, account_name, account_type, expires_at, created_at, updated_at
		 FROM git_connections WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var connections []*GitConnection
	for rows.Next() {
		var c GitConnection
		var expiresAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Provider, &c.InstallationID,
			&c.AccessToken, &c.RefreshToken, &c.AccountName, &c.AccountType,
			&expiresAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if expiresAt.Valid {
			c.ExpiresAt = &expiresAt.Time
		}
		connections = append(connections, &c)
	}
	return connections, len(connections), nil
}

// DeleteGitConnection removes a git connection.
func (s *Service) DeleteGitConnection(ctx context.Context, connectionID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM git_connections WHERE id = ?", connectionID)
	return err
}

// ListRepositories fetches repositories from the Git provider using the connection's access token.
func (s *Service) ListRepositories(ctx context.Context, connectionID string) ([]*GitRepository, error) {
	var accessToken, provider string
	err := s.db.QueryRowContext(ctx,
		`SELECT access_token, provider FROM git_connections WHERE id = ?`, connectionID,
	).Scan(&accessToken, &provider)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("git connection not found")
	}
	if err != nil {
		return nil, err
	}

	if accessToken == "" {
		return nil, fmt.Errorf("git connection has no access token")
	}

	// Call GitHub API to list installation repositories
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/installation/repositories", nil)
	if err != nil {
		return nil, fmt.Errorf("deploy: list repos: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deploy: list repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deploy: github API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Repositories []struct {
			ID            int64  `json:"id"`
			Name          string `json:"name"`
			FullName      string `json:"full_name"`
			Private       bool   `json:"private"`
			DefaultBranch string `json:"default_branch"`
			CloneURL      string `json:"clone_url"`
			HTMLURL       string `json:"html_url"`
			Language      string `json:"language"`
			UpdatedAt     string `json:"updated_at"`
		} `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("deploy: parse github response: %w", err)
	}

	var repos []*GitRepository
	for _, r := range result.Repositories {
		repos = append(repos, &GitRepository{
			ID: r.ID, Name: r.Name, FullName: r.FullName,
			Private: r.Private, DefaultBranch: r.DefaultBranch,
			CloneURL: r.CloneURL, HTMLURL: r.HTMLURL,
			Language: r.Language, UpdatedAt: r.UpdatedAt,
		})
	}
	return repos, nil
}

// ── Environment models + operations ──

// Environment represents a deploy environment (e.g., production, staging, development).
type Environment struct {
	ID        string            `json:"$id"`
	ProjectID string            `json:"projectId"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Branch    string            `json:"branch"`
	Domain    string            `json:"domain"`
	EnvVars   map[string]string `json:"envVars"`
	IsDefault bool              `json:"isDefault"`
	CreatedAt time.Time         `json:"$createdAt"`
	UpdatedAt time.Time         `json:"$updatedAt"`
}

// CreateEnvironment creates a new environment for a project.
func (s *Service) CreateEnvironment(ctx context.Context, projectID, name, slug, branch, domain string, envVars map[string]string) (*Environment, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	if envVars == nil {
		envVars = map[string]string{}
	}
	envJSON, _ := json.Marshal(envVars)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO environments (id, project_id, name, slug, branch, domain, env_vars, is_default, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, FALSE, ?, ?)`,
		id, projectID, name, slug, branch, domain, envJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: create environment: %w", err)
	}
	return &Environment{
		ID: id, ProjectID: projectID, Name: name, Slug: slug,
		Branch: branch, Domain: domain, EnvVars: envVars,
		IsDefault: false, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ListEnvironments returns all environments for a project.
func (s *Service) ListEnvironments(ctx context.Context, projectID string) ([]*Environment, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, slug, branch, domain, env_vars, is_default, created_at, updated_at
		 FROM environments WHERE project_id = ? ORDER BY is_default DESC, created_at ASC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var envs []*Environment
	for rows.Next() {
		var e Environment
		var envJSON []byte
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Slug, &e.Branch,
			&e.Domain, &envJSON, &e.IsDefault, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(envJSON, &e.EnvVars)
		if e.EnvVars == nil {
			e.EnvVars = map[string]string{}
		}
		envs = append(envs, &e)
	}
	return envs, len(envs), nil
}

// GetEnvironment returns a single environment by ID.
func (s *Service) GetEnvironment(ctx context.Context, environmentID, projectID string) (*Environment, error) {
	var e Environment
	var envJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, slug, branch, domain, env_vars, is_default, created_at, updated_at
		 FROM environments WHERE id = ? AND project_id = ?`, environmentID, projectID,
	).Scan(&e.ID, &e.ProjectID, &e.Name, &e.Slug, &e.Branch,
		&e.Domain, &envJSON, &e.IsDefault, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envJSON, &e.EnvVars)
	if e.EnvVars == nil {
		e.EnvVars = map[string]string{}
	}
	return &e, nil
}

// UpdateEnvironment updates an environment.
func (s *Service) UpdateEnvironment(ctx context.Context, environmentID, name, branch, domain string, envVars map[string]string) (*Environment, error) {
	if envVars == nil {
		envVars = map[string]string{}
	}
	envJSON, _ := json.Marshal(envVars)

	_, err := s.db.ExecContext(ctx,
		`UPDATE environments SET name=?, branch=?, domain=?, env_vars=?, updated_at=?
		 WHERE id=?`,
		name, branch, domain, envJSON, time.Now().UTC(), environmentID)
	if err != nil {
		return nil, err
	}

	// Retrieve the updated environment
	var e Environment
	var envJSONOut []byte
	err = s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, slug, branch, domain, env_vars, is_default, created_at, updated_at
		 FROM environments WHERE id = ?`, environmentID,
	).Scan(&e.ID, &e.ProjectID, &e.Name, &e.Slug, &e.Branch,
		&e.Domain, &envJSONOut, &e.IsDefault, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envJSONOut, &e.EnvVars)
	if e.EnvVars == nil {
		e.EnvVars = map[string]string{}
	}
	return &e, nil
}

// DeleteEnvironment deletes an environment. The default environment cannot be deleted.
func (s *Service) DeleteEnvironment(ctx context.Context, environmentID string) error {
	var isDefault bool
	err := s.db.QueryRowContext(ctx,
		`SELECT is_default FROM environments WHERE id = ?`, environmentID,
	).Scan(&isDefault)
	if err == sql.ErrNoRows {
		return fmt.Errorf("environment not found")
	}
	if err != nil {
		return err
	}
	if isDefault {
		return fmt.Errorf("cannot delete the default environment")
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM environments WHERE id = ?", environmentID)
	return err
}

// CreateDefaultEnvironments creates the 3 default environments for a new project.
func (s *Service) CreateDefaultEnvironments(ctx context.Context, projectID string) error {
	now := time.Now().UTC()
	emptyEnv, _ := json.Marshal(map[string]string{})

	defaults := []struct {
		name      string
		slug      string
		branch    string
		isDefault bool
	}{
		{"Production", "production", "main", true},
		{"Staging", "staging", "staging", false},
		{"Development", "development", "develop", false},
	}

	for _, d := range defaults {
		id := uid.New("unique()")
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO environments (id, project_id, name, slug, branch, domain, env_vars, is_default, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?)`,
			id, projectID, d.name, d.slug, d.branch, emptyEnv, d.isDefault, now, now)
		if err != nil {
			return fmt.Errorf("deploy: create default environment %s: %w", d.name, err)
		}
	}
	return nil
}
