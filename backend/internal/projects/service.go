// Package projects implements project management: create/update/delete projects and API keys.
package projects

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/uid"
)

// Service handles project business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new projects Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Create creates a new project.
func (s *Service) Create(ctx context.Context, name, description string) (*model.Project, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO projects (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, name, description, now, now)
	if err != nil {
		return nil, fmt.Errorf("projects: create: %w", err)
	}
	return &model.Project{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Get returns a project by ID.
func (s *Service) Get(ctx context.Context, id string) (*model.Project, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM projects WHERE id = ?", id)
	return scanProject(row)
}

// GetByKey looks up a project by raw API key secret.
func (s *Service) GetByKey(ctx context.Context, secret string) (*model.Project, error) {
	if !strings.HasPrefix(secret, "applad_key_") {
		return nil, fmt.Errorf("projects: invalid key format")
	}
	prefix := secret[:16]
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(secret)))

	rows, err := s.db.QueryContext(ctx,
		"SELECT k.project_id FROM api_keys k WHERE k.secret_prefix = ? AND k.secret_hash = ?",
		prefix, hash)
	if err != nil {
		return nil, fmt.Errorf("projects: getbykey query: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("projects: key not found")
	}
	var projectID string
	if err := rows.Scan(&projectID); err != nil {
		return nil, err
	}
	return s.Get(ctx, projectID)
}

// List returns all projects.
func (s *Service) List(ctx context.Context) ([]*model.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM projects ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("projects: list: %w", err)
	}
	defer rows.Close()
	var projects []*model.Project
	for rows.Next() {
		p, err := scanProjectRow(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// Update updates a project's name and description.
func (s *Service) Update(ctx context.Context, id, name, description string) (*model.Project, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE projects SET name = ?, description = ? WHERE id = ?",
		name, description, id)
	if err != nil {
		return nil, fmt.Errorf("projects: update: %w", err)
	}
	return s.Get(ctx, id)
}

// Delete removes a project by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", id)
	return err
}

// CreateKey creates a new API key for a project. Returns the key model and the raw secret.
func (s *Service) CreateKey(ctx context.Context, projectID, name string, scopes []string) (*model.APIKey, string, error) {
	rawSecret := "applad_key_" + uid.RandomHex(32)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawSecret)))
	prefix := rawSecret
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	scopesJSON, _ := json.Marshal(scopes)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO api_keys (id, project_id, name, secret_hash, secret_prefix, scopes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, projectID, name, hash, prefix, scopesJSON, now)
	if err != nil {
		return nil, "", fmt.Errorf("projects: create key: %w", err)
	}
	key := &model.APIKey{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Secret:    rawSecret,
		Scopes:    scopes,
		CreatedAt: now,
	}
	return key, rawSecret, nil
}

// ListKeys returns all API keys for a project.
func (s *Service) ListKeys(ctx context.Context, projectID string) ([]*model.APIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, scopes, expires_at, created_at FROM api_keys WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, fmt.Errorf("projects: list keys: %w", err)
	}
	defer rows.Close()
	var keys []*model.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// DeleteKey removes an API key.
func (s *Service) DeleteKey(ctx context.Context, projectID, keyID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM api_keys WHERE id = ? AND project_id = ?", keyID, projectID)
	return err
}

// --- scan helpers ---

type projectScanner interface {
	Scan(dest ...interface{}) error
}

func scanProject(row *sql.Row) (*model.Project, error) {
	var p model.Project
	var desc sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &desc, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, err
	}
	p.Description = desc.String
	return &p, nil
}

func scanProjectRow(rows *sql.Rows) (*model.Project, error) {
	var p model.Project
	var desc sql.NullString
	if err := rows.Scan(&p.ID, &p.Name, &desc, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Description = desc.String
	return &p, nil
}

func scanAPIKey(rows *sql.Rows) (*model.APIKey, error) {
	var k model.APIKey
	var scopesJSON []byte
	var expiresAt sql.NullTime
	if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &scopesJSON, &expiresAt, &k.CreatedAt); err != nil {
		return nil, err
	}
	if len(scopesJSON) > 0 {
		json.Unmarshal(scopesJSON, &k.Scopes) //nolint:errcheck
	}
	if k.Scopes == nil {
		k.Scopes = []string{}
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		k.ExpiresAt = &t
	}
	return &k, nil
}
