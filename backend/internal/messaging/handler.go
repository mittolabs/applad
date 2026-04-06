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
