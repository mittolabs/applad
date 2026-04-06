package usage

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for usage analytics.
type Handler struct {
	svc *Service
}

// NewHandler creates a new usage Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the usage router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.getUsage)
	r.Get("/stats", h.getStats)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) getUsage(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	metric := r.URL.Query().Get("metric")
	rangeStr := r.URL.Query().Get("range")

	if metric == "" {
		apperr.BadRequest(w, "metric query parameter is required")
		return
	}
	if rangeStr == "" {
		rangeStr = "24h"
	}

	points, err := h.svc.GetUsage(r.Context(), projectID, metric, rangeStr)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"metric": metric,
		"range":  rangeStr,
		"total":  len(points),
		"data":   points,
	})
}

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	stats, err := h.svc.GetProjectStats(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
