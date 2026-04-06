package console

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
)

// Handler handles HTTP requests for console auth.
type Handler struct {
	svc           *Service
	signupSetting string // "auto", "true", or "false"
}

// NewHandler creates a new console auth Handler.
func NewHandler(svc *Service, signupSetting string) *Handler {
	return &Handler{svc: svc, signupSetting: signupSetting}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the console auth router (all public, no project header).
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/signup", h.signup)
	r.Post("/login", h.login)
	r.Get("/signup-status", h.signupStatus)
	r.Get("/me", h.getMe)
	r.Patch("/me/name", h.updateName)
	r.Patch("/me/email", h.updateEmail)
	r.Patch("/me/password", h.updatePassword)
	r.Delete("/me", h.deleteAccount)
	return r
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

	user, token, err := h.svc.Signup(r.Context(), body.Email, body.Password, body.Name)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			apperr.Conflict(w, "email already in use")
			return
		}
		apperr.Internal(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_console",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

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

	user, token, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		apperr.Write(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "a_session_console",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
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
	// Extract console JWT from Authorization header
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		apperr.Unauthorized(w)
		return
	}
	tokenStr := strings.TrimPrefix(auth, "Bearer ")

	userID, err := h.svc.ValidateToken(tokenStr)
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
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		return "", fmt.Errorf("no token")
	}
	return h.svc.ValidateToken(strings.TrimPrefix(auth, "Bearer "))
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
