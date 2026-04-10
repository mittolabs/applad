// Package tests contains integration tests that run against a real API server.
// These tests require docker compose services to be running.
// Run with: go test -tags=integration ./tests/...
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

func TestHealthEndpoints(t *testing.T) {
	status, body := request(t, "GET", "/health", nil, nil)
	if status != 200 {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["status"] != "pass" {
		t.Fatalf("expected status=pass, got %v", body["status"])
	}
}

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

	// Create API key
	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test-key", "scopes": []string{}}, nil)
	if status != 201 {
		t.Fatalf("create key: expected 201, got %d: %v", status, body)
	}
	apiKey := body["secret"].(string)
	t.Logf("created API key: %s...", apiKey[:16])

	// Delete
	status, _ = request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)
	if status != 204 {
		t.Fatalf("delete: expected 204, got %d", status)
	}
}

func TestAuthFlow(t *testing.T) {
	// Create project first
	status, body := request(t, "POST", "/projects", map[string]string{
		"name": "auth-test",
	}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	headers := map[string]string{"X-Applad-Project": projectID}

	// Create account
	status, body = request(t, "POST", "/account", map[string]string{
		"userId":   "unique()",
		"email":    "test@example.com",
		"password": "testpassword123",
		"name":     "Test User",
	}, headers)
	if status != 201 {
		t.Fatalf("create account: expected 201, got %d: %v", status, body)
	}

	// Login
	status, body = request(t, "POST", "/account/sessions/email", map[string]string{
		"email":    "test@example.com",
		"password": "testpassword123",
	}, headers)
	if status != 201 {
		t.Fatalf("login: expected 201, got %d: %v", status, body)
	}

	// Invalid login
	status, _ = request(t, "POST", "/account/sessions/email", map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}, headers)
	if status != 401 {
		t.Fatalf("bad login: expected 401, got %d", status)
	}
}

func TestDatabaseFlow(t *testing.T) {
	// Create project
	status, body := request(t, "POST", "/projects", map[string]string{"name": "db-test"}, nil)
	if status != 201 {
		t.Skipf("cannot create project: %d", status)
	}
	projectID := body["$id"].(string)
	defer request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil)

	// Create API key for auth
	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test", "scopes": []string{}}, nil)
	if status != 201 {
		t.Skipf("cannot create key: %d", status)
	}
	apiKey := body["secret"].(string)
	headers := map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     apiKey,
	}

	// Create database
	status, body = request(t, "POST", "/databases", map[string]interface{}{
		"databaseId": "unique()",
		"name":       "testdb",
	}, headers)
	if status != 201 {
		t.Fatalf("create db: expected 201, got %d: %v", status, body)
	}
	dbID := body["$id"].(string)

	// Create table
	status, body = request(t, "POST", fmt.Sprintf("/databases/%s/tables", dbID),
		map[string]interface{}{
			"tableId": "unique()",
			"name":         "posts",
			"permissions":  []string{},
		}, headers)
	if status != 201 {
		t.Fatalf("create table: expected 201, got %d: %v", status, body)
	}
	tableID := body["$id"].(string)

	// Create row
	status, body = request(t, "POST",
		fmt.Sprintf("/databases/%s/tables/%s/rows", dbID, tableID),
		map[string]interface{}{
			"rowId":  "unique()",
			"data":        map[string]interface{}{"title": "Hello", "body": "World"},
			"permissions": []string{},
		}, headers)
	if status != 201 {
		t.Fatalf("create row: expected 201, got %d: %v", status, body)
	}
	rowID := body["$id"].(string)

	// Get row
	status, body = request(t, "GET",
		fmt.Sprintf("/databases/%s/tables/%s/rows/%s", dbID, tableID, rowID),
		nil, headers)
	if status != 200 {
		t.Fatalf("get row: expected 200, got %d", status)
	}
	if body["title"] != "Hello" {
		t.Fatalf("expected title='Hello', got %v", body["title"])
	}

	// List rows
	status, body = request(t, "GET",
		fmt.Sprintf("/databases/%s/tables/%s/rows", dbID, tableID),
		nil, headers)
	if status != 200 {
		t.Fatalf("list rows: expected 200, got %d", status)
	}
}
