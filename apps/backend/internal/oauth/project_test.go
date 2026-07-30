package oauth

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/credentials"
	"github.com/mittolabs/applad/internal/db"
)

// The store encrypts client secrets with the credentials vault, which needs a
// key. Set one for the whole package so encrypt/decrypt round-trips.
func TestMain(m *testing.M) {
	os.Setenv("CREDENTIALS_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	os.Exit(m.Run())
}

func newMockService(t *testing.T) (*ProjectOAuthService, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })
	return NewProjectOAuthService(&db.DB{DB: mockDB}), mock
}

// notPlaintextSecret is a sqlmock argument matcher: the value written to the
// client_secret column must be an encrypted token, never the plaintext secret.
type notPlaintextSecret struct{ plaintext string }

func (m notPlaintextSecret) Match(v driver.Value) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	return s != m.plaintext && credentials.IsEncryptedSecret(s)
}

// Set a provider, then read it back: the stored client secret is populated on
// the struct in memory but must never appear in the JSON the API returns.
func TestSetProviderThenGet_SecretNotLeaked(t *testing.T) {
	svc, mock := newMockService(t)
	ctx := context.Background()

	mock.ExpectExec(`INSERT INTO project_oauth_providers`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := svc.SetProvider(ctx, "proj1", "google", "my-client-id", "my-client-secret", nil); err != nil {
		t.Fatalf("SetProvider: %v", err)
	}

	// Store an actual ciphertext so the read path decrypts it back.
	enc, err := credentials.EncryptSecret("my-client-secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret", "extra"}).
			AddRow("op1", "proj1", "google", true, "my-client-id", enc, nil))
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
		t.Errorf("clientSecret should decrypt internally, got %q", cfg.ClientSecret)
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

// The value written to client_secret is ciphertext, not the plaintext secret.
func TestSetProvider_EncryptsSecretAtRest(t *testing.T) {
	svc, mock := newMockService(t)

	secret := "raw-oauth-secret-value"
	mock.ExpectExec(`INSERT INTO project_oauth_providers`).
		WithArgs(
			sqlmock.AnyArg(),           // id
			"proj1",                    // project_id
			"google",                   // provider
			"cid",                      // client_id
			notPlaintextSecret{secret}, // client_secret must be ciphertext
			nil,                        // extra
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.SetProvider(context.Background(), "proj1", "google", "cid", secret, nil); err != nil {
		t.Fatalf("SetProvider: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// The list query never selects the secret column, so a leak is impossible even
// if a caller forgot the json:"-" tag. Extra (non-secret) is returned.
func TestListProviders_OmitsSecret(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE project_id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "extra"}).
			AddRow("op1", "proj1", "google", true, "cid-google", nil).
			AddRow("op2", "proj1", "microsoft", false, "cid-ms", []byte(`{"tenantId":"contoso"}`)))

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
	// Extra round-trips for prefill.
	if cfgs[1].Extra["tenantId"] != "contoso" {
		t.Errorf("extra tenantId = %q, want contoso", cfgs[1].Extra["tenantId"])
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

// ResolveProvider prefers a per-project config over the instance-wide one and
// yields the decrypted secret for token exchange.
func TestResolveProvider_PrefersProjectConfig(t *testing.T) {
	svc, mock := newMockService(t)
	enc, _ := credentials.EncryptSecret("project-secret")
	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret", "extra"}).
			AddRow("op1", "proj1", "google", true, "project-client-id", enc, nil))

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
	if p.ClientSecret != "project-secret" {
		t.Errorf("clientSecret = %q, want decrypted project-secret", p.ClientSecret)
	}
}

// With no per-project row, ResolveProvider falls back to the instance config.
func TestResolveProvider_FallsBackToGlobal(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret", "extra"}))

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

// Microsoft's configured tenant replaces the hardcoded /common/ in both URLs.
func TestResolveProvider_MicrosoftTenantWired(t *testing.T) {
	svc, mock := newMockService(t)
	enc, _ := credentials.EncryptSecret("ms-secret")
	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret", "extra"}).
			AddRow("op1", "proj1", "microsoft", true, "app-id", enc, []byte(`{"tenantId":"contoso.onmicrosoft.com"}`)))

	p := svc.ResolveProvider(context.Background(), "proj1", "microsoft", nil)
	if p == nil {
		t.Fatal("expected a provider")
	}
	if strings.Contains(p.AuthURL, "/common/") || !strings.Contains(p.AuthURL, "/contoso.onmicrosoft.com/") {
		t.Errorf("authURL not tenant-wired: %s", p.AuthURL)
	}
	if strings.Contains(p.TokenURL, "/common/") || !strings.Contains(p.TokenURL, "/contoso.onmicrosoft.com/") {
		t.Errorf("tokenURL not tenant-wired: %s", p.TokenURL)
	}
}

// A Microsoft config without a tenant keeps the multi-tenant /common/ endpoints.
func TestResolveProvider_MicrosoftDefaultsToCommon(t *testing.T) {
	svc, mock := newMockService(t)
	enc, _ := credentials.EncryptSecret("ms-secret")
	mock.ExpectQuery(`SELECT .+ FROM project_oauth_providers WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "project_id", "provider", "enabled", "client_id", "client_secret", "extra"}).
			AddRow("op1", "proj1", "microsoft", true, "app-id", enc, nil))

	p := svc.ResolveProvider(context.Background(), "proj1", "microsoft", nil)
	if p == nil {
		t.Fatal("expected a provider")
	}
	if !strings.Contains(p.AuthURL, "/common/") {
		t.Errorf("authURL should keep /common/: %s", p.AuthURL)
	}
}
