package databases

// Tests for the tenant-isolation fixes: body-supplied RLS roles (H1), the
// SET LOCAL ROLE sandbox (H2), project-scoped table lookups for DDL (H3),
// and SQL editor statement guards (M4).
//
// H2 note: sqlmock can only prove the code ISSUES a plain `SET LOCAL ROLE`
// on the query transaction before the user statement. That the role then
// actually applies (unlike the old DO-block form, which PostgreSQL reverts
// at block exit) must be verified against a live PostgreSQL.

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/middleware"
)

// expectDirectAccessPrelude registers the expected transaction setup issued by
// prepareDirectAccessTx/applySessionContext, up to and including SET LOCAL ROLE.
func expectDirectAccessPrelude(mock sqlmock.Sqlmock, projectID, databaseID, userID string, claimsArg driver.Value, pgRole string) {
	mock.ExpectBegin()
	mock.ExpectExec(`SET LOCAL search_path`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('applad.project_id', $1, true)`)).
		WithArgs(projectID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('applad.database_id', $1, true)`)).
		WithArgs(databaseID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('applad.user_id', $1, true)`)).
		WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`SELECT set_config('request.jwt.claims', $1, true)`)).
		WithArgs(claimsArg).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`)).
		WithArgs(pgRole).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// Anchored: a plain statement, NOT wrapped in a DO block (which would
	// revert SET LOCAL at block exit and leave RLS bypassed).
	mock.ExpectExec(`^SET LOCAL ROLE "` + pgRole + `"$`).WillReturnResult(sqlmock.NewResult(0, 0))
}

// ── H2 ──

func TestExecuteSQL_IssuesPlainSetLocalRoleBeforeUserStatement(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	// Ordered expectations: the role must be applied on the same tx, before
	// the user statement runs.
	expectDirectAccessPrelude(mock, "proj1", "db1", "user1", sqlmock.AnyArg(), "p_proj1_db1_authenticated")
	mock.ExpectQuery(`^SELECT 1$`).
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectCommit()

	result, err := svc.ExecuteSQL(context.Background(), "proj1", "db1", "user1", nil, "SELECT 1", false)
	if err != nil {
		t.Fatalf("ExecuteSQL returned error: %v", err)
	}
	if result.RowCount != 1 {
		t.Fatalf("expected 1 row, got %d", result.RowCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecuteSQL_AnonymousUsesAnonRole(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectDirectAccessPrelude(mock, "proj1", "db1", "", sqlmock.AnyArg(), "p_proj1_db1_anon")
	mock.ExpectQuery(`^SELECT 1$`).
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectCommit()

	if _, err := svc.ExecuteSQL(context.Background(), "proj1", "db1", "", nil, "SELECT 1", false); err != nil {
		t.Fatalf("ExecuteSQL returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecuteSQL_MissingRLSRoleFailsRequest(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`SET LOCAL search_path`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	for range 4 {
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`)).
		WithArgs("p_proj1_db1_authenticated").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	// The user statement must never run without the RLS role in place.
	_, err := svc.ExecuteSQL(context.Background(), "proj1", "db1", "user1", nil, "SELECT 1", false)
	if err == nil {
		t.Fatal("expected error when rls role is missing")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing-role error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ── H1 ──

// notContainsArg matches any string argument NOT containing substr.
type notContainsArg struct{ substr string }

func (m notContainsArg) Match(v driver.Value) bool {
	s, ok := v.(string)
	return ok && !strings.Contains(s, m.substr)
}

func TestExecuteSQLHandler_IgnoresRolesFromBody(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()
	h := NewHandler(svc)

	// No authenticated user in context, so the anon role applies and the RLS
	// claims must not contain the body-supplied "admin" role.
	expectDirectAccessPrelude(mock, "test-project", "db1", "",
		notContainsArg{substr: "admin"}, "p_test_project_db1_anon")
	mock.ExpectQuery(`^SELECT 1$`).
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectCommit()

	body := `{"statement":"SELECT 1","roles":["admin"]}`
	req := httptest.NewRequest(http.MethodPost, "/db1/sql", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Applad-Project", "test-project")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(middleware.ProjectContext)
	mux.Post("/{databaseId}/sql", h.executeSQL)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ── H3 ──

func expectLookupTableNotFound(mock sqlmock.Sqlmock, tableID, projectID string) {
	mock.ExpectQuery(`SELECT id, database_id, project_id, name FROM tables WHERE id = \$1 AND project_id = \$2`).
		WithArgs(tableID, projectID).
		WillReturnError(sql.ErrNoRows)
}

func TestCreateColumn_OtherTenantsTableIsNotFound(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	// The victim table exists but belongs to another project; the scoped
	// lookup must not resolve it, and no DDL may run.
	expectLookupTableNotFound(mock, "victim-table", "attacker-proj")

	_, err := svc.CreateColumn(context.Background(), "attacker-proj", "victim-table", "evil", "string", false, false, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "table not found") {
		t.Fatalf("expected table not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeleteColumn_OtherTenantsTableIsNotFound(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectLookupTableNotFound(mock, "victim-table", "attacker-proj")

	err := svc.DeleteColumn(context.Background(), "attacker-proj", "victim-table", "email")
	if err == nil || !strings.Contains(err.Error(), "table not found") {
		t.Fatalf("expected table not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateIndex_OtherTenantsTableIsNotFound(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectLookupTableNotFound(mock, "victim-table", "attacker-proj")

	_, err := svc.CreateIndex(context.Background(), "attacker-proj", "victim-table", "idx", "btree", []string{"email"}, nil)
	if err == nil || !strings.Contains(err.Error(), "table not found") {
		t.Fatalf("expected table not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLookupProjectTable_ScopesByProject(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectLookupTable(mock, "table1", "db-1", "proj-1", "users")

	table, err := svc.lookupProjectTable(context.Background(), "table1", "proj-1")
	if err != nil {
		t.Fatalf("lookupProjectTable returned error: %v", err)
	}
	if table.Schema != "p_proj_1_db_1" {
		t.Fatalf("expected schema p_proj_1_db_1, got %q", table.Schema)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ── M4 ──

func TestExecuteSQL_RejectsMultipleStatements(t *testing.T) {
	svc := &Service{}
	for _, statement := range []string{
		"SELECT 1; DROP TABLE users",
		"SELECT 1;DELETE FROM users",
		"SELECT 1; SET ROLE postgres",
		`SELECT E'\'; DROP TABLE users; --'`, // conservative: reject ambiguous escapes
	} {
		_, err := svc.ExecuteSQL(context.Background(), "proj1", "db1", "user1", nil, statement, true)
		if err == nil || !strings.Contains(err.Error(), "multiple SQL statements") {
			t.Fatalf("expected multi-statement rejection for %q, got %v", statement, err)
		}
	}
}

func TestExecuteSQL_RejectsSessionControl(t *testing.T) {
	svc := &Service{}
	for _, statement := range []string{
		"SET ROLE postgres",
		"set local role applad",
		"SET SESSION AUTHORIZATION postgres",
		"RESET ROLE",
		"reset all",
		"/* sneaky */ SET ROLE postgres",
	} {
		_, err := svc.ExecuteSQL(context.Background(), "proj1", "db1", "user1", nil, statement, true)
		if err == nil || !strings.Contains(err.Error(), "SET/RESET") {
			t.Fatalf("expected SET/RESET rejection for %q, got %v", statement, err)
		}
	}
}

func TestExecuteSQL_RejectsCommentPrefixedDDL(t *testing.T) {
	svc := &Service{}
	for _, statement := range []string{
		"/* hi */ DROP TABLE users",
		"-- hi\nDROP TABLE users",
	} {
		_, err := svc.ExecuteSQL(context.Background(), "proj1", "db1", "user1", nil, statement, true)
		if err == nil || !strings.Contains(err.Error(), "DDL statements") {
			t.Fatalf("expected DDL rejection for %q, got %v", statement, err)
		}
	}
}

func TestHasMultipleStatements(t *testing.T) {
	cases := []struct {
		statement string
		want      bool
	}{
		{"SELECT 1", false},
		{"SELECT 1;", false},
		{"SELECT 1;   ", false},
		{"SELECT 1; -- trailing comment", false},
		{"SELECT 'a;b'", false},
		{"SELECT 'it''s; fine'", false},
		{`SELECT ";" FROM "ta;ble"`, false},
		{"SELECT $$a;b$$", false},
		{"SELECT $tag$a;b$tag$", false},
		{"SELECT 1 -- ; not a separator", false},
		{"SELECT 1 /* ; not a separator */", false},
		{"SELECT 1; SELECT 2", true},
		{"SELECT 1;DROP TABLE x", true},
		{"SELECT 'a';SELECT 'b'", true},
		{"SELECT $$a$$; SELECT 2", true},
		{"SELECT 1; /* c */ SELECT 2", true},
	}
	for _, tc := range cases {
		if got := hasMultipleStatements(tc.statement); got != tc.want {
			t.Fatalf("hasMultipleStatements(%q) = %v, want %v", tc.statement, got, tc.want)
		}
	}
}

func TestStripLeadingSQLComments(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"SELECT 1", "SELECT 1"},
		{"  /* a */ SELECT 1", "SELECT 1"},
		{"/* a /* nested */ b */ SELECT 1", "SELECT 1"},
		{"-- c\nSELECT 1", "SELECT 1"},
		{"-- only a comment", ""},
		{"/* unterminated", ""},
	}
	for _, tc := range cases {
		if got := stripLeadingSQLComments(tc.in); got != tc.want {
			t.Fatalf("stripLeadingSQLComments(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
