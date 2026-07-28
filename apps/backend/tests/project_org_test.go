//go:build integration

package tests

import "testing"

// A project must belong to an organization: a fresh account (in no org) is
// refused, and can create one only after it has an organization.
func TestProjectRequiresOrganization(t *testing.T) {
	token := consoleToken(t) // skips if signup is closed on this instance
	auth := map[string]string{"Authorization": "Bearer " + token}

	if st, body := request(t, "POST", "/projects", map[string]string{"name": "no-org-project"}, auth); st != 409 {
		t.Fatalf("org-less account should be refused a project (409), got %d %v", st, body)
	}

	if st, body := request(t, "POST", "/organizations", map[string]string{"name": "Test Co"}, auth); st != 201 {
		t.Fatalf("create org: %d %v", st, body)
	}

	if st, body := request(t, "POST", "/projects", map[string]string{"name": "now-allowed"}, auth); st != 201 {
		t.Fatalf("after an org exists, project creation should succeed (201), got %d %v", st, body)
	}
}
