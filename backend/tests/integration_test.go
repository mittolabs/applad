// Package tests contains integration tests that run against a real API server.
// These tests require a running PostgreSQL and Redis instance.
//
// Run locally (with docker compose up first):
//
//	go test -tags=integration -v ./tests/...
//
// In CI the API server is started automatically via GitHub Actions.
//
//go:build integration

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

var baseURL string

func TestMain(m *testing.M) {
	baseURL = os.Getenv("APPLAD_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}
	os.Exit(m.Run())
}

// request sends an HTTP request and returns the status code and decoded body.
func request(t *testing.T, method, path string, body interface{}, headers map[string]string) (int, map[string]interface{}) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil
	}
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

// ── Health ────────────────────────────────────────────────────────────────────

func TestHealthEndpoints(t *testing.T) {
	status, body := request(t, "GET", "/health", nil, nil)
	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["status"] != "pass" {
		t.Fatalf("expected status=pass, got %v", body["status"])
	}
}

// ── Console auth ──────────────────────────────────────────────────────────────

func TestConsoleFlow(t *testing.T) {
	email := "ci-console@example.com"
	password := "supersecret99"

	// Sign up — skip this test if signup is disabled (non-fresh environment)
	status, body := request(t, "POST", "/console/signup", map[string]string{
		"name": "CI Admin", "email": email, "password": password,
	}, nil)
	if status == 403 {
		t.Skipf("console signup disabled (existing install) — skipping: %v", body["message"])
	}
	if status != 201 {
		t.Fatalf("signup: expected 201, got %d: %v", status, body)
	}

	// Login
	status, body = request(t, "POST", "/console/login", map[string]string{
		"email": email, "password": password,
	}, nil)
	if status != 200 {
		t.Fatalf("login: expected 200, got %d: %v", status, body)
	}
	token, ok := body["token"].(string)
	if !ok || token == "" {
		t.Fatalf("login: missing token in response: %v", body)
	}

	// /me
	status, body = request(t, "GET", "/console/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if status != 200 {
		t.Fatalf("/me: expected 200, got %d: %v", status, body)
	}
	if body["email"] != email {
		t.Fatalf("/me: expected email %q, got %v", email, body["email"])
	}

	// Password reset (SMTP not configured → token returned in body)
	status, body = request(t, "POST", "/console/password-reset/request",
		map[string]string{"email": email}, nil)
	if status != 200 {
		t.Fatalf("reset request: expected 200, got %d: %v", status, body)
	}
	resetToken, ok := body["token"].(string)
	if !ok || resetToken == "" {
		t.Fatalf("reset request: expected token in response: %v", body)
	}

	// Confirm reset with new password
	status, body = request(t, "POST", "/console/password-reset/confirm", map[string]string{
		"token": resetToken, "password": "newsecret99",
	}, nil)
	if status != 200 {
		t.Fatalf("reset confirm: expected 200, got %d: %v", status, body)
	}

	// Login with new password
	status, body = request(t, "POST", "/console/login", map[string]string{
		"email": email, "password": "newsecret99",
	}, nil)
	if status != 200 {
		t.Fatalf("login after reset: expected 200, got %d: %v", status, body)
	}
}

// ── Projects + API keys ───────────────────────────────────────────────────────

func TestProjectCRUD(t *testing.T) {
	// Create
	status, body := request(t, "POST", "/projects", map[string]string{
		"name":        "integration-test",
		"description": "test project",
	}, nil)
	if status != 201 {
		t.Fatalf("create: expected 201, got %d: %v", status, body)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)
	t.Logf("created project: %s", projectID)

	// Get
	status, body = request(t, "GET", fmt.Sprintf("/projects/%s", projectID), nil, nil)
	if status != 200 {
		t.Fatalf("get: expected 200, got %d", status)
	}
	if body["name"] != "integration-test" {
		t.Fatalf("expected name 'integration-test', got %v", body["name"])
	}

	// List
	status, body = request(t, "GET", "/projects", nil, nil)
	if status != 200 {
		t.Fatalf("list: expected 200, got %d", status)
	}

	// Update
	status, body = request(t, "PATCH", fmt.Sprintf("/projects/%s", projectID),
		map[string]string{"name": "integration-test-updated", "description": "updated"}, nil)
	if status != 200 {
		t.Fatalf("update: expected 200, got %d: %v", status, body)
	}
	if body["name"] != "integration-test-updated" {
		t.Fatalf("update: expected updated name, got %v", body["name"])
	}

	// Create API key — scopes ["*"] required for full access
	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test-key", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Fatalf("create key: expected 201, got %d: %v", status, body)
	}
	apiKey := body["secret"].(string)
	if len(apiKey) < 20 {
		t.Fatalf("create key: secret too short: %q", apiKey)
	}
	t.Logf("created API key: %s...", apiKey[:16])

	// List keys
	status, body = request(t, "GET", fmt.Sprintf("/projects/%s/keys", projectID), nil, nil)
	if status != 200 {
		t.Fatalf("list keys: expected 200, got %d", status)
	}
	if body["total"].(float64) < 1 {
		t.Fatalf("list keys: expected at least 1 key, got %v", body["total"])
	}
}

// ── API key authentication ────────────────────────────────────────────────────

func TestAPIKeyAuth(t *testing.T) {
	// Setup: project + key with full scope
	status, body := request(t, "POST", "/projects", map[string]string{"name": "apikey-test"}, nil)
	if status != 201 {
		t.Fatalf("create project: %d %v", status, body)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "full-key", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Fatalf("create key: %d %v", status, body)
	}
	fullKey := body["secret"].(string)

	authH := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     fullKey,
	}

	// Valid key — should reach the endpoint (200 or 403 on scope, not 401)
	status, _ = request(t, "GET", "/databases", nil, authH)
	if status == 401 {
		t.Fatalf("valid API key rejected with 401")
	}

	// Invalid key — must get 401
	badH := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     "applad_key_thisisnotarealkey000000000000000000000000000000000000000000",
	}
	status, _ = request(t, "GET", "/databases", nil, badH)
	if status != 401 {
		t.Fatalf("invalid key: expected 401, got %d", status)
	}

	// Key for wrong project — must get 401
	status2, body2 := request(t, "POST", "/projects", map[string]string{"name": "other-proj"}, nil)
	if status2 == 201 {
		otherID := body2["$id"].(string)
		defer request(t, "DELETE", fmt.Sprintf("/projects/%s", otherID), nil, nil)
		crossH := map[string]string{
			"X-Applad-Project": otherID,
			"X-Applad-Key":     fullKey, // key belongs to projectID, not otherID
		}
		status, _ = request(t, "GET", "/databases", nil, crossH)
		if status != 401 {
			t.Fatalf("cross-project key: expected 401, got %d", status)
		}
	}
}

// ── User auth flow ────────────────────────────────────────────────────────────

func TestAuthFlow(t *testing.T) {
	status, body := request(t, "POST", "/projects", map[string]string{"name": "auth-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	h := map[string]string{"X-Applad-Project": projectID}

	// Create account
	status, body = request(t, "POST", "/account", map[string]string{
		"userId": "unique()", "email": "test@example.com",
		"password": "testpassword123", "name": "Test User",
	}, h)
	if status != 201 {
		t.Fatalf("create account: expected 201, got %d: %v", status, body)
	}

	// Login
	status, body = request(t, "POST", "/account/sessions/email", map[string]string{
		"email": "test@example.com", "password": "testpassword123",
	}, h)
	if status != 201 {
		t.Fatalf("login: expected 201, got %d: %v", status, body)
	}

	// Wrong password
	status, _ = request(t, "POST", "/account/sessions/email", map[string]string{
		"email": "test@example.com", "password": "wrongpassword",
	}, h)
	if status != 401 {
		t.Fatalf("bad login: expected 401, got %d", status)
	}
}

// ── Database CRUD ─────────────────────────────────────────────────────────────

func TestDatabaseFlow(t *testing.T) {
	// Project + key
	status, body := request(t, "POST", "/projects", map[string]string{"name": "db-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Skipf("cannot create key: %d", status)
	}
	apiKey := body["secret"].(string)
	h := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     apiKey,
	}

	// Create database
	status, body = request(t, "POST", "/databases",
		map[string]interface{}{"name": "testdb"}, h)
	if status != 201 {
		t.Fatalf("create db: expected 201, got %d: %v", status, body)
	}
	dbID := body["$id"].(string)

	// Create table
	status, body = request(t, "POST", fmt.Sprintf("/databases/%s/tables", dbID),
		map[string]interface{}{"name": "posts"}, h)
	if status != 201 {
		t.Fatalf("create table: expected 201, got %d: %v", status, body)
	}
	tableID := body["$id"].(string)

	// Add columns
	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/columns/string", dbID, tableID),
		map[string]interface{}{"key": "title", "size": 256, "required": true}, h)
	if status != 201 {
		t.Fatalf("create column title: expected 201, got %d: %v", status, body)
	}
	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/columns/string", dbID, tableID),
		map[string]interface{}{"key": "body", "size": 4096, "required": false}, h)
	if status != 201 {
		t.Fatalf("create column body: expected 201, got %d: %v", status, body)
	}

	// Set table permissions (any user can read/write)
	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/permissions", dbID, tableID),
		map[string]interface{}{"permissions": []map[string]string{
			{"role": "any", "action": "create"},
			{"role": "any", "action": "read"},
			{"role": "any", "action": "update"},
			{"role": "any", "action": "delete"},
		}}, h)
	if status != 200 {
		t.Fatalf("set permissions: expected 200, got %d: %v", status, body)
	}

	// Create row
	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/rows", dbID, tableID),
		map[string]interface{}{"data": map[string]interface{}{
			"title": "Hello", "body": "World",
		}}, h)
	if status != 201 {
		t.Fatalf("create row: expected 201, got %d: %v", status, body)
	}
	rowID := body["$id"].(string)

	// Get row
	status, body = request(t, "GET",
		fmt.Sprintf("/databases/%s/tables/%s/rows/%s", dbID, tableID, rowID), nil, h)
	if status != 200 {
		t.Fatalf("get row: expected 200, got %d", status)
	}
	if body["title"] != "Hello" {
		t.Fatalf("get row: expected title='Hello', got %v", body["title"])
	}

	// List rows
	status, body = request(t, "GET",
		fmt.Sprintf("/databases/%s/tables/%s/rows", dbID, tableID), nil, h)
	if status != 200 {
		t.Fatalf("list rows: expected 200, got %d", status)
	}
	if body["total"].(float64) != 1 {
		t.Fatalf("list rows: expected 1, got %v", body["total"])
	}

	// Update row
	status, body = request(t, "PATCH",
		fmt.Sprintf("/databases/%s/tables/%s/rows/%s", dbID, tableID, rowID),
		map[string]interface{}{"data": map[string]interface{}{"title": "Updated"}}, h)
	if status != 200 {
		t.Fatalf("update row: expected 200, got %d: %v", status, body)
	}
	if body["title"] != "Updated" {
		t.Fatalf("update row: expected title='Updated', got %v", body["title"])
	}

	// Delete row
	status, _ = request(t, "DELETE",
		fmt.Sprintf("/databases/%s/tables/%s/rows/%s", dbID, tableID, rowID), nil, h)
	if status != 204 {
		t.Fatalf("delete row: expected 204, got %d", status)
	}

	// List rows — should be empty now
	status, body = request(t, "GET",
		fmt.Sprintf("/databases/%s/tables/%s/rows", dbID, tableID), nil, h)
	if status != 200 {
		t.Fatalf("list after delete: expected 200, got %d", status)
	}
	if body["total"].(float64) != 0 {
		t.Fatalf("list after delete: expected 0, got %v", body["total"])
	}
}

// ── Storage ───────────────────────────────────────────────────────────────────

func TestStorageFlow(t *testing.T) {
	status, body := request(t, "POST", "/projects", map[string]string{"name": "storage-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Skipf("cannot create key: %d", status)
	}
	apiKey := body["secret"].(string)
	h := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     apiKey,
	}

	// Create bucket
	status, body = request(t, "POST", "/storage/buckets",
		map[string]interface{}{"name": "my-bucket", "public": false}, h)
	if status != 201 {
		t.Fatalf("create bucket: expected 201, got %d: %v", status, body)
	}
	bucketID := body["$id"].(string)
	t.Logf("created bucket: %s", bucketID)

	// List buckets
	status, body = request(t, "GET", "/storage/buckets", nil, h)
	if status != 200 {
		t.Fatalf("list buckets: expected 200, got %d", status)
	}
	if body["total"].(float64) < 1 {
		t.Fatalf("list buckets: expected at least 1, got %v", body["total"])
	}

	// Get bucket
	status, body = request(t, "GET", fmt.Sprintf("/storage/buckets/%s", bucketID), nil, h)
	if status != 200 {
		t.Fatalf("get bucket: expected 200, got %d", status)
	}
	if body["name"] != "my-bucket" {
		t.Fatalf("get bucket: expected name 'my-bucket', got %v", body["name"])
	}

	// Update bucket
	status, body = request(t, "PUT", fmt.Sprintf("/storage/buckets/%s", bucketID),
		map[string]interface{}{"name": "renamed-bucket"}, h)
	if status != 200 {
		t.Fatalf("update bucket: expected 200, got %d: %v", status, body)
	}
	if body["name"] != "renamed-bucket" {
		t.Fatalf("update bucket: expected 'renamed-bucket', got %v", body["name"])
	}

	// Delete bucket
	status, _ = request(t, "DELETE", fmt.Sprintf("/storage/buckets/%s", bucketID), nil, h)
	if status != 204 {
		t.Fatalf("delete bucket: expected 204, got %d", status)
	}
}

// ── Teams ─────────────────────────────────────────────────────────────────────

func TestTeamsFlow(t *testing.T) {
	status, body := request(t, "POST", "/projects", map[string]string{"name": "teams-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Skipf("cannot create key: %d", status)
	}
	apiKey := body["secret"].(string)
	h := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     apiKey,
	}

	// Create team
	status, body = request(t, "POST", "/teams",
		map[string]interface{}{"teamId": "unique()", "name": "Alpha Team"}, h)
	if status != 201 {
		t.Fatalf("create team: expected 201, got %d: %v", status, body)
	}
	teamID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/teams/%s", teamID), nil, h)
	t.Logf("created team: %s", teamID)

	// Get team
	status, body = request(t, "GET", fmt.Sprintf("/teams/%s", teamID), nil, h)
	if status != 200 {
		t.Fatalf("get team: expected 200, got %d", status)
	}
	if body["name"] != "Alpha Team" {
		t.Fatalf("get team: expected 'Alpha Team', got %v", body["name"])
	}

	// List teams
	status, body = request(t, "GET", "/teams", nil, h)
	if status != 200 {
		t.Fatalf("list teams: expected 200, got %d", status)
	}
	if body["total"].(float64) < 1 {
		t.Fatalf("list teams: expected at least 1, got %v", body["total"])
	}

	// Update team name
	status, body = request(t, "PUT", fmt.Sprintf("/teams/%s", teamID),
		map[string]interface{}{"name": "Beta Team"}, h)
	if status != 200 {
		t.Fatalf("update team: expected 200, got %d: %v", status, body)
	}
	if body["name"] != "Beta Team" {
		t.Fatalf("update team: expected 'Beta Team', got %v", body["name"])
	}
}

// ── Webhooks ──────────────────────────────────────────────────────────────────

func TestWebhooksFlow(t *testing.T) {
	status, body := request(t, "POST", "/projects", map[string]string{"name": "webhooks-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	// Webhooks use the console session — reuse console project-level endpoints (no API key needed for management)
	// Actually webhooks are project-scoped with API key auth — no API key needed for management endpoints
	// Let's check: from router, webhooks are under /webhooks with project+auth middleware

	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Skipf("cannot create key: %d", status)
	}
	apiKey := body["secret"].(string)
	h := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     apiKey,
	}

	// Create webhook
	status, body = request(t, "POST", "/webhooks",
		map[string]interface{}{
			"name":    "test-hook",
			"url":     "https://example.com/hook",
			"events":  []string{"databases.rows.create"},
			"enabled": true,
		}, h)
	if status != 201 {
		t.Fatalf("create webhook: expected 201, got %d: %v", status, body)
	}
	hookID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/webhooks/%s", hookID), nil, h)
	t.Logf("created webhook: %s", hookID)

	// List webhooks
	status, body = request(t, "GET", "/webhooks", nil, h)
	if status != 200 {
		t.Fatalf("list webhooks: expected 200, got %d", status)
	}
	if body["total"].(float64) < 1 {
		t.Fatalf("list webhooks: expected at least 1, got %v", body["total"])
	}

	// Get webhook
	status, body = request(t, "GET", fmt.Sprintf("/webhooks/%s", hookID), nil, h)
	if status != 200 {
		t.Fatalf("get webhook: expected 200, got %d", status)
	}
	if body["name"] != "test-hook" {
		t.Fatalf("get webhook: expected 'test-hook', got %v", body["name"])
	}

	// Update webhook
	status, body = request(t, "PUT", fmt.Sprintf("/webhooks/%s", hookID),
		map[string]interface{}{"name": "updated-hook", "url": "https://example.com/hook", "events": []string{"databases.rows.create"}, "enabled": true}, h)
	if status != 200 {
		t.Fatalf("update webhook: expected 200, got %d: %v", status, body)
	}
	if body["name"] != "updated-hook" {
		t.Fatalf("update webhook: expected 'updated-hook', got %v", body["name"])
	}
}

// ── Workflows ─────────────────────────────────────────────────────────────────

func TestWorkflowsFlow(t *testing.T) {
	status, body := request(t, "POST", "/projects", map[string]string{"name": "workflows-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Skipf("cannot create key: %d", status)
	}
	apiKey := body["secret"].(string)
	h := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     apiKey,
	}

	// Create workflow
	status, body = request(t, "POST", "/workflows",
		map[string]interface{}{
			"name": "test-workflow",
			"nodes": []map[string]interface{}{
				{
					"id":   "start",
					"type": "http",
					"config": map[string]interface{}{
						"method": "GET",
						"url":    "https://httpbin.org/get",
					},
				},
			},
			"edges":   []interface{}{},
			"trigger": "manual",
		}, h)
	if status != 201 {
		t.Fatalf("create workflow: expected 201, got %d: %v", status, body)
	}
	wfID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/workflows/%s", wfID), nil, h)
	t.Logf("created workflow: %s", wfID)

	// List workflows
	status, body = request(t, "GET", "/workflows", nil, h)
	if status != 200 {
		t.Fatalf("list workflows: expected 200, got %d", status)
	}
	if body["total"].(float64) < 1 {
		t.Fatalf("list workflows: expected at least 1, got %v", body["total"])
	}

	// Get workflow
	status, body = request(t, "GET", fmt.Sprintf("/workflows/%s", wfID), nil, h)
	if status != 200 {
		t.Fatalf("get workflow: expected 200, got %d", status)
	}
	if body["name"] != "test-workflow" {
		t.Fatalf("get workflow: expected 'test-workflow', got %v", body["name"])
	}
}

// ── Scope enforcement ─────────────────────────────────────────────────────────

func TestScopeEnforcement(t *testing.T) {
	status, body := request(t, "POST", "/projects", map[string]string{"name": "scope-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	// Create a key restricted to databases only
	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "db-only", "scopes": []string{"databases.read", "databases.write"}}, nil)
	if status != 201 {
		t.Fatalf("create scoped key: %d %v", status, body)
	}
	dbKey := body["secret"].(string)
	dbH := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     dbKey,
	}

	// Databases endpoint should be accessible
	status, _ = request(t, "GET", "/databases", nil, dbH)
	if status == 401 {
		t.Fatalf("db-scoped key: expected access to /databases, got 401")
	}

	// Storage endpoint should be rejected (403 scope error, not 401 auth error)
	status, _ = request(t, "GET", "/storage/buckets", nil, dbH)
	if status == 200 {
		t.Fatalf("db-scoped key: should NOT have access to /storage, got 200")
	}
	if status == 401 {
		t.Fatalf("db-scoped key: should get scope error (403), not auth error (401)")
	}
}
