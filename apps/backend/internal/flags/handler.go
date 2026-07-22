package flags

import (
	"encoding/json"
	"net/http"
	"strings"

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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the feature flags router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Flag CRUD
	r.Post("/", h.createFlag)
	r.Get("/", h.listFlags)
	r.Get("/{flagId}", h.getFlag)
	r.Put("/{flagId}", h.updateFlag)
	r.Patch("/{flagId}/toggle", h.toggleFlag)
	r.Delete("/{flagId}", h.deleteFlag)

	// Rules
	r.Post("/{flagId}/rules", h.createRule)
	r.Get("/{flagId}/rules", h.listRules)
	r.Delete("/{flagId}/rules/{ruleId}", h.deleteRule)

	// Overrides
	r.Post("/{flagId}/overrides", h.setOverride)
	r.Get("/{flagId}/overrides", h.listOverrides)
	r.Delete("/{flagId}/overrides/{targetType}/{targetId}", h.deleteOverride)

	// Evaluation (the SDK calls these)
	r.Post("/evaluate", h.evaluate)
	r.Post("/evaluate/all", h.evaluateAll)
	r.Get("/evaluate/{key}", h.evaluateByKey)

	// Stats
	r.Get("/{flagId}/stats", h.getStats)

	return r
}

// ── Flag CRUD ────────────────────────────────────────────────────────────────

func (h *Handler) createFlag(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Key          string      `json:"key"`
		Name         string      `json:"name"`
		Description  string      `json:"description"`
		Type         string      `json:"type"`
		DefaultValue interface{} `json:"defaultValue"`
		Tags         []string    `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Key) == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "key is required")
		return
	}
	flag, err := h.svc.CreateFlag(r.Context(), projectID, body.Key, body.Name, body.Description, body.Type, body.DefaultValue, body.Tags)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			apperr.Write(w, http.StatusConflict, "flag_already_exists", err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, flag)
}

func (h *Handler) listFlags(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	flags, err := h.svc.ListFlags(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if flags == nil {
		flags = []*Flag{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(flags), "flags": flags})
}

func (h *Handler) getFlag(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	flagID := chi.URLParam(r, "flagId")
	flag, err := h.svc.GetFlag(r.Context(), flagID, projectID)
	if err != nil {
		apperr.NotFound(w, "flag")
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handler) updateFlag(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	flagID := chi.URLParam(r, "flagId")
	var body struct {
		Name         string      `json:"name"`
		Description  string      `json:"description"`
		DefaultValue interface{} `json:"defaultValue"`
		Enabled      bool        `json:"enabled"`
		Tags         []string    `json:"tags"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	flag, err := h.svc.UpdateFlag(r.Context(), flagID, projectID, body.Name, body.Description, body.DefaultValue, body.Enabled, body.Tags)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handler) toggleFlag(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	flagID := chi.URLParam(r, "flagId")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.ToggleFlag(r.Context(), flagID, projectID, body.Enabled); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"enabled": body.Enabled})
}

func (h *Handler) deleteFlag(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	flagID := chi.URLParam(r, "flagId")
	if err := h.svc.DeleteFlag(r.Context(), flagID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Rules ────────────────────────────────────────────────────────────────────

func (h *Handler) createRule(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flagId")
	var body struct {
		Type       string      `json:"type"`
		Conditions []Condition `json:"conditions"`
		Value      interface{} `json:"value"`
		RolloutPct int         `json:"rolloutPct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "invalid body")
		return
	}
	if body.RolloutPct == 0 {
		body.RolloutPct = 100
	}
	rule, err := h.svc.CreateRule(r.Context(), flagID, body.Type, body.Conditions, body.Value, body.RolloutPct)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) listRules(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flagId")
	rules, err := h.svc.ListRules(r.Context(), flagID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if rules == nil {
		rules = []Rule{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"rules": rules})
}

func (h *Handler) deleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleId")
	if err := h.svc.DeleteRule(r.Context(), ruleID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Overrides ────────────────────────────────────────────────────────────────

func (h *Handler) setOverride(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flagId")
	var body struct {
		TargetType string      `json:"targetType"`
		TargetID   string      `json:"targetId"`
		Value      interface{} `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TargetID == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "targetType, targetId, and value required")
		return
	}
	if err := h.svc.SetOverride(r.Context(), flagID, body.TargetType, body.TargetID, body.Value); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"set": true})
}

func (h *Handler) listOverrides(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flagId")
	overrides, err := h.svc.ListOverrides(r.Context(), flagID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if overrides == nil {
		overrides = []Override{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"overrides": overrides})
}

func (h *Handler) deleteOverride(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flagId")
	targetType := chi.URLParam(r, "targetType")
	targetID := chi.URLParam(r, "targetId")
	if err := h.svc.DeleteOverride(r.Context(), flagID, targetType, targetID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Evaluation (SDK endpoints) ───────────────────────────────────────────────

func (h *Handler) evaluate(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Key     string       `json:"key"`
		Context *EvalContext `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "key is required")
		return
	}
	// Auto-populate user context from auth
	if body.Context == nil {
		body.Context = &EvalContext{}
	}
	if body.Context.UserID == "" {
		body.Context.UserID = middleware.UserFromContext(r.Context())
	}

	result, err := h.svc.Evaluate(r.Context(), projectID, body.Key, body.Context)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) evaluateAll(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Context *EvalContext `json:"context"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Context == nil {
		body.Context = &EvalContext{}
	}
	if body.Context.UserID == "" {
		body.Context.UserID = middleware.UserFromContext(r.Context())
	}

	results, err := h.svc.EvaluateAll(r.Context(), projectID, body.Context)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"flags": results})
}

func (h *Handler) evaluateByKey(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	key := chi.URLParam(r, "key")
	evalCtx := &EvalContext{
		UserID: middleware.UserFromContext(r.Context()),
	}
	result, err := h.svc.Evaluate(r.Context(), projectID, key, evalCtx)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Stats ────────────────────────────────────────────────────────────────────

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	flagID := chi.URLParam(r, "flagId")
	stats, err := h.svc.GetFlagStats(r.Context(), flagID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
