// Package projects implements project management: create/update/delete projects and API keys.
package projects

import (
	"context"
	"crypto/hmac"
	"crypto/sha256" // used by hmac.New
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
	db        *db.DB
	keySecret []byte // pepper for HMAC-SHA256 API key hashing
}

// NewService creates a new projects Service.
// keySecret is the HMAC pepper for API key hashing (API_KEY_SECRET env var).
// Falls back to jwtSecret so unset deployments continue to work.
func NewService(database *db.DB, keySecret, jwtSecret string) *Service {
	pepper := keySecret
	if pepper == "" {
		pepper = jwtSecret
	}
	return &Service{db: database, keySecret: []byte(pepper)}
}

// hashKey returns the HMAC-SHA256 hex digest of a raw API key secret.
func (s *Service) hashKey(raw string) string {
	mac := hmac.New(sha256.New, s.keySecret)
	mac.Write([]byte(raw))
	return fmt.Sprintf("%x", mac.Sum(nil))
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
	hash := s.hashKey(secret)

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
func (s *Service) CreateKey(ctx context.Context, projectID, name string, scopes []string, expiresAt *time.Time) (*model.APIKey, string, error) {
	rawSecret := "applad_key_" + uid.RandomHex(32)
	hash := s.hashKey(rawSecret)
	prefix := rawSecret
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	scopesJSON, _ := json.Marshal(scopes)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO api_keys (id, project_id, name, secret_hash, secret_prefix, scopes, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, projectID, name, hash, prefix, scopesJSON, expiresAt, now)
	if err != nil {
		return nil, "", fmt.Errorf("projects: create key: %w", err)
	}
	key := &model.APIKey{
		ID:           id,
		ProjectID:    projectID,
		Name:         name,
		Secret:       rawSecret,
		SecretPrefix: prefix,
		Scopes:       scopes,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
	}
	return key, rawSecret, nil
}

// ListKeys returns all API keys for a project.
func (s *Service) ListKeys(ctx context.Context, projectID string) ([]*model.APIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, scopes, expires_at, created_at, secret_prefix FROM api_keys WHERE project_id = ? ORDER BY created_at DESC",
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

// GetKey returns a single API key by ID.
func (s *Service) GetKey(ctx context.Context, projectID, keyID string) (*model.APIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, scopes, expires_at, created_at, secret_prefix FROM api_keys WHERE id = ? AND project_id = ?",
		keyID, projectID)
	if err != nil {
		return nil, fmt.Errorf("projects: get key: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("key not found")
	}
	return scanAPIKey(rows)
}

// UpdateKey updates mutable fields of an API key.
// setExpiry controls whether expiresAt is written (allows clearing it with nil).
func (s *Service) UpdateKey(ctx context.Context, projectID, keyID string, name *string, scopes []string, setExpiry bool, expiresAt *time.Time) (*model.APIKey, error) {
	sets := []string{}
	args := []interface{}{}
	if name != nil && strings.TrimSpace(*name) != "" {
		sets = append(sets, "name = ?")
		args = append(args, strings.TrimSpace(*name))
	}
	if scopes != nil {
		scopesJSON, _ := json.Marshal(scopes)
		sets = append(sets, "scopes = ?")
		args = append(args, scopesJSON)
	}
	if setExpiry {
		sets = append(sets, "expires_at = ?")
		args = append(args, expiresAt)
	}
	if len(sets) == 0 {
		return s.GetKey(ctx, projectID, keyID)
	}
	args = append(args, keyID, projectID)
	_, err := s.db.ExecContext(ctx,
		"UPDATE api_keys SET "+strings.Join(sets, ", ")+" WHERE id = ? AND project_id = ?",
		args...)
	if err != nil {
		return nil, fmt.Errorf("projects: update key: %w", err)
	}
	return s.GetKey(ctx, projectID, keyID)
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

// --- search ---

// SearchResult is a single match returned by the cross-resource search.
type SearchResult struct {
	Type     string `json:"type"`     // function, database, bucket, workflow, deployment, user
	ID       string `json:"id"`
	Label    string `json:"label"`
	Subtitle string `json:"subtitle,omitempty"`
}

// Search performs a case-insensitive ILIKE search across the main resource
// types for a project and returns up to limit results ordered by label.
func (s *Service) Search(ctx context.Context, projectID, query string, limit int) ([]*SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	pattern := "%" + strings.ReplaceAll(query, "%", "\\%") + "%"

	const sql = `
		SELECT 'function'   AS type, id, name AS label, runtime            AS subtitle
		FROM   functions    WHERE project_id = ? AND name  ILIKE ?
		UNION ALL
		SELECT 'database',           id, name,           ''                AS subtitle
		FROM   databases    WHERE project_id = ? AND name  ILIKE ?
		UNION ALL
		SELECT 'bucket',             id, name,           ''                AS subtitle
		FROM   buckets      WHERE project_id = ? AND name  ILIKE ?
		UNION ALL
		SELECT 'workflow',           id, name,           trigger_type      AS subtitle
		FROM   workflows    WHERE project_id = ? AND name  ILIKE ?
		UNION ALL
		SELECT 'deployment',         id, name,           type              AS subtitle
		FROM   deploy_targets WHERE project_id = ? AND name ILIKE ?
		UNION ALL
		SELECT 'user',               id,
		       COALESCE(NULLIF(name,''), email, id),
		       COALESCE(email, '')                                         AS subtitle
		FROM   users        WHERE project_id = ? AND (name ILIKE ? OR email ILIKE ?)
		ORDER  BY label
		LIMIT  ?`

	rows, err := s.db.QueryContext(ctx, sql,
		projectID, pattern, // functions
		projectID, pattern, // databases
		projectID, pattern, // buckets
		projectID, pattern, // workflows
		projectID, pattern, // deploy_targets
		projectID, pattern, pattern, // users (name OR email)
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Type, &r.ID, &r.Label, &r.Subtitle); err != nil {
			return nil, fmt.Errorf("search: scan: %w", err)
		}
		results = append(results, &r)
	}
	if results == nil {
		results = []*SearchResult{}
	}
	return results, nil
}

// --- platforms ---

// Platform represents a registered platform for a project.
type Platform struct {
	ID             string    `json:"$id"`
	ProjectID      string    `json:"projectId"`
	Type           string    `json:"type"` // web, ios, android, desktop, server
	Name           string    `json:"name"`
	Hostname       string    `json:"hostname,omitempty"`
	StoreID        string    `json:"storeId,omitempty"`
	DeployTargetID string    `json:"deployTargetId,omitempty"`
	CreatedAt      time.Time `json:"$createdAt"`
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
		"SELECT id, project_id, type, name, hostname, store_id, deploy_target_id, created_at FROM platforms WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, fmt.Errorf("platforms: list: %w", err)
	}
	defer rows.Close()
	var platforms []*Platform
	for rows.Next() {
		var p Platform
		var hostname, storeID, deployTargetID sql.NullString
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Type, &p.Name, &hostname, &storeID, &deployTargetID, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Hostname = hostname.String
		p.StoreID = storeID.String
		p.DeployTargetID = deployTargetID.String
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

// GetPlatform fetches a single platform by ID.
func (s *Service) GetPlatform(ctx context.Context, projectID, platformID string) (*Platform, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, project_id, type, name, hostname, store_id, deploy_target_id, created_at FROM platforms WHERE id = ? AND project_id = ?",
		platformID, projectID)
	var p Platform
	var hostname, storeID, deployTargetID sql.NullString
	if err := row.Scan(&p.ID, &p.ProjectID, &p.Type, &p.Name, &hostname, &storeID, &deployTargetID, &p.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("platform not found")
		}
		return nil, fmt.Errorf("platforms: get: %w", err)
	}
	p.Hostname = hostname.String
	p.StoreID = storeID.String
	p.DeployTargetID = deployTargetID.String
	return &p, nil
}

// UpdatePlatform updates mutable fields of a platform.
func (s *Service) UpdatePlatform(ctx context.Context, projectID, platformID string, name, hostname, deployTargetID *string) (*Platform, error) {
	sets := []string{}
	args := []interface{}{}
	if name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *name)
	}
	if hostname != nil {
		sets = append(sets, "hostname = ?")
		args = append(args, *hostname)
	}
	if deployTargetID != nil {
		sets = append(sets, "deploy_target_id = ?")
		if *deployTargetID == "" {
			args = append(args, nil) // store NULL when disconnecting
		} else {
			args = append(args, *deployTargetID)
		}
	}
	if len(sets) == 0 {
		return s.GetPlatform(ctx, projectID, platformID)
	}
	args = append(args, platformID, projectID)
	_, err := s.db.ExecContext(ctx,
		"UPDATE platforms SET "+strings.Join(sets, ", ")+" WHERE id = ? AND project_id = ?",
		args...)
	if err != nil {
		return nil, fmt.Errorf("platforms: update: %w", err)
	}
	return s.GetPlatform(ctx, projectID, platformID)
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
	if err := rows.Scan(&k.ID, &k.ProjectID, &k.Name, &scopesJSON, &expiresAt, &k.CreatedAt, &k.SecretPrefix); err != nil {
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
