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

	"github.com/go-chi/chi/v5"
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
