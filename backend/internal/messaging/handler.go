package messaging

import (
	"encoding/json"
	"net/http"
	"strconv"

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
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
		Draft   bool     `json:"draft"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Subject == "" {
		apperr.BadRequest(w, "subject is required")
		return
	}

	status := "processing"
	if body.Draft {
		status = "draft"
	}

	msg, err := h.svc.CreateMessage(r.Context(), projectID, "email", body.Subject, body.HTML, body.To, status)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	if !body.Draft {
		go func() {
			sendErr := h.svc.SendEmail(r.Context(), body.To, body.Subject, body.HTML)
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
		To    string `json:"to"`
		Body  string `json:"body"`
		Draft bool   `json:"draft"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Body == "" {
		apperr.BadRequest(w, "body is required")
		return
	}

	status := "processing"
	if body.Draft {
		status = "draft"
	}
	recipients := []string{}
	if body.To != "" {
		recipients = []string{body.To}
	}

	msg, err := h.svc.CreateMessage(r.Context(), projectID, "sms", "", body.Body, recipients, status)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	if !body.Draft {
		go func() {
			sendErr := h.svc.SendSMS(r.Context(), body.To, body.Body)
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
		Token string `json:"token"`
		Title string `json:"title"`
		Body  string `json:"body"`
		Draft bool   `json:"draft"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Title == "" {
		apperr.BadRequest(w, "title is required")
		return
	}

	status := "processing"
	if body.Draft {
		status = "draft"
	}
	recipients := []string{}
	if body.Token != "" {
		recipients = []string{body.Token}
	}

	msg, err := h.svc.CreateMessage(r.Context(), projectID, "push", body.Title, body.Body, recipients, status)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	if !body.Draft {
		go func() {
			sendErr := h.svc.SendPush(r.Context(), body.Token, body.Title, body.Body)
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
