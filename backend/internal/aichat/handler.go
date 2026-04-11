package aichat

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/console"
)

// Handler handles AI chat HTTP requests.
type Handler struct {
	svc        *Service
	consoleSvc *console.Service
}

// NewHandler creates a new AI chat Handler.
func NewHandler(svc *Service, consoleSvc *console.Service) *Handler {
	return &Handler{svc: svc, consoleSvc: consoleSvc}
}

// Routes returns the AI chat router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/chat", h.chat)
	return r
}

func (h *Handler) chat(w http.ResponseWriter, r *http.Request) {
	// Validate console JWT
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		apperr.Unauthorized(w)
		return
	}
	if _, err := h.consoleSvc.ValidateToken(token); err != nil {
		apperr.Unauthorized(w)
		return
	}

	var body struct {
		Messages []Message `json:"messages"`
		Context  string    `json:"context"` // page context hint
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) == 0 {
		apperr.BadRequest(w, "messages required")
		return
	}

	reply, err := h.svc.Chat(r.Context(), body.Messages, body.Context)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": reply})
}
