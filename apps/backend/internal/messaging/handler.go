package messaging

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	mw "github.com/mittolabs/applad/internal/middleware"
)

// Handler handles HTTP requests for messaging.
type Handler struct {
	svc *Service
}

// NewHandler creates a new messaging Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the messaging router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Providers
	r.Get("/providers", h.listProviders)
	r.Post("/providers", h.createProvider)
	r.Get("/providers/{providerId}", h.getProvider)
	r.Put("/providers/{providerId}", h.updateProvider)
	r.Delete("/providers/{providerId}", h.deleteProvider)

	// Messages
	r.Get("/messages", h.listMessages)
	r.Post("/messages/email", h.createEmail)
	r.Post("/messages/sms", h.createSMS)
	r.Post("/messages/push", h.createPush)
	r.Get("/messages/{msgId}", h.getMessage)
	r.Delete("/messages/{msgId}", h.deleteMessage)

	// Legacy direct-send endpoints (backwards compat)
	r.Post("/email", h.sendEmailLegacy)
	r.Post("/sms", h.sendSMSLegacy)
	r.Post("/push", h.sendPushLegacy)

	// Topics
	r.Get("/topics", h.listTopics)
	r.Post("/topics", h.createTopic)
	r.Get("/topics/{topicId}", h.getTopic)
	r.Post("/topics/{topicId}/subscribers", h.addSubscriber)
	r.Post("/topics/{topicId}/messages", h.sendToTopic)

	// Templates
	r.Get("/templates", h.listTemplates)
	r.Post("/templates", h.createTemplate)
	r.Get("/templates/{templateId}", h.getTemplate)
	r.Put("/templates/{templateId}", h.updateTemplate)
	r.Delete("/templates/{templateId}", h.deleteTemplate)
	r.Post("/templates/{templateId}/send", h.sendTemplate)

	return r
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	limit := queryInt(r, "limit", 25)
	offset := queryInt(r, "offset", 0)
	search := r.URL.Query().Get("search")

	msgs, total, err := h.svc.ListMessages(r.Context(), projectID, limit, offset, search)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"messages": msgs,
		"total":    total,
	})
}

func (h *Handler) createEmail(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	var body struct {
		To          []string `json:"to"`
		Subject     string   `json:"subject"`
		HTML        string   `json:"html"`
		Draft       bool     `json:"draft"`
		ScheduledAt *string  `json:"scheduledAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Subject == "" {
		apperr.BadRequest(w, "subject is required")
		return
	}
	scheduledAt, err := parseScheduledAt(body.ScheduledAt)
	if err != nil {
		apperr.BadRequest(w, "scheduledAt must be an RFC 3339 timestamp")
		return
	}

	status, sendNow := sendDecision(body.Draft, scheduledAt)

	msg, err := h.svc.CreateMessage(r.Context(), projectID, "email", body.Subject, body.HTML, body.To, status, scheduledAt)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	if sendNow {
		go func() {
			sendErr := h.svc.SendEmailForProject(r.Context(), projectID, body.To, body.Subject, body.HTML)
			newStatus := "sent"
			if sendErr != nil {
				newStatus = "failed"
			}
			_ = h.svc.UpdateMessageStatus(r.Context(), msg.ID, newStatus)
			msg.Status = newStatus
		}()
	}

	writeJSON(w, http.StatusCreated, msg)
}

func (h *Handler) createSMS(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	var body struct {
		To          string  `json:"to"`
		Body        string  `json:"body"`
		Draft       bool    `json:"draft"`
		ScheduledAt *string `json:"scheduledAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Body == "" {
		apperr.BadRequest(w, "body is required")
		return
	}
	scheduledAt, err := parseScheduledAt(body.ScheduledAt)
	if err != nil {
		apperr.BadRequest(w, "scheduledAt must be an RFC 3339 timestamp")
		return
	}

	status, sendNow := sendDecision(body.Draft, scheduledAt)
	recipients := []string{}
	if body.To != "" {
		recipients = []string{body.To}
	}

	msg, err := h.svc.CreateMessage(r.Context(), projectID, "sms", "", body.Body, recipients, status, scheduledAt)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	if sendNow {
		go func() {
			sendErr := h.svc.SendSMSForProject(r.Context(), projectID, body.To, body.Body)
			newStatus := "sent"
			if sendErr != nil {
				newStatus = "failed"
			}
			_ = h.svc.UpdateMessageStatus(r.Context(), msg.ID, newStatus)
		}()
	}

	writeJSON(w, http.StatusCreated, msg)
}

func (h *Handler) createPush(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	var body struct {
		Token       string  `json:"token"`
		Title       string  `json:"title"`
		Body        string  `json:"body"`
		Draft       bool    `json:"draft"`
		ScheduledAt *string `json:"scheduledAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Title == "" {
		apperr.BadRequest(w, "title is required")
		return
	}
	scheduledAt, err := parseScheduledAt(body.ScheduledAt)
	if err != nil {
		apperr.BadRequest(w, "scheduledAt must be an RFC 3339 timestamp")
		return
	}

	status, sendNow := sendDecision(body.Draft, scheduledAt)
	recipients := []string{}
	if body.Token != "" {
		recipients = []string{body.Token}
	}

	msg, err := h.svc.CreateMessage(r.Context(), projectID, "push", body.Title, body.Body, recipients, status, scheduledAt)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	if sendNow {
		go func() {
			sendErr := h.svc.SendPushForProject(r.Context(), projectID, body.Token, body.Title, body.Body)
			newStatus := "sent"
			if sendErr != nil {
				newStatus = "failed"
			}
			_ = h.svc.UpdateMessageStatus(r.Context(), msg.ID, newStatus)
		}()
	}

	writeJSON(w, http.StatusCreated, msg)
}

func (h *Handler) getMessage(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	msgID := chi.URLParam(r, "msgId")
	msg, err := h.svc.GetMessage(r.Context(), projectID, msgID)
	if err != nil {
		apperr.NotFound(w, "message")
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	msgID := chi.URLParam(r, "msgId")
	if err := h.svc.DeleteMessage(r.Context(), projectID, msgID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Legacy direct-send (no persistence)
// ---------------------------------------------------------------------------

func (h *Handler) sendEmailLegacy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if len(body.To) == 0 || body.Subject == "" {
		apperr.BadRequest(w, "to and subject are required")
		return
	}
	if err := h.svc.SendEmail(r.Context(), body.To, body.Subject, body.HTML); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) sendSMSLegacy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To   string `json:"to"`
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.To == "" || body.Body == "" {
		apperr.BadRequest(w, "to and body are required")
		return
	}
	if err := h.svc.SendSMS(r.Context(), body.To, body.Body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) sendPushLegacy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Token == "" || body.Title == "" {
		apperr.BadRequest(w, "token and title are required")
		return
	}
	if err := h.svc.SendPush(r.Context(), body.Token, body.Title, body.Body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------

func (h *Handler) listProviders(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	providers, err := h.svc.ListProviders(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
		"total":     len(providers),
	})
}

func (h *Handler) createProvider(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	var body struct {
		Name     string          `json:"name"`
		Type     string          `json:"type"`
		Provider string          `json:"provider"`
		Config   json.RawMessage `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Name == "" || body.Type == "" || body.Provider == "" {
		apperr.BadRequest(w, "name, type, and provider are required")
		return
	}
	p, err := h.svc.CreateProvider(r.Context(), projectID, body.Name, body.Type, body.Provider, body.Config)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) getProvider(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	providerID := chi.URLParam(r, "providerId")
	p, err := h.svc.GetProvider(r.Context(), projectID, providerID)
	if err != nil {
		apperr.NotFound(w, "provider")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) updateProvider(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	providerID := chi.URLParam(r, "providerId")
	var body struct {
		Name    string          `json:"name"`
		Config  json.RawMessage `json:"config"`
		Enabled bool            `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	p, err := h.svc.UpdateProvider(r.Context(), projectID, providerID, body.Name, body.Config, body.Enabled)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deleteProvider(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	providerID := chi.URLParam(r, "providerId")
	if err := h.svc.DeleteProvider(r.Context(), projectID, providerID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Topics
// ---------------------------------------------------------------------------

func (h *Handler) listTopics(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	topics, err := h.svc.ListTopics(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"topics": topics,
		"total":  len(topics),
	})
}

func (h *Handler) createTopic(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Name == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	topic, err := h.svc.CreateTopic(r.Context(), projectID, body.Name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, topic)
}

func (h *Handler) getTopic(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	topicID := chi.URLParam(r, "topicId")
	topic, err := h.svc.GetTopic(r.Context(), projectID, topicID)
	if err != nil {
		apperr.NotFound(w, "topic")
		return
	}
	writeJSON(w, http.StatusOK, topic)
}

func (h *Handler) addSubscriber(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	topicID := chi.URLParam(r, "topicId")
	var body struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Target == "" {
		apperr.BadRequest(w, "target is required")
		return
	}
	topic, err := h.svc.AddSubscriber(r.Context(), projectID, topicID, body.Target)
	if err != nil {
		apperr.NotFound(w, "topic")
		return
	}
	writeJSON(w, http.StatusOK, topic)
}

func (h *Handler) sendToTopic(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	topicID := chi.URLParam(r, "topicId")
	var body struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Body == "" {
		apperr.BadRequest(w, "body is required")
		return
	}
	if err := h.svc.SendToTopic(r.Context(), projectID, topicID, body.Subject, body.Body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseScheduledAt reads an optional RFC 3339 "scheduledAt" value. A missing or
// empty value returns (nil, nil); a malformed one returns an error so the
// handler can reject it rather than silently sending now.
func parseScheduledAt(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, err
	}
	u := t.UTC()
	return &u, nil
}

// sendDecision resolves the message status to store and whether to deliver
// inline. A draft is never sent; a scheduledAt in the future is queued for the
// per-minute sweep; anything else (absent or already-past scheduledAt) keeps
// the immediate-send behaviour.
func sendDecision(draft bool, scheduledAt *time.Time) (status string, sendNow bool) {
	switch {
	case draft:
		return "draft", false
	case scheduledAt != nil && scheduledAt.After(time.Now()):
		return "scheduled", false
	default:
		return "processing", true
	}
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}

// ---------------------------------------------------------------------------
// Template handlers
// ---------------------------------------------------------------------------

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	templates, total, err := h.svc.ListTemplates(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     total,
		"templates": templates,
	})
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	var body struct {
		TemplateID string   `json:"templateId"`
		Name       string   `json:"name"`
		Type       string   `json:"type"`
		Subject    string   `json:"subject"`
		Body       string   `json:"body"`
		Variables  []string `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Name == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Type == "" {
		body.Type = "email"
	}
	t, err := h.svc.CreateTemplate(r.Context(), projectID, body.TemplateID, body.Name, body.Type, body.Subject, body.Body, body.Variables)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) getTemplate(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	templateID := chi.URLParam(r, "templateId")
	t, err := h.svc.GetTemplate(r.Context(), templateID, projectID)
	if err != nil {
		apperr.NotFound(w, "template")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	templateID := chi.URLParam(r, "templateId")
	var body struct {
		Name      string   `json:"name"`
		Type      string   `json:"type"`
		Subject   string   `json:"subject"`
		Body      string   `json:"body"`
		Variables []string `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	t, err := h.svc.UpdateTemplate(r.Context(), templateID, projectID, body.Name, body.Type, body.Subject, body.Body, body.Variables)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	templateID := chi.URLParam(r, "templateId")
	if err := h.svc.DeleteTemplate(r.Context(), templateID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) sendTemplate(w http.ResponseWriter, r *http.Request) {
	projectID := mw.ProjectFromContext(r.Context())
	templateID := chi.URLParam(r, "templateId")
	var body struct {
		To        []string          `json:"to"`
		Variables map[string]string `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if len(body.To) == 0 {
		apperr.BadRequest(w, "to is required")
		return
	}
	if err := h.svc.SendTemplate(r.Context(), templateID, projectID, body.To, body.Variables); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
