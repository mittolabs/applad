//go:build integration

package tests

import (
	"fmt"
	"testing"
)

func TestTeamsFlow(t *testing.T) {
	projectID, apiKey := projectWithKey(t, "teams-test")
	h := authHeader(projectID, apiKey)

	// Create team
	status, body := request(t, "POST", "/teams",
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
