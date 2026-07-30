package console

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

// PATCH /console/me updates the signed-in admin's name and email in one request
// (the account page's single "Save changes"). Before this route existed the
// console called a path with no handler and every profile edit 405'd.
func TestUpdateProfile_PatchMe(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(&db.DB{DB: mockDB}, "test-jwt-secret-32chars-long!!")
	h := &Handler{svc: svc}

	// An empty session id means ValidateSession trusts the JWT without a
	// session-table lookup, so no DB expectation is needed for auth.
	token, err := svc.signJWT("uid1", "old@test.com", "")
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec("UPDATE console_users SET name = \\$1 WHERE id = \\$2").
		WithArgs("New Name", "uid1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE console_users SET email = \\$1 WHERE id = \\$2").
		WithArgs("new@test.com", "uid1").WillReturnResult(sqlmock.NewResult(0, 1))
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT .+ FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "mfa_enabled", "created_at", "updated_at"}).
			AddRow("uid1", "new@test.com", "New Name", false, ts, ts))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/me", strings.NewReader(`{"name":"New Name","email":"new@test.com"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	Routes(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "new@test.com") {
		t.Errorf("expected updated user in response, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// An unauthenticated PATCH /console/me is rejected — the update never reaches
// the database.
func TestUpdateProfile_PatchMe_Unauthorized(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(&db.DB{DB: mockDB}, "test-jwt-secret-32chars-long!!")
	h := &Handler{svc: svc}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/me", strings.NewReader(`{"name":"x"}`))
	Routes(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

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
