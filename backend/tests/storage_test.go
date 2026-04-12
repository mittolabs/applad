//go:build integration

package tests

import (
	"fmt"
	"testing"
)

func TestStorageFlow(t *testing.T) {
	projectID, apiKey := projectWithKey(t, "storage-test")
	h := authHeader(projectID, apiKey)

	// Create bucket
	status, body := request(t, "POST", "/storage/buckets",
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
