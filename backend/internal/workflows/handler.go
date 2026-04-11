package workflows

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for workflows.
type Handler struct {
	svc *Service
}

// NewHandler creates a new workflow Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the authenticated workflow router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{workflowId}", h.get)
	r.Put("/{workflowId}", h.update)
	r.Delete("/{workflowId}", h.delete)
	r.Post("/{workflowId}/execute", h.execute)
	r.Post("/{workflowId}/webhook-secret", h.regenerateWebhookSecret)
	r.Get("/{workflowId}/executions", h.listExecutions)
	r.Get("/{workflowId}/executions/{executionId}", h.getExecution)
	// Versioning
	r.Get("/{workflowId}/versions", h.listVersions)
	// Sharing
	r.Post("/{workflowId}/shares", h.shareWorkflow)
	r.Get("/{workflowId}/shares", h.listShares)
	r.Delete("/{workflowId}/shares/{userId}", h.unshareWorkflow)
	// Folders
	r.Post("/folders", h.createFolder)
	r.Get("/folders", h.listFolders)
	// Templates
	r.Get("/templates", h.listTemplates)
	r.Get("/templates/{templateId}", h.getTemplate)
	return r
}

// WebhookRoutes returns the public webhook trigger router (no auth required).
func WebhookRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/{workflowId}", h.webhookTrigger)
	return r
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name          string                 `json:"name"`
		Description   string                 `json:"description"`
		TriggerType   string                 `json:"triggerType"`
		TriggerConfig map[string]interface{} `json:"triggerConfig"`
		Nodes         []Node                 `json:"nodes"`
		Edges         []Edge                 `json:"edges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.TriggerType == "" {
		body.TriggerType = "manual"
	}

	wf, err := h.svc.Create(r.Context(), projectID, body.Name, body.Description, body.TriggerType, body.TriggerConfig, body.Nodes, body.Edges)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, wf)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	workflows, total, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if workflows == nil {
		workflows = []*Workflow{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     total,
		"workflows": workflows,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "workflowId")
	wf, err := h.svc.Get(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "workflow")
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "workflowId")
	var body struct {
		Name          string                 `json:"name"`
		Description   string                 `json:"description"`
		Status        string                 `json:"status"`
		TriggerType   string                 `json:"triggerType"`
		TriggerConfig map[string]interface{} `json:"triggerConfig"`
		Nodes         []Node                 `json:"nodes"`
		Edges         []Edge                 `json:"edges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Status == "" {
		body.Status = "draft"
	}
	if body.TriggerType == "" {
		body.TriggerType = "manual"
	}

	wf, err := h.svc.Update(r.Context(), id, projectID, body.Name, body.Description, body.Status, body.TriggerType, body.TriggerConfig, body.Nodes, body.Edges)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "workflowId")
	if err := h.svc.Delete(r.Context(), id, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) regenerateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "workflowId")
	secret, err := h.svc.RegenerateWebhookSecret(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "workflow")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"webhookSecret": secret})
}

func (h *Handler) execute(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "workflowId")

	// Verify workflow exists and is active or draft
	wf, err := h.svc.Get(r.Context(), id, projectID)
	if err != nil {
		apperr.NotFound(w, "workflow")
		return
	}
	if wf.Status == "paused" {
		apperr.BadRequest(w, "workflow is paused")
		return
	}

	var body struct {
		TriggerData map[string]interface{} `json:"triggerData"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	exec, err := h.svc.Execute(r.Context(), id, projectID, body.TriggerData)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, exec)
}

func (h *Handler) listExecutions(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	workflowID := chi.URLParam(r, "workflowId")
	execs, total, err := h.svc.ListExecutions(r.Context(), workflowID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if execs == nil {
		execs = []*Execution{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":      total,
		"executions": execs,
	})
}

func (h *Handler) getExecution(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	workflowID := chi.URLParam(r, "workflowId")
	execID := chi.URLParam(r, "executionId")
	exec, err := h.svc.GetExecution(r.Context(), execID, workflowID, projectID)
	if err != nil {
		apperr.NotFound(w, "execution")
		return
	}
	writeJSON(w, http.StatusOK, exec)
}

func (h *Handler) webhookTrigger(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")

	// Look up workflow without project scoping — webhook URLs are public
	wf, err := h.svc.GetByID(r.Context(), workflowID)
	if err != nil {
		apperr.NotFound(w, "workflow")
		return
	}
	if wf.Status != "active" {
		apperr.BadRequest(w, "workflow is not active")
		return
	}
	if wf.TriggerType != "webhook" {
		apperr.BadRequest(w, "workflow trigger type is not webhook")
		return
	}

	// Read body first so we can verify the HMAC signature before processing.
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		apperr.BadRequest(w, "failed to read request body")
		return
	}

	// Verify HMAC-SHA256 signature when the workflow has a secret configured.
	// Clients send: X-Applad-Signature: <hex(hmac-sha256(secret, body))>
	if wf.WebhookSecret != "" {
		sig := r.Header.Get("X-Applad-Signature")
		if sig == "" {
			apperr.Write(w, http.StatusUnauthorized, "missing_signature",
				"X-Applad-Signature header is required for this webhook.")
			return
		}
		mac := hmac.New(sha256.New, []byte(wf.WebhookSecret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(sig)) {
			apperr.Write(w, http.StatusUnauthorized, "invalid_signature",
				"Webhook signature verification failed.")
			return
		}
	}

	// Parse trigger data from the pre-read body
	var triggerData map[string]interface{}
	json.Unmarshal(body, &triggerData) //nolint:errcheck
	if triggerData == nil {
		triggerData = map[string]interface{}{}
	}
	triggerData["_method"] = r.Method
	triggerData["_headers"] = flattenHeaders(r.Header)

	exec, err := h.svc.Execute(r.Context(), wf.ID, wf.ProjectID, triggerData)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, exec)
}

// ── Versioning ──

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")
	versions, err := h.svc.ListVersions(r.Context(), workflowID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if versions == nil {
		versions = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"versions": versions})
}

// ── Sharing ──

func (h *Handler) shareWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")
	var body struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "userId is required")
		return
	}
	if body.Role == "" {
		body.Role = "viewer"
	}
	if err := h.svc.ShareWorkflow(r.Context(), workflowID, body.UserID, body.Role); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"shared": true})
}

func (h *Handler) listShares(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")
	shares, err := h.svc.ListShares(r.Context(), workflowID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if shares == nil {
		shares = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"shares": shares})
}

func (h *Handler) unshareWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowId")
	userID := chi.URLParam(r, "userId")
	if err := h.svc.UnshareWorkflow(r.Context(), workflowID, userID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"unshared": true})
}

// ── Folders ──

func (h *Handler) createFolder(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name     string `json:"name"`
		ParentID string `json:"parentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	id, err := h.svc.CreateFolder(r.Context(), projectID, body.Name, body.ParentID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"$id": id, "name": body.Name})
}

func (h *Handler) listFolders(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	folders, err := h.svc.ListFolders(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if folders == nil {
		folders = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"folders": folders})
}

// ── Templates ──

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.svc.ListTemplates(r.Context())
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if templates == nil {
		templates = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates})
}

func (h *Handler) getTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := chi.URLParam(r, "templateId")
	tpl, err := h.svc.GetTemplate(r.Context(), templateID)
	if err != nil {
		apperr.NotFound(w, "template")
		return
	}
	writeJSON(w, http.StatusOK, tpl)
}
