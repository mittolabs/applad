package databases

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func TestSessionClaims_IncludesRoles(t *testing.T) {
	roles := normalizeRoles("user1", []string{"admin"})
	joinedRoles := strings.Join(roles, ",")
	for _, expected := range []string{"admin", "any", "user:user1", "users"} {
		if !strings.Contains(joinedRoles, expected) {
			t.Fatalf("expected roles to include %q, got %v", expected, roles)
		}
	}

	claims := sessionClaims{
		Role:      "applad_user",
		ProjectID: "proj1",
		UserID:    "user1",
		Roles:     roles,
	}
	data, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal sessionClaims: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["project_id"] != "proj1" {
		t.Fatalf("expected project_id proj1, got %v", m["project_id"])
	}
	if m["role"] != "applad_user" {
		t.Fatalf("expected role applad_user, got %v", m["role"])
	}
}

func TestPolicyRoleExpression_HandlesBuiltInRoles(t *testing.T) {
	expr := policyRoleExpression([]string{"users", "admin", "user:user1"})
	checks := []string{
		"current_setting('applad.user_id', true)",
		"jsonb_exists(",
		"'admin'",
		"= 'user1'",
	}
	for _, check := range checks {
		if !strings.Contains(expr, check) {
			t.Fatalf("expected expression %q to contain %q", expr, check)
		}
	}
}

func TestLookupTableContext_ComputesSchema(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectQuery(`SELECT id, database_id, project_id, name, row_security FROM tables WHERE id =`).
		WithArgs("table1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "database_id", "project_id", "name", "row_security"}).
			AddRow("table1", "db-1", "proj-1", "users", false))

	table, err := svc.lookupTableContext(context.Background(), "table1")
	if err != nil {
		t.Fatalf("lookupTableContext returned error: %v", err)
	}
	if table.Schema != "p_proj_1_db_1" {
		t.Fatalf("expected schema p_proj_1_db_1, got %q", table.Schema)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateIndex_DefaultTypeUsesStandardIndex(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectLookupTable(mock, "table1", "db1", "proj1", "users")
	mock.ExpectExec(regexp.QuoteMeta(`CREATE INDEX IF NOT EXISTS "users_email_idx" ON "p_proj1_db1"."users" ("email")`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO indexes`).
		WithArgs(sqlmock.AnyArg(), "table1", "users_email_idx", "btree", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	index, err := svc.CreateIndex(context.Background(), "proj1", "table1", "users_email_idx", "btree", []string{"email"}, []string{"ASC"})
	if err != nil {
		t.Fatalf("CreateIndex returned error: %v", err)
	}
	if index.Type != "btree" {
		t.Fatalf("expected type btree, got %q", index.Type)
	}
	if len(index.Columns) != 1 || index.Columns[0] != "email" {
		t.Fatalf("expected email column, got %v", index.Columns)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateIndex_UniqueTypeUsesUniqueDDL(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectLookupTable(mock, "table1", "db1", "proj1", "users")
	mock.ExpectExec(regexp.QuoteMeta(`CREATE UNIQUE INDEX IF NOT EXISTS "users_email_unique" ON "p_proj1_db1"."users" ("email")`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO indexes`).
		WithArgs(sqlmock.AnyArg(), "table1", "users_email_unique", "unique", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	index, err := svc.CreateIndex(context.Background(), "proj1", "table1", "users_email_unique", "unique", []string{"email"}, []string{"ASC"})
	if err != nil {
		t.Fatalf("CreateIndex returned error: %v", err)
	}
	if index.Type != "unique" {
		t.Fatalf("expected type unique, got %q", index.Type)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateIndex_FullTextSingleColumnUsesGIN(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectLookupTable(mock, "table1", "db1", "proj1", "articles")
	mock.ExpectExec(regexp.QuoteMeta(`CREATE INDEX IF NOT EXISTS "articles_body_search" ON "p_proj1_db1"."articles" USING GIN (to_tsvector('english', "body"))`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO indexes`).
		WithArgs(sqlmock.AnyArg(), "table1", "articles_body_search", "fulltext", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	index, err := svc.CreateIndex(context.Background(), "proj1", "table1", "articles_body_search", "fulltext", []string{"body"}, []string{"ASC"})
	if err != nil {
		t.Fatalf("CreateIndex returned error: %v", err)
	}
	if index.Type != "fulltext" {
		t.Fatalf("expected type fulltext, got %q", index.Type)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecuteSQL_RejectsDDL(t *testing.T) {
	svc := &Service{}
	_, err := svc.ExecuteSQL(context.Background(), "proj1", "db1", "user1", []string{"admin"}, "CREATE TABLE users (id text)", false)
	if err == nil {
		t.Fatal("expected DDL rejection")
	}
	if !strings.Contains(err.Error(), "DDL statements are not allowed") {
		t.Fatalf("expected DDL error, got %v", err)
	}
}

// ── Query builder: between / select / cursorAfter ───────────────────────────

func TestQueryToWhereSQL_BetweenEmitsBoundedRange(t *testing.T) {
	where, args := queryToWhereSQL([]Query{
		{Field: "age", Method: "between", Values: []interface{}{"18", "65"}},
	})
	if !strings.Contains(where, `"age" BETWEEN $1 AND $2`) {
		t.Fatalf("expected inclusive BETWEEN clause, got %q", where)
	}
	if len(args) != 2 || args[0] != "18" || args[1] != "65" {
		t.Fatalf("expected two bound params [18 65], got %v", args)
	}
}

func TestQueryToWhereSQL_BetweenSkipsMalformedBounds(t *testing.T) {
	where, args := queryToWhereSQL([]Query{
		{Field: "age", Method: "between", Values: []interface{}{"18"}},
	})
	if strings.Contains(where, "BETWEEN") {
		t.Fatalf("a single bound is malformed and must not emit BETWEEN, got %q", where)
	}
	if len(args) != 0 {
		t.Fatalf("expected no bound params for malformed between, got %v", args)
	}
}

func TestParseSelectColumns_DropsRelationExpansion(t *testing.T) {
	cols := parseSelectColumns("id, title, author(name)")
	if len(cols) != 2 || cols[0] != "id" || cols[1] != "title" {
		t.Fatalf("expected [id title] with the relation dropped, got %v", cols)
	}
	if parseSelectColumns("   ") != nil {
		t.Fatal("blank select should parse to no columns")
	}
}

func TestSelectProjection_LimitsToRequestedPlusSystemColumns(t *testing.T) {
	proj := selectProjection(parseSelectColumns("id, title, author(name)"), false)
	for _, want := range []string{`'id', t."id"`, `'title', t."title"`, `'created_at', t."created_at"`, `'updated_at', t."updated_at"`} {
		if !strings.Contains(proj, want) {
			t.Fatalf("expected projection to include %q, got %q", want, proj)
		}
	}
	if strings.Contains(proj, "author") {
		t.Fatalf("relation expansion must not appear in the projection, got %q", proj)
	}
	if strings.Contains(proj, rowPermColumn) {
		t.Fatalf("a non-document-security table must not project %s, got %q", rowPermColumn, proj)
	}
}

func TestSelectProjection_IncludesPermissionsForDocumentSecurity(t *testing.T) {
	proj := selectProjection(parseSelectColumns("title"), true)
	if !strings.Contains(proj, rowPermColumn) {
		t.Fatalf("a document-security table must project %s, got %q", rowPermColumn, proj)
	}
}

func TestListRowsWithAuth_BetweenBindsTwoParams(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectListRowsPrelude(mock, "t1", "db1", "proj1", "posts", false)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "posts" WHERE "age" BETWEEN $1 AND $2`)).
		WithArgs("18", "65").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`"age" BETWEEN $1 AND $2`)).
		WithArgs("18", "65", 25, 0).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"r1","age":30}`)))
	expectListRowsPostlude(mock, "t1")

	params := ListParams{Queries: []Query{{Field: "age", Method: "between", Values: []interface{}{"18", "65"}}}}
	rows, total, err := svc.ListRowsWithQuery(context.Background(), "proj1", "db1", "t1", params)
	if err != nil {
		t.Fatalf("ListRowsWithQuery returned error: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != "r1" {
		t.Fatalf("expected one row r1 (total 1), got total=%d rows=%v", total, rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListRowsWithAuth_SelectLimitsProjection(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectListRowsPrelude(mock, "t1", "db1", "proj1", "posts", false)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "posts"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`jsonb_build_object('id', t."id", 'title', t."title"`)).
		WithArgs(25, 0).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"r1","title":"Hi"}`)))
	expectListRowsPostlude(mock, "t1")

	params := ListParams{Select: "id, title"}
	rows, total, err := svc.ListRowsWithQuery(context.Background(), "proj1", "db1", "t1", params)
	if err != nil {
		t.Fatalf("ListRowsWithQuery returned error: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Data["title"] != "Hi" {
		t.Fatalf("expected projected row, got total=%d rows=%v", total, rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListRowsWithAuth_CursorAfterAddsKeysetPagination(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectListRowsPrelude(mock, "t1", "db1", "proj1", "posts", false)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "posts"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	// Keyset predicate + total-order ORDER BY, cursor bound once and no OFFSET.
	mock.ExpectQuery(regexp.QuoteMeta(`("created_at", id) < ((SELECT "created_at" FROM "posts" WHERE id = $1), $1) ORDER BY "created_at" DESC, id DESC LIMIT $2`)).
		WithArgs("r100", 25).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"r101"}`)))
	expectListRowsPostlude(mock, "t1")

	params := ListParams{CursorAfter: "r100"}
	rows, total, err := svc.ListRowsWithQuery(context.Background(), "proj1", "db1", "t1", params)
	if err != nil {
		t.Fatalf("ListRowsWithQuery returned error: %v", err)
	}
	if total != 5 || len(rows) != 1 || rows[0].ID != "r101" {
		t.Fatalf("expected next page row r101 (total 5), got total=%d rows=%v", total, rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// expectListRowsPrelude mocks lookupTableContext plus the transaction setup that
// prepareDirectAccessTx performs before any row query runs (userID empty → anon).
func expectListRowsPrelude(mock sqlmock.Sqlmock, tableID, databaseID, projectID, tableName string, rowSecurity bool) {
	mock.ExpectQuery(`SELECT id, database_id, project_id, name, row_security FROM tables WHERE id =`).
		WithArgs(tableID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "database_id", "project_id", "name", "row_security"}).
			AddRow(tableID, databaseID, projectID, tableName, rowSecurity))
	mock.ExpectBegin()
	mock.ExpectExec(`SET LOCAL search_path`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('applad.project_id'`).WithArgs(projectID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('applad.database_id'`).WithArgs(databaseID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('applad.user_id'`).WithArgs("").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('request.jwt.claims'`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(schemaName(projectID, databaseID) + "_anon").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`SET LOCAL ROLE`).WillReturnResult(sqlmock.NewResult(0, 0))
}

// expectListRowsPostlude mocks the column-permission read that runs after the
// data query and the deferred transaction rollback.
func expectListRowsPostlude(mock sqlmock.Sqlmock, tableID string) {
	mock.ExpectQuery(`SELECT key_name, COALESCE\(permissions`).
		WithArgs(tableID).
		WillReturnRows(sqlmock.NewRows([]string{"key_name", "permissions"}))
	mock.ExpectRollback()
}

func newMockService(t *testing.T) (*Service, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	database := &db.DB{DB: mockDB}
	return NewService(database), mock, mockDB
}

func expectLookupTable(mock sqlmock.Sqlmock, tableID, databaseID, projectID, name string) {
	mock.ExpectQuery(`SELECT id, database_id, project_id, name FROM tables WHERE id = \$1 AND project_id = \$2`).
		WithArgs(tableID, projectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "database_id", "project_id", "name"}).
			AddRow(tableID, databaseID, projectID, name))
}

// expectTableContext mocks lookupTableContext with an explicit row-security flag.
func expectTableContext(mock sqlmock.Sqlmock, tableID, databaseID, projectID, name string, rowSecurity bool) {
	mock.ExpectQuery(`SELECT id, database_id, project_id, name, row_security FROM tables WHERE id =`).
		WithArgs(tableID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "database_id", "project_id", "name", "row_security"}).
			AddRow(tableID, databaseID, projectID, name, rowSecurity))
}

// expectAuthedTx mocks prepareDirectAccessTx for a signed-in user: the
// authenticated per-database role and a non-empty user_id claim (the anon prelude
// above covers the server/API-key case with an empty user).
func expectAuthedTx(mock sqlmock.Sqlmock, databaseID, projectID, userID string) {
	mock.ExpectBegin()
	mock.ExpectExec(`SET LOCAL search_path`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('applad.project_id'`).WithArgs(projectID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('applad.database_id'`).WithArgs(databaseID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('applad.user_id'`).WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`set_config\('request.jwt.claims'`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(schemaName(projectID, databaseID) + "_authenticated").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(`SET LOCAL ROLE`).WillReturnResult(sqlmock.NewResult(0, 0))
}

// A user session (userID != "") listing a table with document security OFF must
// hold table-level read, exactly as Get/Create require. RLS is not enabled for
// such a table and its GRANT SELECT is unconditional, so without the list-path
// gate a caller lacking read would be handed every row. Here the caller has no
// read grant and must be denied rather than shown the whole table.
func TestListRowsWithAuth_UserWithoutTableReadIsDenied(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectTableContext(mock, "t1", "db1", "proj1", "posts", false)
	// enforcePermission: no table-level read grant for the caller's roles.
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM permissions`).
		WithArgs("proj1", "table", "t1", "read", "any", "users", "user:u1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	// enforcePermission then re-reads row_security; still off, so no row-level fallback.
	mock.ExpectQuery(`SELECT row_security FROM tables WHERE id =`).
		WithArgs("t1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"row_security"}).AddRow(false))

	rows, total, err := svc.ListRowsWithAuth(context.Background(), "proj1", "db1", "t1", "u1", []string{}, ListParams{})
	if err == nil {
		t.Fatalf("expected permission denied, got total=%d rows=%v", total, rows)
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected permission denied error, got %v", err)
	}
	if rows != nil {
		t.Fatalf("no rows may be returned on denial, got %v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// The same user, once granted table-level read, still lists rows normally: the
// gate denies the attacker without blocking the legitimate caller.
func TestListRowsWithAuth_UserWithTableReadListsRows(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectTableContext(mock, "t1", "db1", "proj1", "posts", false)
	// enforcePermission: table-level read granted → allowed, no row_security re-read.
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM permissions`).
		WithArgs("proj1", "table", "t1", "read", "any", "users", "user:u1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	expectAuthedTx(mock, "db1", "proj1", "u1")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "posts"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_jsonb(t) FROM "posts" AS t`)).
		WithArgs(25, 0).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"r1"}`)))
	expectListRowsPostlude(mock, "t1")

	rows, total, err := svc.ListRowsWithAuth(context.Background(), "proj1", "db1", "t1", "u1", []string{}, ListParams{})
	if err != nil {
		t.Fatalf("ListRowsWithAuth returned error: %v", err)
	}
	if total != 2 || len(rows) != 1 || rows[0].ID != "r1" {
		t.Fatalf("expected row r1 (total 2), got total=%d rows=%v", total, rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A server API key (userID == "") keeps broad access: it is never RLS-filtered,
// so the list path must not run the table-level read gate for it. The absence of
// any permissions COUNT query here proves the check is scoped to user sessions.
func TestListRowsWithAuth_ServerKeySkipsReadGate(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectListRowsPrelude(mock, "t1", "db1", "proj1", "posts", false)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "posts"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_jsonb(t) FROM "posts" AS t`)).
		WithArgs(25, 0).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"r1"}`)))
	expectListRowsPostlude(mock, "t1")

	rows, total, err := svc.ListRowsWithAuth(context.Background(), "proj1", "db1", "t1", "", []string{"service"}, ListParams{})
	if err != nil {
		t.Fatalf("ListRowsWithAuth returned error: %v", err)
	}
	if total != 3 || len(rows) != 1 || rows[0].ID != "r1" {
		t.Fatalf("expected row r1 (total 3), got total=%d rows=%v", total, rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A document-security table is RLS-filtered per row, so the blanket table-level
// gate is skipped for a user session: a caller with only row-level read must
// still list the rows they are permitted to read. The absence of a permissions
// COUNT query proves the list path does not over-block here — RLS decides which
// rows come back.
func TestListRowsWithAuth_DocumentSecurityListsPermittedRows(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectTableContext(mock, "t1", "db1", "proj1", "posts", true)
	expectAuthedTx(mock, "db1", "proj1", "u1")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM "posts"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_jsonb(t) FROM "posts" AS t`)).
		WithArgs(25, 0).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"mine"}`)))
	expectListRowsPostlude(mock, "t1")

	rows, total, err := svc.ListRowsWithAuth(context.Background(), "proj1", "db1", "t1", "u1", []string{}, ListParams{})
	if err != nil {
		t.Fatalf("ListRowsWithAuth returned error: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != "mine" {
		t.Fatalf("expected the caller's permitted row, got total=%d rows=%v", total, rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
