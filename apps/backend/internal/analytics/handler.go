package analytics

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the analytics HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new analytics Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the analytics router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Event ingestion
	r.Post("/events", h.track)
	r.Post("/events/batch", h.trackBatch)
	r.Get("/events", h.listEvents)

	// Aggregations
	r.Get("/stats", h.stats)
	r.Get("/realtime", h.realtime)
	r.Get("/events/counts", h.eventCounts)
	r.Get("/dau", h.dau)

	// Funnels
	r.Post("/funnels", h.createFunnel)
	r.Get("/funnels", h.listFunnels)
	r.Get("/funnels/{funnelId}/analyze", h.analyzeFunnel)
	r.Delete("/funnels/{funnelId}", h.deleteFunnel)

	// Overview
	r.Get("/overview", h.overview)

	// Request performance, measured by the platform itself
	r.Get("/performance", h.getPerformance)
	r.Post("/performance", h.recordPerf)

	// Uptime monitors
	r.Get("/uptime", h.listUptime)
	r.Post("/uptime", h.createUptime)
	r.Delete("/uptime/{monitorId}", h.deleteUptime)

	// Cron monitors
	r.Get("/crons", h.listCrons)
	r.Post("/crons", h.createCron)
	r.Patch("/crons/{monitorId}/toggle", h.toggleCron)
	r.Delete("/crons/{monitorId}", h.deleteCron)
	r.Post("/crons/{monitorId}/checkin", h.cronCheckin)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) track(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil || e.Event == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "event name is required")
		return
	}
	e.ProjectID = projectID
	if e.UserID == "" {
		e.UserID = middleware.UserFromContext(r.Context())
	}
	created, err := h.svc.Track(r.Context(), e)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) trackBatch(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Events []Event `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "invalid body")
		return
	}
	for i := range body.Events {
		body.Events[i].ProjectID = projectID
		if body.Events[i].UserID == "" {
			body.Events[i].UserID = middleware.UserFromContext(r.Context())
		}
	}
	count, err := h.svc.TrackBatch(r.Context(), body.Events)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"processed": count})
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit == 0 {
		limit = 100
	}
	from, to := parseTimeRange(q.Get("from"), q.Get("to"))
	events, total, err := h.svc.QueryEvents(r.Context(), projectID, q.Get("event"), q.Get("userId"), from, to, limit, offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if events == nil {
		events = []*Event{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": total, "limit": limit, "offset": offset, "events": events,
	})
}

func (h *Handler) eventCounts(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	from, to := parseTimeRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	counts, err := h.svc.EventCounts(r.Context(), projectID, from, to)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"counts": counts})
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	h.eventCounts(w, r)
}

func (h *Handler) dau(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	from, to := parseTimeRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	data, err := h.svc.DAU(r.Context(), projectID, from, to)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if data == nil {
		data = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (h *Handler) realtime(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	summary, err := h.svc.RealtimeSummary(r.Context(), projectID, time.Now().UTC().Add(-5*time.Minute))
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) createFunnel(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name  string   `json:"name"`
		Steps []string `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || len(body.Steps) < 2 {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name and at least 2 steps are required")
		return
	}
	f, err := h.svc.CreateFunnel(r.Context(), projectID, body.Name, body.Steps)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) listFunnels(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	funnels, err := h.svc.ListFunnels(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if funnels == nil {
		funnels = []*Funnel{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(funnels), "funnels": funnels})
}

func (h *Handler) analyzeFunnel(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	funnelID := chi.URLParam(r, "funnelId")
	from, to := parseTimeRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	result, err := h.svc.AnalyzeFunnel(r.Context(), projectID, funnelID, from, to)
	if err != nil {
		apperr.NotFound(w, "funnel")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) deleteFunnel(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	funnelID := chi.URLParam(r, "funnelId")
	if err := h.svc.DeleteFunnel(r.Context(), funnelID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseTimeRange parses ISO-8601 or RFC3339 strings; defaults to last 7 days.
func parseTimeRange(fromStr, toStr string) (from, to time.Time) {
	to = time.Now().UTC()
	from = to.AddDate(0, 0, -7)
	if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
		from = t
	}
	if t, err := time.Parse(time.RFC3339, toStr); err == nil {
		to = t
	}
	return
}

// ── Overview ─────────────────────────────────────────────────────────────────

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	data, err := h.svc.GetOverview(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// ── Performance ──────────────────────────────────────────────────────────────

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

// ── Uptime ───────────────────────────────────────────────────────────────────

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
	writeJSON(w, http.StatusOK, map[string]interface{}{"monitors": monitors})
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

// ── Crons ────────────────────────────────────────────────────────────────────

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
	writeJSON(w, http.StatusOK, map[string]interface{}{"monitors": monitors})
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"toggled": true})
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
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
	if err := h.svc.CronCheckin(r.Context(), monitorID, req); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"received": true})
}
