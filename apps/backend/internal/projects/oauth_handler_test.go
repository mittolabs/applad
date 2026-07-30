package projects

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	mw "github.com/mittolabs/applad/internal/middleware"
	oauthpkg "github.com/mittolabs/applad/internal/oauth"
)

// --- fakes -----------------------------------------------------------------

type fakeValidator struct{ userID string }

func (f fakeValidator) ValidateSession(_ context.Context, _ string) (string, string, error) {
	return f.userID, "sess1", nil
}

type fakeAccess struct{ allow bool }

func (f fakeAccess) CanAccessProject(_ context.Context, _, _ string) (bool, error) {
	return f.allow, nil
}
func (f fakeAccess) IsOrgMember(_ context.Context, _, _ string) (bool, error) { return f.allow, nil }
func (f fakeAccess) UserOrgIDs(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (f fakeAccess) ProjectOrgs(_ context.Context) (map[string]string, error) { return nil, nil }
func (f fakeAccess) DefaultOrg(_ context.Context, _ string) (string, error)   { return "org1", nil }

// fakeOAuthStore records writes and, like the real store, never returns the
// secret from ListProviders.
type fakeOAuthStore struct {
	set      map[string]oauthpkg.ProjectOAuthConfig // provider -> config (with secret, kept internally)
	disabled map[string]bool
	deleted  []string
}

func newFakeOAuthStore() *fakeOAuthStore {
	return &fakeOAuthStore{set: map[string]oauthpkg.ProjectOAuthConfig{}, disabled: map[string]bool{}}
}

func (s *fakeOAuthStore) ListProviders(_ context.Context, projectID string) ([]oauthpkg.ProjectOAuthConfig, error) {
	out := []oauthpkg.ProjectOAuthConfig{}
	for name, c := range s.set {
		out = append(out, oauthpkg.ProjectOAuthConfig{
			ID: "op_" + name, ProjectID: projectID, ProviderName: name,
			Enabled: !s.disabled[name], ClientID: c.ClientID, Extra: c.Extra, // secret intentionally omitted
		})
	}
	return out, nil
}

func (s *fakeOAuthStore) SetProvider(_ context.Context, projectID, provider, clientID, clientSecret string, extra map[string]string) error {
	s.set[provider] = oauthpkg.ProjectOAuthConfig{ClientID: clientID, ClientSecret: clientSecret, Extra: extra}
	delete(s.disabled, provider)
	return nil
}

func (s *fakeOAuthStore) DisableProvider(_ context.Context, _, provider string) error {
	s.disabled[provider] = true
	return nil
}

func (s *fakeOAuthStore) DeleteProvider(_ context.Context, _, provider string) error {
	s.deleted = append(s.deleted, provider)
	delete(s.set, provider)
	return nil
}

func oauthTestRouter(t *testing.T, access AccessChecker, store OAuthConfigStore) http.Handler {
	t.Helper()
	h := NewHandler(NewService(nil, "", "test-secret"))
	h.SetAccess(access)
	h.SetOAuthConfig(store)
	r := chi.NewRouter()
	r.Use(mw.RequireConsoleAuth(fakeValidator{userID: "user1"}))
	r.Mount("/projects", Routes(h))
	return r
}

func do(t *testing.T, router http.Handler, method, path, body string, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.Header.Set("Authorization", "Bearer token")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// --- tests -----------------------------------------------------------------

// The end-to-end happy path: set a provider through PUT, then GET it back. The
// GET must expose enabled + clientId and never the secret that was stored.
func TestOAuthConfig_SetThenGet_SecretNotLeaked(t *testing.T) {
	store := newFakeOAuthStore()
	router := oauthTestRouter(t, fakeAccess{allow: true}, store)

	secret := "top-secret-value-xyz"
	w := do(t, router, http.MethodPut, "/projects/proj1/auth/oauth/google",
		`{"clientId":"my-client-id","clientSecret":"`+secret+`","enabled":true}`, true)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := store.set["google"]; got.ClientID != "my-client-id" || got.ClientSecret != secret {
		t.Fatalf("store did not persist provider: %+v", got)
	}

	w = do(t, router, http.MethodGet, "/projects/proj1/auth/oauth", "", true)
	if w.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, secret) {
		t.Fatalf("secret leaked in GET response: %s", body)
	}
	if !strings.Contains(body, "my-client-id") {
		t.Fatalf("clientId missing from GET response: %s", body)
	}
	var resp struct {
		Total     int `json:"total"`
		Providers []struct {
			Provider string `json:"provider"`
			Enabled  bool   `json:"enabled"`
			ClientID string `json:"clientId"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 1 || len(resp.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %+v", resp)
	}
	if resp.Providers[0].Provider != "google" || !resp.Providers[0].Enabled {
		t.Fatalf("unexpected provider view: %+v", resp.Providers[0])
	}
}

func TestOAuthConfig_Disable(t *testing.T) {
	store := newFakeOAuthStore()
	router := oauthTestRouter(t, fakeAccess{allow: true}, store)

	w := do(t, router, http.MethodPut, "/projects/proj1/auth/oauth/github",
		`{"clientId":"cid","clientSecret":"sec","enabled":false}`, true)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !store.disabled["github"] {
		t.Fatal("provider should have been disabled")
	}
}

func TestOAuthConfig_Delete(t *testing.T) {
	store := newFakeOAuthStore()
	store.set["google"] = oauthpkg.ProjectOAuthConfig{ClientID: "cid"}
	router := oauthTestRouter(t, fakeAccess{allow: true}, store)

	w := do(t, router, http.MethodDelete, "/projects/proj1/auth/oauth/google", "", true)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if len(store.deleted) != 1 || store.deleted[0] != "google" {
		t.Fatalf("provider was not deleted: %+v", store.deleted)
	}
}

func TestOAuthConfig_EnableRequiresClientID(t *testing.T) {
	router := oauthTestRouter(t, fakeAccess{allow: true}, newFakeOAuthStore())
	w := do(t, router, http.MethodPut, "/projects/proj1/auth/oauth/google",
		`{"clientId":"","enabled":true}`, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthConfig_UnknownProvider(t *testing.T) {
	router := oauthTestRouter(t, fakeAccess{allow: true}, newFakeOAuthStore())
	w := do(t, router, http.MethodPut, "/projects/proj1/auth/oauth/notreal",
		`{"clientId":"cid","enabled":true}`, true)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Authorization: a caller who is not a member of the project's org is refused,
// and no write reaches the store.
func TestOAuthConfig_Forbidden(t *testing.T) {
	store := newFakeOAuthStore()
	router := oauthTestRouter(t, fakeAccess{allow: false}, store)

	w := do(t, router, http.MethodPut, "/projects/proj1/auth/oauth/google",
		`{"clientId":"cid","clientSecret":"sec","enabled":true}`, true)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if len(store.set) != 0 {
		t.Fatal("forbidden caller must not write to the store")
	}
}

func TestOAuthConfig_Unauthenticated(t *testing.T) {
	router := oauthTestRouter(t, fakeAccess{allow: true}, newFakeOAuthStore())
	w := do(t, router, http.MethodGet, "/projects/proj1/auth/oauth", "", false)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
