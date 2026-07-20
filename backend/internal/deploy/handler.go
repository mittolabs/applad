package deploy

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
		r.Get("/{targetId}/logs", h.getTargetLogs)
	})

	// Pipelines
	r.Route("/pipelines", func(r chi.Router) {
		r.Post("/", h.createPipeline)
		r.Get("/", h.listPipelines)
		r.Get("/{pipelineId}", h.getPipeline)
		r.Put("/{pipelineId}", h.updatePipeline)
		r.Delete("/{pipelineId}", h.deletePipeline)
		r.Post("/{pipelineId}/trigger", h.triggerPipeline)
		r.Post("/{pipelineId}/source", h.uploadSource)
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

	// Custom domains (web deploy targets)
	r.Post("/targets/{targetId}/domains", h.createDomain)
	r.Get("/targets/{targetId}/domains", h.listDomains)
	r.Post("/targets/{targetId}/domains/{domainId}/verify", h.verifyDomain)
	r.Delete("/targets/{targetId}/domains/{domainId}", h.deleteDomain)

	// Registry images (container deploy targets)
	r.Get("/targets/{targetId}/images", h.listImages)
	r.Post("/targets/{targetId}/images", h.pushImage)
	r.Delete("/targets/{targetId}/images/{imageId}", h.deleteImage)

	// Build agents
	r.Post("/agents", h.registerAgent)
	r.Get("/agents", h.listAgents)
	r.Post("/agents/{agentId}/heartbeat", h.heartbeatAgent)
	r.Delete("/agents/{agentId}", h.deleteAgent)

	// Deploy templates
	r.Get("/templates", h.listDeployTemplates)
	r.Get("/templates/{templateId}", h.getDeployTemplate)

	// Git connections
	r.Route("/git/connections", func(r chi.Router) {
		r.Post("/", h.createGitConnection)
		r.Get("/", h.listGitConnections)
		r.Delete("/{connectionId}", h.deleteGitConnection)
		r.Get("/{connectionId}/repos", h.listRepositories)
		r.Post("/{connectionId}/webhook-secret", h.generateWebhookSecret)
	})

	// Preview releases
	r.Get("/targets/{targetId}/releases/previews", h.listPreviewReleases)
	r.Delete("/releases/{releaseId}/preview", h.destroyPreviewRelease)

	// Environments
	r.Route("/environments", func(r chi.Router) {
		r.Post("/", h.createEnvironment)
		r.Get("/", h.listEnvironments)
		r.Get("/{envId}", h.getEnvironment)
		r.Put("/{envId}", h.updateEnvironment)
		r.Delete("/{envId}", h.deleteEnvironment)
	})

	return r
}

// WebhookRoutes returns a router for public inbound git webhook events.
// Mounted separately outside the project-auth middleware.
func WebhookRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/{connectionId}", h.handleGitWebhook)
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

// SourceDir returns the directory uploaded pipeline sources are stored in. The
// API and the builds worker share this volume, so the worker can read what was
// uploaded here.
func SourceDir() string {
	base := os.Getenv("STORAGE_PATH")
	if base == "" {
		base = "/var/applad/storage"
	}
	return filepath.Join(base, "deploy-sources")
}

// SourceArchivePath is where a given pipeline's uploaded source tarball lives.
func SourceArchivePath(pipelineID string) string {
	return filepath.Join(SourceDir(), pipelineID+".tar.gz")
}

// uploadSource accepts a gzipped tar of the app's source for pipelines with
// sourceType "upload" (the default). Deploying from a local folder needs no git
// remote this way.
func (h *Handler) uploadSource(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	pipelineID := chi.URLParam(r, "pipelineId")

	// Confirm the pipeline belongs to this project before writing anything.
	if _, err := h.svc.GetPipeline(r.Context(), pipelineID, projectID); err != nil {
		apperr.NotFound(w, "pipeline")
		return
	}

	if err := os.MkdirAll(SourceDir(), 0o755); err != nil {
		apperr.Internal(w, err)
		return
	}
	dest := SourceArchivePath(pipelineID)
	f, err := os.Create(dest)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	defer f.Close()

	// 512MB ceiling so a bad upload can't fill the volume.
	written, err := io.Copy(f, io.LimitReader(r.Body, 512<<20))
	if err != nil {
		os.Remove(dest)
		apperr.Internal(w, err)
		return
	}
	if written == 0 {
		os.Remove(dest)
		apperr.BadRequest(w, "empty source archive")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"pipelineId": pipelineID,
		"bytes":      written,
	})
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

// ── Target logs handler ──

func (h *Handler) getTargetLogs(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	result, err := h.svc.GetTargetLogs(r.Context(), targetID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Custom domain handlers (web deploy targets) ──

func (h *Handler) createDomain(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	var body struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Domain) == "" {
		apperr.BadRequest(w, "domain is required")
		return
	}
	d, err := h.svc.CreateCustomDomain(r.Context(), projectID, targetID, body.Domain)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) listDomains(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	domains, total, err := h.svc.ListDomains(r.Context(), projectID, targetID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if domains == nil {
		domains = []*CustomDomain{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   total,
		"domains": domains,
	})
}

func (h *Handler) verifyDomain(w http.ResponseWriter, r *http.Request) {
	domainID := chi.URLParam(r, "domainId")
	d, err := h.svc.VerifyDomain(r.Context(), domainID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "domain")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) deleteDomain(w http.ResponseWriter, r *http.Request) {
	domainID := chi.URLParam(r, "domainId")
	if err := h.svc.DeleteDomain(r.Context(), domainID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Registry image handlers (container deploy targets) ──

func (h *Handler) pushImage(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	var body struct {
		Repository string `json:"repository"`
		Tag        string `json:"tag"`
		Digest     string `json:"digest"`
		SizeBytes  int64  `json:"sizeBytes"`
		Platform   string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Repository) == "" {
		apperr.BadRequest(w, "repository is required")
		return
	}
	if body.Tag == "" {
		body.Tag = "latest"
	}
	if body.Platform == "" {
		body.Platform = "linux/amd64"
	}

	img, err := h.svc.PushImage(r.Context(), targetID, projectID, body.Repository, body.Tag, body.Digest, body.SizeBytes, body.Platform)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, img)
}

func (h *Handler) listImages(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")
	images, total, err := h.svc.ListImages(r.Context(), targetID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if images == nil {
		images = []*RegistryImage{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"images": images,
	})
}

func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	if err := h.svc.DeleteImage(r.Context(), imageID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Build agent handlers ──

func (h *Handler) registerAgent(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name   string   `json:"name"`
		Labels []string `json:"labels"`
		OS     string   `json:"os"`
		Arch   string   `json:"arch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.OS == "" {
		body.OS = "linux"
	}
	if body.Arch == "" {
		body.Arch = "amd64"
	}

	agent, err := h.svc.RegisterAgent(r.Context(), projectID, body.Name, body.Labels, body.OS, body.Arch)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	agents, total, err := h.svc.ListAgents(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if agents == nil {
		agents = []*Agent{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"agents": agents,
	})
}

func (h *Handler) heartbeatAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	if err := h.svc.HeartbeatAgent(r.Context(), agentID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
	})
}

func (h *Handler) deleteAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentId")
	if err := h.svc.DeleteAgent(r.Context(), agentID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Deploy template handlers ──

func (h *Handler) listDeployTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	framework := r.URL.Query().Get("framework")

	templates, total, err := h.svc.ListDeployTemplates(r.Context(), category, framework)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if templates == nil {
		templates = []*DeployTemplate{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     total,
		"templates": templates,
	})
}

func (h *Handler) getDeployTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "templateId")
	t, err := h.svc.GetDeployTemplate(r.Context(), id)
	if err != nil {
		apperr.NotFound(w, "template")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// ── Git connection handlers ──

func (h *Handler) createGitConnection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Provider       string `json:"provider"`
		InstallationID string `json:"installationId"`
		AccessToken    string `json:"accessToken"`
		RefreshToken   string `json:"refreshToken"`
		AccountName    string `json:"accountName"`
		AccountType    string `json:"accountType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Provider) == "" {
		apperr.BadRequest(w, "provider is required")
		return
	}
	if body.Provider != "github" && body.Provider != "gitlab" && body.Provider != "bitbucket" {
		apperr.BadRequest(w, "provider must be github, gitlab, or bitbucket")
		return
	}
	if strings.TrimSpace(body.AccessToken) == "" {
		apperr.BadRequest(w, "accessToken is required")
		return
	}

	conn, err := h.svc.CreateGitConnection(r.Context(), projectID, body.Provider, body.InstallationID,
		body.AccessToken, body.RefreshToken, body.AccountName, body.AccountType)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, conn)
}

func (h *Handler) listGitConnections(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	connections, total, err := h.svc.ListGitConnections(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if connections == nil {
		connections = []*GitConnection{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":       total,
		"connections": connections,
	})
}

func (h *Handler) deleteGitConnection(w http.ResponseWriter, r *http.Request) {
	connectionID := chi.URLParam(r, "connectionId")
	if err := h.svc.DeleteGitConnection(r.Context(), connectionID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listRepositories(w http.ResponseWriter, r *http.Request) {
	connectionID := chi.URLParam(r, "connectionId")
	repos, err := h.svc.ListRepositories(r.Context(), connectionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "git connection")
			return
		}
		apperr.Internal(w, err)
		return
	}
	if repos == nil {
		repos = []*GitRepository{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":        len(repos),
		"repositories": repos,
	})
}

// ── Environment handlers ──

func (h *Handler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name    string            `json:"name"`
		Slug    string            `json:"slug"`
		Branch  string            `json:"branch"`
		Domain  string            `json:"domain"`
		EnvVars map[string]string `json:"envVars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if strings.TrimSpace(body.Slug) == "" {
		apperr.BadRequest(w, "slug is required")
		return
	}

	env, err := h.svc.CreateEnvironment(r.Context(), projectID, body.Name, body.Slug, body.Branch, body.Domain, body.EnvVars)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, env)
}

func (h *Handler) listEnvironments(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	envs, total, err := h.svc.ListEnvironments(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if envs == nil {
		envs = []*Environment{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":        total,
		"environments": envs,
	})
}

func (h *Handler) getEnvironment(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "envId")
	env, err := h.svc.GetEnvironment(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "environment")
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (h *Handler) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "envId")
	var body struct {
		Name    string            `json:"name"`
		Branch  string            `json:"branch"`
		Domain  string            `json:"domain"`
		EnvVars map[string]string `json:"envVars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}

	env, err := h.svc.UpdateEnvironment(r.Context(), id, body.Name, body.Branch, body.Domain, body.EnvVars)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (h *Handler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "envId")
	if err := h.svc.DeleteEnvironment(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "environment")
			return
		}
		if strings.Contains(err.Error(), "cannot delete") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Git webhook handlers ──

// handleGitWebhook is the public inbound endpoint for GitHub/GitLab push and PR events.
// It is mounted at POST /v1/deploy/git/webhook/{connectionId} without project-auth middleware.
func (h *Handler) handleGitWebhook(w http.ResponseWriter, r *http.Request) {
	connectionID := chi.URLParam(r, "connectionId")

	// Detect provider from headers.
	event := r.Header.Get("X-GitHub-Event")
	signature := r.Header.Get("X-Hub-Signature-256")
	if event == "" {
		// GitLab sends X-Gitlab-Event and X-Gitlab-Token.
		event = r.Header.Get("X-Gitlab-Event")
		signature = r.Header.Get("X-Gitlab-Token")
		// Normalise GitLab event names to match our handler.
		switch event {
		case "Push Hook":
			event = "push"
		case "Merge Request Hook":
			event = "pull_request"
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20)) // 5 MB limit
	if err != nil {
		apperr.BadRequest(w, "failed to read request body")
		return
	}

	triggered, err := h.svc.HandleGitWebhook(r.Context(), connectionID, event, body, signature)
	if err != nil {
		if strings.Contains(err.Error(), "signature verification failed") || strings.Contains(err.Error(), "no secret configured") {
			apperr.Write(w, http.StatusUnauthorized, "webhook_unauthorized", err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "git connection")
			return
		}
		apperr.Internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"triggered": triggered,
	})
}

// generateWebhookSecret creates (or rotates) the HMAC secret for a git connection.
func (h *Handler) generateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	connectionID := chi.URLParam(r, "connectionId")

	secret, err := h.svc.GenerateWebhookSecret(r.Context(), connectionID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"webhookSecret": secret,
		"webhookUrl":    "/v1/deploy/git/webhook/" + connectionID,
	})
}

// ── Preview release handlers ──

func (h *Handler) listPreviewReleases(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	targetID := chi.URLParam(r, "targetId")

	releases, total, err := h.svc.ListPreviewReleases(r.Context(), projectID, targetID)
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

func (h *Handler) destroyPreviewRelease(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	releaseID := chi.URLParam(r, "releaseId")

	if err := h.svc.DestroyPreviewRelease(r.Context(), releaseID, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "preview release")
			return
		}
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
