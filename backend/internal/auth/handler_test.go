package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/middleware"
)

// withProjectCtx sets the project ID in the request context for testing.
func withProjectCtx(r *http.Request, projectID string) *http.Request {
	ctx := context.WithValue(r.Context(), projectCtxKey, projectID)
	return r.WithContext(ctx)
}

// projectCtxKey matches the unexported key used by middleware.ProjectContext.
// We use middleware.ProjectContext in tests instead.

func TestCreateAccount_MissingFields(t *testing.T) {
	// Create a handler with a nil service — we expect validation to fail before DB call
	svc := &Service{} // no DB, will panic if actually called
	h := NewHandler(svc)

	tests := []struct {
		name string
		body map[string]string
		want int
	}{
		{"empty body", map[string]string{}, http.StatusBadRequest},
		{"missing password", map[string]string{"email": "a@b.com"}, http.StatusBadRequest},
		{"missing email", map[string]string{"password": "pass"}, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			// Wrap with project context middleware
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := r.Context()
					// Inject project ID using the middleware package
					r = r.WithContext(context.WithValue(ctx, projectCtxKeyType(4), "test-project"))
					next.ServeHTTP(w, r)
				})
			})
			mux.Post("/", h.createAccount)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

// projectCtxKeyType matches middleware.contextKey
type projectCtxKeyType int

func TestCreateAccount_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Post("/", h.createAccount)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAccountRoutes_Structure(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)
	router := AccountRoutes(h)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestUserRoutes_Structure(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)
	router := UserRoutes(h)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["hello"] != "world" {
		t.Fatalf("expected 'world', got %s", result["hello"])
	}
}

func TestCreateEmailSession_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/sessions/email", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Post("/sessions/email", h.createEmailSession)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetAccount_NoUser(t *testing.T) {
	// getAccount reads UserFromContext - if no user is set, it should 404
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Chain: ProjectContext -> Authenticate -> RequireAuth -> handler
	// Without auth, RequireAuth blocks it
	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Use(middleware.RequireAuth)
	mux.Get("/", h.getAccount)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
