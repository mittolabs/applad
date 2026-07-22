//go:build integration

package tests

import (
	"fmt"
	"testing"
)

func TestWorkflowsFlow(t *testing.T) {
	projectID, apiKey := projectWithKey(t, "workflows-test")
	h := authHeader(projectID, apiKey)

	// Create workflow
	status, body := request(t, "POST", "/workflows",
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
