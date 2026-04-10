// Package realtime implements Applad's WebSocket pub/sub service.
// Clients subscribe to channels; the hub broadcasts events from services.
// When a Redis address is provided, events are fanned through Redis pub/sub
// so multiple API instances share a single event stream — enabling horizontal
// scaling of the WebSocket layer.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const redisChannel = "applad:realtime"

// Event is a realtime event broadcast to subscribers.
type Event struct {
	Type      string      `json:"type"`      // e.g. "databases.rows.create"
	Channel   string      `json:"channel"`   // e.g. "databases.db1.tables.t1.rows"
	Timestamp string      `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Hub manages WebSocket client connections and event broadcasting.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	channels   map[string]map[*Client]bool // channel -> set of clients
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	rdb        *redis.Client // nil when Redis pub/sub is not configured
}

// NewHub creates a hub. Pass databaseDSN to enable PostgreSQL CDC and
// redisAddr (host:port) to enable cross-instance Redis pub/sub.
func NewHub(databaseDSN, redisAddr string) *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
		broadcast:  make(chan Event, 512),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	if strings.TrimSpace(redisAddr) != "" {
		h.rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
		go h.subscribeRedis()
	}

	go h.run()

	if strings.TrimSpace(databaseDSN) != "" {
		go h.listenPostgres(databaseDSN)
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
	databaseID, _ := message["database_id"].(string)
	tableName, _ := message["table"].(string)

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

	eventType := "databases.rows." + action

	// Publish to project-wide channel (broad subscription)
	h.Publish(Event{
		Type:      eventType,
		Channel:   "projects." + projectID + ".databases.rows",
		Timestamp: timestamp,
		Payload:   message,
	})

	// Publish to database-scoped channel
	if databaseID != "" {
		h.Publish(Event{
			Type:      eventType,
			Channel:   "databases." + projectID + "." + databaseID,
			Timestamp: timestamp,
			Payload:   message,
		})
	}

	// Publish to table-scoped channel (most specific — used by onInsert/onUpdate/onDelete)
	if databaseID != "" && tableName != "" {
		h.Publish(Event{
			Type:      eventType,
			Channel:   "databases." + projectID + "." + databaseID + "." + tableName,
			Timestamp: timestamp,
			Payload:   message,
		})
	}
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

// Publish sends an event to subscribers. When Redis is configured the event
// is published to the Redis channel so all API instances receive it; otherwise
// it is dispatched directly to this instance's local clients.
func (h *Hub) Publish(event Event) {
	if h.rdb != nil {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("realtime: marshal event: %v", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := h.rdb.Publish(ctx, redisChannel, string(data)).Err(); err != nil {
			log.Printf("realtime: redis publish: %v", err)
			// Fall back to local delivery on Redis failure
			h.broadcast <- event
		}
		return
	}
	h.broadcast <- event
}

// subscribeRedis receives events from the Redis channel and fans them out
// to local WebSocket clients. This is the delivery path in multi-instance mode.
func (h *Hub) subscribeRedis() {
	for {
		ctx := context.Background()
		sub := h.rdb.Subscribe(ctx, redisChannel)
		ch := sub.Channel()
		for msg := range ch {
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("realtime: redis unmarshal: %v", err)
				continue
			}
			h.broadcast <- event
		}
		sub.Close() //nolint:errcheck
		log.Printf("realtime: redis subscription closed, reconnecting in 2s")
		time.Sleep(2 * time.Second)
	}
}

// Stats returns the number of connected clients and active channels.
func (h *Hub) Stats() (clients int, channels int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients), len(h.channels)
}
