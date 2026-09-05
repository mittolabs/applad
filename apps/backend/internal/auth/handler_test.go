package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/middleware"
)

// stubOAuthProvider is a test double for the OAuthProvider interface.
type stubOAuthProvider struct {
	exchangeErr error
}

func (s *stubOAuthProvider) GetAuthURL(redirectURI, state string) string {
	return "https://provider.example/authorize?state=" + url.QueryEscape(state)
}
func (s *stubOAuthProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (string, error) {
	return "access-token", s.exchangeErr
}
func (s *stubOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (OAuthUserInfo, error) {
	return OAuthUserInfo{}, nil
}

// TestOAuthRedirect_SetsStateNonceCookie: initiation stores a nonce cookie and
// embeds the same nonce as the first field of the OAuth state.
func TestOAuthRedirect_SetsStateNonceCookie(t *testing.T) {
	h := NewHandler(&Service{})
	h.SetOAuthProviders(map[string]OAuthProvider{"google": &stubOAuthProvider{}})

	req := httptest.NewRequest(http.MethodGet, "/sessions/oauth/google?success=/ok&failure=/fail", nil)
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Get("/sessions/oauth/{provider}", h.oauthRedirect)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", w.Code)
	}

	var nonce string
	for _, c := range w.Result().Cookies() {
		if c.Name == oauthStateCookie {
			nonce = c.Value
		}
	}
	if nonce == "" {
		t.Fatal("expected an oauth state nonce cookie to be set")
	}

	loc, _ := url.Parse(w.Header().Get("Location"))
	state := loc.Query().Get("state")
	if !strings.HasPrefix(state, nonce+"|") {
		t.Fatalf("expected state to begin with the cookie nonce, got %q", state)
	}
}

// oauthCallbackReq drives oauthCallback with the given state and optional cookie.
func oauthCallbackReq(t *testing.T, h *Handler, state, cookieNonce string) *httptest.ResponseRecorder {
	t.Helper()
	target := fmt.Sprintf("/sessions/oauth/google/callback?code=abc&state=%s", url.QueryEscape(state))
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookieNonce != "" {
		req.AddCookie(&http.Cookie{Name: oauthStateCookie, Value: cookieNonce})
	}
	w := httptest.NewRecorder()
	mux := chi.NewMux()
	mux.Get("/sessions/oauth/{provider}/callback", h.oauthCallback)
	mux.ServeHTTP(w, req)
	return w
}

// TestOAuthCallback_MissingNonceCookie: no cookie means the flow was not started
// by this browser, so the callback is rejected.
func TestOAuthCallback_MissingNonceCookie(t *testing.T) {
	h := NewHandler(&Service{})
	h.SetOAuthProviders(map[string]OAuthProvider{"google": &stubOAuthProvider{}})

	w := oauthCallbackReq(t, h, "somenonce|test-project|/ok|/fail", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with no nonce cookie, got %d", w.Code)
	}
}

// TestOAuthCallback_MismatchedNonce: a forged state whose nonce does not match
// the cookie is rejected.
func TestOAuthCallback_MismatchedNonce(t *testing.T) {
	h := NewHandler(&Service{})
	h.SetOAuthProviders(map[string]OAuthProvider{"google": &stubOAuthProvider{}})

	w := oauthCallbackReq(t, h, "attacker-nonce|test-project|/ok|/fail", "victim-nonce")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a mismatched nonce, got %d", w.Code)
	}
}

// TestOAuthCallback_MatchingNonce_Proceeds: a matching nonce passes the CSRF
// gate and proceeds to the token exchange (here made to fail, proving we got
// past the gate rather than being rejected at it).
func TestOAuthCallback_MatchingNonce_Proceeds(t *testing.T) {
	h := NewHandler(&Service{})
	h.SetOAuthProviders(map[string]OAuthProvider{"google": &stubOAuthProvider{exchangeErr: fmt.Errorf("boom")}})

	w := oauthCallbackReq(t, h, "good-nonce|test-project|/ok|/fail", "good-nonce")
	if w.Code == http.StatusForbidden {
		t.Fatalf("did not expect a CSRF rejection for a matching nonce, got %d", w.Code)
	}
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected a redirect after passing the CSRF gate, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "token_exchange_failed") {
		t.Fatalf("expected redirect to the failure URL, got %q", loc)
	}
}

// withProject is a chi middleware that injects a test project ID into the context
// using the same mechanism as middleware.ProjectContext.
func withProject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Applad-Project", "test-project")
		// Call the real ProjectContext middleware logic by setting the header
		// and relying on the handler reading from middleware.ProjectFromContext
		next.ServeHTTP(w, r)
	})
}

func TestCreateAccount_MissingFields(t *testing.T) {
	// Create a handler with a nil service — we expect validation to fail before DB call
	svc := &Service{} // no DB, will panic if actually called
	h := NewHandler(svc)

	tests := []struct {
		name string
		body map[string]string
		want int
	}{
		{"empty body", map[string]string{}, http.StatusBadRequest},
		{"missing password", map[string]string{"email": "a@b.com"}, http.StatusBadRequest},
		{"missing email", map[string]string{"password": "pass"}, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			// Wrap with project context middleware
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := r.Context()
					// Inject project ID using the middleware package
					r = r.WithContext(context.WithValue(ctx, projectCtxKeyType(4), "test-project"))
					next.ServeHTTP(w, r)
				})
			})
			mux.Post("/", h.createAccount)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

// projectCtxKeyType matches middleware.contextKey
type projectCtxKeyType int

func TestCreateAccount_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Post("/", h.createAccount)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAccountRoutes_Structure(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)
	router := AccountRoutes(h)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestUserRoutes_Structure(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)
	router := UserRoutes(h)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

// --- auth message templates ------------------------------------------------

// fakeMailer captures the last email so a test can assert what was sent.
type fakeMailer struct {
	to      []string
	subject string
	body    string
	calls   int
}

func (m *fakeMailer) SendEmail(_ context.Context, to []string, subject, htmlBody string) error {
	m.to, m.subject, m.body, m.calls = to, subject, htmlBody, m.calls+1
	return nil
}

// fakeTemplateResolver returns a fixed template for one key, mimicking a
// project that saved custom copy.
type fakeTemplateResolver struct {
	key     string
	subject string
	body    string
}

func (r fakeTemplateResolver) AuthEmailTemplate(_ context.Context, _, key string) (string, string, bool) {
	if key == r.key {
		return r.subject, r.body, true
	}
	return "", "", false
}

// magicLinkHandler builds a handler wired to a sqlmock service so the
// magic-link flow reaches the mailer without a real database.
func magicLinkHandler(t *testing.T, mailer EmailSender, resolver AuthTemplateResolver) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })
	// User exists, so CreateMagicLinkToken skips the insert and mints a token.
	mock.ExpectQuery("SELECT id FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	mock.ExpectExec("INSERT INTO auth_tokens").
		WillReturnResult(sqlmock.NewResult(1, 1))

	h := NewHandler(NewService(&db.DB{DB: mockDB}, "test-secret"))
	h.SetMailer(mailer)
	// The link carries a credential, so its callback has to be a target the
	// project registered. These tests use the one postMagicLink asks for.
	h.SetRedirectAllowlist(fixedAllowlist{"https://app.example/callback"})
	if resolver != nil {
		h.SetTemplateResolver(resolver)
	}
	return h, mock
}

// fixedAllowlist stands in for the projects service in a handler unit test.
type fixedAllowlist []string

func (f fixedAllowlist) RedirectURIsForProject(context.Context, string) ([]string, error) {
	return f, nil
}

func postMagicLink(h *Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/sessions/magic",
		bytes.NewReader([]byte(`{"email":"user@example.com","url":"https://app.example/callback"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Post("/sessions/magic", h.createMagicLink)
	mux.ServeHTTP(w, req)
	return w
}

// A project with a saved magic-link template: the mailer must receive the
// rendered custom subject and body, with {{url}} substituted.
func TestMagicLink_UsesCustomTemplate(t *testing.T) {
	mailer := &fakeMailer{}
	resolver := fakeTemplateResolver{
		key:     "magic",
		subject: "Your {{email}} sign-in link",
		body:    `<p>Tap <a href="{{url}}">here</a> to sign in.</p>`,
	}
	h, _ := magicLinkHandler(t, mailer, resolver)

	if w := postMagicLink(h); w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if mailer.calls != 1 {
		t.Fatalf("expected 1 email, got %d", mailer.calls)
	}
	if mailer.subject != "Your user@example.com sign-in link" {
		t.Fatalf("custom subject not rendered: %q", mailer.subject)
	}
	if !strings.Contains(mailer.body, `href="https://app.example/callback?secret=`) {
		t.Fatalf("custom body missing rendered url: %q", mailer.body)
	}
	if strings.Contains(mailer.body, "Sign in to Applad") {
		t.Fatalf("built-in body leaked despite custom template: %q", mailer.body)
	}
	if strings.Contains(mailer.body, "{{url}}") {
		t.Fatalf("placeholder left unrendered: %q", mailer.body)
	}
}

// Without a resolver (or when the project saved nothing) the built-in copy is
// used, so the fallback keeps working.
func TestMagicLink_FallsBackToBuiltInCopy(t *testing.T) {
	mailer := &fakeMailer{}
	// Resolver only has copy for "recovery", so "magic" falls through.
	resolver := fakeTemplateResolver{key: "recovery", subject: "x", body: "y"}
	h, _ := magicLinkHandler(t, mailer, resolver)

	if w := postMagicLink(h); w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if mailer.subject != "Sign in to Applad" {
		t.Fatalf("expected built-in subject, got %q", mailer.subject)
	}
	if !strings.Contains(mailer.body, "Sign in to Applad") {
		t.Fatalf("expected built-in body, got %q", mailer.body)
	}
}

// A variable rendered into a custom subject must not be able to inject a header:
// a value carrying "\r\nBcc: ..." is folded so the resolved subject stays one
// line and contains no newline, while the body may keep its newlines.
func TestResolveAuthMessage_SubjectInjection_Stripped(t *testing.T) {
	h := &Handler{templates: fakeTemplateResolver{
		key:     "verification",
		subject: "Verify {{email}}",
		body:    "<p>Hi {{email}}</p>",
	}}
	vars := map[string]string{"email": "victim@test.com\r\nBcc: attacker@evil.com"}
	subject, _ := h.resolveAuthMessage(context.Background(), "proj", "verification",
		"Verify your email", "<p>default</p>", vars)
	if strings.ContainsAny(subject, "\r\n") {
		t.Fatalf("subject still contains a newline: %q", subject)
	}
	if !strings.HasPrefix(subject, "Verify victim@test.com") {
		t.Fatalf("subject not folded as expected: %q", subject)
	}
}

// A legitimate custom subject with a clean variable renders unchanged, so the
// sanitization does not disturb the normal path.
func TestResolveAuthMessage_LegitimateSubject_Unchanged(t *testing.T) {
	h := &Handler{templates: fakeTemplateResolver{
		key:     "verification",
		subject: "Verify {{email}}",
		body:    "<p>Hi {{email}}</p>",
	}}
	vars := map[string]string{"email": "user@test.com"}
	subject, body := h.resolveAuthMessage(context.Background(), "proj", "verification",
		"Verify your email", "<p>default</p>", vars)
	if subject != "Verify user@test.com" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if body != "<p>Hi user@test.com</p>" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestRenderAuthMessage(t *testing.T) {
	got := renderAuthMessage("Hi {{name}}, code {{otp}}", map[string]string{"name": "Sam", "otp": "123456"})
	if got != "Hi Sam, code 123456" {
		t.Fatalf("unexpected render: %q", got)
	}
	// Unknown placeholders are preserved rather than blanked.
	if got := renderAuthMessage("{{missing}}", map[string]string{}); got != "{{missing}}" {
		t.Fatalf("expected placeholder preserved, got %q", got)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected application/json; charset=utf-8, got %s", ct)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["hello"] != "world" {
		t.Fatalf("expected 'world', got %s", result["hello"])
	}
}

func TestCreateEmailSession_InvalidJSON(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/sessions/email", bytes.NewReader([]byte("{invalid")))
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Post("/sessions/email", h.createEmailSession)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetAccount_NoUser(t *testing.T) {
	// getAccount reads UserFromContext - if no user is set, it should 404
	svc := &Service{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Chain: ProjectContext -> Authenticate -> RequireAuth -> handler
	// Without auth, RequireAuth blocks it
	mux := chi.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
			next.ServeHTTP(w, r)
		})
	})
	mux.Use(middleware.RequireAuth)
	mux.Get("/", h.getAccount)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// A callback the project did not register is refused rather than emailed.
//
// The link carries a single-use credential and the caller chooses both the URL
// and the address the mail goes to, so an unregistered target has to be a
// refusal — not a link that is quietly sent somewhere else.
func TestMagicLink_RefusesUnregisteredCallback(t *testing.T) {
	mailer := &fakeMailer{}
	h, _ := magicLinkHandler(t, mailer, nil)
	// The harness registers https://app.example/callback and nothing else.

	req := httptest.NewRequest(http.MethodPost, "/sessions/magic",
		bytes.NewReader([]byte(`{"email":"user@example.com","url":"https://attacker.example/collect"}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := chi.NewMux()
	mux.Post("/sessions/magic", h.createMagicLink)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unregistered callback, got %d: %s", w.Code, w.Body.String())
	}
	if mailer.calls != 0 {
		t.Fatalf("nothing should have been sent, got %d emails", mailer.calls)
	}
}
