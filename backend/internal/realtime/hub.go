// Package realtime implements Applad's WebSocket pub/sub service.
// Clients subscribe to channels; the hub broadcasts events from services.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Event is a realtime event broadcast to subscribers.
type Event struct {
	Type      string      `json:"type"`      // e.g. "databases.rows.create"
	Channel   string      `json:"channel"`   // e.g. "databases.db1.tables.t1.rows"
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Hub manages WebSocket client connections and event broadcasting.
type Hub struct {
	mu          sync.RWMutex
	clients     map[*Client]bool
	channels    map[string]map[*Client]bool // channel -> set of clients
	broadcast   chan Event
	register    chan *Client
	unregister  chan *Client
}

// NewHub creates a hub and optionally starts a PostgreSQL LISTEN loop.
func NewHub(databaseDSN ...string) *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go h.run()
	if len(databaseDSN) > 0 && strings.TrimSpace(databaseDSN[0]) != "" {
		go h.listenPostgres(databaseDSN[0])
	}
	return h
}

func (h *Hub) listenPostgres(databaseDSN string) {
	for {
		if err := h.listenLoop(databaseDSN); err != nil {
			log.Printf("realtime: postgres listener stopped: %v", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func (h *Hub) listenLoop(databaseDSN string) error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseDSN)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "LISTEN applad_changes"); err != nil {
		return err
	}

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		h.publishDatabaseNotification(notification.Payload)
	}
}

func (h *Hub) publishDatabaseNotification(payload string) {
	var message map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &message); err != nil {
		log.Printf("realtime: invalid notification payload: %v", err)
		return
	}
	projectID, _ := message["project_id"].(string)
	if projectID == "" {
		return
	}
	action, _ := message["action"].(string)
	action = strings.ToLower(action)
	switch action {
	case "insert":
		action = "create"
	case "update":
		action = "update"
	case "delete":
		action = "delete"
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	if value, ok := message["timestamp"].(string); ok && value != "" {
		timestamp = value
	}
	h.Publish(Event{
		Type:      "databases.rows." + action,
		Channel:   "projects." + projectID + ".databases.rows",
		Timestamp: timestamp,
		Payload:   message,
	})
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// Remove from all channels
				for ch, subs := range h.channels {
					delete(subs, client)
					if len(subs) == 0 {
						delete(h.channels, ch)
					}
				}
				close(client.send)
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			h.mu.RLock()
			subs := h.channels[event.Channel]
			data, err := json.Marshal(event)
			if err != nil {
				h.mu.RUnlock()
				log.Printf("realtime: marshal error: %v", err)
				continue
			}
			for client := range subs {
				select {
				case client.send <- data:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// Subscribe adds a client to a channel.
func (h *Hub) Subscribe(c *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels[channel] == nil {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][c] = true
}

// Unsubscribe removes a client from a channel.
func (h *Hub) Unsubscribe(c *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.channels[channel]; ok {
		delete(subs, c)
		if len(subs) == 0 {
			delete(h.channels, channel)
		}
	}
}

// Publish sends an event to all subscribers of the given channel.
func (h *Hub) Publish(event Event) {
	h.broadcast <- event
}

// Stats returns the number of connected clients and active channels.
func (h *Hub) Stats() (clients int, channels int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients), len(h.channels)
}
