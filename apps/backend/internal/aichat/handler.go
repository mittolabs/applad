package aichat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/console"
)

// Handler handles AI chat HTTP requests.
type Handler struct {
	svc        *Service
	consoleSvc *console.Service
	executor   *ToolExecutor
}

// NewHandler creates a new AI chat Handler.
// port is the API server port used by the ToolExecutor to call internal endpoints.
func NewHandler(svc *Service, consoleSvc *console.Service, port string) *Handler {
	return &Handler{
		svc:        svc,
		consoleSvc: consoleSvc,
		executor:   NewToolExecutor(port, &http.Client{Timeout: 30 * time.Second}),
	}
}

// Routes returns the AI chat router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/config", h.getConfig)
	r.Post("/chat", h.chat)     // non-streaming (legacy)
	r.Post("/stream", h.stream) // SSE streaming
	return r
}

// getConfig returns provider info so the frontend can display the model name.
func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"model":      h.svc.ModelName(),
		"configured": h.svc.IsConfigured(),
	})
}

// chat is the legacy non-streaming endpoint (kept for backward compatibility).
func (h *Handler) chat(w http.ResponseWriter, r *http.Request) {
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
		Context  string    `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) == 0 {
		apperr.BadRequest(w, "messages required")
		return
	}

	reply, err := h.svc.Chat(r.Context(), body.Messages, body.Context, token, h.executor)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": reply})
}

// stream handles SSE streaming chat.
// Response: text/event-stream
//
//	data: {"delta":"token"}\n\n  — each chunk
//	data: [DONE]\n\n             — end of stream
//	data: {"error":"..."}\n\n   — on error
func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
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
		Context  string    `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) == 0 {
		apperr.BadRequest(w, "messages required")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	send := func(data string) {
		fmt.Fprintf(w, "data: %s\n\n", data)
		if canFlush {
			flusher.Flush()
		}
	}

	out := make(chan string, 64)
	errCh := make(chan error, 1)

	go func() {
		errCh <- h.svc.StreamChat(r.Context(), body.Messages, body.Context, token, h.executor, out)
	}()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	done := false
	for !done {
		select {
		case delta, ok := <-out:
			if !ok {
				done = true
				break
			}
			b, _ := json.Marshal(map[string]string{"delta": delta})
			send(string(b))
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			if canFlush {
				flusher.Flush()
			}
		case <-r.Context().Done():
			done = true
		}
	}

	if err := <-errCh; err != nil {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		send(string(b))
	}

	send("[DONE]")
}
