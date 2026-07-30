package oauth

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/mittolabs/applad/internal/credentials"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// ProjectOAuthConfig is a per-project OAuth provider configuration stored in DB.
type ProjectOAuthConfig struct {
	ID           string `json:"$id"`
	ProjectID    string `json:"projectId"`
	ProviderName string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"-"` // never exposed in API responses
	// Extra carries auxiliary, non-secret provider fields (Microsoft tenant id,
	// Apple key id / team id). It is safe to expose: secret material never lives
	// here — Apple's .p8 key is stored encrypted in ClientSecret.
	Extra map[string]string `json:"extra,omitempty"`
}

// ProjectOAuthService manages per-project OAuth provider configurations.
type ProjectOAuthService struct {
	db *db.DB
}

// NewProjectOAuthService creates a new project OAuth service.
func NewProjectOAuthService(database *db.DB) *ProjectOAuthService {
	return &ProjectOAuthService{db: database}
}

// SetProvider creates or updates an OAuth provider for a project. An empty
// clientSecret on update keeps the stored one, so the console — which never
// receives the secret back on GET — can re-save a provider (e.g. to change the
// client id) without wiping it. Likewise a nil/empty extra keeps the stored
// aux fields, so a plain enable/disable toggle does not drop them.
//
// The secret is encrypted at rest with the credentials vault mechanism before
// it touches the database; only the ciphertext is stored.
func (s *ProjectOAuthService) SetProvider(ctx context.Context, projectID, provider, clientID, clientSecret string, extra map[string]string) error {
	storedSecret := ""
	if clientSecret != "" {
		enc, err := credentials.EncryptSecret(clientSecret)
		if err != nil {
			return err
		}
		storedSecret = enc
	}

	var extraArg interface{} // nil => keep stored extra
	if len(extra) > 0 {
		b, err := json.Marshal(extra)
		if err != nil {
			return err
		}
		extraArg = string(b)
	}

	id := uid.New("unique()")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO project_oauth_providers (id, project_id, provider, enabled, client_id, client_secret, extra)
		 VALUES (?, ?, ?, TRUE, ?, ?, ?)
		 ON CONFLICT (project_id, provider) DO UPDATE SET
		   client_id = EXCLUDED.client_id,
		   client_secret = CASE WHEN EXCLUDED.client_secret = '' THEN project_oauth_providers.client_secret ELSE EXCLUDED.client_secret END,
		   extra = CASE WHEN EXCLUDED.extra IS NULL THEN project_oauth_providers.extra ELSE EXCLUDED.extra END,
		   enabled = TRUE`,
		id, projectID, provider, clientID, storedSecret, extraArg)
	return err
}

// DisableProvider disables an OAuth provider for a project.
func (s *ProjectOAuthService) DisableProvider(ctx context.Context, projectID, provider string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE project_oauth_providers SET enabled = FALSE WHERE project_id = ? AND provider = ?",
		projectID, provider)
	return err
}

// DeleteProvider removes an OAuth provider config for a project.
func (s *ProjectOAuthService) DeleteProvider(ctx context.Context, projectID, provider string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM project_oauth_providers WHERE project_id = ? AND provider = ?",
		projectID, provider)
	return err
}

// GetProvider returns a project's OAuth provider config with the client secret
// decrypted for in-process use (token exchange). The secret is never serialized.
func (s *ProjectOAuthService) GetProvider(ctx context.Context, projectID, provider string) (*ProjectOAuthConfig, error) {
	var cfg ProjectOAuthConfig
	var storedSecret string
	var extraRaw []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, project_id, provider, enabled, client_id, client_secret, extra FROM project_oauth_providers WHERE project_id = ? AND provider = ? AND enabled = TRUE",
		projectID, provider).Scan(&cfg.ID, &cfg.ProjectID, &cfg.ProviderName, &cfg.Enabled, &cfg.ClientID, &storedSecret, &extraRaw)
	if err == sql.ErrNoRows {
		return nil, nil // not configured
	}
	if err != nil {
		return nil, err
	}
	if storedSecret != "" {
		dec, err := credentials.DecryptSecret(storedSecret)
		if err != nil {
			return nil, err
		}
		cfg.ClientSecret = dec
	}
	if len(extraRaw) > 0 {
		_ = json.Unmarshal(extraRaw, &cfg.Extra)
	}
	return &cfg, nil
}

// ListProviders returns all configured OAuth providers for a project. The
// client secret column is never selected, so a secret cannot leak here; the
// non-secret Extra fields are returned so the console can prefill them.
func (s *ProjectOAuthService) ListProviders(ctx context.Context, projectID string) ([]ProjectOAuthConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, provider, enabled, client_id, extra FROM project_oauth_providers WHERE project_id = ? ORDER BY provider",
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []ProjectOAuthConfig
	for rows.Next() {
		var cfg ProjectOAuthConfig
		var extraRaw []byte
		if err := rows.Scan(&cfg.ID, &cfg.ProjectID, &cfg.ProviderName, &cfg.Enabled, &cfg.ClientID, &extraRaw); err != nil {
			return nil, err
		}
		if len(extraRaw) > 0 {
			_ = json.Unmarshal(extraRaw, &cfg.Extra)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// ResolveProvider returns a fully configured Provider for a project + provider name.
// Falls back to global env-var config if no project-level config exists.
func (s *ProjectOAuthService) ResolveProvider(ctx context.Context, projectID, providerName string, globalProviders map[string]*Provider) *Provider {
	// Try project-level config first
	cfg, err := s.GetProvider(ctx, projectID, providerName)
	if err == nil && cfg != nil {
		defs := AllProviderDefinitions()
		if def, ok := defs[providerName]; ok {
			p := def.ToProvider(cfg.ClientID, cfg.ClientSecret)
			applyAuxConfig(p, providerName, cfg)
			return p
		}
	}

	// Fall back to global config
	if p, ok := globalProviders[providerName]; ok {
		return p
	}

	return nil
}
