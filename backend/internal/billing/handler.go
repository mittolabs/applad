package billing

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the billing HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new billing Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the billing router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Plans (public read, admin write handled externally)
	r.Get("/plans", h.listPlans)

	// Subscription
	r.Get("/subscription", h.getSubscription)
	r.Post("/subscription", h.subscribe)
	r.Delete("/subscription", h.cancelSubscription)

	// Usage / metering
	r.Post("/events", h.recordEvent)
	r.Get("/usage", h.getUsage)

	// Invoices
	r.Get("/invoices", h.listInvoices)
	r.Get("/invoices/{invoiceId}", h.getInvoice)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) listPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListPlans(r.Context())
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if plans == nil {
		plans = []*Plan{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(plans), "plans": plans})
}

func (h *Handler) getSubscription(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	sub, err := h.svc.GetSubscription(r.Context(), projectID)
	if err != nil {
		apperr.Write(w, http.StatusNotFound, "subscription_not_found", "no active subscription")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) subscribe(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		PlanID string `json:"planId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PlanID == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "planId is required")
		return
	}
	sub, err := h.svc.Subscribe(r.Context(), projectID, body.PlanID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) cancelSubscription(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	if err := h.svc.CancelSubscription(r.Context(), projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"cancelAtPeriodEnd": true})
}

func (h *Handler) recordEvent(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		EventType string                 `json:"eventType"`
		Quantity  int64                  `json:"quantity"`
		Unit      string                 `json:"unit"`
		Metadata  map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.EventType == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "eventType is required")
		return
	}
	if body.Quantity == 0 {
		body.Quantity = 1
	}
	e, err := h.svc.RecordEvent(r.Context(), projectID, body.EventType, body.Quantity, body.Unit, body.Metadata)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) getUsage(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	q := r.URL.Query()
	from := time.Now().UTC().AddDate(0, -1, 0)
	to := time.Now().UTC()
	if t, err := time.Parse(time.RFC3339, q.Get("from")); err == nil {
		from = t
	}
	if t, err := time.Parse(time.RFC3339, q.Get("to")); err == nil {
		to = t
	}
	summary, err := h.svc.GetUsageSummary(r.Context(), projectID, from, to)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) listInvoices(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit == 0 {
		limit = 20
	}
	invoices, total, err := h.svc.ListInvoices(r.Context(), projectID, limit, offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if invoices == nil {
		invoices = []*Invoice{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": total, "limit": limit, "offset": offset, "invoices": invoices,
	})
}

func (h *Handler) getInvoice(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	invoiceID := chi.URLParam(r, "invoiceId")
	invoice, err := h.svc.GetInvoice(r.Context(), projectID, invoiceID)
	if err != nil {
		apperr.NotFound(w, "invoice")
		return
	}
	writeJSON(w, http.StatusOK, invoice)
}

// ── Admin plan creation (no project auth needed) ──────────────────────────────

// AdminRoutes returns routes for admin plan management (mount under /console or similar).
func AdminRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/plans", h.adminCreatePlan)
	return r
}

func (h *Handler) adminCreatePlan(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string                 `json:"name"`
		Slug         string                 `json:"slug"`
		PriceMonthly int                    `json:"priceMonthly"`
		PriceYearly  int                    `json:"priceYearly"`
		Limits       map[string]interface{} `json:"limits"`
		Features     []string               `json:"features"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Slug == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "slug is required")
		return
	}
	p, err := h.svc.CreatePlan(r.Context(), body.Name, body.Slug, body.PriceMonthly, body.PriceYearly, body.Limits, body.Features)
	if err != nil {
		apperr.Write(w, http.StatusConflict, "plan_exists", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}
