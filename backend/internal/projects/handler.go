package projects

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
)

// Handler handles HTTP requests for project management.
type Handler struct {
	svc *Service
}

// NewHandler creates a new projects Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the projects router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createProject)
	r.Get("/", h.listProjects)
	r.Get("/{projectId}", h.getProject)
	r.Patch("/{projectId}", h.updateProject)
	r.Delete("/{projectId}", h.deleteProject)
	r.Post("/{projectId}/keys", h.createKey)
	r.Get("/{projectId}/keys", h.listKeys)
	r.Delete("/{projectId}/keys/{keyId}", h.deleteKey)
	r.Get("/{projectId}/usage", h.getUsage)
	r.Post("/{projectId}/platforms", h.createPlatform)
	r.Get("/{projectId}/platforms", h.listPlatforms)
	r.Delete("/{projectId}/platforms/{platformId}", h.deletePlatform)
	r.Patch("/{projectId}/auth", h.updateAuthConfig)
	r.Patch("/{projectId}/services", h.updateServicesConfig)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	p, err := h.svc.Create(r.Context(), body.Name, body.Description)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.List(r.Context())
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    len(projects),
		"projects": projects,
	})
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "project")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	p, err := h.svc.Update(r.Context(), id, body.Name, body.Description)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "project")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createKey(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	var body struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Scopes == nil {
		body.Scopes = []string{}
	}
	key, _, err := h.svc.CreateKey(r.Context(), projectID, body.Name, body.Scopes)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, key)
}

func (h *Handler) listKeys(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	keys, err := h.svc.ListKeys(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": len(keys),
		"keys":  keys,
	})
}

func (h *Handler) deleteKey(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	keyID := chi.URLParam(r, "keyId")
	if err := h.svc.DeleteKey(r.Context(), projectID, keyID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getUsage(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	usage, err := h.svc.GetUsage(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (h *Handler) createPlatform(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	var body struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		StoreID  string `json:"storeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Type) == "" {
		apperr.BadRequest(w, "name and type are required")
		return
	}
	p, err := h.svc.CreatePlatform(r.Context(), projectID, body.Type, body.Name, body.Hostname, body.StoreID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) listPlatforms(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	platforms, err := h.svc.ListPlatforms(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(platforms),
		"platforms": platforms,
	})
}

func (h *Handler) deletePlatform(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	platformID := chi.URLParam(r, "platformId")
	if err := h.svc.DeletePlatform(r.Context(), projectID, platformID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateAuthConfig(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.UpdateAuthConfig(r.Context(), projectID, body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (h *Handler) updateServicesConfig(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.UpdateServicesConfig(r.Context(), projectID, body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}
