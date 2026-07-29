package edge

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the edge functions HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new edge Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the edge router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Functions
	r.Post("/functions", h.create)
	r.Get("/functions", h.list)
	r.Get("/functions/{functionId}", h.get)
	r.Put("/functions/{functionId}", h.update)
	r.Delete("/functions/{functionId}", h.delete)

	// Deployments
	r.Post("/functions/{functionId}/invoke", h.invoke)
	r.Get("/functions/{functionId}/executions", h.listExecutions)
	r.Post("/functions/{functionId}/deploy", h.deploy)
	r.Get("/functions/{functionId}/deployments", h.listDeployments)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name    string            `json:"name"`
		Slug    string            `json:"slug"`
		Route   string            `json:"route"`
		Code    string            `json:"code"`
		Runtime string            `json:"runtime"`
		Regions []string          `json:"regions"`
		EnvVars map[string]string `json:"envVars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	if body.Slug == "" {
		body.Slug = edgeSlug(body.Route, body.Name)
	}
	if body.Slug == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "slug could not be derived")
		return
	}
	f, err := h.svc.Create(r.Context(), projectID, body.Name, body.Slug, body.Code, body.Runtime, body.Regions, body.EnvVars)
	if err != nil {
		apperr.Write(w, http.StatusConflict, "edge_function_exists", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functions, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if functions == nil {
		functions = []*EdgeFunction{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(functions), "functions": functions})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	f, err := h.svc.Get(r.Context(), functionID, projectID)
	if err != nil {
		apperr.NotFound(w, "edge_function")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	var body struct {
		Name    string            `json:"name"`
		Code    string            `json:"code"`
		Regions []string          `json:"regions"`
		EnvVars map[string]string `json:"envVars"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	f, err := h.svc.Update(r.Context(), functionID, projectID, body.Name, body.Code, body.Regions, body.EnvVars)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	if err := h.svc.Delete(r.Context(), functionID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deploy(w http.ResponseWriter, r *http.Request) {
	// Edge functions can be managed (create/list/update/delete) but there is
	// no edge execution/deployment engine wired up on this instance. The old
	// handler flipped a status column to "deployed"/"active" without deploying
	// anything, reporting a success that never happened. Report honestly
	// instead of fabricating one.
	apperr.Write(w, http.StatusNotImplemented, "edge_not_implemented",
		"edge function deployment is not implemented on this instance")
}

func (h *Handler) invoke(w http.ResponseWriter, r *http.Request) {
	// The old handler echoed the request body back as "output" with status
	// "completed", so callers received an HTTP 200 for code that never ran.
	// There is no edge execution engine (the API process holds no Docker
	// socket, and no edge worker consumes edge invocations), so report that
	// invocation is not implemented rather than faking a run.
	apperr.Write(w, http.StatusNotImplemented, "edge_not_implemented",
		"edge function invocation is not implemented on this instance")
}

func (h *Handler) listExecutions(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	deployments, err := h.svc.ListDeployments(r.Context(), functionID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if deployments == nil {
		deployments = []*Deployment{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(deployments), "executions": deployments})
}

func (h *Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	functionID := chi.URLParam(r, "functionId")
	deployments, err := h.svc.ListDeployments(r.Context(), functionID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if deployments == nil {
		deployments = []*Deployment{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(deployments), "deployments": deployments})
}

func edgeSlug(route, name string) string {
	base := strings.TrimSpace(route)
	if base == "" {
		base = name
	}
	base = strings.ToLower(strings.Trim(base, "/"))
	base = strings.ReplaceAll(base, "/", "-")
	base = strings.ReplaceAll(base, "_", "-")
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	base = re.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	for strings.Contains(base, "--") {
		base = strings.ReplaceAll(base, "--", "-")
	}
	return base
}
