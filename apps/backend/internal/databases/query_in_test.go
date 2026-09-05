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
