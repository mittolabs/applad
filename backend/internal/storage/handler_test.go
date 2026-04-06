package storage

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

func TestCreateBucket_MissingName(t *testing.T) {
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
			req := httptest.NewRequest(http.MethodPost, "/buckets", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(withProject)
			mux.Post("/buckets", h.createBucket)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateFile_NoMultipart(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/buckets/b1/files", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Post("/buckets/{bucketId}/files", h.createFile)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
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
	writeJSON(w, http.StatusCreated, map[string]string{"id": "abc"})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["id"] != "abc" {
		t.Fatalf("expected 'abc', got %s", result["id"])
	}
}
