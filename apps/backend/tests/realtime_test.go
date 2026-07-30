//go:build integration

package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsURL converts the REST baseURL (http://host/v1) into the realtime WebSocket
// URL (ws://host/v1/realtime), carrying the project as a query param so the
// ProjectContext middleware scopes the connection even before headers are read.
func wsURL(projectID string) string {
	u := strings.Replace(baseURL, "http://", "ws://", 1)
	u = strings.Replace(u, "https://", "wss://", 1)
	return fmt.Sprintf("%s/realtime?project=%s", u, projectID)
}

// TestRealtimeRowChangeDelivered is the full NOTIFY delivery path end to end:
// a row written over the REST API fires the applad_notify_change() Postgres
// trigger, pg_notify reaches the API server's realtime hub LISTEN loop, and the
// change is delivered over a real WebSocket to a subscribed client. This is the
// exact path migration 032 fixed (a dropped database_id), and nothing exercised
// it at any level before.
//
// The connection authenticates with the project API key (headers on the
// handshake), which the realtime handler treats as an authenticated,
// broad-access client — so the database-scoped and table-scoped subscriptions
// are authorized without per-row filtering.
func TestRealtimeRowChangeDelivered(t *testing.T) {
	projectID, apiKey := projectWithKey(t, "realtime-test")
	h := authHeader(projectID, apiKey)

	// Schema: database -> table "events" (title column) with open permissions.
	status, body := request(t, "POST", "/databases", map[string]interface{}{"name": "rtdb"}, h)
	if status != 201 {
		t.Fatalf("create db: expected 201, got %d: %v", status, body)
	}
	dbID := body["$id"].(string)

	status, body = request(t, "POST", fmt.Sprintf("/databases/%s/tables", dbID),
		map[string]interface{}{"name": "events"}, h)
	if status != 201 {
		t.Fatalf("create table: expected 201, got %d: %v", status, body)
	}
	tableID := body["$id"].(string)

	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/columns/string", dbID, tableID),
		map[string]interface{}{"key": "title", "size": 256, "required": true}, h)
	if status != 201 {
		t.Fatalf("create column: expected 201, got %d: %v", status, body)
	}

	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/permissions", dbID, tableID),
		map[string]interface{}{"permissions": []map[string]string{
			{"role": "any", "action": "create"},
			{"role": "any", "action": "read"},
		}}, h)
	if status != 200 {
		t.Fatalf("set permissions: expected 200, got %d: %v", status, body)
	}

	// Open the WebSocket, authenticating with the project API key.
	header := http.Header{}
	header.Set("X-Applad-Project", projectID)
	header.Set("X-Applad-Key", apiKey)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(projectID), header)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		t.Fatalf("dial realtime WS: %v (http %d)", err, code)
	}
	defer conn.Close()

	// Subscribe to both the table-scoped and database-scoped channels. "events"
	// is the physical table name the trigger reports as TG_TABLE_NAME.
	tableChannel := fmt.Sprintf("databases.%s.%s.events", projectID, dbID)
	dbChannel := fmt.Sprintf("databases.%s.%s", projectID, dbID)
	sub, _ := json.Marshal(map[string]interface{}{
		"type":     "subscribe",
		"channels": []string{tableChannel, dbChannel},
	})
	if err := conn.WriteMessage(websocket.TextMessage, sub); err != nil {
		t.Fatalf("send subscribe: %v", err)
	}

	// Give the server a moment to register the subscription before the write, so
	// the NOTIFY cannot race ahead of it.
	time.Sleep(300 * time.Millisecond)

	// Write a row — this is what fires the trigger.
	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/rows", dbID, tableID),
		map[string]interface{}{"data": map[string]interface{}{"title": "realtime-hello"}}, h)
	if status != 201 {
		t.Fatalf("create row: expected 201, got %d: %v", status, body)
	}

	// Read change events until we see our row create (or time out). Ignore any
	// unrelated frames; fail fast on an error frame (a rejected subscription).
	conn.SetReadDeadline(time.Now().Add(8 * time.Second))
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("did not receive row change over WS within deadline: %v", err)
		}
		var ev struct {
			Type    string                 `json:"type"`
			Channel string                 `json:"channel"`
			Payload map[string]interface{} `json:"payload"`
		}
		if err := json.Unmarshal(msg, &ev); err != nil {
			continue
		}
		if ev.Type == "error" {
			t.Fatalf("subscription rejected: %v", ev.Payload)
		}
		if ev.Type != "databases.rows.create" {
			continue
		}
		// Assert shape: channel is one we subscribed to, action created, and the
		// new row carries our value.
		if ev.Channel != tableChannel && ev.Channel != dbChannel {
			continue
		}
		if ev.Payload["database_id"] != dbID {
			t.Fatalf("expected database_id %s in payload, got %v", dbID, ev.Payload["database_id"])
		}
		row, ok := ev.Payload["new"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected new row object in payload, got %v", ev.Payload["new"])
		}
		if row["title"] != "realtime-hello" {
			t.Fatalf("expected new.title 'realtime-hello', got %v", row["title"])
		}
		return // success
	}
}
