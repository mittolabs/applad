package audit

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes audit log HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new audit Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the audit log router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/{logId}", h.get)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit == 0 {
		limit = 50
	}
	logs, total, err := h.svc.List(r.Context(), projectID,
		q.Get("action"), q.Get("resourceType"), q.Get("userId"),
		limit, offset,
	)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if logs == nil {
		logs = []*Log{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"logs":   logs,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	logID := chi.URLParam(r, "logId")
	l, err := h.svc.Get(r.Context(), logID, projectID)
	if err != nil {
		apperr.NotFound(w, "audit_log")
		return
	}
	writeJSON(w, http.StatusOK, l)
}
