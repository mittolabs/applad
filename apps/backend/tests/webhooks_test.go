//go:build integration

package tests

import (
	"fmt"
	"testing"
)

func TestWebhooksFlow(t *testing.T) {
	projectID, apiKey := projectWithKey(t, "webhooks-test")
	h := authHeader(projectID, apiKey)

	// Create webhook
	status, body := request(t, "POST", "/webhooks",
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
		map[string]interface{}{
			"name":    "updated-hook",
			"url":     "https://example.com/hook",
			"events":  []string{"databases.rows.create"},
			"enabled": true,
		}, h)
	if status != 200 {
		t.Fatalf("update webhook: expected 200, got %d: %v", status, body)
	}
	if body["name"] != "updated-hook" {
		t.Fatalf("update webhook: expected 'updated-hook', got %v", body["name"])
	}
}
