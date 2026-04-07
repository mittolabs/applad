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
	r.Get("/events/counts", h.eventCounts)
	r.Get("/dau", h.dau)

	// Funnels
	r.Post("/funnels", h.createFunnel)
	r.Get("/funnels", h.listFunnels)
	r.Get("/funnels/{funnelId}/analyze", h.analyzeFunnel)
	r.Delete("/funnels/{funnelId}", h.deleteFunnel)

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
