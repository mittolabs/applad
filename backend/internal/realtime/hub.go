// Package realtime implements Applad's WebSocket pub/sub service.
// Clients subscribe to channels; the hub broadcasts events from services.
package realtime

import (
	"encoding/json"
	"log"
	"sync"
)

// Event is a realtime event broadcast to subscribers.
type Event struct {
	Type      string      `json:"type"`      // e.g. "databases.documents.create"
	Channel   string      `json:"channel"`   // e.g. "databases.db1.collections.c1.documents"
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

// NewHub creates and starts a new realtime Hub.
func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go h.run()
	return h
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
