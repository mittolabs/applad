package avatars

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
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

// ---------------------------------------------------------------------------
// SSRF guard: the favicon + image proxies fetch attacker-controlled URLs and
// must route through netguard, which refuses loopback / link-local / private
// destinations even after DNS resolution and across redirects.
// ---------------------------------------------------------------------------

// A loopback target (stand-in for an internal compose service or the metadata
// endpoint) must not be streamed back: the image proxy returns 502 and never
// echoes the internal body.
func TestRemoteImage_BlocksLoopback(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "SECRET-INTERNAL-RESPONSE")
	}))
	defer internal.Close()

	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/image?url="+url.QueryEscape(internal.URL), nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for a loopback target, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "SECRET-INTERNAL-RESPONSE") {
		t.Fatal("internal response leaked through the image proxy")
	}
}

// The favicon proxy swallows fetch failures into a default icon, so a blocked
// loopback fetch must yield the default PNG rather than the target's body.
func TestFavicon_BlocksLoopback(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		io.WriteString(w, "INTERNAL-FAVICON-BYTES")
	}))
	defer internal.Close()

	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/favicon?url="+url.QueryEscape(internal.URL), nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (default favicon), got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "INTERNAL-FAVICON-BYTES") {
		t.Fatal("internal favicon body leaked through the proxy")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "image/png") {
		t.Fatalf("expected the default favicon (image/png), got %q", ct)
	}
}

// The success path: a permitted address is fetched and streamed through
// unchanged. netguard refuses every local address, so we relax the egress
// policy in a child process (ALLOW_PRIVATE_EGRESS=true) and let a loopback test
// server stand in for a genuine public host — the only way to exercise the
// allow path in-process without weakening the guard for the block tests above.
func TestRemoteImage_AllowsPermittedAddress(t *testing.T) {
	if os.Getenv("AVATARS_EGRESS_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run", "^TestRemoteImage_AllowsPermittedAddress$", "-test.v")
		cmd.Env = append(os.Environ(), "AVATARS_EGRESS_CHILD=1", "ALLOW_PRIVATE_EGRESS=true")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child run failed: %v\n%s", err, out)
		}
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	pngBytes := buf.Bytes()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBytes)
	}))
	defer origin.Close()

	srv := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/image?url="+url.QueryEscape(origin.URL), nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a permitted fetch, got %d", rec.Code)
	}
	if !bytes.Equal(rec.Body.Bytes(), pngBytes) {
		t.Fatal("expected the fetched image to stream through unchanged")
	}
}
