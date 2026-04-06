package databases

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// projectCtxKeyType matches the middleware's context key type.
type projectCtxKeyType int

func withProject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
		next.ServeHTTP(w, r)
	})
}

func TestCreateDatabase_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"empty name", `{"name":""}`, http.StatusBadRequest},
		{"whitespace name", `{"name":"   "}`, http.StatusBadRequest},
		{"invalid json", `bad`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(withProject)
			mux.Post("/", h.createDatabase)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateCollection_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"empty name", `{"name":""}`, http.StatusBadRequest},
		{"invalid json", `{bad}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/db1/collections", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(withProject)
			mux.Post("/{databaseId}/collections", h.createCollection)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateAttribute_MissingKey(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"no key", `{"required":true}`, http.StatusBadRequest},
		{"invalid json", `bad`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/db1/collections/c1/attributes/string", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Post("/{databaseId}/collections/{collectionId}/attributes/string", h.createAttr("string"))
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateIndex_MissingKey(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/db1/collections/c1/indexes", bytes.NewReader([]byte(`{"type":"key"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Post("/{databaseId}/collections/{collectionId}/indexes", h.createIndex)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateDocument_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/db1/collections/c1/documents", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Post("/{databaseId}/collections/{collectionId}/documents", h.createDocument)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
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
	writeJSON(w, http.StatusOK, map[string]int{"count": 5})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]int
	json.NewDecoder(w.Body).Decode(&result)
	if result["count"] != 5 {
		t.Fatalf("expected count=5, got %d", result["count"])
	}
}
