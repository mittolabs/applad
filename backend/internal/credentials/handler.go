package credentials

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for credentials.
type Handler struct {
	svc *Service
}

// NewHandler creates a new credential Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the authenticated credential router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Post("/rotate", h.rotateKeys)
	r.Get("/{credentialId}", h.get)
	r.Put("/{credentialId}", h.update)
	r.Delete("/{credentialId}", h.delete)
	r.Get("/{credentialId}/accesses", h.listAccesses)
	return r
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Description string  `json:"description"`
		Data        string  `json:"data"`
		Protected   bool    `json:"protected"`
		ExpiresAt   *string `json:"expiresAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if strings.TrimSpace(body.Type) == "" {
		apperr.BadRequest(w, "type is required")
		return
	}
	if strings.TrimSpace(body.Data) == "" {
		apperr.BadRequest(w, "data is required")
		return
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			apperr.BadRequest(w, "expiresAt must be RFC3339 format")
			return
		}
		expiresAt = &t
	}

	cred, err := h.svc.Create(r.Context(), projectID, body.Name, body.Type, body.Description, body.Data, body.Protected, expiresAt)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = strings.Split(xff, ",")[0]
	}
	h.svc.LogAccess(r.Context(), cred.ID, projectID, "create", "", "api", ip, r.UserAgent())

	writeJSON(w, http.StatusCreated, cred)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	p := middleware.ParsePagination(r)

	creds, total, err := h.svc.List(r.Context(), projectID, p.Limit, p.Offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":       total,
		"credentials": creds,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "credentialId")

	// Require API key for any credential retrieval (data is decrypted)
	isAPIKey := middleware.IsAPIKey(r.Context())

	cred, err := h.svc.Get(r.Context(), id, projectID, isAPIKey)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "credential not found":
			apperr.NotFound(w, "credential")
		case "credential expired":
			apperr.BadRequest(w, "credential has expired")
		case "credential requires API key authentication":
			apperr.Write(w, http.StatusForbidden, "credential_protected", "this credential requires API key authentication")
		default:
			apperr.Internal(w, err)
		}
		return
	}

	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = strings.Split(xff, ",")[0]
	}
	h.svc.LogAccess(r.Context(), id, projectID, "read", "", actorType(isAPIKey), ip, r.UserAgent())

	writeJSON(w, http.StatusOK, cred)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "credentialId")

	var body struct {
		Name        string  `json:"name"`
		Type        string  `json:"type"`
		Description string  `json:"description"`
		Data        string  `json:"data"`
		Protected   bool    `json:"protected"`
		ExpiresAt   *string `json:"expiresAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if strings.TrimSpace(body.Data) == "" {
		apperr.BadRequest(w, "data is required")
		return
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			apperr.BadRequest(w, "expiresAt must be RFC3339 format")
			return
		}
		expiresAt = &t
	}

	cred, err := h.svc.Update(r.Context(), id, projectID, body.Name, body.Type, body.Description, body.Data, body.Protected, expiresAt)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = strings.Split(xff, ",")[0]
	}
	h.svc.LogAccess(r.Context(), id, projectID, "update", "", "api", ip, r.UserAgent())

	writeJSON(w, http.StatusOK, cred)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "credentialId")

	if err := h.svc.Delete(r.Context(), id, projectID); err != nil {
		if err.Error() == "credential not found" {
			apperr.NotFound(w, "credential")
			return
		}
		apperr.Internal(w, err)
		return
	}

	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = strings.Split(xff, ",")[0]
	}
	h.svc.LogAccess(r.Context(), id, projectID, "delete", "", "api", ip, r.UserAgent())

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rotateKeys(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())

	rotated, err := h.svc.RotateKeys(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = strings.Split(xff, ",")[0]
	}
	h.svc.LogAccess(r.Context(), "", projectID, "rotate", "", "api", ip, r.UserAgent())

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rotated": rotated,
	})
}

func (h *Handler) listAccesses(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "credentialId")
	p := middleware.ParsePagination(r)

	accesses, total, err := h.svc.ListAccesses(r.Context(), id, projectID, p.Limit, p.Offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    total,
		"accesses": accesses,
	})
}

func actorType(isAPIKey bool) string {
	if isAPIKey {
		return "api_key"
	}
	return "session"
}
