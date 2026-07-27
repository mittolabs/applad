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
