package realtime

import (
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
	return r
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("realtime: upgrade error: %v", err)
		return
	}

	projectID := middleware.ProjectFromContext(r.Context())
	userID := middleware.UserFromContext(r.Context())

	client := NewClient(h.hub, conn, projectID, userID)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
