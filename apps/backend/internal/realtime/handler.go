package realtime

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/mittolabs/applad/internal/middleware"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS is handled at the middleware level
	},
}

// Handler handles WebSocket connections.
type Handler struct {
	hub *Hub
}

// NewHandler creates a new realtime Handler.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// Routes returns the realtime router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.handleWebSocket)
	r.Get("/stats", h.stats)
	return r
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	clients, channels := h.hub.Stats()
	channelList := h.hub.ChannelStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connections": clients,
		"channels":    channels,
		"channelList": channelList,
	})
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("realtime: upgrade error: %v", err)
		return
	}

	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	isAPIKey := middleware.IsAPIKey(ctx)
	// A console administrator is validated for this project's org by the
	// Authenticate middleware but carries no end-user id; treat it as an
	// authenticated, broad-access connection.
	isConsoleAdmin := middleware.IsConsoleAdmin(ctx)

	authenticated := userID != "" || isConsoleAdmin
	broadAccess := isAPIKey || isConsoleAdmin

	client := NewClient(h.hub, conn, projectID, userID, authenticated, broadAccess)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
