package databases

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParsePermissionString(t *testing.T) {
	cases := []struct {
		in           string
		action, role string
		ok           bool
	}{
		{`read("team:X")`, "read", "team:X", true},
		{`delete("user:abc")`, "delete", "user:abc", true},
		{`create('users')`, "create", "users", true},
		{`read("any")`, "read", "any", true},
		{`read()`, "", "", false},
		{`garbage`, "", "", false},
		{`("x")`, "", "", false},
	}
	for _, c := range cases {
		a, r, ok := parsePermissionString(c.in)
		if ok != c.ok || (ok && (a != c.action || r != c.role)) {
			t.Errorf("parsePermissionString(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.in, a, r, ok, c.action, c.role, c.ok)
		}
	}
}

func TestParsePermissionStrings_WriteShorthandAndValidation(t *testing.T) {
	got, err := parsePermissionStrings([]string{`write("team:X")`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// write expands to update + delete.
	seen := map[string]bool{}
	for _, p := range got {
		seen[p.Action] = true
		if p.Role != "team:X" {
			t.Errorf("role = %q, want team:X", p.Role)
		}
	}
	if !seen["update"] || !seen["delete"] || seen["write"] {
		t.Errorf("write should expand to update+delete, got %+v", got)
	}

	if _, err := parsePermissionStrings([]string{`frobnicate("users")`}); err == nil {
		t.Error("expected error for unknown action")
	}
}

func TestRowPermissionsJSON_RoundTrip(t *testing.T) {
	raw, err := rowPermissionsJSON([]string{
		`read("team:X")`, `read("user:a")`, `update("user:a")`, `delete("user:a")`,
		`create("users")`, // create is table-level; must be dropped from a row
	})
	if err != nil {
		t.Fatalf("rowPermissionsJSON: %v", err)
	}
	var m map[string][]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["create"]; ok {
		t.Error("create must not appear in row permissions")
	}
	if len(m["read"]) != 2 || len(m["update"]) != 1 || len(m["delete"]) != 1 {
		t.Errorf("grouping wrong: %+v", m)
	}

	// Reconstruct the strings from the object form (what mapToRow returns).
	var generic map[string]interface{}
	_ = json.Unmarshal(raw, &generic)
	strs := rowPermissionsToStrings(generic)
	joined := strings.Join(strs, ",")
	for _, want := range []string{`read("team:X")`, `update("user:a")`, `delete("user:a")`} {
		if !strings.Contains(joined, want) {
			t.Errorf("reconstructed %q missing %q", joined, want)
		}
	}
}

func TestCombinePolicyExprs(t *testing.T) {
	if got := combinePolicyExprs("", "FALSE"); got != "" {
		t.Errorf("all-empty should be empty, got %q", got)
	}
	if got := combinePolicyExprs("A", ""); got != "A" {
		t.Errorf("single kept should be bare, got %q", got)
	}
	if got := combinePolicyExprs("A", "B"); got != "(A OR B)" {
		t.Errorf("two kept should OR, got %q", got)
	}
	if got := combinePolicyExprs("FALSE", "B"); got != "B" {
		t.Errorf("FALSE should be dropped, got %q", got)
	}
}

func TestRowPermExpression_UsesJsonbExistsNotOperator(t *testing.T) {
	expr := rowPermExpression("read")
	if strings.Contains(expr, "?") {
		t.Errorf("must not use the ? operator (rewritten to a placeholder): %q", expr)
	}
	for _, want := range []string{"jsonb_exists(", rowPermColumn, "'read'", "request.jwt.claims"} {
		if !strings.Contains(expr, want) {
			t.Errorf("expr %q missing %q", expr, want)
		}
	}
}

func TestPolicyRoleExpression_TeamRoleUsesJsonbExists(t *testing.T) {
	expr := policyRoleExpression([]string{"team:abc"})
	if strings.Contains(expr, " ? ") {
		t.Errorf("team role must not use the ? operator: %q", expr)
	}
	if !strings.Contains(expr, "jsonb_exists(") || !strings.Contains(expr, "'team:abc'") {
		t.Errorf("team role should resolve via jsonb_exists: %q", expr)
	}
}

func TestCheckRowPermission_QuotedRoles(t *testing.T) {
	perms := []string{`read("team:X")`, `delete("user:a")`}
	// The author (holds user:a) may delete.
	if !checkRowPermission(perms, []string{"any", "users", "user:a"}, "delete") {
		t.Error("author should be permitted to delete")
	}
	// A different member (team:X but not user:a) may not delete.
	if checkRowPermission(perms, []string{"any", "users", "user:b", "team:X"}, "delete") {
		t.Error("non-author must not be permitted to delete")
	}
	// team:X member may read.
	if !checkRowPermission(perms, []string{"any", "users", "user:b", "team:X"}, "read") {
		t.Error("team member should be permitted to read")
	}
	// A public read grant matches anyone.
	if !checkRowPermission([]string{`read("any")`}, []string{"any"}, "read") {
		t.Error("any should match")
	}
}

// A counter that anyone may move must not also be a row anyone may rewrite.
//
// Atomic increments used to authorize as row updates, so the only way to let
// every user like a post was to let every user edit it. `increment` is a
// separate action for exactly that reason, and these assert the separation
// holds in both directions.
func TestIncrementIsItsOwnPermission(t *testing.T) {
	parsed, err := parsePermissionStrings([]string{`increment("users")`})
	if err != nil {
		t.Fatalf("increment should be a valid action: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Action != "increment" || parsed[0].Role != "users" {
		t.Fatalf("parsed = %+v, want one increment/users", parsed)
	}

	// The grant must not leak into update or delete: that is the whole point.
	perms := []string{`increment("users")`}
	roles := []string{"users", "user:someone-else"}
	if !checkRowPermission(perms, roles, "increment") {
		t.Error("increment(users) should allow a signed-in user to increment")
	}
	for _, action := range []string{"update", "delete", "read"} {
		if checkRowPermission(perms, roles, action) {
			t.Errorf("increment(users) must not grant %q", action)
		}
	}

	// And the row still stores it, or the grant would vanish on write.
	raw, err := rowPermissionsJSON([]string{`increment("users")`, `update("user:author")`})
	if err != nil {
		t.Fatalf("rowPermissionsJSON: %v", err)
	}
	var stored map[string][]string
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal stored permissions: %v", err)
	}
	if len(stored["increment"]) != 1 || stored["increment"][0] != "users" {
		t.Errorf("stored increment = %v, want [users]", stored["increment"])
	}
	if len(stored["update"]) != 1 || stored["update"][0] != "user:author" {
		t.Errorf("stored update = %v, want [user:author]", stored["update"])
	}
}
