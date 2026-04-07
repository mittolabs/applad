package migrations

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for migrations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new migrations Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the migrations router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Post("/{id}/retry", h.retry)
	r.Delete("/{id}", h.delete)
	r.Post("/validate", h.validate)
	return r
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Source string                 `json:"source"`
		Config map[string]interface{} `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Source) == "" {
		apperr.BadRequest(w, "source is required")
		return
	}
	if body.Config == nil {
		body.Config = map[string]interface{}{}
	}

	m, err := h.svc.Create(r.Context(), projectID, body.Source, body.Config)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	migrations, total, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if migrations == nil {
		migrations = []*Migration{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      total,
		"migrations": migrations,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "id")
	m, err := h.svc.Get(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "migration")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) retry(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "id")
	m, err := h.svc.Retry(r.Context(), id, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "migration")
			return
		}
		if strings.Contains(err.Error(), "can only retry") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id, projectID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "migration")
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

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Source string                 `json:"source"`
		Config map[string]interface{} `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Source) == "" {
		apperr.BadRequest(w, "source is required")
		return
	}
	if body.Config == nil {
		body.Config = map[string]interface{}{}
	}

	report, err := h.svc.ValidateReport(r.Context(), projectID, body.Source, body.Config)
	if err != nil {
		if strings.Contains(err.Error(), "unsupported") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}
