package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// e164Re validates E.164 phone numbers: +<country code><number>, 8–15 digits total.
var e164Re = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

// safeRedirectURL returns the URL only if it is relative (no scheme/host).
// This prevents open-redirect attacks where attacker-controlled URLs are used.
func safeRedirectURL(raw string) string {
	if raw == "" {
		return "/"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "/"
	}
	return raw
}

// EmailSender sends emails for auth flows.
type EmailSender interface {
	SendEmail(ctx context.Context, to []string, subject, htmlBody string) error
}

// SMSSender sends SMS for phone OTP flows.
type SMSSender interface {
	SendSMS(ctx context.Context, to, body string) error
}

// Handler handles HTTP requests for auth.
type Handler struct {
	svc            *Service
	oauthProviders map[string]OAuthProvider
	mailer         EmailSender
	smsSender      SMSSender
}

// SetMailer sets the email sender for auth flows (magic link, verification, reset).
func (h *Handler) SetMailer(m EmailSender) {
	h.mailer = m
}

// SetSMSSender sets the SMS sender for phone OTP flows.
func (h *Handler) SetSMSSender(s SMSSender) {
	h.smsSender = s
}

// OAuthProvider is the interface for OAuth2 provider operations.
type OAuthProvider interface {
	GetAuthURL(redirectURI, state string) string
	ExchangeCode(ctx context.Context, code, redirectURI string) (string, error)
	GetUserInfo(ctx context.Context, accessToken string) (OAuthUserInfo, error)
}

// OAuthUserInfo is normalized OAuth user info.
type OAuthUserInfo struct {
	ID       string
	Email    string
	Name     string
	Provider string
}

// NewHandler creates a new auth Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// AccountRoutes returns the account router (client-side).
func AccountRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Public endpoints
	r.Post("/", h.createAccount)
	r.Post("/sessions/email", h.createEmailSession)
	r.Post("/sessions/anonymous", h.createAnonymousSession)
	r.Get("/sessions/oauth/{provider}", h.oauthRedirect)
	r.Get("/sessions/oauth/{provider}/callback", h.oauthCallback)

	// Phone OTP
	r.Post("/sessions/phone", h.sendPhoneOTP)
	r.Put("/sessions/phone", h.verifyPhoneOTP)

	// Public auth flows
	r.Post("/sessions/magic-link", h.createMagicLink)
	r.Put("/sessions/magic-link", h.redeemMagicLink)
	r.Post("/verification", h.createEmailVerification)
	r.Put("/verification", h.verifyEmail)
	r.Post("/recovery", h.createPasswordReset)
	r.Put("/recovery", h.resetPassword)

	// Authenticated endpoints
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Get("/", h.getAccount)
		r.Patch("/name", h.updateName)
		r.Patch("/email", h.updateEmail)
		r.Patch("/password", h.updatePassword)
		r.Patch("/prefs", h.updatePrefs)
		r.Delete("/", h.deleteAccount)
		r.Get("/sessions", h.listSessions)
		r.Get("/sessions/{sessionId}", h.getSession)
		r.Delete("/sessions/{sessionId}", h.deleteSession)
		r.Delete("/sessions", h.deleteSessions)
		r.Post("/jwt", h.getJWT)
		r.Post("/mfa", h.enableMFA)
		r.Put("/mfa", h.verifyMFA)
		r.Delete("/mfa", h.disableMFA)
	})

	return r
}

// UserRoutes returns the users router (server-side).
func UserRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createUser)
	r.Get("/", h.listUsers)
	r.Get("/{userId}", h.getUser)
	r.Delete("/{userId}", h.deleteUser)
	r.Patch("/{userId}/name", h.updateUserName)
	r.Patch("/{userId}/email", h.updateUserEmail)
	r.Patch("/{userId}/password", h.updateUserPassword)
	r.Patch("/{userId}/status", h.updateUserStatus)
	r.Patch("/{userId}/prefs", h.updateUserPrefs)
	r.Get("/{userId}/sessions", h.listUserSessions)
	r.Delete("/{userId}/sessions", h.deleteUserSessions)
	return r
}

// --- account handlers ---

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		UserID   string `json:"userId"`
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
	u, err := h.svc.CreateAccount(r.Context(), projectID, body.UserID, body.Email, body.Password, body.Name)
	if err != nil {
		if strings.Contains(err.Error(), "user_already_exists") {
			apperr.Conflict(w, "email already in use")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	u, err := h.svc.GetAccount(ctx, userID, projectID)
	if err != nil {
		apperr.NotFound(w, "user")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updateName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	u, err := h.svc.UpdateName(ctx, userID, projectID, body.Name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updateEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	u, err := h.svc.UpdateEmail(ctx, userID, projectID, body.Email, body.Password)
	if err != nil {
		apperr.BadRequest(w, "invalid email or password")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updatePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	var body struct {
		Password    string `json:"password"`
		OldPassword string `json:"oldPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	u, err := h.svc.UpdatePassword(ctx, userID, projectID, body.Password, body.OldPassword)
	if err != nil {
		apperr.BadRequest(w, "invalid current password")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updatePrefs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	var body struct {
		Prefs map[string]interface{} `json:"prefs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	u, err := h.svc.UpdatePrefs(ctx, userID, projectID, body.Prefs)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	if err := h.svc.DeleteAccount(ctx, userID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createEmailSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	ip := r.RemoteAddr
	ua := r.UserAgent()
	sess, token, err := h.svc.CreateEmailSession(r.Context(), projectID, body.Email, body.Password, ip, ua)
	if err != nil {
		apperr.Write(w, http.StatusUnauthorized, "user_invalid_credentials", "invalid credentials")
		return
	}
	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_" + projectID,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600, // 30 days
	})
	// Also hand the token back in the body: a browser gets the cookie, everything
	// else (mobile, a script) needs the secret to authenticate by header.
	sess.Secret = token
	writeJSON(w, http.StatusCreated, sess)
}

func (h *Handler) createAnonymousSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	ip := r.RemoteAddr
	ua := r.UserAgent()
	sess, token, err := h.svc.CreateAnonymousSession(r.Context(), projectID, ip, ua)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	// Set project-specific session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_" + projectID,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600, // 30 days
	})
	// Set generic a_session cookie as fallback for middleware
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600, // 30 days
	})
	sess.Secret = token
	writeJSON(w, http.StatusCreated, sess)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	sessionID := chi.URLParam(r, "sessionId")
	sess, err := h.svc.GetSession(ctx, sessionID, userID, projectID)
	if err != nil {
		apperr.NotFound(w, "session")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	sessions, err := h.svc.ListSessions(ctx, userID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    len(sessions),
		"sessions": sessions,
	})
}

func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	sessionID := chi.URLParam(r, "sessionId")
	if err := h.svc.DeleteSession(ctx, sessionID, userID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	if err := h.svc.DeleteSessions(ctx, userID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getJWT(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	token, err := h.svc.GetJWT(ctx, userID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"jwt": token})
}

// --- user management handlers (server-side) ---

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		UserID   string `json:"userId"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	u, err := h.svc.CreateUser(r.Context(), projectID, body.UserID, body.Email, body.Phone, body.Password, body.Name)
	if err != nil {
		if strings.Contains(err.Error(), "user_already_exists") {
			apperr.Conflict(w, "email already in use")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	pg := middleware.ParsePagination(r)
	search := r.URL.Query().Get("search")
	users, total, err := h.svc.ListUsers(ctx, projectID, pg.Limit, pg.Offset, search)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": total,
		"users": users,
	})
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	u, err := h.svc.GetUser(ctx, userID, projectID)
	if err != nil {
		apperr.NotFound(w, "user")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	if err := h.svc.DeleteUser(ctx, userID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateUserName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	u, err := h.svc.UpdateName(ctx, userID, projectID, body.Name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updateUserEmail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	u, err := h.svc.UpdateUserEmailAdmin(ctx, userID, projectID, body.Email)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updateUserPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	var body struct {
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	u, err := h.svc.UpdatePassword(ctx, userID, projectID, body.Password, "")
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	var body struct {
		Status bool `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	u, err := h.svc.UpdateUserStatus(ctx, userID, projectID, body.Status)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) updateUserPrefs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	var body struct {
		Prefs map[string]interface{} `json:"prefs"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	u, err := h.svc.UpdatePrefs(ctx, userID, projectID, body.Prefs)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) listUserSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	sessions, err := h.svc.ListUserSessions(ctx, userID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    len(sessions),
		"sessions": sessions,
	})
}

func (h *Handler) deleteUserSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := chi.URLParam(r, "userId")
	if err := h.svc.DeleteUserSessions(ctx, userID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- MFA handlers ---

func (h *Handler) enableMFA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	secret, recovery, err := h.svc.EnableMFA(ctx, userID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"secret":        secret,
		"recoveryCodes": recovery,
		"uri":           fmt.Sprintf("otpauth://totp/Applad?secret=%s&issuer=Applad", secret),
	})
}

func (h *Handler) verifyMFA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.VerifyMFA(ctx, userID, projectID, body.Code); err != nil {
		apperr.BadRequest(w, "invalid MFA code")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

func (h *Handler) disableMFA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.UserFromContext(ctx)
	projectID := middleware.ProjectFromContext(ctx)
	if err := h.svc.DisableMFA(ctx, userID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Magic link, email verification, password reset handlers ---

func (h *Handler) createMagicLink(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Email string `json:"email"`
		URL   string `json:"url"` // callback URL to append token to
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Email == "" {
		apperr.BadRequest(w, "email is required")
		return
	}
	token, err := h.svc.CreateMagicLinkToken(r.Context(), projectID, body.Email)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	// Send magic link email
	if h.mailer != nil {
		link := token
		if body.URL != "" {
			link = body.URL + "?secret=" + token
		}
		html := fmt.Sprintf(`<h2>Sign in to Applad</h2><p>Click the link below to sign in:</p><p><a href="%s">Sign In</a></p><p>This link expires in 15 minutes.</p>`, link)
		h.mailer.SendEmail(r.Context(), []string{body.Email}, "Sign in to Applad", html)
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "sent"})
}

func (h *Handler) redeemMagicLink(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Secret string `json:"secret"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Secret == "" {
		apperr.BadRequest(w, "secret is required")
		return
	}
	sess, token, err := h.svc.RedeemMagicLink(r.Context(), projectID, body.Secret, r.RemoteAddr, r.UserAgent())
	if err != nil {
		apperr.BadRequest(w, "invalid or expired magic link")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_" + projectID,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600,
	})
	writeJSON(w, http.StatusOK, sess)
}

func (h *Handler) createEmailVerification(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		UserID string `json:"userId"`
		URL    string `json:"url"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.UserID == "" {
		apperr.BadRequest(w, "userId is required")
		return
	}
	token, err := h.svc.CreateEmailVerificationToken(r.Context(), body.UserID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	// Send verification email
	if h.mailer != nil {
		// Get user email
		user, _ := h.svc.GetAccount(r.Context(), body.UserID, projectID)
		if user != nil && user.Email != "" {
			link := token
			if body.URL != "" {
				link = body.URL + "?secret=" + token
			}
			html := fmt.Sprintf(`<h2>Verify your email</h2><p>Click the link below to verify your email address:</p><p><a href="%s">Verify Email</a></p><p>This link expires in 24 hours.</p>`, link)
			h.mailer.SendEmail(r.Context(), []string{user.Email}, "Verify your email", html)
		}
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "sent"})
}

func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Secret string `json:"secret"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Secret == "" {
		apperr.BadRequest(w, "secret is required")
		return
	}
	if err := h.svc.VerifyEmail(r.Context(), projectID, body.Secret); err != nil {
		apperr.BadRequest(w, "invalid or expired verification token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

func (h *Handler) createPasswordReset(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Email string `json:"email"`
		URL   string `json:"url"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Email == "" {
		apperr.BadRequest(w, "email is required")
		return
	}
	token, err := h.svc.CreatePasswordResetToken(r.Context(), projectID, body.Email)
	if err != nil {
		// Don't reveal whether email exists
		writeJSON(w, http.StatusCreated, map[string]string{"status": "sent"})
		return
	}

	// Send password reset email
	if h.mailer != nil {
		link := token
		if body.URL != "" {
			link = body.URL + "?secret=" + token
		}
		html := fmt.Sprintf(`<h2>Reset your password</h2><p>Click the link below to reset your password:</p><p><a href="%s">Reset Password</a></p><p>This link expires in 1 hour. If you didn't request this, ignore this email.</p>`, link)
		h.mailer.SendEmail(r.Context(), []string{body.Email}, "Reset your password", html)
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "sent"})
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Secret   string `json:"secret"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Secret == "" || body.Password == "" {
		apperr.BadRequest(w, "secret and password are required")
		return
	}
	if err := h.svc.ResetPassword(r.Context(), projectID, body.Secret, body.Password); err != nil {
		apperr.BadRequest(w, "invalid or expired reset token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

// --- OAuth2 handlers ---

// SetOAuthProviders sets the available OAuth providers on the handler.
func (h *Handler) SetOAuthProviders(providers map[string]OAuthProvider) {
	h.oauthProviders = providers
}

func (h *Handler) oauthRedirect(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := h.oauthProviders[providerName]
	if !ok {
		apperr.BadRequest(w, "unsupported OAuth provider: "+providerName)
		return
	}

	projectID := middleware.ProjectFromContext(r.Context())
	// Validate redirect URLs are relative-only to prevent open-redirect attacks.
	successURL := safeRedirectURL(r.URL.Query().Get("success"))
	failureURL := safeRedirectURL(r.URL.Query().Get("failure"))

	// Build callback URL
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	callbackURL := fmt.Sprintf("%s://%s/v1/account/sessions/oauth/%s/callback", scheme, r.Host, providerName)

	// State encodes project + redirect URLs
	state := fmt.Sprintf("%s|%s|%s", projectID, successURL, failureURL)

	authURL := provider.GetAuthURL(callbackURL, state)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := h.oauthProviders[providerName]
	if !ok {
		apperr.BadRequest(w, "unsupported OAuth provider")
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		apperr.BadRequest(w, "OAuth authorization failed")
		return
	}

	// Parse state: projectID|successURL|failureURL
	// Re-validate URLs from state to guard against tampered state parameters.
	parts := strings.SplitN(state, "|", 3)
	projectID := ""
	successURL := "/"
	failureURL := "/"
	if len(parts) >= 1 {
		projectID = parts[0]
	}
	if len(parts) >= 2 {
		successURL = safeRedirectURL(parts[1])
	}
	if len(parts) >= 3 {
		failureURL = safeRedirectURL(parts[2])
	}

	// Build callback URL for token exchange
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	callbackURL := fmt.Sprintf("%s://%s/v1/account/sessions/oauth/%s/callback", scheme, r.Host, providerName)

	// Exchange code for token
	ctx := r.Context()
	accessToken, err := provider.ExchangeCode(ctx, code, callbackURL)
	if err != nil {
		http.Redirect(w, r, failureURL+"?error=token_exchange_failed", http.StatusTemporaryRedirect)
		return
	}

	// Get user info
	userInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		http.Redirect(w, r, failureURL+"?error=userinfo_failed", http.StatusTemporaryRedirect)
		return
	}

	// Create or link user and session
	sess, token, err := h.svc.CreateOAuthSession(ctx, projectID, providerName, userInfo.ID, userInfo.Email, userInfo.Name, r.RemoteAddr, r.UserAgent())
	if err != nil {
		http.Redirect(w, r, failureURL+"?error=session_failed", http.StatusTemporaryRedirect)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_" + projectID,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600, // 30 days
	})

	_ = sess // session is set via cookie
	http.Redirect(w, r, successURL, http.StatusTemporaryRedirect)
}

// ── Phone OTP ────────────────────────────────────────────────────────────────

func (h *Handler) sendPhoneOTP(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Phone == "" {
		apperr.BadRequest(w, "phone is required")
		return
	}
	if !e164Re.MatchString(body.Phone) {
		apperr.BadRequest(w, "phone must be in E.164 format (e.g. +15551234567)")
		return
	}
	code, err := h.svc.SendPhoneOTP(r.Context(), projectID, body.Phone)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	// Send SMS if sender configured
	if h.smsSender != nil {
		msg := fmt.Sprintf("Your Applad verification code is: %s", code)
		if err := h.smsSender.SendSMS(r.Context(), body.Phone, msg); err != nil {
			apperr.Internal(w, fmt.Errorf("failed to send SMS: %w", err))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sent": true, "phone": body.Phone})
}

func (h *Handler) verifyPhoneOTP(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Phone == "" || body.Code == "" {
		apperr.BadRequest(w, "phone and code are required")
		return
	}
	sess, token, err := h.svc.VerifyPhoneOTP(r.Context(), projectID, body.Phone, body.Code, r.RemoteAddr, r.UserAgent())
	if err != nil {
		apperr.Write(w, http.StatusUnauthorized, "auth_invalid_otp", "invalid or expired OTP code")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 3600, // 30 days
	})
	writeJSON(w, http.StatusCreated, map[string]interface{}{"session": sess, "token": token})
}
