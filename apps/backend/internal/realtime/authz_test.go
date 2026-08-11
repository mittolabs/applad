package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// authedClient builds a registered client with an explicit auth posture.
func authedClient(hub *Hub, projectID, userID string, authenticated, broad bool) *Client {
	c := &Client{
		hub:           hub,
		send:          make(chan []byte, 64),
		projectID:     projectID,
		userID:        userID,
		authenticated: authenticated,
		broadAccess:   broad,
	}
	hub.register <- c
	time.Sleep(10 * time.Millisecond)
	return c
}

// subscribed reports whether c is registered on channel in the hub.
func (h *Hub) subscribed(c *Client, channel string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.channels[channel][c]
}

// drainError returns the first frame on the client's send buffer, or "" if none
// arrives promptly.
func drainError(c *Client) *Event {
	select {
	case msg := <-c.send:
		var ev Event
		if json.Unmarshal(msg, &ev) == nil {
			return &ev
		}
	case <-time.After(100 * time.Millisecond):
	}
	return nil
}

// stubReadAuth is a configurable ReadAuthorizer.
type stubReadAuth struct {
	fn func(projectID, databaseID, tableName, userID string) (TableReadDecision, error)
}

func (s stubReadAuth) AuthorizeTableRead(_ context.Context, projectID, databaseID, tableName, userID string) (TableReadDecision, error) {
	return s.fn(projectID, databaseID, tableName, userID)
}

// stubReleaseVerifier ties a release to a project only when ownerProject matches.
type stubReleaseVerifier struct{ ownerProject string }

func (s stubReleaseVerifier) ReleaseBelongsToProject(_ context.Context, _ /*releaseID*/, projectID string) (bool, error) {
	return projectID == s.ownerProject, nil
}

// --- Project scoping ---------------------------------------------------------

func TestSubscribe_RejectsCrossProjectChannel(t *testing.T) {
	hub := NewHub("", "")
	// An authenticated client of proj1 tries to read proj2's rows.
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "databases.proj2.db1.posts")

	if hub.subscribed(c, "databases.proj2.db1.posts") {
		t.Fatal("cross-project subscription was accepted; expected rejection")
	}
	ev := drainError(c)
	if ev == nil || ev.Type != "error" {
		t.Fatalf("expected an error frame, got %+v", ev)
	}
	if code, _ := ev.Payload.(map[string]interface{})["code"].(string); code != "realtime_forbidden_project" {
		t.Fatalf("expected realtime_forbidden_project, got %v", ev.Payload)
	}
}

func TestSubscribe_RejectsCrossProjectResourceChannel(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "projects.proj2.storage.files")

	if hub.subscribed(c, "projects.proj2.storage.files") {
		t.Fatal("cross-project resource subscription was accepted; expected rejection")
	}
}

// --- Authentication required -------------------------------------------------

func TestSubscribe_RejectsUnauthenticatedDataChannel(t *testing.T) {
	hub := NewHub("", "")
	// Same project, but no credential — only the (non-secret) project header.
	c := authedClient(hub, "proj1", "", false, false)

	hub.subscribeClient(c, "databases.proj1.db1.posts")

	if hub.subscribed(c, "databases.proj1.db1.posts") {
		t.Fatal("unauthenticated data-channel subscription was accepted; expected rejection")
	}
	ev := drainError(c)
	if ev == nil {
		t.Fatal("expected an error frame for unauthenticated subscription")
	}
	if code, _ := ev.Payload.(map[string]interface{})["code"].(string); code != "realtime_unauthenticated" {
		t.Fatalf("expected realtime_unauthenticated, got %v", ev.Payload)
	}
}

func TestSubscribe_RejectsUnauthenticatedResourceChannel(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "", false, false)

	hub.subscribeClient(c, "projects.proj1.storage.files")

	if hub.subscribed(c, "projects.proj1.storage.files") {
		t.Fatal("unauthenticated resource subscription was accepted; expected rejection")
	}
}

// --- Legitimate access preserved --------------------------------------------

func TestSubscribe_AllowsSameProjectAuthenticatedResourceChannel(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "user1", true, false)
	channel := "projects.proj1.storage.files"

	hub.subscribeClient(c, channel)

	if !hub.subscribed(c, channel) {
		t.Fatal("legitimate same-project authenticated subscription was rejected")
	}

	hub.Publish(Event{Type: "storage.files.create", Channel: channel, Payload: map[string]string{"id": "f1"}})

	select {
	case msg := <-c.send:
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Fatalf("bad event: %v", err)
		}
		if ev.Channel != channel {
			t.Fatalf("expected channel %s, got %s", channel, ev.Channel)
		}
	case <-time.After(time.Second):
		t.Fatal("legitimate subscriber did not receive its event")
	}
}

func TestSubscribe_AllowsTableReadWithTableGrant(t *testing.T) {
	hub := NewHub("", "")
	hub.SetReadAuthorizer(stubReadAuth{fn: func(_, _, _, _ string) (TableReadDecision, error) {
		return TableReadDecision{AllowAll: true}, nil
	}})
	c := authedClient(hub, "proj1", "user1", true, false)
	channel := "databases.proj1.db1.posts"

	hub.subscribeClient(c, channel)
	if !hub.subscribed(c, channel) {
		t.Fatal("table-read-granted subscription was rejected")
	}
	// No filter attached — every row delivered.
	hub.Publish(Event{Type: "databases.rows.create", Channel: channel, Payload: map[string]interface{}{
		"new": map[string]interface{}{"id": "r1"},
	}})
	if drainError(c) == nil {
		t.Fatal("AllowAll subscriber did not receive its event")
	}
}

// --- Per-row read filtering --------------------------------------------------

func TestSubscribe_RowFilteringDeliversOnlyReadableRows(t *testing.T) {
	hub := NewHub("", "")
	// The stub grants no table-level read but row security is on, resolving the
	// caller's roles to any/users/user:user1 (no team membership).
	hub.SetReadAuthorizer(stubReadAuth{fn: func(_, _, _, userID string) (TableReadDecision, error) {
		return TableReadDecision{RowFiltered: true, Roles: []string{"any", "users", "user:" + userID}}, nil
	}})
	c := authedClient(hub, "proj1", "user1", true, false)
	channel := "databases.proj1.db1.posts"

	hub.subscribeClient(c, channel)
	if !hub.subscribed(c, channel) {
		t.Fatal("row-filtered subscription was rejected")
	}

	// A row readable only by team:secret — user1 is not a member, must NOT see it.
	hub.Publish(Event{Type: "databases.rows.create", Channel: channel, Payload: map[string]interface{}{
		"new": map[string]interface{}{
			"id":           "secret-row",
			"_permissions": map[string]interface{}{"read": []interface{}{"team:secret"}},
		},
	}})
	// A row readable by user1 — must be delivered.
	hub.Publish(Event{Type: "databases.rows.create", Channel: channel, Payload: map[string]interface{}{
		"new": map[string]interface{}{
			"id":           "mine",
			"_permissions": map[string]interface{}{"read": []interface{}{"user:user1"}},
		},
	}})

	got := map[string]bool{}
	deadline := time.After(500 * time.Millisecond)
	for len(got) < 1 {
		select {
		case msg := <-c.send:
			var ev Event
			if err := json.Unmarshal(msg, &ev); err != nil {
				t.Fatalf("bad event: %v", err)
			}
			row, _ := ev.Payload.(map[string]interface{})["new"].(map[string]interface{})
			got[row["id"].(string)] = true
		case <-deadline:
			t.Fatal("timed out before the readable row arrived")
		}
	}

	if got["secret-row"] {
		t.Fatal("subscriber received a row it may not read")
	}
	if !got["mine"] {
		t.Fatal("subscriber did not receive a row it may read")
	}
}

func TestSubscribe_RejectsTableWithNoReadAccess(t *testing.T) {
	hub := NewHub("", "")
	// Neither table-level nor row security grants access.
	hub.SetReadAuthorizer(stubReadAuth{fn: func(_, _, _, _ string) (TableReadDecision, error) {
		return TableReadDecision{}, nil
	}})
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "databases.proj1.db1.posts")

	if hub.subscribed(c, "databases.proj1.db1.posts") {
		t.Fatal("subscription to an unreadable table was accepted")
	}
}

func TestSubscribe_BroadAccessBypassesRowFiltering(t *testing.T) {
	hub := NewHub("", "")
	called := false
	hub.SetReadAuthorizer(stubReadAuth{fn: func(_, _, _, _ string) (TableReadDecision, error) {
		called = true
		return TableReadDecision{}, nil
	}})
	// A server API key: authenticated + broad access.
	c := authedClient(hub, "proj1", "api:key1", true, true)
	channel := "databases.proj1.db1.posts"

	hub.subscribeClient(c, channel)
	if !hub.subscribed(c, channel) {
		t.Fatal("broad-access subscription was rejected")
	}
	if called {
		t.Fatal("read authorizer was consulted for a broad-access connection")
	}
	// Delivered unfiltered, even for a row with restrictive permissions.
	hub.Publish(Event{Type: "databases.rows.create", Channel: channel, Payload: map[string]interface{}{
		"new": map[string]interface{}{"id": "r1", "_permissions": map[string]interface{}{"read": []interface{}{"team:x"}}},
	}})
	if drainError(c) == nil {
		t.Fatal("broad-access subscriber did not receive an event")
	}
}

// --- Aggregate database feeds require broad access ---------------------------

// A plain end-user session must NOT be able to subscribe to the table-less
// aggregate database channel, which would leak every other user's row changes
// bypassing table grants and row security.
func TestSubscribe_RejectsUserSessionOnAggregateDatabaseChannel(t *testing.T) {
	hub := NewHub("", "")
	hub.SetReadAuthorizer(stubReadAuth{fn: func(_, _, _, _ string) (TableReadDecision, error) {
		t.Fatal("read authorizer must not be consulted for the aggregate channel")
		return TableReadDecision{}, nil
	}})
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "databases.proj1.db1")

	if hub.subscribed(c, "databases.proj1.db1") {
		t.Fatal("user session subscribed to the aggregate database feed; expected rejection")
	}
	ev := drainError(c)
	if ev == nil || ev.Type != "error" {
		t.Fatalf("expected an error frame, got %+v", ev)
	}
	if code, _ := ev.Payload.(map[string]interface{})["code"].(string); code != "realtime_forbidden_read" {
		t.Fatalf("expected realtime_forbidden_read, got %v", ev.Payload)
	}
}

// A broad-access connection (server API key) retains access to the aggregate
// database feed.
func TestSubscribe_AllowsBroadAccessOnAggregateDatabaseChannel(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "api:key1", true, true)
	channel := "databases.proj1.db1"

	hub.subscribeClient(c, channel)
	if !hub.subscribed(c, channel) {
		t.Fatal("broad-access subscription to the aggregate database feed was rejected")
	}
	hub.Publish(Event{Type: "databases.rows.create", Channel: channel, Payload: map[string]interface{}{
		"new": map[string]interface{}{"id": "r1"},
	}})
	if drainError(c) == nil {
		t.Fatal("broad-access subscriber did not receive its event")
	}
}

// The same user session that is denied the aggregate feed must still reach a
// specific table channel, delivered per-row filtered.
func TestSubscribe_UserSessionStillReachesTableChannel(t *testing.T) {
	hub := NewHub("", "")
	hub.SetReadAuthorizer(stubReadAuth{fn: func(_, _, _, userID string) (TableReadDecision, error) {
		return TableReadDecision{RowFiltered: true, Roles: []string{"user:" + userID}}, nil
	}})
	c := authedClient(hub, "proj1", "user1", true, false)
	channel := "databases.proj1.db1.posts"

	hub.subscribeClient(c, channel)
	if !hub.subscribed(c, channel) {
		t.Fatal("user session was rejected from a specific table channel")
	}
	// A row readable only by user1 is delivered; a foreign row is not.
	hub.Publish(Event{Type: "databases.rows.create", Channel: channel, Payload: map[string]interface{}{
		"new": map[string]interface{}{"id": "other", "_permissions": map[string]interface{}{"read": []interface{}{"user:someoneelse"}}},
	}})
	hub.Publish(Event{Type: "databases.rows.create", Channel: channel, Payload: map[string]interface{}{
		"new": map[string]interface{}{"id": "mine", "_permissions": map[string]interface{}{"read": []interface{}{"user:user1"}}},
	}})

	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case msg := <-c.send:
			var ev Event
			if err := json.Unmarshal(msg, &ev); err != nil {
				t.Fatalf("bad event: %v", err)
			}
			row, _ := ev.Payload.(map[string]interface{})["new"].(map[string]interface{})
			if row["id"] == "other" {
				t.Fatal("table subscriber received a row it may not read")
			}
			if row["id"] == "mine" {
				return // got the readable row, filtering works
			}
		case <-deadline:
			t.Fatal("timed out before the readable row arrived")
		}
	}
}

// A plain end-user session must NOT subscribe to the project-wide row feed.
func TestSubscribe_RejectsUserSessionOnProjectWideRowFeed(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "projects.proj1.databases.rows")

	if hub.subscribed(c, "projects.proj1.databases.rows") {
		t.Fatal("user session subscribed to the project-wide row feed; expected rejection")
	}
	ev := drainError(c)
	if ev == nil || ev.Type != "error" {
		t.Fatalf("expected an error frame, got %+v", ev)
	}
	if code, _ := ev.Payload.(map[string]interface{})["code"].(string); code != "realtime_forbidden_read" {
		t.Fatalf("expected realtime_forbidden_read, got %v", ev.Payload)
	}
}

// A broad-access connection retains access to the project-wide row feed.
func TestSubscribe_AllowsBroadAccessOnProjectWideRowFeed(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "api:key1", true, true)
	channel := "projects.proj1.databases.rows"

	hub.subscribeClient(c, channel)
	if !hub.subscribed(c, channel) {
		t.Fatal("broad-access subscription to the project-wide row feed was rejected")
	}
}

// --- Deploy channels ---------------------------------------------------------

func TestSubscribe_DeployChannelRejectsOtherProjectRelease(t *testing.T) {
	hub := NewHub("", "")
	hub.SetReleaseVerifier(stubReleaseVerifier{ownerProject: "proj2"})
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "deploy.rel123")

	if hub.subscribed(c, "deploy.rel123") {
		t.Fatal("deploy subscription for another project's release was accepted")
	}
}

func TestSubscribe_DeployChannelAllowsOwnRelease(t *testing.T) {
	hub := NewHub("", "")
	hub.SetReleaseVerifier(stubReleaseVerifier{ownerProject: "proj1"})
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "deploy.rel123")

	if !hub.subscribed(c, "deploy.rel123") {
		t.Fatal("deploy subscription for the connection's own release was rejected")
	}
}

func TestSubscribe_DeployChannelRejectsUnauthenticated(t *testing.T) {
	hub := NewHub("", "")
	hub.SetReleaseVerifier(stubReleaseVerifier{ownerProject: "proj1"})
	c := authedClient(hub, "proj1", "", false, false)

	hub.subscribeClient(c, "deploy.rel123")

	if hub.subscribed(c, "deploy.rel123") {
		t.Fatal("unauthenticated deploy subscription was accepted")
	}
}

// --- Chat channels: authorized by conversation membership, not project scoping --

// stubChatVerifier reports membership only for the configured
// (projectID, conversationID, userID) triples.
type stubChatVerifier struct {
	members map[string]bool // "projectID|conversationID|userID" -> true
	err     error
}

func (s stubChatVerifier) IsConversationMember(_ context.Context, projectID, conversationID, userID string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.members[projectID+"|"+conversationID+"|"+userID], nil
}

func TestSubscribe_ChatChannelAllowsConversationMember(t *testing.T) {
	hub := NewHub("", "")
	hub.SetChatVerifier(stubChatVerifier{members: map[string]bool{"proj1|conv1|user1": true}})
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "chat.conv1")

	if !hub.subscribed(c, "chat.conv1") {
		t.Fatal("chat subscription for a conversation member was rejected")
	}
}

func TestSubscribe_ChatChannelRejectsNonMember(t *testing.T) {
	hub := NewHub("", "")
	hub.SetChatVerifier(stubChatVerifier{members: map[string]bool{"proj1|conv1|user1": true}})
	c := authedClient(hub, "proj1", "user2", true, false)

	hub.subscribeClient(c, "chat.conv1")

	if hub.subscribed(c, "chat.conv1") {
		t.Fatal("chat subscription for a non-member was accepted")
	}
}

func TestSubscribe_ChatChannelRejectsUnauthenticated(t *testing.T) {
	hub := NewHub("", "")
	hub.SetChatVerifier(stubChatVerifier{members: map[string]bool{"proj1|conv1|": true}})
	c := authedClient(hub, "proj1", "", false, false)

	hub.subscribeClient(c, "chat.conv1")

	if hub.subscribed(c, "chat.conv1") {
		t.Fatal("unauthenticated chat subscription was accepted")
	}
}

// A broad-access connection (server API key) gets no special treatment on a
// chat channel: chat participants are real users, and an API key's synthetic
// identity is never a real conversation member, so it must still be denied.
func TestSubscribe_ChatChannelBroadAccessStillRequiresMembership(t *testing.T) {
	hub := NewHub("", "")
	hub.SetChatVerifier(stubChatVerifier{members: map[string]bool{}})
	c := authedClient(hub, "proj1", "api:key1", true, true)

	hub.subscribeClient(c, "chat.conv1")

	if hub.subscribed(c, "chat.conv1") {
		t.Fatal("broad-access connection without conversation membership was accepted")
	}
}

func TestSubscribe_ChatChannelFailsClosedWithoutVerifier(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "chat.conv1")

	if hub.subscribed(c, "chat.conv1") {
		t.Fatal("chat subscription was accepted with no verifier wired")
	}
	if ev := drainError(c); ev == nil || ev.Type != "error" {
		t.Fatal("expected an error frame when chat delivery is not configured")
	}
}

func TestSubscribe_ChatChannelFailsClosedOnVerifierError(t *testing.T) {
	hub := NewHub("", "")
	hub.SetChatVerifier(stubChatVerifier{err: fmt.Errorf("db unavailable")})
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "chat.conv1")

	if hub.subscribed(c, "chat.conv1") {
		t.Fatal("chat subscription was accepted despite the membership check erroring")
	}
}

// --- Unknown channels fail closed -------------------------------------------

func TestSubscribe_RejectsUnknownChannel(t *testing.T) {
	hub := NewHub("", "")
	c := authedClient(hub, "proj1", "user1", true, false)

	hub.subscribeClient(c, "mystery.channel")

	if hub.subscribed(c, "mystery.channel") {
		t.Fatal("unknown channel subscription was accepted; expected fail-closed rejection")
	}
}

// --- Channel parsing ---------------------------------------------------------

func TestParseChannel(t *testing.T) {
	cases := []struct {
		in   string
		kind channelKind
		proj string
		db   string
		tbl  string
		rel  string
	}{
		{"databases.p1.d1.posts", kindDatabaseData, "p1", "d1", "posts", ""},
		{"databases.p1.d1", kindDatabaseData, "p1", "d1", "", ""},
		{"projects.p1.databases.rows", kindProjectData, "p1", "", "", ""},
		{"projects.p1.storage.files", kindProjectData, "p1", "", "", ""},
		{"deploy.rel9", kindDeploy, "", "", "", "rel9"},
		{"mystery", kindUnknown, "", "", "", ""},
	}
	for _, tc := range cases {
		pc := parseChannel(tc.in)
		if pc.kind != tc.kind || pc.projectID != tc.proj || pc.databaseID != tc.db || pc.tableName != tc.tbl || pc.releaseID != tc.rel {
			t.Errorf("parseChannel(%q) = %+v, want kind=%d proj=%q db=%q tbl=%q rel=%q",
				tc.in, pc, tc.kind, tc.proj, tc.db, tc.tbl, tc.rel)
		}
	}
}

func TestParseChannel_Chat(t *testing.T) {
	pc := parseChannel("chat.conv123")
	if pc.kind != kindChat || pc.conversationID != "conv123" {
		t.Errorf("parseChannel(chat.conv123) = %+v, want kind=kindChat conversationID=conv123", pc)
	}
}
