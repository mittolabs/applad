package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for auth.
type Handler struct {
	svc *Service
}

// NewHandler creates a new auth Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
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
		apperr.BadRequest(w, err.Error())
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
		apperr.BadRequest(w, err.Error())
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
		SameSite: http.SameSiteLaxMode,
	})
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
	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_" + projectID,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
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
