package databases

import (
	"strings"
	"testing"
)

// An IN filter must be parameterised per member, and an empty list must match
// nothing rather than quietly returning the whole table.
func TestBuildWhere_InOperator(t *testing.T) {
	where, args := queryToWhereSQL([]Query{{Field: "author_id", Method: "in", Values: []interface{}{"a", "b", "c"}}})
	if !strings.Contains(where, `IN ($1, $2, $3)`) {
		t.Errorf("where = %q, want three placeholders", where)
	}
	if len(args) != 3 || args[0] != "a" || args[2] != "c" {
		t.Errorf("args = %v, want [a b c]", args)
	}
	// Nothing of the caller's data may appear in the SQL text itself.
	for _, v := range []string{"a", "b", "c"} {
		if strings.Contains(where, `'`+v+`'`) {
			t.Errorf("value %q was interpolated into %q", v, where)
		}
	}

	emptyIn, emptyArgs := queryToWhereSQL([]Query{{Field: "author_id", Method: "in", Values: []interface{}{}}})
	if !strings.Contains(emptyIn, "FALSE") {
		t.Errorf("empty in() = %q, want FALSE — an empty list matches nothing", emptyIn)
	}
	if len(emptyArgs) != 0 {
		t.Errorf("empty in() bound %d args, want 0", len(emptyArgs))
	}

	emptyNot, _ := queryToWhereSQL([]Query{{Field: "author_id", Method: "notIn", Values: []interface{}{}}})
	if !strings.Contains(emptyNot, "TRUE") {
		t.Errorf("empty notIn() = %q, want TRUE — excluding nothing excludes nothing", emptyNot)
	}

	notIn, notArgs := queryToWhereSQL([]Query{{Field: "id", Method: "notIn", Values: []string{"x", "y"}}})
	if !strings.Contains(notIn, "NOT IN ($1, $2)") {
		t.Errorf("notIn = %q", notIn)
	}
	if len(notArgs) != 2 {
		t.Errorf("notIn args = %v", notArgs)
	}
}

// The empty case has to survive parsing, not just the WHERE builder.
//
// The first cut returned nil for in("field") with no members, so the filter was
// dropped and the caller got the whole table — the exact failure the empty-list
// handling exists to prevent. Testing queryToWhereSQL directly had missed it,
// because nothing could reach it with an empty list.
func TestParseQuery_EmptyInSurvivesParsing(t *testing.T) {
	q := parseQueryString(`in("author_id")`)
	if q == nil {
		t.Fatal(`in("author_id") parsed to nil — the filter would be dropped`)
	}
	if q.Field != "author_id" || q.Method != "in" {
		t.Fatalf("parsed = %+v", q)
	}
	if len(listValues(q.Values)) != 0 {
		t.Fatalf("values = %v, want empty", q.Values)
	}

	where, args := queryToWhereSQL([]Query{*q})
	if !strings.Contains(where, "FALSE") {
		t.Errorf("where = %q, want FALSE", where)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want none", args)
	}
}

// Rows come back with $id and $createdAt, so filtering on those names is the
// natural thing to write. They used to fail as unknown columns.
func TestQueryFieldName_MapsDollarNames(t *testing.T) {
	cases := map[string]string{
		"$id":        "id",
		"$createdAt": "created_at",
		"$updatedAt": "updated_at",
		"author_id":  "author_id",
	}
	for in, want := range cases {
		if got := queryFieldName(in); got != want {
			t.Errorf("queryFieldName(%q) = %q, want %q", in, got, want)
		}
	}

	where, args := queryToWhereSQL([]Query{{Field: "$id", Method: "in", Values: []interface{}{"r1", "r2"}}})
	if !strings.Contains(where, `"id" IN ($1, $2)`) {
		t.Errorf("where = %q, want a filter on the id column", where)
	}
	if len(args) != 2 {
		t.Errorf("args = %v", args)
	}
}
