package webhooks

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for webhooks.
type Handler struct {
	svc *Service
}

// NewHandler creates a new webhooks Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the webhooks router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createWebhook)
	r.Get("/", h.listWebhooks)
	r.Get("/{id}", h.getWebhook)
	r.Put("/{id}", h.updateWebhook)
	r.Delete("/{id}", h.deleteWebhook)
	r.Get("/{id}/deliveries", h.listDeliveries)
	r.Post("/{id}/deliveries/{deliveryId}/retry", h.retryDelivery)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) createWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name    string   `json:"name"`
		URL     string   `json:"url"`
		Events  []string `json:"events"`
		Secret  string   `json:"secret"`
		Enabled *bool    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.URL) == "" {
		apperr.BadRequest(w, "name and url are required")
		return
	}
	if body.Events == nil {
		body.Events = []string{}
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	webhook, err := h.svc.Create(r.Context(), projectID, body.Name, body.URL, body.Events, body.Secret, enabled)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, webhook)
}

func (h *Handler) listWebhooks(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	webhooks, total, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "webhooks": webhooks})
}

func (h *Handler) getWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "id")
	webhook, err := h.svc.Get(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "webhook")
		return
	}
	writeJSON(w, http.StatusOK, webhook)
}

func (h *Handler) updateWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var body struct {
		Name    string   `json:"name"`
		URL     string   `json:"url"`
		Events  []string `json:"events"`
		Secret  string   `json:"secret"`
		Enabled bool     `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Events == nil {
		body.Events = []string{}
	}
	webhook, err := h.svc.Update(r.Context(), id, projectID, body.Name, body.URL, body.Events, body.Secret, body.Enabled)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, webhook)
}

func (h *Handler) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listDeliveries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deliveries, total, err := h.svc.ListDeliveries(r.Context(), id)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "deliveries": deliveries})
}

func (h *Handler) retryDelivery(w http.ResponseWriter, r *http.Request) {
	deliveryID := chi.URLParam(r, "deliveryId")
	delivery, err := h.svc.RetryDelivery(r.Context(), deliveryID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "delivery")
			return
		}
		if strings.Contains(err.Error(), "max retry") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, delivery)
}
