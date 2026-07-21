package console

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The hint cookie is what lets the marketing site on the parent domain know
// someone is signed in. It must be scoped to that parent domain, carry no
// token, and stay readable by JavaScript — unlike the real session cookie.
func TestSignedInCookies(t *testing.T) {
	h := &Handler{cookies: CookieConfig{Domain: ".applad.io"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/console/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https") // as our proxy sets it
	h.setSignedIn(rec, req, "jwt-value-here")

	var session, hint *http.Cookie
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case "a_session_console":
			session = c
		case SessionHintCookie:
			hint = c
		}
	}
	if session == nil || hint == nil {
		t.Fatalf("expected both cookies, got %v", rec.Result().Cookies())
	}

	if hint.Value == "jwt-value-here" || strings.Contains(hint.Value, ".") {
		t.Errorf("hint cookie must not carry the token, got %q", hint.Value)
	}
	if hint.HttpOnly {
		t.Error("hint cookie must be readable by JavaScript")
	}
	if !session.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}
	for _, c := range []*http.Cookie{session, hint} {
		// Go drops the leading dot when writing the header. Per RFC 6265 a
		// Domain attribute always matches subdomains, so "applad.io" still
		// covers console.applad.io — which is the point of setting it.
		if c.Domain != "applad.io" {
			t.Errorf("%s domain = %q, want applad.io", c.Name, c.Domain)
		}
		if !c.Secure {
			t.Errorf("%s should be Secure in production", c.Name)
		}
		// Strict would withhold the cookie when arriving from a link on the
		// marketing site, which is the main way people reach the console.
		if c.SameSite != http.SameSiteLaxMode {
			t.Errorf("%s SameSite = %v, want Lax", c.Name, c.SameSite)
		}
	}

	rec = httptest.NewRecorder()
	h.clearSignedIn(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge >= 0 {
			t.Errorf("%s should be expired on logout, MaxAge = %d", c.Name, c.MaxAge)
		}
	}
}

// A Secure cookie sent over plain http is silently dropped and nobody stays
// signed in. This must follow the request scheme, not the environment: a
// self-hosted install runs APP_ENV=production over http on a bare VPS.
func TestCookiesFollowRequestScheme(t *testing.T) {
	h := &Handler{cookies: CookieConfig{}}

	rec := httptest.NewRecorder()
	h.setSignedIn(rec, httptest.NewRequest("POST", "/console/login", nil), "token")
	for _, c := range rec.Result().Cookies() {
		if c.Secure {
			t.Errorf("%s must not be Secure over plain http", c.Name)
		}
	}

	rec = httptest.NewRecorder()
	tls := httptest.NewRequest("POST", "/console/login", nil)
	tls.Header.Set("X-Forwarded-Proto", "https")
	h.setSignedIn(rec, tls, "token")
	for _, c := range rec.Result().Cookies() {
		if !c.Secure {
			t.Errorf("%s must be Secure behind TLS", c.Name)
		}
	}
}

// A console at console.<parent> must share cookies with <parent>, or the
// marketing site cannot tell that anyone is signed in. Anything reached by IP
// or bare hostname — a self-hosted install — must stay host-only, so no
// configuration is needed there.
func TestCookieDomainDerivedFromHost(t *testing.T) {
	h := &Handler{}
	tests := map[string]string{
		"console.applad.io":           ".applad.io",
		"console.applad.io.localhost": ".applad.io.localhost",
		"console.example.com:8080":    ".example.com",
		"applad.io":                   "", // not the console host
		"localhost":                   "",
		"192.168.1.10":                "",
		"console.localhost":           "", // no parent domain to share with
	}
	for host, want := range tests {
		r := httptest.NewRequest("POST", "/console/login", nil)
		r.Host = host
		if got := h.cookieDomain(r); got != want {
			t.Errorf("cookieDomain(%q) = %q, want %q", host, got, want)
		}
	}

	// An explicit setting always wins.
	forced := &Handler{cookies: CookieConfig{Domain: ".override.test"}}
	r := httptest.NewRequest("POST", "/console/login", nil)
	r.Host = "console.applad.io"
	if got := forced.cookieDomain(r); got != ".override.test" {
		t.Errorf("configured domain = %q, want .override.test", got)
	}
}
