package db

import "testing"

// The rebinder turns ? placeholders into $N for Postgres. JSONB has operators
// spelled with a question mark — ?, ?| and ?& all test for keys — so a query
// already written positionally must be left alone, or it becomes silently
// wrong rather than failing loudly.
func TestRebindLeavesPositionalQueriesAlone(t *testing.T) {
	query := `SELECT name FROM tests WHERE project_id = $1 AND tags ?| ARRAY['smoke']`
	if got := rebind(query); got != query {
		t.Errorf("rebind rewrote a positional query:\n got: %s\nwant: %s", got, query)
	}
}

func TestRebindConvertsPlaceholders(t *testing.T) {
	got := rebind("SELECT * FROM t WHERE a = ? AND b = ?")
	want := "SELECT * FROM t WHERE a = $1 AND b = $2"
	if got != want {
		t.Errorf("rebind() = %q, want %q", got, want)
	}
}

func TestRebindIgnoresQueriesWithoutPlaceholders(t *testing.T) {
	query := "SELECT 1"
	if got := rebind(query); got != query {
		t.Errorf("rebind() = %q, want it unchanged", got)
	}
}
