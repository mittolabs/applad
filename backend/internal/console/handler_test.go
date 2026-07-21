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
	h := &Handler{cookies: CookieConfig{Domain: ".applad.io", Secure: true}}

	rec := httptest.NewRecorder()
	h.setSignedIn(rec, "jwt-value-here")

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
	h.clearSignedIn(rec)
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge >= 0 {
			t.Errorf("%s should be expired on logout, MaxAge = %d", c.Name, c.MaxAge)
		}
	}
}

// Development serves the console over plain http, where a Secure cookie is
// silently dropped and nobody can stay signed in.
func TestCookiesNotSecureInDevelopment(t *testing.T) {
	h := &Handler{cookies: CookieConfig{Domain: "", Secure: false}}
	rec := httptest.NewRecorder()
	h.setSignedIn(rec, "token")
	for _, c := range rec.Result().Cookies() {
		if c.Secure {
			t.Errorf("%s must not be Secure in development", c.Name)
		}
	}
}
