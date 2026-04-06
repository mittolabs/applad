package messaging

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
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
	r.Post("/email", h.sendEmail)
	r.Post("/sms", h.sendSMS)
	r.Post("/push", h.sendPush)
	r.Post("/topics", h.createTopic)
	r.Post("/topics/{topicId}/subscribers", h.addSubscriber)
	r.Post("/topics/{topicId}/messages", h.sendToTopic)
	return r
}

func (h *Handler) sendEmail(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) sendSMS(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) sendPush(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) createTopic(w http.ResponseWriter, r *http.Request) {
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
	topic := h.svc.CreateTopic(body.Name)
	writeJSON(w, http.StatusCreated, topic)
}

func (h *Handler) addSubscriber(w http.ResponseWriter, r *http.Request) {
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
	topic, err := h.svc.AddSubscriber(topicID, body.Target)
	if err != nil {
		apperr.NotFound(w, "topic")
		return
	}
	writeJSON(w, http.StatusOK, topic)
}

func (h *Handler) sendToTopic(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.SendToTopic(r.Context(), topicID, body.Subject, body.Body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
