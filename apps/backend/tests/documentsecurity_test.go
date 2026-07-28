//go:build integration

package tests

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

// TestDocumentSecurityChatFlow exercises the whole per-row security stack the
// chat example depends on, end to end against a real API + Postgres:
//
//   - a session returns a bearer secret usable off the browser (G1)
//   - a document-security table enforces each row's own read permission (G7)
//   - team membership becomes an RLS role, so read("team:X") admits members (G2)
//   - the membership lifecycle: creator is enrolled, invitee joins (G3)
//   - update/delete are author-scoped
//
// It is the committed form of what was first proven by hand.
func TestDocumentSecurityChatFlow(t *testing.T) {
	token := consoleToken(t)
	pid, key := projectWithConsoleToken(t, token, "docsec")
	k := authHeader(pid, key)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	db := "chat" + suffix
	tbl := "messages" + suffix
	rows := fmt.Sprintf("/databases/%s/tables/%s/rows", db, tbl)

	// --- schema, created by the operator (API key) ---
	if st, b := request(t, "POST", "/databases", map[string]string{"name": "chat", "databaseId": db}, k); st != 201 {
		t.Fatalf("create database: %d %v", st, b)
	}
	if st, b := request(t, "POST", fmt.Sprintf("/databases/%s/tables", db), map[string]interface{}{
		"name": tbl, "tableId": tbl, "permissions": []string{`create("users")`}, "documentSecurity": true,
	}, k); st != 201 {
		t.Fatalf("create table: %d %v", st, b)
	}
	for _, col := range []struct {
		key  string
		size int
	}{{"channel_id", 128}, {"user_id", 128}, {"author_name", 256}, {"body", 8000}} {
		if st, b := request(t, "POST", fmt.Sprintf("/databases/%s/tables/%s/columns/string", db, tbl),
			map[string]interface{}{"key": col.key, "size": col.size, "required": true}, k); st != 201 {
			t.Fatalf("create column %s: %d %v", col.key, st, b)
		}
	}

	// --- two users ---
	uidA, secA := signupLogin(t, pid, "a"+suffix+"@example.com")
	_, secB := signupLogin(t, pid, "b"+suffix+"@example.com")
	if secA == "" || secB == "" {
		t.Fatal("expected a session secret for each user (G1)")
	}
	authA := sessionHeader(pid, secA)
	authB := sessionHeader(pid, secB)

	// --- A creates a channel (team) and posts a message scoped to it ---
	st, team := request(t, "POST", "/teams", map[string]string{"name": "general", "teamId": "unique()"}, authA)
	if st != 201 {
		t.Fatalf("create team: %d %v", st, team)
	}
	teamID := team["$id"].(string)

	// G6: a channel (team) must appear only to its members. The creator sees it;
	// an unrelated user must not learn it exists via the team list.
	if got := teamCount(t, authA); got != 1 {
		t.Fatalf("A (member) should list 1 team, listed %d", got)
	}
	if got := teamCount(t, authB); got != 0 {
		t.Fatalf("B (non-member) must list 0 teams, listed %d [team-list scoping, G6]", got)
	}

	st, msg := request(t, "POST", rows, map[string]interface{}{
		"rowId": "unique()",
		"data":  map[string]string{"channel_id": teamID, "user_id": uidA, "author_name": "A", "body": "hello channel"},
		"permissions": []string{
			fmt.Sprintf(`read("team:%s")`, teamID),
			fmt.Sprintf(`update("user:%s")`, uidA),
			fmt.Sprintf(`delete("user:%s")`, uidA),
		},
	}, authA)
	if st != 201 {
		t.Fatalf("A posts message: %d %v", st, msg)
	}
	msgID := msg["$id"].(string)

	// --- the core assertions ---
	if got := listCount(t, rows, teamID, authA); got != 1 {
		t.Fatalf("A (member) should see 1 message, saw %d", got)
	}
	if got := listCount(t, rows, teamID, authB); got != 0 {
		t.Fatalf("B (non-member) must see 0 messages, saw %d [per-row RLS, G7]", got)
	}

	// --- invite B, B joins, B now sees it ---
	st, inv := request(t, "POST", fmt.Sprintf("/teams/%s/memberships", teamID), map[string]string{"email": "b" + suffix + "@example.com"}, authA)
	if st != 201 || inv["secret"] == nil {
		t.Fatalf("invite B: %d %v", st, inv)
	}
	if st, b := request(t, "PATCH", fmt.Sprintf("/teams/%s/memberships/%s/status", teamID, inv["$id"].(string)),
		map[string]string{"secret": inv["secret"].(string)}, authB); st != 200 {
		t.Fatalf("B accepts invite: %d %v", st, b)
	}
	if got := listCount(t, rows, teamID, authB); got != 1 {
		t.Fatalf("B should see 1 message after joining, saw %d [membership -> role -> RLS, G2/G3]", got)
	}

	// --- author-scoped writes ---
	if st, _ := request(t, "DELETE", rows+"/"+msgID, nil, authB); st != 403 && st != 404 {
		t.Fatalf("B (non-author) must not delete A's message, got %d", st)
	}
	if st, _ := request(t, "DELETE", rows+"/"+msgID, nil, authA); st != 204 && st != 200 {
		t.Fatalf("A (author) should delete their own message, got %d", st)
	}
}

// consoleToken signs up (or skips if the instance is closed) and returns a
// console session token, needed because project creation is an authenticated
// operation.
func consoleToken(t *testing.T) string {
	t.Helper()
	email := fmt.Sprintf("op%d@example.com", time.Now().UnixNano())
	st, body := request(t, "POST", "/console/signup", map[string]string{
		"name": "Op", "email": email, "password": "password123",
	}, nil)
	if st == 403 {
		t.Skipf("console signup closed on this instance — skipping: %v", body["message"])
	}
	if st != 201 {
		t.Fatalf("console signup: %d %v", st, body)
	}
	return body["token"].(string)
}

func projectWithConsoleToken(t *testing.T, token, name string) (projectID, apiKey string) {
	t.Helper()
	auth := map[string]string{"Authorization": "Bearer " + token}
	st, body := request(t, "POST", "/projects", map[string]string{"name": name}, auth)
	if st != 201 {
		t.Fatalf("create project: %d %v", st, body)
	}
	projectID = body["$id"].(string)
	t.Cleanup(func() { request(t, "DELETE", "/projects/"+projectID, nil, auth) })

	st, body = request(t, "POST", "/projects/"+projectID+"/keys",
		map[string]interface{}{"name": "boot", "scopes": []string{"*"}}, auth)
	if st != 201 {
		t.Fatalf("create key: %d %v", st, body)
	}
	return projectID, body["secret"].(string)
}

// signupLogin creates an end-user account and opens a session, returning the
// user id and the session bearer secret.
func signupLogin(t *testing.T, projectID, email string) (userID, secret string) {
	t.Helper()
	ph := map[string]string{"X-Applad-Project": projectID}
	if st, b := request(t, "POST", "/account", map[string]string{
		"userId": "unique()", "email": email, "password": "password123", "name": email,
	}, ph); st != 201 {
		t.Fatalf("create account %s: %d %v", email, st, b)
	}
	st, b := request(t, "POST", "/account/sessions/email", map[string]string{
		"email": email, "password": "password123",
	}, ph)
	if st != 201 {
		t.Fatalf("login %s: %d %v", email, st, b)
	}
	if b["userId"] != nil {
		userID = b["userId"].(string)
	}
	if b["secret"] != nil {
		secret = b["secret"].(string)
	}
	return userID, secret
}

func sessionHeader(projectID, secret string) map[string]string {
	return map[string]string{"X-Applad-Project": projectID, "Authorization": "Bearer " + secret}
}

func teamCount(t *testing.T, headers map[string]string) int {
	t.Helper()
	st, body := request(t, "GET", "/teams", nil, headers)
	if st != 200 {
		t.Fatalf("list teams: %d %v", st, body)
	}
	if teams, ok := body["teams"].([]interface{}); ok {
		return len(teams)
	}
	return 0
}

func listCount(t *testing.T, rowsPath, channelID string, headers map[string]string) int {
	t.Helper()
	q := url.QueryEscape(fmt.Sprintf(`equal("channel_id","%s")`, channelID))
	st, body := request(t, "GET", rowsPath+"?queries[]="+q, nil, headers)
	if st != 200 {
		t.Fatalf("list rows: %d %v", st, body)
	}
	if rows, ok := body["rows"].([]interface{}); ok {
		return len(rows)
	}
	return 0
}
