//go:build integration

package tests

import (
	"fmt"
	"testing"
)

func TestDatabaseFlow(t *testing.T) {
	projectID, apiKey := projectWithKey(t, "db-test")
	h := authHeader(projectID, apiKey)

	// Create database
	status, body := request(t, "POST", "/databases",
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
