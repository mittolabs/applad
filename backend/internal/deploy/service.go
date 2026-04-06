// Package deploy implements Applad's deployment service:
// deployments, builds, and release management.
package deploy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Deployment represents a deployment configuration.
type Deployment struct {
	ID        string                 `json:"$id"`
	ProjectID string                 `json:"projectId"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // web, function, container
	Status    string                 `json:"status"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"$createdAt"`
	UpdatedAt time.Time              `json:"$updatedAt"`
}

// Service handles deploy business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new deploy Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Create creates a new deployment.
func (s *Service) Create(ctx context.Context, projectID, name, deployType string, config map[string]interface{}) (*Deployment, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	cfgJSON, _ := json.Marshal(config)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO deployments (id, project_id, name, type, status, config, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'pending', ?, ?, ?)`,
		id, projectID, name, deployType, cfgJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("deploy: create: %w", err)
	}
	return &Deployment{
		ID: id, ProjectID: projectID, Name: name,
		Type: deployType, Status: "pending", Config: config,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Get returns a deployment by ID.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Deployment, error) {
	var d Deployment
	var cfgJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, project_id, name, type, status, config, created_at, updated_at FROM deployments WHERE id = ? AND project_id = ?",
		id, projectID).Scan(&d.ID, &d.ProjectID, &d.Name, &d.Type, &d.Status, &cfgJSON, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("deployment not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(cfgJSON, &d.Config)
	if d.Config == nil {
		d.Config = map[string]interface{}{}
	}
	return &d, nil
}

// List returns all deployments for a project.
func (s *Service) List(ctx context.Context, projectID string) ([]*Deployment, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, type, status, config, created_at, updated_at FROM deployments WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var deployments []*Deployment
	for rows.Next() {
		var d Deployment
		var cfgJSON []byte
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Name, &d.Type, &d.Status, &cfgJSON, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(cfgJSON, &d.Config)
		if d.Config == nil {
			d.Config = map[string]interface{}{}
		}
		deployments = append(deployments, &d)
	}
	return deployments, len(deployments), nil
}

// UpdateStatus updates the status of a deployment.
func (s *Service) UpdateStatus(ctx context.Context, id, projectID, status string) (*Deployment, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE deployments SET status = ?, updated_at = ? WHERE id = ? AND project_id = ?",
		status, time.Now().UTC(), id, projectID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id, projectID)
}

// Delete removes a deployment.
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM deployments WHERE id = ? AND project_id = ?", id, projectID)
	return err
}
