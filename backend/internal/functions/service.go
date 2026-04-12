// Package functions implements Applad's serverless function execution service:
// function management, execution tracking, and build queue integration.
package functions

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

// Function represents a serverless function configuration.
type Function struct {
	ID         string            `json:"$id"`
	ProjectID  string            `json:"projectId"`
	Name       string            `json:"name"`
	Runtime    string            `json:"runtime"`    // node-18, python-3, go-1
	Entrypoint string            `json:"entrypoint"` // e.g. index.handler
	Timeout    int               `json:"timeout"`    // seconds
	EnvVars    map[string]string `json:"envVars"`
	SourceType string            `json:"sourceType"` // inline or git
	Source     string            `json:"source"`     // stored code (inline only)
	Repository string            `json:"repository"` // git repo URL (git only)
	Branch     string            `json:"branch"`     // git branch (git only)
	Cron       string            `json:"cron"`       // cron schedule expression (empty = manual only)
	Status     string            `json:"status"`     // active, inactive, building
	CreatedAt  time.Time         `json:"$createdAt"`
	UpdatedAt  time.Time         `json:"$updatedAt"`
}

// FunctionExecution represents a single execution of a function.
type FunctionExecution struct {
	ID         string    `json:"$id"`
	FunctionID string    `json:"functionId"`
	ProjectID  string    `json:"projectId"`
	Status     string    `json:"status"` // pending, processing, completed, failed
	Output     string    `json:"output"`
	Errors     string    `json:"errors"`
	Duration   float64   `json:"duration"` // seconds
	CreatedAt  time.Time `json:"$createdAt"`
}

// Service handles functions business logic.
type Service struct {
	db    *db.DB
	queue *queue.Queue
}

// NewService creates a new functions Service.
func NewService(database *db.DB, q *queue.Queue) *Service {
	return &Service{db: database, queue: q}
}

// Create creates a new function.
func (s *Service) Create(ctx context.Context, projectID, name, runtime, entrypoint string, timeout int, envVars map[string]string, sourceType, source, repository, branch, cron string) (*Function, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	envJSON, _ := json.Marshal(envVars)
	if sourceType == "" {
		sourceType = "inline"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO functions (id, project_id, name, runtime, entrypoint, timeout, env_vars, source_type, source, repository, branch, cron, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'building', $13, $14)`,
		id, projectID, name, runtime, entrypoint, timeout, envJSON, sourceType, source, repository, branch, cron, now, now)
	if err != nil {
		return nil, fmt.Errorf("functions: create: %w", err)
	}

	// Pre-warm: push a build job so the image + warm container are ready before first invocation
	hasSource := (sourceType == "git" && repository != "") || (sourceType == "inline" && source != "")
	if s.queue != nil && hasSource {
		s.queue.Push(ctx, "builds", queue.Job{
			ID:   uid.New("unique()"),
			Type: "function_build",
			Payload: map[string]interface{}{
				"functionId": id,
				"projectId":  projectID,
				"runtime":    runtime,
				"entrypoint": entrypoint,
				"sourceType": sourceType,
				"source":     source,
				"repository": repository,
				"branch":     branch,
			},
			CreatedAt: now,
		})
	}

	return &Function{
		ID: id, ProjectID: projectID, Name: name,
		Runtime: runtime, Entrypoint: entrypoint, Timeout: timeout,
		EnvVars: envVars, SourceType: sourceType, Source: source,
		Repository: repository, Branch: branch, Cron: cron, Status: "building",
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Get returns a function by ID.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Function, error) {
	var f Function
	var envJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, project_id, name, runtime, entrypoint, timeout, env_vars, COALESCE(source_type,'inline'), COALESCE(source,''), COALESCE(repository,''), COALESCE(branch,''), COALESCE(cron,''), status, created_at, updated_at FROM functions WHERE id = $1 AND project_id = $2",
		id, projectID).Scan(&f.ID, &f.ProjectID, &f.Name, &f.Runtime, &f.Entrypoint, &f.Timeout, &envJSON, &f.SourceType, &f.Source, &f.Repository, &f.Branch, &f.Cron, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("function not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(envJSON, &f.EnvVars)
	if f.EnvVars == nil {
		f.EnvVars = map[string]string{}
	}
	return &f, nil
}

// List returns all functions for a project.
func (s *Service) List(ctx context.Context, projectID string) ([]*Function, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, runtime, entrypoint, timeout, env_vars, COALESCE(source_type,'inline'), COALESCE(source,''), COALESCE(repository,''), COALESCE(branch,''), COALESCE(cron,''), status, created_at, updated_at FROM functions WHERE project_id = $1 ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var functions []*Function
	for rows.Next() {
		var f Function
		var envJSON []byte
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Runtime, &f.Entrypoint, &f.Timeout, &envJSON, &f.SourceType, &f.Source, &f.Repository, &f.Branch, &f.Cron, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(envJSON, &f.EnvVars)
		if f.EnvVars == nil {
			f.EnvVars = map[string]string{}
		}
		functions = append(functions, &f)
	}
	return functions, len(functions), nil
}

// Update updates an existing function.
func (s *Service) Update(ctx context.Context, id, projectID string, name, runtime, entrypoint string, timeout int, envVars map[string]string, sourceType, source, repository, branch, cron string) (*Function, error) {
	now := time.Now().UTC()
	envJSON, _ := json.Marshal(envVars)
	if sourceType == "" {
		sourceType = "inline"
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE functions SET name = $1, runtime = $2, entrypoint = $3, timeout = $4, env_vars = $5, source_type = $6, source = $7, repository = $8, branch = $9, cron = $10, status = 'building', updated_at = $11
		 WHERE id = $12 AND project_id = $13`,
		name, runtime, entrypoint, timeout, envJSON, sourceType, source, repository, branch, cron, now, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("functions: update: %w", err)
	}

	// Re-build and pre-warm on source/runtime change
	hasSource := (sourceType == "git" && repository != "") || (sourceType == "inline" && source != "")
	if s.queue != nil && hasSource {
		s.queue.Push(ctx, "builds", queue.Job{
			ID:   uid.New("unique()"),
			Type: "function_build",
			Payload: map[string]interface{}{
				"functionId": id,
				"projectId":  projectID,
				"runtime":    runtime,
				"entrypoint": entrypoint,
				"sourceType": sourceType,
				"source":     source,
				"repository": repository,
				"branch":     branch,
			},
			CreatedAt: now,
		})
	}

	return s.Get(ctx, id, projectID)
}

// Delete removes a function and its executions.
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM function_executions WHERE function_id = $1 AND project_id = $2", id, projectID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "DELETE FROM functions WHERE id = $1 AND project_id = $2", id, projectID)
	return err
}

// Execute creates a pending execution and pushes a job to the builds queue.
func (s *Service) Execute(ctx context.Context, functionID, projectID string) (*FunctionExecution, error) {
	// Verify function exists
	fn, err := s.Get(ctx, functionID, projectID)
	if err != nil {
		return nil, err
	}

	execID := uid.New("unique()")
	now := time.Now().UTC()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO function_executions (id, function_id, project_id, status, output, errors, duration, created_at)
		 VALUES ($1, $2, $3, 'pending', '', '', 0, $4)`,
		execID, functionID, projectID, now)
	if err != nil {
		return nil, fmt.Errorf("functions: execute: %w", err)
	}

	// Push job to builds queue for async processing
	job := queue.Job{
		ID:   execID,
		Type: "function_execution",
		Payload: map[string]interface{}{
			"executionId": execID,
			"functionId":  functionID,
			"projectId":   projectID,
			"runtime":     fn.Runtime,
			"entrypoint":  fn.Entrypoint,
			"timeout":     fn.Timeout,
			"sourceType":  fn.SourceType,
			"source":      fn.Source,
			"repository":  fn.Repository,
			"branch":      fn.Branch,
		},
		CreatedAt: now,
	}
	if err := s.queue.Push(ctx, "builds", job); err != nil {
		return nil, fmt.Errorf("functions: enqueue execution: %w", err)
	}

	return &FunctionExecution{
		ID: execID, FunctionID: functionID, ProjectID: projectID,
		Status: "pending", Output: "", Errors: "", Duration: 0,
		CreatedAt: now,
	}, nil
}

// GetExecution returns a single execution by ID.
func (s *Service) GetExecution(ctx context.Context, executionID, functionID, projectID string) (*FunctionExecution, error) {
	var e FunctionExecution
	err := s.db.QueryRowContext(ctx,
		"SELECT id, function_id, project_id, status, output, errors, duration, created_at FROM function_executions WHERE id = $1 AND function_id = $2 AND project_id = $3",
		executionID, functionID, projectID).Scan(&e.ID, &e.FunctionID, &e.ProjectID, &e.Status, &e.Output, &e.Errors, &e.Duration, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListExecutions returns all executions for a function.
func (s *Service) ListExecutions(ctx context.Context, functionID, projectID string) ([]*FunctionExecution, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, function_id, project_id, status, output, errors, duration, created_at FROM function_executions WHERE function_id = $1 AND project_id = $2 ORDER BY created_at DESC",
		functionID, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var executions []*FunctionExecution
	for rows.Next() {
		var e FunctionExecution
		if err := rows.Scan(&e.ID, &e.FunctionID, &e.ProjectID, &e.Status, &e.Output, &e.Errors, &e.Duration, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		executions = append(executions, &e)
	}
	return executions, len(executions), nil
}
