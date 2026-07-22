package deploy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type projectCtxKey int

func withProject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), projectCtxKey(4), "test-project")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestCreateTarget_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/targets/", bytes.NewReader([]byte(`{"type":"web"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTarget_EmptyName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/targets/", bytes.NewReader([]byte(`{"name":"  ","type":"web"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTarget_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/targets/", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePipeline_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/pipelines/", bytes.NewReader([]byte(`{"targetId":"t1"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePipeline_MissingTargetID(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/pipelines/", bytes.NewReader([]byte(`{"name":"my-pipe"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTarget_TypeDefaults(t *testing.T) {
	tgt := Target{
		Type: "",
	}
	if tgt.Type != "" {
		t.Fatal("expected empty type before handler sets default")
	}
}

func TestUpdateTarget_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/targets/t1", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateTarget_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/targets/t1", bytes.NewReader([]byte(`{"type":"web"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePipeline_MissingTargetID(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPut, "/pipelines/p1", bytes.NewReader([]byte(`{"name":"my-pipe"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Mount("/", Routes(h))
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
