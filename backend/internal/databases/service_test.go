package databases

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/db"
)

func TestSignedPostgRESTJWT_IncludesClaims(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret"}
	tokenString, err := svc.signedPostgRESTJWT("proj1", "user1", []string{"admin"})
	if err != nil {
		t.Fatalf("signedPostgRESTJWT returned error: %v", err)
	}
	if tokenString == "" {
		t.Fatal("expected signed token")
	}

	token, err := jwt.ParseWithClaims(tokenString, &postgrestClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	claims, ok := token.Claims.(*postgrestClaims)
	if !ok || !token.Valid {
		t.Fatal("expected valid postgrest claims")
	}
	if claims.ProjectID != "proj1" {
		t.Fatalf("expected project_id proj1, got %q", claims.ProjectID)
	}
	if claims.UserID != "user1" {
		t.Fatalf("expected user_id user1, got %q", claims.UserID)
	}
	if claims.Role != "applad_user" {
		t.Fatalf("expected db role applad_user, got %q", claims.Role)
	}
	joinedRoles := strings.Join(claims.Roles, ",")
	for _, expected := range []string{"admin", "any", "user:user1", "users"} {
		if !strings.Contains(joinedRoles, expected) {
			t.Fatalf("expected roles to include %q, got %v", expected, claims.Roles)
		}
	}
}

func TestPolicyRoleExpression_HandlesBuiltInRoles(t *testing.T) {
	expr := policyRoleExpression([]string{"users", "admin", "user:user1"})
	checks := []string{
		"current_setting('applad.user_id', true)",
		"? 'admin'",
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

	mock.ExpectQuery(`SELECT id, database_id, project_id, name FROM tables WHERE id =`).
		WithArgs("table1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "database_id", "project_id", "name"}).
			AddRow("table1", "db-1", "proj-1", "users"))

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
	mock.ExpectExec(regexp.QuoteMeta("NOTIFY pgrst, 'reload schema'")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	index, err := svc.CreateIndex(context.Background(), "table1", "users_email_idx", "btree", []string{"email"}, []string{"ASC"})
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
	mock.ExpectExec(regexp.QuoteMeta("NOTIFY pgrst, 'reload schema'")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	index, err := svc.CreateIndex(context.Background(), "table1", "users_email_unique", "unique", []string{"email"}, []string{"ASC"})
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
	mock.ExpectExec(regexp.QuoteMeta("NOTIFY pgrst, 'reload schema'")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	index, err := svc.CreateIndex(context.Background(), "table1", "articles_body_search", "fulltext", []string{"body"}, []string{"ASC"})
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
	mock.ExpectQuery(`SELECT id, database_id, project_id, name FROM tables WHERE id =`).
		WithArgs(tableID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "database_id", "project_id", "name"}).
			AddRow(tableID, databaseID, projectID, name))
}
