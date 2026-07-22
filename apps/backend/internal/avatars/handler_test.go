package avatars

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestRouter() http.Handler {
	r := chi.NewRouter()
	h := NewHandler()
	r.Mount("/", Routes(h))
	return r
}

func TestInitials_ReturnsImage(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/initials?name=John+Doe&width=100&height=100", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "image/png") {
		t.Fatalf("expected Content-Type containing image/png, got %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected non-empty body")
	}
}

func TestInitials_DefaultSize(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/initials?name=A", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "image/png") {
		t.Fatalf("expected Content-Type containing image/png, got %q", ct)
	}
}

func TestQR_ReturnsSVG(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/qr?text=hello", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "svg") {
		t.Fatalf("expected Content-Type containing svg, got %q", ct)
	}
}

func TestQR_MissingText(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/qr", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreditCard_ValidCode(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/credit-cards/visa", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "svg") {
		t.Fatalf("expected Content-Type containing svg, got %q", ct)
	}
}

func TestCreditCard_UnknownCode(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/credit-cards/unknown", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Unknown card codes return 404.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestFlag_ValidCode(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/flags/US", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "svg") {
		t.Fatalf("expected Content-Type containing svg, got %q", ct)
	}
}

func TestFavicon_MissingURL(t *testing.T) {
	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/favicon", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// Missing url parameter returns 400.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
