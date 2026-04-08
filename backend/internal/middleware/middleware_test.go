package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/model"
)

type testAPIKeyProvider struct {
	key *model.APIKey
	err error
}

func (p testAPIKeyProvider) GetKeyBySecret(_ context.Context, _ string) (*model.APIKey, error) {
	return p.key, p.err
}

func TestCORS_SetsHeaders(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected CORS allow origin header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected CORS allow methods header")
	}
}

func TestCORS_Preflight(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestProjectContext_Missing(t *testing.T) {
	handler := ProjectContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without project header")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestProjectContext_Present(t *testing.T) {
	var gotProjectID string
	handler := ProjectContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProjectID = ProjectFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Applad-Project", "proj123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotProjectID != "proj123" {
		t.Fatalf("expected project ID 'proj123', got %s", gotProjectID)
	}
}

func TestRequireAuth_NoAuth(t *testing.T) {
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler without auth")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_JWT(t *testing.T) {
	secret := "test-secret"

	// Create a valid JWT
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        "sess456",
		},
		SessionID: "sess456",
		ProjectID: "proj789",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	var gotUser, gotSession string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		gotSession = SessionFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Authenticate(secret, nil)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if gotUser != "user123" {
		t.Fatalf("expected user 'user123', got %s", gotUser)
	}
	if gotSession != "sess456" {
		t.Fatalf("expected session 'sess456', got %s", gotSession)
	}
}

func TestAuthenticate_APIKey(t *testing.T) {
	var gotUser string
	var gotIsAPIKey bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		gotIsAPIKey = IsAPIKey(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Authenticate("secret", nil)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Applad-Key", "applad_key_abc123def456")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !gotIsAPIKey {
		t.Fatal("expected IsAPIKey to be true")
	}
	if gotUser == "" {
		t.Fatal("expected user to be set from API key")
	}
}

func TestAuthenticate_APIKeyRejectsProjectMismatch(t *testing.T) {
	provider := testAPIKeyProvider{key: &model.APIKey{ID: "key1", ProjectID: "proj-a", Scopes: []string{"databases"}}}
	handler := Authenticate("secret", provider)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for project mismatch")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/databases", nil)
	req.Header.Set("X-Applad-Key", "applad_key_abc123def456")
	req.Header.Set("X-Applad-Project", "proj-b")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthenticate_APIKeyRejectsMissingScope(t *testing.T) {
	provider := testAPIKeyProvider{key: &model.APIKey{ID: "key1", ProjectID: "proj-a", Scopes: []string{"storage.read"}}}
	handler := Authenticate("secret", provider)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without required scope")
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/databases", nil)
	req.Header.Set("X-Applad-Key", "applad_key_abc123def456")
	req.Header.Set("X-Applad-Project", "proj-a")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAuthenticate_APIKeyAllowsReadScope(t *testing.T) {
	provider := testAPIKeyProvider{key: &model.APIKey{ID: "key1", ProjectID: "proj-a", Scopes: []string{"databases.read"}}}
	called := false
	handler := Authenticate("secret", provider)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/databases", nil)
	req.Header.Set("X-Applad-Key", "applad_key_abc123def456")
	req.Header.Set("X-Applad-Project", "proj-a")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestAuthenticate_InvalidJWT(t *testing.T) {
	var gotUser string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Authenticate("secret", nil)(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should pass through without setting user (Authenticate doesn't block)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotUser != "" {
		t.Fatal("expected empty user for invalid JWT")
	}
}

func TestAuthenticate_ExpiredJWT(t *testing.T) {
	secret := "test-secret"
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	var gotUser string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Authenticate(secret, nil)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if gotUser != "" {
		t.Fatal("expected empty user for expired JWT")
	}
}

func TestRequireAuth_WithAuth(t *testing.T) {
	// Chain: Authenticate -> RequireAuth -> handler
	secret := "test-secret"
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := Authenticate(secret, nil)(RequireAuth(inner))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Fatal("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	WriteJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["key"] != "value" {
		t.Fatalf("expected value, got %s", result["key"])
	}
}
