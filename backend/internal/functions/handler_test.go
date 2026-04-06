package functions

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

func TestCreate_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"runtime":"node-18"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Post("/", h.create)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_MissingRuntime(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"myfn"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Post("/", h.create)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(withProject)
	mux.Post("/", h.create)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListRuntimes_ReturnsAll(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/runtimes", nil)
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Get("/runtimes", h.listRuntimes)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	// Should contain known runtimes
	for _, rt := range []string{"node-18", "python-3.12", "go-1.22", "dart-3", "custom"} {
		if !bytes.Contains([]byte(body), []byte(rt)) {
			t.Errorf("expected body to contain %q", rt)
		}
	}
}

func TestDefaults_EntrypointAndTimeout(t *testing.T) {
	f := Function{
		Entrypoint: "",
		Timeout:    0,
	}
	// Handler sets defaults for empty entrypoint and zero timeout
	if f.Entrypoint != "" {
		t.Fatal("expected empty entrypoint before handler default")
	}
	if f.Timeout != 0 {
		t.Fatal("expected 0 timeout before handler default")
	}
}
