//go:build integration

package tests

import (
	"fmt"
	"testing"
)

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
	status, _ = request(t, "GET", "/projects", nil, nil)
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

func TestAPIKeyAuth(t *testing.T) {
	projectID, fullKey := projectWithKey(t, "apikey-test")
	h := authHeader(projectID, fullKey)

	// Valid key — should reach the endpoint (200 or 403 on scope, not 401)
	status, _ := request(t, "GET", "/databases", nil, h)
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

func TestScopeEnforcement(t *testing.T) {
	projectID, _ := projectWithKey(t, "scope-test")

	// Create a key restricted to databases only
	status, body := request(t, "POST", fmt.Sprintf("/projects/%s/keys", projectID),
		map[string]interface{}{"name": "db-only", "scopes": []string{"databases.read", "databases.write"}}, nil)
	if status != 201 {
		t.Fatalf("create scoped key: %d %v", status, body)
	}
	dbH := authHeader(projectID, body["secret"].(string))

	// Databases endpoint should be accessible
	status, _ = request(t, "GET", "/databases", nil, dbH)
	if status == 401 {
		t.Fatalf("db-scoped key: expected access to /databases, got 401")
	}

	// Storage endpoint should be rejected with a scope error (not 401 auth error)
	status, _ = request(t, "GET", "/storage/buckets", nil, dbH)
	if status == 200 {
		t.Fatalf("db-scoped key: should NOT have access to /storage, got 200")
	}
	if status == 401 {
		t.Fatalf("db-scoped key: should get scope error (403), not auth error (401)")
	}
}
