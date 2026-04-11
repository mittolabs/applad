package functions

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/runtime"
)

// Handler handles HTTP requests for functions.
type Handler struct {
	svc *Service
}

// NewHandler creates a new functions Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the functions router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/runtimes", h.listRuntimes)
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{functionId}", h.get)
	r.Put("/{functionId}", h.update)
	r.Delete("/{functionId}", h.delete)
	r.Post("/{functionId}/executions", h.execute)
	r.Get("/{functionId}/executions", h.listExecutions)
	r.Get("/{functionId}/executions/{executionId}", h.getExecution)
	return r
}

func (h *Handler) listRuntimes(w http.ResponseWriter, r *http.Request) {
	runtimes := runtime.SupportedRuntimes()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    len(runtimes),
		"runtimes": runtimes,
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name       string            `json:"name"`
		Runtime    string            `json:"runtime"`
		Entrypoint string            `json:"entrypoint"`
		Timeout    int               `json:"timeout"`
		EnvVars    map[string]string `json:"envVars"`
		SourceType string            `json:"sourceType"`
		Source     string            `json:"source"`
		Repository string            `json:"repository"`
		Branch     string            `json:"branch"`
		Cron       string            `json:"cron"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Runtime == "" {
		apperr.BadRequest(w, "runtime is required")
		return
	}
	if body.Entrypoint == "" {
		body.Entrypoint = "main"
	}
	if body.Timeout <= 0 {
		body.Timeout = 15
	}
	if body.EnvVars == nil {
		body.EnvVars = map[string]string{}
	}
	if body.SourceType == "" {
		body.SourceType = "inline"
	}
	f, err := h.svc.Create(r.Context(), projectID, body.Name, body.Runtime, body.Entrypoint, body.Timeout, body.EnvVars, body.SourceType, body.Source, body.Repository, body.Branch, body.Cron)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functions, total, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if functions == nil {
		functions = []*Function{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     total,
		"functions": functions,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "functionId")
	f, err := h.svc.Get(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "function")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "functionId")

	// Verify function exists
	existing, err := h.svc.Get(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "function")
		return
	}

	var body struct {
		Name       string            `json:"name"`
		Runtime    string            `json:"runtime"`
		Entrypoint string            `json:"entrypoint"`
		Timeout    int               `json:"timeout"`
		EnvVars    map[string]string `json:"envVars"`
		SourceType string            `json:"sourceType"`
		Source     string            `json:"source"`
		Repository string            `json:"repository"`
		Branch     string            `json:"branch"`
		Cron       *string           `json:"cron"` // pointer so explicit "" is preserved
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}

	// Use existing values as defaults for unset fields
	if strings.TrimSpace(body.Name) == "" {
		body.Name = existing.Name
	}
	if body.Runtime == "" {
		body.Runtime = existing.Runtime
	}
	if body.Entrypoint == "" {
		body.Entrypoint = existing.Entrypoint
	}
	if body.Timeout <= 0 {
		body.Timeout = existing.Timeout
	}
	if body.EnvVars == nil {
		body.EnvVars = existing.EnvVars
	}
	if body.SourceType == "" {
		body.SourceType = existing.SourceType
	}
	if body.Source == "" {
		body.Source = existing.Source
	}
	if body.Repository == "" {
		body.Repository = existing.Repository
	}
	if body.Branch == "" {
		body.Branch = existing.Branch
	}
	cron := existing.Cron
	if body.Cron != nil {
		cron = *body.Cron
	}

	f, err := h.svc.Update(r.Context(), id, projectID, body.Name, body.Runtime, body.Entrypoint, body.Timeout, body.EnvVars, body.SourceType, body.Source, body.Repository, body.Branch, cron)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "functionId")
	if err := h.svc.Delete(r.Context(), id, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) execute(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	exec, err := h.svc.Execute(r.Context(), functionID, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "function")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, exec)
}

func (h *Handler) listExecutions(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	executions, total, err := h.svc.ListExecutions(r.Context(), functionID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if executions == nil {
		executions = []*FunctionExecution{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      total,
		"executions": executions,
	})
}

func (h *Handler) getExecution(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	executionID := chi.URLParam(r, "executionId")
	exec, err := h.svc.GetExecution(r.Context(), executionID, functionID, projectID)
	if err != nil {
		apperr.NotFound(w, "execution")
		return
	}
	writeJSON(w, http.StatusOK, exec)
}
