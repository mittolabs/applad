package realtime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// newTestClient creates a Client with no WebSocket connection, suitable for unit tests.
// It only needs the send channel to receive broadcast messages.
func newTestClient(hub *Hub) *Client {
	return &Client{
		hub:       hub,
		conn:      nil,
		send:      make(chan []byte, 64),
		projectID: "proj1",
		userID:    "user1",
	}
}

func TestHub_PublishToSubscribers(t *testing.T) {
	hub := NewHub()
	client := newTestClient(hub)

	// Register client
	hub.register <- client
	// Give the hub goroutine time to process
	time.Sleep(10 * time.Millisecond)

	// Subscribe to a channel
	hub.Subscribe(client, "databases.docs")

	// Publish an event
	hub.Publish(Event{
		Type:      "databases.rows.create",
		Channel:   "databases.docs",
		Timestamp: "2026-04-06T00:00:00Z",
		Payload:   map[string]string{"id": "row1"},
	})

	// Wait for the event to arrive
	select {
	case msg := <-client.send:
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}
		if ev.Type != "databases.rows.create" {
			t.Errorf("expected type 'databases.rows.create', got '%s'", ev.Type)
		}
		if ev.Channel != "databases.docs" {
			t.Errorf("expected channel 'databases.docs', got '%s'", ev.Channel)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestHub_UnsubscribeStopsEvents(t *testing.T) {
	hub := NewHub()
	client := newTestClient(hub)

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Subscribe(client, "storage.files")
	hub.Unsubscribe(client, "storage.files")

	// Publish an event to the channel we just unsubscribed from
	hub.Publish(Event{
		Type:    "storage.files.create",
		Channel: "storage.files",
	})

	// Should NOT receive anything
	select {
	case msg := <-client.send:
		t.Fatalf("expected no event after unsubscribe, but got: %s", string(msg))
	case <-time.After(100 * time.Millisecond):
		// Good: no event received
	}
}

func TestHub_Stats(t *testing.T) {
	hub := NewHub()
	c1 := newTestClient(hub)
	c2 := newTestClient(hub)

	hub.register <- c1
	hub.register <- c2
	time.Sleep(10 * time.Millisecond)

	hub.Subscribe(c1, "ch1")
	hub.Subscribe(c2, "ch2")

	clients, channels := hub.Stats()
	if clients != 2 {
		t.Errorf("expected 2 clients, got %d", clients)
	}
	if channels != 2 {
		t.Errorf("expected 2 channels, got %d", channels)
	}
}

func TestPublishResourceEvent_FormatsCorrectly(t *testing.T) {
	hub := NewHub()
	client := newTestClient(hub)

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Subscribe to the channel that PublishResourceEvent will target
	channel := "projects.proj1.databases.rows"
	hub.Subscribe(client, channel)

	PublishResourceEvent(hub, "databases", "rows", "create", "proj1", "row123", map[string]string{"key": "val"})

	select {
	case msg := <-client.send:
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if ev.Type != "databases.rows.create" {
			t.Errorf("expected type 'databases.rows.create', got '%s'", ev.Type)
		}
		if ev.Channel != "projects.proj1.databases.rows" {
			t.Errorf("expected channel 'projects.proj1.databases.rows', got '%s'", ev.Channel)
		}
		if ev.Timestamp == "" {
			t.Error("expected non-empty timestamp")
		}
		// Verify timestamp is RFC3339
		if !strings.Contains(ev.Timestamp, "T") {
			t.Errorf("expected RFC3339 timestamp, got '%s'", ev.Timestamp)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPublishResourceEvent_NilPublisher(t *testing.T) {
	// Should not panic when publisher is nil
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PublishResourceEvent panicked with nil publisher: %v", r)
		}
	}()

	PublishResourceEvent(nil, "databases", "rows", "create", "proj1", "row1", nil)
}

func TestHub_MultipleChannels(t *testing.T) {
	hub := NewHub()
	client := newTestClient(hub)

	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Subscribe(client, "ch-a")
	hub.Subscribe(client, "ch-b")

	// Publish to ch-a only
	hub.Publish(Event{Type: "test.a", Channel: "ch-a"})

	select {
	case msg := <-client.send:
		var ev Event
		json.Unmarshal(msg, &ev)
		if ev.Type != "test.a" {
			t.Errorf("expected type 'test.a', got '%s'", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ch-a event")
	}

	// Publish to ch-b
	hub.Publish(Event{Type: "test.b", Channel: "ch-b"})

	select {
	case msg := <-client.send:
		var ev Event
		json.Unmarshal(msg, &ev)
		if ev.Type != "test.b" {
			t.Errorf("expected type 'test.b', got '%s'", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ch-b event")
	}
}
