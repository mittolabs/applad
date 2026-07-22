package appcache

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the managed cache HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new cache Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the cache router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/stats", h.stats)
	r.Delete("/", h.flush)
	r.Post("/invalidate", h.invalidateByTag)
	r.Get("/{key}", h.get)
	r.Put("/{key}", h.set)
	r.Delete("/{key}", h.delete)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	key := chi.URLParam(r, "key")
	entry, err := h.svc.Get(r.Context(), projectID, key)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if entry == nil {
		apperr.Write(w, http.StatusNotFound, "cache_miss", "key not found or expired")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	key := chi.URLParam(r, "key")
	var body struct {
		Value interface{} `json:"value"`
		TTL   int         `json:"ttl"` // seconds; 0 = no expiry
		Tags  []string    `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "invalid body")
		return
	}
	ttl := time.Duration(body.TTL) * time.Second
	if err := h.svc.Set(r.Context(), projectID, key, body.Value, ttl, body.Tags); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"key": key, "set": true})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	key := chi.URLParam(r, "key")
	if err := h.svc.Delete(r.Context(), projectID, key); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) invalidateByTag(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Tag == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "tag is required")
		return
	}
	count, err := h.svc.InvalidateByTag(r.Context(), projectID, body.Tag)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"invalidated": count})
}

func (h *Handler) flush(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	count, err := h.svc.Flush(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"flushed": count})
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	stats, err := h.svc.Stats(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
