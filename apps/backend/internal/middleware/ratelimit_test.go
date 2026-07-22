package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimit_AllowsNormal(t *testing.T) {
	handler := RateLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimit_BlocksExcessive(t *testing.T) {
	handler := RateLimit(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "5.6.7.8:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if i < 5 && w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
		if i == 5 && w.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d: expected 429, got %d", i, w.Code)
		}
	}
}

func TestRateLimit_DifferentIPs(t *testing.T) {
	handler := RateLimit(2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP 1: 2 requests OK
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("ip1 request %d: expected 200, got %d", i, w.Code)
		}
	}

	// IP 2: still allowed
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ip2: expected 200, got %d", w.Code)
	}
}

// A console page issues twenty or more requests to render, so a signed-in
// caller cannot share a bucket sized for anonymous floods — a few refreshes
// exhausted it and the whole page failed.
func TestRateLimitTiersSignedInCallersSeparately(t *testing.T) {
	anon := newInMemoryLimiter(2, time.Minute)
	authed := newInMemoryLimiter(50, time.Minute)

	// The anonymous bucket is small on purpose.
	if !anon.allow("1.2.3.4") || !anon.allow("1.2.3.4") {
		t.Fatal("the first two anonymous requests should be allowed")
	}
	if anon.allow("1.2.3.4") {
		t.Error("the third anonymous request should be refused")
	}

	// The same address, signed in, is counted against its own larger budget.
	for i := 0; i < 50; i++ {
		if !authed.allow("1.2.3.4") {
			t.Fatalf("signed-in request %d was refused", i+1)
		}
	}
}

func TestHasSessionCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/v1/projects", nil)
	if hasSessionCookie(r) {
		t.Error("a request with no cookies was treated as signed in")
	}
	r.AddCookie(&http.Cookie{Name: "applad_session", Value: "x"})
	if !hasSessionCookie(r) {
		t.Error("a session cookie was not recognised")
	}
}
