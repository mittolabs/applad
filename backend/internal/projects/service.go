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
	key, err := s.GetKeyBySecret(ctx, secret)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, key.ProjectID)
}

// GetKeyBySecret looks up an API key by raw secret.
func (s *Service) GetKeyBySecret(ctx context.Context, secret string) (*model.APIKey, error) {
	if !strings.HasPrefix(secret, "applad_key_") {
		return nil, fmt.Errorf("projects: invalid key format")
	}
	prefix := secret[:16]
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(secret)))

	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, scopes, expires_at, created_at FROM api_keys WHERE secret_prefix = ? AND secret_hash = ?",
		prefix, hash)
	if err != nil {
		return nil, fmt.Errorf("projects: get key by secret query: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("projects: key not found")
	}
	key, err := scanAPIKey(rows)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// List returns all projects, optionally filtered by org ID.
func (s *Service) List(ctx context.Context, orgID ...string) ([]*model.Project, error) {
	var rows *sql.Rows
	var err error
	if len(orgID) > 0 && orgID[0] != "" {
		rows, err = s.db.QueryContext(ctx,
			"SELECT id, name, description, created_at, updated_at FROM projects WHERE org_id = ? ORDER BY created_at DESC", orgID[0])
	} else {
		rows, err = s.db.QueryContext(ctx,
			"SELECT id, name, description, created_at, updated_at FROM projects ORDER BY created_at DESC")
	}
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

// UsageStats holds aggregated project statistics.
type UsageStats struct {
	ProjectID    string `json:"projectId"`
	Users        int    `json:"users"`
	Sessions     int    `json:"sessions"`
	Databases    int    `json:"databases"`
	Tables       int    `json:"tables"`
	Rows         int    `json:"rows"`
	Buckets      int    `json:"buckets"`
	Files        int    `json:"files"`
	StorageBytes int64  `json:"storageBytes"`
	Teams        int    `json:"teams"`
	Workflows    int    `json:"workflows"`
	Executions   int    `json:"executions"`
	Functions    int    `json:"functions"`
	Deployments  int    `json:"deployments"`
}

// GetUsage returns aggregated usage stats for a project.
func (s *Service) GetUsage(ctx context.Context, projectID string) (*UsageStats, error) {
	u := &UsageStats{ProjectID: projectID}
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE project_id = ?", projectID).Scan(&u.Users)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE project_id = ?", projectID).Scan(&u.Sessions)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM databases WHERE project_id = ?", projectID).Scan(&u.Databases)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tables WHERE project_id = ?", projectID).Scan(&u.Tables)
	s.db.QueryRowContext(ctx, "SELECT 0").Scan(&u.Rows)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM buckets WHERE project_id = ?", projectID).Scan(&u.Buckets)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE project_id = ?", projectID).Scan(&u.Files)
	s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(size), 0) FROM files WHERE project_id = ?", projectID).Scan(&u.StorageBytes)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams WHERE project_id = ?", projectID).Scan(&u.Teams)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflows WHERE project_id = ?", projectID).Scan(&u.Workflows)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workflow_executions WHERE project_id = ?", projectID).Scan(&u.Executions)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM functions WHERE project_id = ?", projectID).Scan(&u.Functions)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM deployments WHERE project_id = ?", projectID).Scan(&u.Deployments)
	return u, nil
}

// --- platforms ---

// Platform represents a registered platform for a project.
type Platform struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Type      string    `json:"type"` // web, flutter-ios, flutter-android, flutter-web, server
	Name      string    `json:"name"`
	Hostname  string    `json:"hostname,omitempty"`
	StoreID   string    `json:"storeId,omitempty"`
	CreatedAt time.Time `json:"$createdAt"`
}

// CreatePlatform registers a new platform for a project.
func (s *Service) CreatePlatform(ctx context.Context, projectID, pType, name, hostname, storeID string) (*Platform, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO platforms (id, project_id, type, name, hostname, store_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, projectID, pType, name, hostname, storeID, now)
	if err != nil {
		return nil, fmt.Errorf("platforms: create: %w", err)
	}
	return &Platform{
		ID: id, ProjectID: projectID, Type: pType, Name: name,
		Hostname: hostname, StoreID: storeID, CreatedAt: now,
	}, nil
}

// ListPlatforms returns all platforms for a project.
func (s *Service) ListPlatforms(ctx context.Context, projectID string) ([]*Platform, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, type, name, hostname, store_id, created_at FROM platforms WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, fmt.Errorf("platforms: list: %w", err)
	}
	defer rows.Close()
	var platforms []*Platform
	for rows.Next() {
		var p Platform
		var hostname, storeID sql.NullString
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Type, &p.Name, &hostname, &storeID, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Hostname = hostname.String
		p.StoreID = storeID.String
		platforms = append(platforms, &p)
	}
	if platforms == nil {
		platforms = []*Platform{}
	}
	return platforms, nil
}

// DeletePlatform removes a platform.
func (s *Service) DeletePlatform(ctx context.Context, projectID, platformID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM platforms WHERE id = ? AND project_id = ?", platformID, projectID)
	return err
}

// AuthSecurity holds project-level auth security policy settings.
type AuthSecurity struct {
	UsersLimit                 int  `json:"usersLimit"`                 // 0 = unlimited
	SessionLengthSeconds       int  `json:"sessionLengthSeconds"`       // default 31536000 (365 days)
	SessionsPerUser            int  `json:"sessionsPerUser"`            // 0 = unlimited
	PasswordMinLength          int  `json:"passwordMinLength"`          // default 8
	PasswordHistory            int  `json:"passwordHistory"`            // 0 = disabled
	PasswordDictionary         bool `json:"passwordDictionary"`
	PasswordPersonalData       bool `json:"passwordPersonalData"`
	MFARequired                bool `json:"mfaRequired"`
	SessionAlerts              bool `json:"sessionAlerts"`
	InvalidateOnPasswordChange bool `json:"invalidateOnPasswordChange"`
}

// defaultAuthSecurity returns sensible defaults.
func defaultAuthSecurity() AuthSecurity {
	return AuthSecurity{
		UsersLimit:                 0,
		SessionLengthSeconds:       365 * 24 * 3600,
		SessionsPerUser:            10,
		PasswordMinLength:          8,
		PasswordHistory:            0,
		PasswordDictionary:         false,
		PasswordPersonalData:       false,
		MFARequired:                false,
		SessionAlerts:              false,
		InvalidateOnPasswordChange: true,
	}
}

// GetAuthSecurity reads the security sub-key from a project's auth_config.
func (s *Service) GetAuthSecurity(ctx context.Context, projectID string) (AuthSecurity, error) {
	var raw sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT auth_config FROM projects WHERE id = ?", projectID).Scan(&raw)
	if err != nil {
		return defaultAuthSecurity(), fmt.Errorf("projects: get auth security: %w", err)
	}
	sec := defaultAuthSecurity()
	if !raw.Valid || raw.String == "" || raw.String == "null" {
		return sec, nil
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw.String), &cfg); err != nil {
		return sec, nil
	}
	if secRaw, ok := cfg["security"]; ok {
		_ = json.Unmarshal(secRaw, &sec)
	}
	return sec, nil
}

// UpdateAuthSecurity merges the security sub-key into a project's auth_config.
func (s *Service) UpdateAuthSecurity(ctx context.Context, projectID string, sec AuthSecurity) error {
	var raw sql.NullString
	_ = s.db.QueryRowContext(ctx,
		"SELECT auth_config FROM projects WHERE id = ?", projectID).Scan(&raw)
	cfg := map[string]interface{}{}
	if raw.Valid && raw.String != "" && raw.String != "null" {
		_ = json.Unmarshal([]byte(raw.String), &cfg)
	}
	cfg["security"] = sec
	configJSON, _ := json.Marshal(cfg)
	_, err := s.db.ExecContext(ctx,
		"UPDATE projects SET auth_config = ? WHERE id = ?", configJSON, projectID)
	if err != nil {
		return fmt.Errorf("projects: update auth security: %w", err)
	}
	return nil
}

// UpdateAuthConfig updates the auth_config JSON for a project.
func (s *Service) UpdateAuthConfig(ctx context.Context, projectID string, config map[string]interface{}) error {
	configJSON, _ := json.Marshal(config)
	_, err := s.db.ExecContext(ctx,
		"UPDATE projects SET auth_config = ? WHERE id = ?", configJSON, projectID)
	if err != nil {
		return fmt.Errorf("projects: update auth config: %w", err)
	}
	return nil
}

// UpdateServicesConfig updates the services_config JSON for a project.
func (s *Service) UpdateServicesConfig(ctx context.Context, projectID string, config map[string]interface{}) error {
	configJSON, _ := json.Marshal(config)
	_, err := s.db.ExecContext(ctx,
		"UPDATE projects SET services_config = ? WHERE id = ?", configJSON, projectID)
	if err != nil {
		return fmt.Errorf("projects: update services config: %w", err)
	}
	return nil
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
