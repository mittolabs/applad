package realtime

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client represents a single WebSocket connection.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	projectID string
	userID    string
	// authenticated is true when the connection presented a valid credential
	// (a user session, a server API key, or a console administrator). A bare
	// project header does not make a connection authenticated.
	authenticated bool
	// broadAccess is true for connections that hold project-wide read access
	// (server API keys, console administrators). Such connections are still
	// project-scoped but skip per-row read filtering.
	broadAccess bool
	// subFilters holds the per-channel row filter for subscriptions that must
	// be filtered per event. A channel absent here (nil filter) is delivered
	// unconditionally. Written under Hub.mu; read on the broadcast path under
	// Hub.mu's read lock.
	subFilters map[string]*rowFilter
}

// NewClient creates a new client. authenticated reports whether the connection
// carried a valid credential; broadAccess reports whether it holds project-wide
// read access (API key or console admin) and so bypasses per-row filtering.
func NewClient(hub *Hub, conn *websocket.Conn, projectID, userID string, authenticated, broadAccess bool) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 64),
		projectID:     projectID,
		userID:        userID,
		authenticated: authenticated,
		broadAccess:   broadAccess,
	}
}

// allows reports whether an event may be delivered to this client on its
// channel. Called on the broadcast path with Hub.mu held for reading.
func (c *Client) allows(ev Event) bool {
	f := c.subFilters[ev.Channel]
	if f == nil {
		return true
	}
	return f.allows(ev)
}

// sendError delivers a non-fatal error frame to the client (e.g. a rejected
// subscription). Non-blocking: a full or closed send buffer drops it.
func (c *Client) sendError(channel, code, message string) {
	ev := Event{
		Type:      "error",
		Channel:   channel,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   map[string]string{"code": code, "message": message},
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

// clientMessage is the message format clients send to subscribe/unsubscribe.
type clientMessage struct {
	Type     string   `json:"type"` // "subscribe" or "unsubscribe"
	Channels []string `json:"channels"`
}

// ReadPump reads messages from the WebSocket connection.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("realtime: read error: %v", err)
			}
			break
		}

		var msg clientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "subscribe":
			for _, ch := range msg.Channels {
				c.hub.subscribeClient(c, ch)
			}
		case "unsubscribe":
			for _, ch := range msg.Channels {
				c.hub.Unsubscribe(c, ch)
			}
		}
	}
}

// WritePump writes messages to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
