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
	hub := NewHub("", "")
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
	hub := NewHub("", "")
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
	hub := NewHub("", "")
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
	hub := NewHub("", "")
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
	hub := NewHub("", "")
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

// TestHub_DatabaseNotificationRoutesByDatabaseID feeds a realistic pg_notify
// payload — the exact shape applad_notify_change() emits (project_id,
// database_id, schema, table, action, new, timestamp) — and asserts the change
// fans out to all three derived channels: project-wide, database-scoped, and
// table-scoped. The database-scoped and table-scoped channels both depend on
// database_id being present in the payload; migration 032 regressed exactly
// that field, so this test guards the full NOTIFY -> hub routing path.
func TestHub_DatabaseNotificationRoutesByDatabaseID(t *testing.T) {
	hub := NewHub("", "")

	projectWide := newTestClient(hub)
	dbScoped := newTestClient(hub)
	tableScoped := newTestClient(hub)
	for _, c := range []*Client{projectWide, dbScoped, tableScoped} {
		hub.register <- c
	}
	time.Sleep(10 * time.Millisecond)

	hub.Subscribe(projectWide, "projects.proj1.databases.rows")
	hub.Subscribe(dbScoped, "databases.proj1.db1")
	hub.Subscribe(tableScoped, "databases.proj1.db1.posts")

	// The literal payload PostgreSQL sends for an INSERT into a project table.
	payload := `{"project_id":"proj1","database_id":"db1","schema":"p_proj1_db1",` +
		`"table":"posts","action":"insert","old":null,` +
		`"new":{"id":"row1","title":"Hello","_permissions":{"read":["any"]}},` +
		`"timestamp":"2026-07-30T12:00:00Z"}`
	hub.publishDatabaseNotification(payload)

	// receive reads one event off a client's send channel or fails.
	receive := func(name string, c *Client) Event {
		select {
		case msg := <-c.send:
			var ev Event
			if err := json.Unmarshal(msg, &ev); err != nil {
				t.Fatalf("%s: unmarshal: %v", name, err)
			}
			return ev
		case <-time.After(time.Second):
			t.Fatalf("%s: timed out waiting for change event", name)
			return Event{}
		}
	}

	// assertRow checks the event carries the create type, the expected channel,
	// the propagated timestamp, and the new row body (project_id/database_id are
	// what routed it here in the first place).
	assertRow := func(name string, ev Event, wantChannel string) {
		if ev.Type != "databases.rows.create" {
			t.Errorf("%s: expected type 'databases.rows.create', got '%s'", name, ev.Type)
		}
		if ev.Channel != wantChannel {
			t.Errorf("%s: expected channel '%s', got '%s'", name, wantChannel, ev.Channel)
		}
		if ev.Timestamp != "2026-07-30T12:00:00Z" {
			t.Errorf("%s: expected payload timestamp propagated, got '%s'", name, ev.Timestamp)
		}
		msg, ok := ev.Payload.(map[string]interface{})
		if !ok {
			t.Fatalf("%s: payload is not an object: %T", name, ev.Payload)
		}
		if msg["database_id"] != "db1" {
			t.Errorf("%s: expected database_id 'db1' in payload, got %v", name, msg["database_id"])
		}
		row, _ := msg["new"].(map[string]interface{})
		if row == nil || row["title"] != "Hello" {
			t.Errorf("%s: expected new.title 'Hello', got %v", name, msg["new"])
		}
	}

	assertRow("project-wide", receive("project-wide", projectWide), "projects.proj1.databases.rows")
	assertRow("database-scoped", receive("database-scoped", dbScoped), "databases.proj1.db1")
	assertRow("table-scoped", receive("table-scoped", tableScoped), "databases.proj1.db1.posts")
}

// TestHub_DatabaseNotificationWithoutDatabaseID confirms that when database_id
// is absent (the migration-032 failure mode), only the project-wide channel
// fires — the database- and table-scoped subscribers get nothing — proving the
// scoped channels are genuinely gated on database_id rather than always firing.
func TestHub_DatabaseNotificationWithoutDatabaseID(t *testing.T) {
	hub := NewHub("", "")

	projectWide := newTestClient(hub)
	dbScoped := newTestClient(hub)
	hub.register <- projectWide
	hub.register <- dbScoped
	time.Sleep(10 * time.Millisecond)

	hub.Subscribe(projectWide, "projects.proj1.databases.rows")
	hub.Subscribe(dbScoped, "databases.proj1.db1")

	hub.publishDatabaseNotification(`{"project_id":"proj1","table":"posts","action":"insert","new":{"id":"row1"}}`)

	select {
	case <-projectWide.send:
		// Good: the project-wide channel always fires on a project row change.
	case <-time.After(time.Second):
		t.Fatal("project-wide subscriber timed out")
	}

	select {
	case msg := <-dbScoped.send:
		t.Fatalf("database-scoped subscriber should get nothing without database_id, got: %s", string(msg))
	case <-time.After(100 * time.Millisecond):
		// Good: no database_id means no database-scoped delivery.
	}
}

func TestHub_DeployLogNotificationRoutes(t *testing.T) {
	hub := NewHub("", "")
	client := newTestClient(hub)
	hub.register <- client
	time.Sleep(10 * time.Millisecond)
	hub.Subscribe(client, "deploy.rel123")

	// A worker's pg_notify payload for a build log line.
	hub.publishDatabaseNotification(`{"kind":"deploy_log","release_id":"rel123","seq":1,"line":"Step 1/5"}`)

	select {
	case msg := <-client.send:
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if ev.Type != "deploy.log" || ev.Channel != "deploy.rel123" {
			t.Fatalf("expected deploy.log on deploy.rel123, got %s on %s", ev.Type, ev.Channel)
		}
		payload, _ := ev.Payload.(map[string]interface{})
		if payload["line"] != "Step 1/5" {
			t.Fatalf("expected line 'Step 1/5', got %v", payload["line"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for deploy log event")
	}
}
