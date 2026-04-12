// Package edge implements edge function management — Cloudflare Workers-style
// serverless functions deployed to edge nodes close to users.
package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// EdgeFunction is a deployable edge function definition.
type EdgeFunction struct {
	ID        string            `json:"$id"`
	ProjectID string            `json:"projectId"`
	Name      string            `json:"name"`
	Slug      string            `json:"slug"`
	Code      string            `json:"code"`
	Runtime   string            `json:"runtime"` // js | ts | wasm
	Regions   []string          `json:"regions"`
	EnvVars   map[string]string `json:"envVars,omitempty"`
	Status    string            `json:"status"` // draft | deployed | error
	CreatedAt time.Time         `json:"$createdAt"`
	UpdatedAt time.Time         `json:"$updatedAt"`
}

// Deployment is a specific deployed version of an edge function.
type Deployment struct {
	ID         string    `json:"$id"`
	FunctionID string    `json:"functionId"`
	ProjectID  string    `json:"projectId"`
	Version    int       `json:"version"`
	Status     string    `json:"status"` // deploying | active | failed
	Regions    []string  `json:"regions"`
	DeployedAt *time.Time `json:"deployedAt,omitempty"`
	CreatedAt  time.Time `json:"$createdAt"`
}

// Service manages edge functions and their deployments.
type Service struct {
	db *db.DB
}

// NewService creates a new edge Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ── Functions ─────────────────────────────────────────────────────────────────

// Create creates a new edge function.
func (s *Service) Create(ctx context.Context, projectID, name, slug, code, runtime string, regions []string, envVars map[string]string) (*EdgeFunction, error) {
	f := &EdgeFunction{
		ID: uid.New(""), ProjectID: projectID, Name: name, Slug: slug,
		Code: code, Runtime: runtime, Regions: regions, EnvVars: envVars,
		Status: "draft", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if f.Runtime == "" {
		f.Runtime = "js"
	}
	regionsJSON, _ := json.Marshal(regions)
	envJSON, _ := json.Marshal(envVars)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO edge_functions (id, project_id, name, slug, code, runtime, regions, env_vars, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		f.ID, f.ProjectID, f.Name, f.Slug, f.Code, f.Runtime,
		nullBytes(regionsJSON), nullBytes(envJSON), f.Status,
		f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, fmt.Errorf("edge: function with slug %q already exists", slug)
		}
		return nil, fmt.Errorf("edge: create: %w", err)
	}
	return f, nil
}

// Get fetches an edge function by ID.
func (s *Service) Get(ctx context.Context, functionID, projectID string) (*EdgeFunction, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, slug, code, runtime, COALESCE(regions,'[]'), COALESCE(env_vars,'{}'), status, created_at, updated_at
		 FROM edge_functions WHERE id = $1 AND project_id = $2`, functionID, projectID)
	return scanFunction(row)
}

// List returns all edge functions for a project.
func (s *Service) List(ctx context.Context, projectID string) ([]*EdgeFunction, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, slug, code, runtime, COALESCE(regions,'[]'), COALESCE(env_vars,'{}'), status, created_at, updated_at
		 FROM edge_functions WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*EdgeFunction
	for rows.Next() {
		f, err := scanFunction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// Update updates an edge function's code, regions, or env vars.
func (s *Service) Update(ctx context.Context, functionID, projectID, name, code string, regions []string, envVars map[string]string) (*EdgeFunction, error) {
	f, err := s.Get(ctx, functionID, projectID)
	if err != nil {
		return nil, err
	}
	if name != "" {
		f.Name = name
	}
	if code != "" {
		f.Code = code
	}
	if regions != nil {
		f.Regions = regions
	}
	if envVars != nil {
		f.EnvVars = envVars
	}
	regionsJSON, _ := json.Marshal(f.Regions)
	envJSON, _ := json.Marshal(f.EnvVars)
	_, err = s.db.ExecContext(ctx,
		"UPDATE edge_functions SET name=$1, code=$2, regions=$3, env_vars=$4, updated_at=$5 WHERE id=$6",
		f.Name, f.Code, nullBytes(regionsJSON), nullBytes(envJSON), time.Now().UTC(), f.ID,
	)
	return f, err
}

// Delete removes an edge function.
func (s *Service) Delete(ctx context.Context, functionID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM edge_functions WHERE id=$1 AND project_id=$2", functionID, projectID)
	return err
}

// ── Deployments ───────────────────────────────────────────────────────────────

// Deploy creates a deployment record for an edge function.
func (s *Service) Deploy(ctx context.Context, functionID, projectID string, regions []string) (*Deployment, error) {
	f, err := s.Get(ctx, functionID, projectID)
	if err != nil {
		return nil, fmt.Errorf("edge: function not found")
	}

	// Get current version number
	var maxVer int
	s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM edge_deployments WHERE function_id=$1", functionID).Scan(&maxVer) //nolint:errcheck
	newVer := maxVer + 1

	if len(regions) == 0 {
		regions = f.Regions
	}
	now := time.Now().UTC()
	d := &Deployment{
		ID: uid.New(""), FunctionID: functionID, ProjectID: projectID,
		Version: newVer, Status: "deploying", Regions: regions,
		DeployedAt: &now, CreatedAt: now,
	}
	regionsJSON, _ := json.Marshal(regions)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO edge_deployments (id, function_id, project_id, version, status, regions, deployed_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		d.ID, d.FunctionID, d.ProjectID, d.Version, d.Status,
		nullBytes(regionsJSON), d.DeployedAt, d.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("edge: deploy: %w", err)
	}

	// Mark as deployed (in a real system this would be async)
	s.db.ExecContext(ctx, "UPDATE edge_functions SET status='deployed', updated_at=$1 WHERE id=$2", now, functionID)       //nolint:errcheck
	s.db.ExecContext(ctx, "UPDATE edge_deployments SET status='active' WHERE id=$1", d.ID) //nolint:errcheck
	d.Status = "active"
	return d, nil
}

// ListDeployments returns deployment history for an edge function.
func (s *Service) ListDeployments(ctx context.Context, functionID, projectID string) ([]*Deployment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, function_id, project_id, version, status, COALESCE(regions,'[]'), deployed_at, created_at
		 FROM edge_deployments WHERE function_id=$1 AND project_id=$2 ORDER BY version DESC`, functionID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Deployment
	for rows.Next() {
		d := &Deployment{}
		var regionsRaw []byte
		if err := rows.Scan(&d.ID, &d.FunctionID, &d.ProjectID, &d.Version, &d.Status,
			&regionsRaw, &d.DeployedAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(regionsRaw, &d.Regions) //nolint:errcheck
		out = append(out, d)
	}
	return out, nil
}

// ── scanners ──────────────────────────────────────────────────────────────────

func scanFunction(row interface{ Scan(...interface{}) error }) (*EdgeFunction, error) {
	f := &EdgeFunction{}
	var regionsRaw, envRaw []byte
	if err := row.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Slug, &f.Code, &f.Runtime,
		&regionsRaw, &envRaw, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(regionsRaw, &f.Regions) //nolint:errcheck
	json.Unmarshal(envRaw, &f.EnvVars)     //nolint:errcheck
	return f, nil
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
