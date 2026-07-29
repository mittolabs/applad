package oauth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func newMockService(t *testing.T) (*ProjectOAuthService, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })
	return NewProjectOAuthService(&db.DB{DB: mockDB}), mock
}

// Set a provider, then read it back: the stored client secret is populated on
// the struct in memory but must never appear in the JSON the API returns.
func TestSetProviderThenGet_SecretNotLeaked(t *testing.T) {
	svc, mock := newMockService(t)
	ctx := context.Background()

	mock.ExpectExec(`INSERT INTO project_oauth_providers`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := svc.SetProvider(ctx, "proj1", "google", "my-client-id", "my-client-secret"); err != nil {
		t.Fatalf("SetProvider: %v", err)
	}

	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret"}).
			AddRow("op1", "proj1", "google", true, "my-client-id", "my-client-secret"))
	cfg, err := svc.GetProvider(ctx, "proj1", "google")
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected a config, got nil")
	}
	if cfg.ClientID != "my-client-id" {
		t.Errorf("clientId = %q, want my-client-id", cfg.ClientID)
	}
	// The secret is available in-process (needed for token exchange)...
	if cfg.ClientSecret != "my-client-secret" {
		t.Errorf("clientSecret should be populated internally, got %q", cfg.ClientSecret)
	}
	// ...but must not be serialized to the client.
	out, _ := json.Marshal(cfg)
	if strings.Contains(string(out), "my-client-secret") {
		t.Fatalf("client secret leaked in JSON: %s", out)
	}
	if strings.Contains(string(out), "clientSecret") {
		t.Fatalf("clientSecret key present in JSON: %s", out)
	}
	if !strings.Contains(string(out), "my-client-id") {
		t.Fatalf("expected clientId in JSON: %s", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// The list query never selects the secret column, so a leak is impossible even
// if a caller forgot the json:"-" tag.
func TestListProviders_OmitsSecret(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE project_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id"}).
			AddRow("op1", "proj1", "google", true, "cid-google").
			AddRow("op2", "proj1", "github", false, "cid-github"))

	cfgs, err := svc.ListProviders(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(cfgs) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(cfgs))
	}
	for _, c := range cfgs {
		if c.ClientSecret != "" {
			t.Errorf("provider %s carries a secret: %q", c.ProviderName, c.ClientSecret)
		}
	}
	out, _ := json.Marshal(cfgs)
	if strings.Contains(string(out), "clientSecret") || strings.Contains(string(out), "client_secret") {
		t.Fatalf("secret field present in list JSON: %s", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestProjectOAuthConfig_JSONRedactsSecret(t *testing.T) {
	cfg := ProjectOAuthConfig{
		ID: "op1", ProjectID: "proj1", ProviderName: "google",
		Enabled: true, ClientID: "cid", ClientSecret: "super-secret-value",
	}
	out, _ := json.Marshal(cfg)
	if strings.Contains(string(out), "super-secret-value") {
		t.Fatalf("secret leaked: %s", out)
	}
}

// ResolveProvider prefers a per-project config over the instance-wide one.
func TestResolveProvider_PrefersProjectConfig(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret"}).
			AddRow("op1", "proj1", "google", true, "project-client-id", "project-secret"))

	global := map[string]*Provider{
		"google": AllProviderDefinitions()["google"].ToProvider("global-client-id", "global-secret"),
	}
	p := svc.ResolveProvider(context.Background(), "proj1", "google", global)
	if p == nil {
		t.Fatal("expected a provider")
	}
	if p.ClientID != "project-client-id" {
		t.Errorf("clientId = %q, want project-client-id (per-project should win)", p.ClientID)
	}
}

// With no per-project row, ResolveProvider falls back to the instance config.
func TestResolveProvider_FallsBackToGlobal(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret"}))

	global := map[string]*Provider{
		"google": AllProviderDefinitions()["google"].ToProvider("global-client-id", "global-secret"),
	}
	p := svc.ResolveProvider(context.Background(), "proj1", "google", global)
	if p == nil {
		t.Fatal("expected fallback provider")
	}
	if p.ClientID != "global-client-id" {
		t.Errorf("clientId = %q, want global-client-id", p.ClientID)
	}
}
