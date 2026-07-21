package console

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	oauthpkg "github.com/mittolabs/applad/internal/oauth"
)

// SMTPConfig holds optional SMTP settings for sending password-reset emails.
type SMTPConfig struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// Handler handles HTTP requests for console auth.
type Handler struct {
	svc           *Service
	signupSetting string // "auto", "true", or "false"
	providers     map[string]*oauthpkg.Provider
	smtp          SMTPConfig
	cookies       CookieConfig
}

// CookieConfig controls how console cookies are scoped.
type CookieConfig struct {
	// Domain scopes cookies to a parent domain, e.g. ".applad.io" so the
	// marketing site can see that someone is signed in. Empty keeps them
	// host-only.
	Domain string
	// Secure marks cookies HTTPS-only. Off in development, where the console
	// is served over plain http and a Secure cookie would simply be dropped.
	Secure bool
}

// SessionHintCookie is a deliberately non-sensitive marker that someone is
// signed in. It carries no token — it exists so pages on the parent domain
// (the marketing site) can render "Go to console" instead of "Get started".
// Readable by JavaScript by design; the real session never is.
const SessionHintCookie = "applad_session"

// NewHandler creates a new console auth Handler.
func NewHandler(svc *Service, signupSetting string, smtpCfg SMTPConfig, cookies CookieConfig) *Handler {
	return &Handler{
		svc:           svc,
		signupSetting: signupSetting,
		providers:     map[string]*oauthpkg.Provider{},
		smtp:          smtpCfg,
		cookies:       cookies,
	}
}

// setSignedIn writes both the session cookie and the public hint cookie.
func (h *Handler) setSignedIn(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_console",
		Value:    token,
		Path:     "/",
		Domain:   h.cookies.Domain,
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		// Lax rather than Strict: the console is reached by following a link
		// from the marketing site, and Strict would withhold the cookie on
		// that first navigation.
		SameSite: http.SameSiteLaxMode,
		MaxAge:   8 * 3600,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     SessionHintCookie,
		Value:    "1",
		Path:     "/",
		Domain:   h.cookies.Domain,
		HttpOnly: false, // the marketing site reads this from JavaScript
		Secure:   h.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   8 * 3600,
	})
}

// clearSignedIn expires both cookies.
func (h *Handler) clearSignedIn(w http.ResponseWriter) {
	for _, name := range []string{"a_session_console", SessionHintCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Domain:   h.cookies.Domain,
			HttpOnly: name == "a_session_console",
			Secure:   h.cookies.Secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
	}
}

// SetProviders registers OAuth providers available for console login.
func (h *Handler) SetProviders(p map[string]*oauthpkg.Provider) {
	h.providers = p
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the console auth router (all public, no project header).
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/signup", h.signup)
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Get("/signup-status", h.signupStatus)
	r.Get("/auth-providers", h.listAuthProviders)
	r.Get("/auth/{provider}", h.oauthRedirect)
	r.Get("/auth/{provider}/callback", h.oauthCallback)
	r.Get("/me", h.getMe)
	r.Patch("/me/name", h.updateName)
	r.Patch("/me/email", h.updateEmail)
	r.Patch("/me/password", h.updatePassword)
	r.Delete("/me", h.deleteAccount)
	r.Get("/sessions", h.listSessions)
	r.Delete("/sessions/{id}", h.revokeSession)
	r.Post("/password-reset/request", h.requestPasswordReset)
	r.Post("/password-reset/confirm", h.confirmPasswordReset)
	return r
}

// bearerToken extracts the raw JWT from the Authorization header.
func bearerToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(auth, "Bearer "), true
}

// clientIP returns the best-guess client IP, honouring the reverse proxy.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return xr
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) {
	// Check if signup is enabled
	enabled, err := h.svc.SignupEnabled(r.Context(), h.signupSetting)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if !enabled {
		apperr.Write(w, http.StatusForbidden, "signup_disabled", "Signup is disabled. Contact your administrator.")
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Email == "" || body.Password == "" {
		apperr.BadRequest(w, "email and password are required")
		return
	}
	if len(body.Password) < 8 {
		apperr.BadRequest(w, "password must be at least 8 characters")
		return
	}

	user, _, err := h.svc.Signup(r.Context(), body.Email, body.Password, body.Name)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			apperr.Conflict(w, "email already in use")
			return
		}
		apperr.Internal(w, err)
		return
	}

	token, err := h.svc.CreateSessionToken(r.Context(), user.ID, user.Email, r.UserAgent(), clientIP(r))
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	h.setSignedIn(w, token)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Email == "" || body.Password == "" {
		apperr.BadRequest(w, "email and password are required")
		return
	}

	user, _, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		apperr.Write(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	token, err := h.svc.CreateSessionToken(r.Context(), user.ID, user.Email, r.UserAgent(), clientIP(r))
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	h.setSignedIn(w, token)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.clearSignedIn(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// listAuthProviders returns the names of OAuth providers configured for console login.
func (h *Handler) listAuthProviders(w http.ResponseWriter, r *http.Request) {
	names := make([]string, 0, len(h.providers))
	for name := range h.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	writeJSON(w, http.StatusOK, map[string]interface{}{"providers": names})
}

// oauthRedirect starts the OAuth2 flow for console login.
func (h *Handler) oauthRedirect(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	p, ok := h.providers[providerName]
	if !ok {
		// This is a full-page navigation from the login screen, so send the
		// user back to a rendered page instead of a raw JSON error.
		http.Redirect(w, r, "/login?error=oauth_unavailable", http.StatusTemporaryRedirect)
		return
	}
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	callbackURL := fmt.Sprintf("%s://%s/v1/console/auth/%s/callback", scheme, r.Host, providerName)
	// State carries the post-login redirect target (validated on callback).
	state := safeConsoleRedirect(r.URL.Query().Get("redirect"))
	http.Redirect(w, r, p.GetAuthURL(callbackURL, state), http.StatusTemporaryRedirect)
}

// oauthCallback handles the OAuth2 provider callback for console login.
func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	p, ok := h.providers[providerName]
	if !ok {
		http.Redirect(w, r, "/login?error=oauth_unavailable", http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	// Re-validate the state redirect to prevent open-redirect via tampered state.
	redirectTo := safeConsoleRedirect(r.URL.Query().Get("state"))

	if code == "" {
		http.Redirect(w, r, "/login?error=oauth_cancelled", http.StatusTemporaryRedirect)
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	callbackURL := fmt.Sprintf("%s://%s/v1/console/auth/%s/callback", scheme, r.Host, providerName)

	ctx := r.Context()
	accessToken, err := p.ExchangeCode(ctx, code, callbackURL)
	if err != nil {
		http.Redirect(w, r, "/login?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	userInfo, err := p.GetUserInfo(ctx, accessToken)
	if err != nil {
		http.Redirect(w, r, "/login?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	user, _, err := h.svc.LoginOrCreateByOAuth(ctx, userInfo.Email, userInfo.Name, providerName, h.signupSetting)
	if err != nil {
		if strings.Contains(err.Error(), "signup disabled") {
			http.Redirect(w, r, "/login?error=signup_disabled", http.StatusTemporaryRedirect)
		} else {
			http.Redirect(w, r, "/login?error=oauth_failed", http.StatusTemporaryRedirect)
		}
		return
	}

	token, err := h.svc.CreateSessionToken(ctx, user.ID, user.Email, r.UserAgent(), clientIP(r))
	if err != nil {
		http.Redirect(w, r, "/login?error=oauth_failed", http.StatusTemporaryRedirect)
		return
	}

	h.setSignedIn(w, token)

	// Pass the token via query param so the Flutter SPA can capture it,
	// store in localStorage, and complete the login without a round-trip.
	dest := redirectTo
	if dest == "/" || dest == "" {
		dest = "/login"
	}
	http.Redirect(w, r, dest+"?console_token="+url.QueryEscape(token), http.StatusTemporaryRedirect)
}

// safeConsoleRedirect returns the URL only if relative (no scheme/host).
func safeConsoleRedirect(raw string) string {
	if raw == "" {
		return "/login"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "/login"
	}
	return raw
}

func (h *Handler) signupStatus(w http.ResponseWriter, r *http.Request) {
	enabled, err := h.svc.SignupEnabled(r.Context(), h.signupSetting)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"signupEnabled": enabled,
	})
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		apperr.Unauthorized(w)
		return
	}

	user, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		apperr.NotFound(w, "console user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) extractUserID(r *http.Request) (string, error) {
	tok, ok := bearerToken(r)
	if !ok {
		return "", fmt.Errorf("no token")
	}
	// ValidateSession also rejects revoked sessions, so revoking a session logs
	// that device out on its next request.
	userID, _, err := h.svc.ValidateSession(r.Context(), tok)
	return userID, err
}

// listSessions returns the caller's active sessions, flagging the current one.
func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	tok, ok := bearerToken(r)
	if !ok {
		apperr.Unauthorized(w)
		return
	}
	userID, sessionID, err := h.svc.ValidateSession(r.Context(), tok)
	if err != nil {
		apperr.Unauthorized(w)
		return
	}
	sessions, err := h.svc.ListSessions(r.Context(), userID, sessionID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": sessions})
}

// revokeSession signs the given session out (ownership enforced).
func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		apperr.Unauthorized(w)
		return
	}
	if err := h.svc.RevokeSession(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		apperr.NotFound(w, "session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) updateName(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		apperr.Unauthorized(w)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.UpdateName(r.Context(), userID, body.Name); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) updateEmail(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		apperr.Unauthorized(w)
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.UpdateEmail(r.Context(), userID, body.Email); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) updatePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		apperr.Unauthorized(w)
		return
	}
	var body struct {
		OldPassword string `json:"oldPassword"`
		Password    string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Password == "" {
		apperr.BadRequest(w, "password is required")
		return
	}
	if err := h.svc.UpdatePassword(r.Context(), userID, body.OldPassword, body.Password); err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, err := h.extractUserID(r)
	if err != nil {
		apperr.Unauthorized(w)
		return
	}
	if err := h.svc.DeleteUser(r.Context(), userID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requestPasswordReset generates a reset token for the given email.
// If SMTP is configured the token is emailed; otherwise it is returned in the
// response body so the admin can share it out-of-band.
func (h *Handler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Email == "" {
		apperr.BadRequest(w, "email is required")
		return
	}

	token, found, err := h.svc.RequestPasswordReset(r.Context(), body.Email)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if !found {
		// Don't reveal whether the email exists.
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"emailSent": false,
			"message":   "If that email is registered, a reset link has been sent.",
		})
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	resetURL := fmt.Sprintf("%s://%s/login?reset_token=%s", scheme, r.Host, token)

	emailSent := false
	if h.smtp.Host != "" {
		if err := h.sendResetEmail(body.Email, resetURL); err == nil {
			emailSent = true
		}
	}

	if !emailSent {
		// SMTP not configured — log the reset URL to server output only.
		// Never return the token in the HTTP response.
		log.Printf("[console] SMTP not configured. Password reset URL for %s: %s", body.Email, resetURL)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"emailSent": emailSent,
		"message":   "If that email is registered, a reset link has been sent.",
	})
}

// confirmPasswordReset validates a reset token and sets a new password.
func (h *Handler) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Token == "" || body.Password == "" {
		apperr.BadRequest(w, "token and password are required")
		return
	}
	if len(body.Password) < 8 {
		apperr.BadRequest(w, "password must be at least 8 characters")
		return
	}
	if err := h.svc.ConfirmPasswordReset(r.Context(), body.Token, body.Password); err != nil {
		apperr.Write(w, http.StatusBadRequest, "invalid_token", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Password updated. Please sign in."})
}

func (h *Handler) sendResetEmail(to, resetURL string) error {
	auth := smtp.PlainAuth("", h.smtp.User, h.smtp.Pass, h.smtp.Host)
	msg := []byte("Subject: Reset your Applad console password\r\n" +
		"From: " + h.smtp.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		"Hi,\r\n\r\n" +
		"Click the link below to reset your Applad console password.\r\n" +
		"This link expires in 1 hour.\r\n\r\n" +
		resetURL + "\r\n\r\n" +
		"If you did not request this, you can safely ignore this email.\r\n\r\n" +
		"— Applad Console\r\n")
	return smtp.SendMail(h.smtp.Host+":"+h.smtp.Port, auth, h.smtp.From, []string{to}, msg)
}
