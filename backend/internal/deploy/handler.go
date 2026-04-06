package deploy

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for deployments.
type Handler struct {
	svc *Service
}

// NewHandler creates a new deploy Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the deploy router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Targets
	r.Route("/targets", func(r chi.Router) {
		r.Post("/", h.createTarget)
		r.Get("/", h.listTargets)
		r.Get("/{targetId}", h.getTarget)
		r.Put("/{targetId}", h.updateTarget)
		r.Delete("/{targetId}", h.deleteTarget)
		r.Post("/{targetId}/executions", h.invokeTarget)
		r.Get("/{targetId}/executions", h.listExecutions)
		r.Get("/{targetId}/executions/{execId}", h.getExecution)
		r.Get("/{targetId}/stats", h.getTargetStats)
		r.Get("/{targetId}/stats/detailed", h.getTargetDetailedStats)
	})

	// Pipelines
	r.Route("/pipelines", func(r chi.Router) {
		r.Post("/", h.createPipeline)
		r.Get("/", h.listPipelines)
		r.Get("/{pipelineId}", h.getPipeline)
		r.Put("/{pipelineId}", h.updatePipeline)
		r.Delete("/{pipelineId}", h.deletePipeline)
		r.Post("/{pipelineId}/trigger", h.triggerPipeline)
	})

	// Releases
	r.Route("/releases", func(r chi.Router) {
		r.Get("/", h.listReleases)
		r.Get("/{releaseId}", h.getRelease)
		r.Get("/{releaseId}/logs", h.getReleaseLogs)
		r.Post("/{releaseId}/rollback", h.rollbackRelease)
	})

	// Runtimes
	r.Get("/runtimes", h.listRuntimes)

	// Aggregate stats
	r.Get("/stats", h.getAggregateStats)

	return r
}

// ── Target handlers ──

func (h *Handler) createTarget(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name        string            `json:"name"`
		Type        string            `json:"type"`
		Runtime     string            `json:"runtime"`
		Entrypoint  string            `json:"entrypoint"`
		TimeoutMs   int               `json:"timeoutMs"`
		MemoryMB    int               `json:"memoryMb"`
		EnvVars     map[string]string `json:"envVars"`
		Permissions json.RawMessage   `json:"permissions"`
		Cron        string            `json:"cron"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Type == "" {
		body.Type = "web"
	}
	if body.TimeoutMs == 0 {
		body.TimeoutMs = 30000
	}
	if body.MemoryMB == 0 {
		body.MemoryMB = 256
	}

	t, err := h.svc.CreateTarget(r.Context(), projectID, body.Name, body.Type, body.Runtime,
		body.Entrypoint, body.TimeoutMs, body.MemoryMB, body.EnvVars, body.Permissions, body.Cron)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) listTargets(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targets, total, err := h.svc.ListTargets(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if targets == nil {
		targets = []*Target{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   total,
		"targets": targets,
	})
}

func (h *Handler) getTarget(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "targetId")
	t, err := h.svc.GetTarget(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "target")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) updateTarget(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "targetId")
	var body struct {
		Name        string            `json:"name"`
		Type        string            `json:"type"`
		Runtime     string            `json:"runtime"`
		Entrypoint  string            `json:"entrypoint"`
		TimeoutMs   int               `json:"timeoutMs"`
		MemoryMB    int               `json:"memoryMb"`
		EnvVars     map[string]string `json:"envVars"`
		Permissions json.RawMessage   `json:"permissions"`
		Cron        string            `json:"cron"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Type == "" {
		body.Type = "web"
	}

	t, err := h.svc.UpdateTarget(r.Context(), id, projectID, body.Name, body.Type, body.Runtime,
		body.Entrypoint, body.TimeoutMs, body.MemoryMB, body.EnvVars, body.Permissions, body.Cron)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTarget(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "targetId")
	if err := h.svc.DeleteTarget(r.Context(), id, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Execution handlers ──

func (h *Handler) invokeTarget(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	var body struct {
		Request string `json:"request"`
		Trigger string `json:"trigger"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Trigger == "" {
		body.Trigger = "api"
	}

	exec, err := h.svc.InvokeTarget(r.Context(), targetID, projectID, body.Request, body.Trigger)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "target")
			return
		}
		if strings.Contains(err.Error(), "only serverless") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, exec)
}

func (h *Handler) listExecutions(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	execs, total, err := h.svc.ListExecutions(r.Context(), targetID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if execs == nil {
		execs = []*Execution{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      total,
		"executions": execs,
	})
}

func (h *Handler) getExecution(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	execID := chi.URLParam(r, "execId")
	exec, err := h.svc.GetExecution(r.Context(), execID, targetID, projectID)
	if err != nil {
		apperr.NotFound(w, "execution")
		return
	}
	writeJSON(w, http.StatusOK, exec)
}

func (h *Handler) getTargetStats(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	stats, err := h.svc.GetTargetStats(r.Context(), targetID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ── Pipeline handlers ──

func (h *Handler) createPipeline(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		TargetID   string            `json:"targetId"`
		Name       string            `json:"name"`
		SourceType string            `json:"sourceType"`
		SourceURL  string            `json:"sourceUrl"`
		Branch     string            `json:"branch"`
		BuildCmd   string            `json:"buildCmd"`
		OutputDir  string            `json:"outputDir"`
		EnvVars    map[string]string `json:"envVars"`
		TriggerOn  json.RawMessage   `json:"triggerOn"`
		CacheDirs  json.RawMessage   `json:"cacheDirs"`
		TimeoutMs  int               `json:"timeoutMs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if strings.TrimSpace(body.TargetID) == "" {
		apperr.BadRequest(w, "targetId is required")
		return
	}
	if body.SourceType == "" {
		body.SourceType = "upload"
	}
	if body.TimeoutMs == 0 {
		body.TimeoutMs = 300000
	}

	p, err := h.svc.CreatePipeline(r.Context(), projectID, body.TargetID, body.Name, body.SourceType,
		body.SourceURL, body.Branch, body.BuildCmd, body.OutputDir, body.EnvVars,
		body.TriggerOn, body.CacheDirs, body.TimeoutMs)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) listPipelines(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	pipelines, total, err := h.svc.ListPipelines(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if pipelines == nil {
		pipelines = []*Pipeline{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     total,
		"pipelines": pipelines,
	})
}

func (h *Handler) getPipeline(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "pipelineId")
	p, err := h.svc.GetPipeline(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "pipeline")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) updatePipeline(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "pipelineId")
	var body struct {
		TargetID   string            `json:"targetId"`
		Name       string            `json:"name"`
		SourceType string            `json:"sourceType"`
		SourceURL  string            `json:"sourceUrl"`
		Branch     string            `json:"branch"`
		BuildCmd   string            `json:"buildCmd"`
		OutputDir  string            `json:"outputDir"`
		EnvVars    map[string]string `json:"envVars"`
		TriggerOn  json.RawMessage   `json:"triggerOn"`
		CacheDirs  json.RawMessage   `json:"cacheDirs"`
		TimeoutMs  int               `json:"timeoutMs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if strings.TrimSpace(body.TargetID) == "" {
		apperr.BadRequest(w, "targetId is required")
		return
	}

	p, err := h.svc.UpdatePipeline(r.Context(), id, projectID, body.TargetID, body.Name, body.SourceType,
		body.SourceURL, body.Branch, body.BuildCmd, body.OutputDir, body.EnvVars,
		body.TriggerOn, body.CacheDirs, body.TimeoutMs)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deletePipeline(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "pipelineId")
	if err := h.svc.DeletePipeline(r.Context(), id, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) triggerPipeline(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	pipelineID := chi.URLParam(r, "pipelineId")
	var body struct {
		TriggerType string `json:"triggerType"`
		Actor       string `json:"actor"`
		CommitSHA   string `json:"commitSha"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.TriggerType == "" {
		body.TriggerType = "manual"
	}

	rel, err := h.svc.TriggerPipeline(r.Context(), pipelineID, projectID, body.TriggerType, body.Actor, body.CommitSHA)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "pipeline")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, rel)
}

// ── Release handlers ──

func (h *Handler) listReleases(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	pipelineID := r.URL.Query().Get("pipelineId")
	targetID := r.URL.Query().Get("targetId")
	status := r.URL.Query().Get("status")

	releases, total, err := h.svc.ListReleases(r.Context(), projectID, pipelineID, targetID, status)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if releases == nil {
		releases = []*Release{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    total,
		"releases": releases,
	})
}

func (h *Handler) getRelease(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "releaseId")
	rel, err := h.svc.GetRelease(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "release")
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

func (h *Handler) getReleaseLogs(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "releaseId")
	rel, err := h.svc.GetRelease(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "release")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"$id":       rel.ID,
		"buildLog":  rel.BuildLog,
		"deployLog": rel.DeployLog,
		"error":     rel.Error,
	})
}

func (h *Handler) rollbackRelease(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	releaseID := chi.URLParam(r, "releaseId")
	var body struct {
		Actor string `json:"actor"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	rel, err := h.svc.RollbackRelease(r.Context(), releaseID, projectID, body.Actor)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "release")
			return
		}
		if strings.Contains(err.Error(), "only rollback") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, rel)
}

// ── Aggregate stats handler ──

func (h *Handler) getAggregateStats(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	stats, err := h.svc.GetAggregateStats(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ── Runtimes handler ──

func (h *Handler) listRuntimes(w http.ResponseWriter, r *http.Request) {
	runtimes := ListRuntimes()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    len(runtimes),
		"runtimes": runtimes,
	})
}

// ── Detailed stats handler ──

func (h *Handler) getTargetDetailedStats(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "24h"
	}
	stats, err := h.svc.GetTargetDetailedStats(r.Context(), targetID, projectID, timeRange)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
