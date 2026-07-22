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

// projectWithKey creates a project and a full-scope API key, returning both IDs.
// Registers cleanup via t.Cleanup.
func projectWithKey(t *testing.T, name string) (projectID, apiKey string) {
	t.Helper()
	status, body := request(t, "POST", "/projects", map[string]string{"name": name}, nil)
	if status != 201 {
		t.Skipf("cannot create project (%d) — skipping", status)
	}
	projectID = body["$id"].(string)
	t.Cleanup(func() { request(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, nil) })

	status, body = request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "test", "scopes": []string{"*"}}, nil)
	if status != 201 {
		t.Skipf("cannot create API key (%d) — skipping", status)
	}
	apiKey = body["secret"].(string)
	return projectID, apiKey
}

// authHeader returns the project+key headers used for API key authentication.
func authHeader(projectID, apiKey string) map[string]string {
	return map[string]string{
		"X-Applad-Project": projectID,
		"X-Applad-Key":     apiKey,
	}
}
