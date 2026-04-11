package observe

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the observe router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Overview
	r.Get("/overview", h.getOverview)

	// Errors
	r.Get("/errors", h.listErrors)
	r.Post("/errors", h.captureError)
	r.Get("/errors/{errorId}", h.getError)
	r.Patch("/errors/{errorId}/resolve", h.resolveError)
	r.Patch("/errors/{errorId}/ignore", h.ignoreError)
	r.Patch("/errors/{errorId}/unresolve", h.unresolveError)
	r.Patch("/errors/{errorId}/priority", h.setErrorPriority)
	r.Patch("/errors/{errorId}/assign", h.assignError)
	r.Post("/errors/{errorId}/activity", h.addActivity)
	r.Post("/errors/bulk", h.bulkUpdateErrors)

	// Logs
	r.Get("/logs", h.listLogs)
	r.Post("/logs", h.captureLog)

	// Performance
	r.Get("/performance", h.getPerformance)
	r.Post("/performance", h.recordPerf)
	r.Post("/performance/vitals", h.recordVitals)

	// Releases
	r.Get("/releases", h.listReleases)
	r.Post("/releases", h.createRelease)
	r.Get("/releases/{releaseId}", h.getRelease)

	// Replays
	r.Get("/replays", h.listReplays)
	r.Post("/replays", h.createReplay)
	r.Get("/replays/{replayId}", h.getReplay)

	// Uptime
	r.Get("/uptime", h.listUptime)
	r.Post("/uptime", h.createUptime)
	r.Delete("/uptime/{monitorId}", h.deleteUptime)

	// Crons
	r.Get("/crons", h.listCrons)
	r.Post("/crons", h.createCron)
	r.Patch("/crons/{monitorId}/toggle", h.toggleCron)
	r.Delete("/crons/{monitorId}", h.deleteCron)
	r.Post("/crons/{monitorId}/checkin", h.cronCheckin)

	// Alerts
	r.Get("/alerts", h.listAlerts)
	r.Post("/alerts", h.createAlert)
	r.Patch("/alerts/{ruleId}/toggle", h.toggleAlert)
	r.Delete("/alerts/{ruleId}", h.deleteAlert)

	return r
}

// ── Overview ─────────────────────────────────────────────────────────────────

func (h *Handler) getOverview(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	data, err := h.svc.GetOverview(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// ── Errors ────────────────────────────────────────────────────────────────────

func (h *Handler) listErrors(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	status := r.URL.Query().Get("status")
	level := r.URL.Query().Get("level")
	search := r.URL.Query().Get("search")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	_ = search // search filtering not yet implemented in service
	errs, err := h.svc.ListErrors(r.Context(), projectID, status, level, limit)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if errs == nil {
		errs = []Error{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(errs), "errors": errs})
}

func (h *Handler) captureError(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req CaptureErrorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "title is required")
		return
	}
	e, err := h.svc.CaptureError(r.Context(), projectID, req)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) getError(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	errorID := chi.URLParam(r, "errorId")
	e, err := h.svc.GetError(r.Context(), projectID, errorID)
	if err != nil || e == nil {
		apperr.NotFound(w, "error")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) resolveError(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	errorID := chi.URLParam(r, "errorId")
	user := middleware.UserFromContext(r.Context())
	if err := h.svc.UpdateErrorStatus(r.Context(), projectID, errorID, "resolved", user); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "resolved"})
}

func (h *Handler) ignoreError(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	errorID := chi.URLParam(r, "errorId")
	user := middleware.UserFromContext(r.Context())
	if err := h.svc.UpdateErrorStatus(r.Context(), projectID, errorID, "ignored", user); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ignored"})
}

func (h *Handler) unresolveError(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	errorID := chi.URLParam(r, "errorId")
	user := middleware.UserFromContext(r.Context())
	if err := h.svc.UpdateErrorStatus(r.Context(), projectID, errorID, "unresolved", user); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "unresolved"})
}

func (h *Handler) setErrorPriority(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	errorID := chi.URLParam(r, "errorId")
	var body struct {
		Priority string `json:"priority"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.SetErrorPriority(r.Context(), projectID, errorID, body.Priority); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"priority": body.Priority})
}

func (h *Handler) assignError(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	errorID := chi.URLParam(r, "errorId")
	var body struct {
		Assignee string `json:"assignee"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.AssignError(r.Context(), projectID, errorID, body.Assignee); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assignee": body.Assignee})
}

func (h *Handler) addActivity(w http.ResponseWriter, r *http.Request) {
	errorID := chi.URLParam(r, "errorId")
	user := middleware.UserFromContext(r.Context())
	var body struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.AddActivity(r.Context(), errorID, "note", user, body.Text); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"created": true})
}

func (h *Handler) bulkUpdateErrors(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	user := middleware.UserFromContext(r.Context())
	var body struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"` // resolve | ignore | unresolve
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.IDs) == 0 {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "ids and action required")
		return
	}
	status := body.Action
	if status == "unresolve" {
		status = "unresolved"
	}
	for _, id := range body.IDs {
		_ = h.svc.UpdateErrorStatus(r.Context(), projectID, id, status, user)
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": len(body.IDs)})
}

// ── Logs ──────────────────────────────────────────────────────────────────────

func (h *Handler) listLogs(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	level := r.URL.Query().Get("level")
	source := r.URL.Query().Get("source")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	logs, err := h.svc.ListLogs(r.Context(), projectID, level, source, limit)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if logs == nil {
		logs = []LogEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(logs), "logs": logs})
}

func (h *Handler) captureLog(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req CaptureLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "message is required")
		return
	}
	entry, err := h.svc.CaptureLog(r.Context(), projectID, req)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

// ── Performance ───────────────────────────────────────────────────────────────

func (h *Handler) getPerformance(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	data, err := h.svc.GetPerformance(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) recordPerf(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req RecordPerfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "invalid body")
		return
	}
	if err := h.svc.RecordPerf(r.Context(), projectID, req); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) recordVitals(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req WebVitalsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "invalid body")
		return
	}
	if err := h.svc.RecordWebVitals(r.Context(), projectID, req); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Releases ──────────────────────────────────────────────────────────────────

func (h *Handler) listReleases(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	releases, err := h.svc.ListReleases(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if releases == nil {
		releases = []Release{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(releases), "releases": releases})
}

func (h *Handler) createRelease(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req CreateReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "version is required")
		return
	}
	rel, err := h.svc.CreateRelease(r.Context(), projectID, req)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rel)
}

func (h *Handler) getRelease(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	releaseID := chi.URLParam(r, "releaseId")
	rel, err := h.svc.GetRelease(r.Context(), projectID, releaseID)
	if err != nil || rel == nil {
		apperr.NotFound(w, "release")
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

// ── Replays ───────────────────────────────────────────────────────────────────

func (h *Handler) listReplays(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	replays, err := h.svc.ListReplays(r.Context(), projectID, limit)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if replays == nil {
		replays = []Replay{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(replays), "replays": replays})
}

func (h *Handler) createReplay(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req CreateReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "invalid body")
		return
	}
	rep, err := h.svc.CreateReplay(r.Context(), projectID, req)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rep)
}

func (h *Handler) getReplay(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	replayID := chi.URLParam(r, "replayId")
	rep, err := h.svc.GetReplay(r.Context(), projectID, replayID)
	if err != nil || rep == nil {
		apperr.NotFound(w, "replay")
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// ── Uptime ────────────────────────────────────────────────────────────────────

func (h *Handler) listUptime(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	monitors, err := h.svc.ListUptimeMonitors(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if monitors == nil {
		monitors = []UptimeMonitor{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"monitors": monitors})
}

func (h *Handler) createUptime(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req CreateUptimeMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.URL == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name and url are required")
		return
	}
	mon, err := h.svc.CreateUptimeMonitor(r.Context(), projectID, req)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, mon)
}

func (h *Handler) deleteUptime(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	monitorID := chi.URLParam(r, "monitorId")
	if err := h.svc.DeleteUptimeMonitor(r.Context(), projectID, monitorID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Crons ─────────────────────────────────────────────────────────────────────

func (h *Handler) listCrons(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	monitors, err := h.svc.ListCronMonitors(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if monitors == nil {
		monitors = []CronMonitor{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"monitors": monitors})
}

func (h *Handler) createCron(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req CreateCronMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Schedule == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name and schedule are required")
		return
	}
	mon, err := h.svc.CreateCronMonitor(r.Context(), projectID, req)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, mon)
}

func (h *Handler) toggleCron(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	monitorID := chi.URLParam(r, "monitorId")
	if err := h.svc.ToggleCronMonitor(r.Context(), projectID, monitorID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"toggled": true})
}

func (h *Handler) deleteCron(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	monitorID := chi.URLParam(r, "monitorId")
	if err := h.svc.DeleteCronMonitor(r.Context(), projectID, monitorID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) cronCheckin(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorId")
	var req CronCheckinRequest
	json.NewDecoder(r.Body).Decode(&req)
	if err := h.svc.CronCheckin(r.Context(), monitorID, req); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"received": true})
}

// ── Alerts ────────────────────────────────────────────────────────────────────

func (h *Handler) listAlerts(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	data, err := h.svc.ListAlerts(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) createAlert(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var req CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	rule, err := h.svc.CreateAlertRule(r.Context(), projectID, req)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) toggleAlert(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleId")
	if err := h.svc.ToggleAlertRule(r.Context(), projectID, ruleID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"toggled": true})
}

func (h *Handler) deleteAlert(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleId")
	if err := h.svc.DeleteAlertRule(r.Context(), projectID, ruleID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
