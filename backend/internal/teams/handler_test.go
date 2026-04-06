package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// projectCtxKeyType matches middleware.contextKey
type projectCtxKeyType int

func withProject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
		next.ServeHTTP(w, r)
	})
}

func TestCreateTeam_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"empty name", `{"name":""}`, http.StatusBadRequest},
		{"whitespace name", `{"name":"  "}`, http.StatusBadRequest},
		{"invalid json", `bad`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(withProject)
			mux.Post("/", h.createTeam)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateMembership_MissingEmail(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"no email", `{"roles":["admin"]}`, http.StatusBadRequest},
		{"empty email", `{"email":""}`, http.StatusBadRequest},
		{"invalid json", `bad`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/t1/memberships", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(withProject)
			mux.Post("/{teamId}/memberships", h.createMembership)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestRoutes_Structure(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)
	router := Routes(h)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]int{"total": 0})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]int
	json.NewDecoder(w.Body).Decode(&result)
	if result["total"] != 0 {
		t.Fatalf("expected total=0, got %d", result["total"])
	}
}
