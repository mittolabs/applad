package databases

// Tests for three database-route authorization fixes:
//   - /transactions now permission-checks each op for a user session (P0 #1)
//   - /sql is gated to a server API key or console admin (P0 #2)
//   - increment/decrement/append are a single atomic UPDATE ... RETURNING (P1 #10)

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/middleware"
)

// expectAnonTx mocks prepareDirectAccessTx for a server API key (no user), which
// runs under the per-database _anon role with an empty user_id claim.
func expectAnonTx(mock sqlmock.Sqlmock, databaseID, projectID string) {
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

// injectContext wraps a handler so the request carries the given context values,
// standing in for the Authenticate middleware in a handler unit test.
func injectContext(fn func(ctx context.Context) context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(fn(r.Context())))
		})
	}
}

// ── P0 #1: transactions enforce per-op permission for a user session ──

// A user session updating a non-row-security table via a transaction op must hold
// table-level update. Non-RLS tables carry unconditional DML grants, so without
// this check the op would rewrite any row. The whole transaction is rolled back.
func TestExecuteTransaction_UserWithoutPermissionRejected(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectAuthedTx(mock, "db1", "proj1", "u1")
	expectTableContext(mock, "t1", "db1", "proj1", "posts", false)
	// existingRowPermissions on a non-RLS table: a plain existence check.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM "p_proj1_db1"."posts" WHERE id = $1`)).
		WithArgs("r1").
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	// enforcePermission: no update grant, and row security is off, so no fallback.
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM permissions`).
		WithArgs("proj1", "table", "t1", "update", "any", "users", "user:u1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT row_security FROM tables WHERE id =`).
		WithArgs("t1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"row_security"}).AddRow(false))
	mock.ExpectRollback()

	ops := []TransactionOp{{Method: "UPDATE", TableID: "t1", RowID: "r1", Data: map[string]interface{}{"title": "x"}}}
	_, err := svc.ExecuteTransaction(context.Background(), "proj1", "db1", "u1", nil, ops)
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// The same user, granted table-level create, runs the op normally: the gate
// denies the attacker without blocking a legitimate caller.
func TestExecuteTransaction_AuthorizedUserSucceeds(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectAuthedTx(mock, "db1", "proj1", "u1")
	expectTableContext(mock, "t1", "db1", "proj1", "posts", false)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM permissions`).
		WithArgs("proj1", "table", "t1", "create", "any", "users", "user:u1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO "posts"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_jsonb(t) FROM "posts" AS t WHERE id = $1`)).
		WithArgs("r1").
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"r1","title":"x"}`)))
	mock.ExpectCommit()

	ops := []TransactionOp{{Method: "CREATE", TableID: "t1", RowID: "r1", Data: map[string]interface{}{"title": "x"}}}
	results, err := svc.ExecuteTransaction(context.Background(), "proj1", "db1", "u1", nil, ops)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(results) != 1 || results[0].Status != http.StatusCreated {
		t.Fatalf("expected one created result, got %+v", results)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A server API key (userID == "") keeps full access: no permission query runs
// before the op. The absence of any permissions COUNT proves the check is scoped
// to user sessions.
func TestExecuteTransaction_ServerKeySkipsPermissionChecks(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectAnonTx(mock, "db1", "proj1")
	expectTableContext(mock, "t1", "db1", "proj1", "posts", false)
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "posts" WHERE id = $1`)).
		WithArgs("r1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ops := []TransactionOp{{Method: "DELETE", TableID: "t1", RowID: "r1"}}
	results, err := svc.ExecuteTransaction(context.Background(), "proj1", "db1", "", nil, ops)
	if err != nil {
		t.Fatalf("expected success for server key, got %v", err)
	}
	if len(results) != 1 || results[0].Status != http.StatusNoContent {
		t.Fatalf("expected one no-content result, got %+v", results)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A read/list op on a document-security table is filtered per row by RLS, so the
// blanket table-level gate is skipped for a user session — a caller with only
// row-level read must still read the rows they are permitted to. The absence of a
// permissions COUNT proves the read path does not over-block here.
func TestExecuteTransaction_DocumentSecurityReadSkipsGate(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectAuthedTx(mock, "db1", "proj1", "u1")
	expectTableContext(mock, "t1", "db1", "proj1", "posts", true)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT to_jsonb(t) FROM "posts" AS t ORDER BY created_at DESC LIMIT $1 OFFSET $2`)).
		WithArgs(25, 0).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow([]byte(`{"id":"mine"}`)))
	mock.ExpectCommit()

	ops := []TransactionOp{{Method: "GET", TableID: "t1"}}
	results, err := svc.ExecuteTransaction(context.Background(), "proj1", "db1", "u1", nil, ops)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(results) != 1 || results[0].Status != http.StatusOK {
		t.Fatalf("expected one ok result, got %+v", results)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ── P0 #2: /sql is gated to a server API key or console admin ──

// A plain end-user session must be refused with 403 before any SQL runs: the raw
// endpoint applies no table-level permission check and would otherwise expose
// every non-RLS table. The service is never reached, so it can be empty.
func TestExecuteSQLHandler_UserSessionForbidden(t *testing.T) {
	h := NewHandler(&Service{})

	req := httptest.NewRequest(http.MethodPost, "/db1/sql", bytes.NewReader([]byte(`{"statement":"SELECT 1"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Applad-Project", "test-project")
	w := httptest.NewRecorder()

	mux := chi.NewMux()
	mux.Use(middleware.ProjectContext)
	mux.Use(injectContext(func(ctx context.Context) context.Context {
		return middleware.ContextWithUser(ctx, "u1")
	}))
	mux.Post("/{databaseId}/sql", h.executeSQL)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a user session, got %d: %s", w.Code, w.Body.String())
	}
	// The API-key-allowed side is proven by TestExecuteSQLHandler_IgnoresRolesFromBody,
	// which marks the caller as a key and gets 200.
}

// ── P1 #10: atomic ops are a single UPDATE ... RETURNING, not read-then-write ──

// The increment must be one statement so concurrent increments cannot lose each
// other's writes. Asserting the exact SQL (COALESCE(col,0)+$delta ... RETURNING)
// proves the fix replaced the old GetRow-then-UpdateRow pair.
func TestAtomicNumericOp_EmitsSingleAtomicUpdate(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectTableContext(mock, "t1", "db1", "proj1", "counters", false)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(validation, '{}') FROM columns WHERE table_id = $1 AND key_name = $2`)).
		WithArgs("t1", "views").
		WillReturnRows(sqlmock.NewRows([]string{"validation"}).AddRow([]byte(`{}`)))
	expectAnonTx(mock, "db1", "proj1")
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "counters" SET "views" = COALESCE("views", 0) + $1, updated_at = NOW() WHERE id = $2 RETURNING to_jsonb("counters".*)`)).
		WithArgs(float64(5), "r1").
		WillReturnRows(sqlmock.NewRows([]string{"to_jsonb"}).AddRow([]byte(`{"id":"r1","views":5}`)))
	mock.ExpectCommit()

	row, err := svc.AtomicNumericOp(context.Background(), "proj1", "db1", "t1", "r1", "views", 5, "", nil)
	if err != nil {
		t.Fatalf("AtomicNumericOp returned error: %v", err)
	}
	if row.ID != "r1" || row.Data["views"] != float64(5) {
		t.Fatalf("expected row r1 with views=5, got %+v", row)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A min/max bound on the column clamps the result inside the same statement.
func TestAtomicNumericOp_ClampsToConfiguredBounds(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectTableContext(mock, "t1", "db1", "proj1", "counters", false)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(validation, '{}') FROM columns WHERE table_id = $1 AND key_name = $2`)).
		WithArgs("t1", "views").
		WillReturnRows(sqlmock.NewRows([]string{"validation"}).AddRow([]byte(`{"min":0,"max":10}`)))
	expectAnonTx(mock, "db1", "proj1")
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "counters" SET "views" = LEAST(GREATEST(COALESCE("views", 0) + $1, $2), $3), updated_at = NOW() WHERE id = $4 RETURNING to_jsonb("counters".*)`)).
		WithArgs(float64(-5), float64(0), float64(10), "r1").
		WillReturnRows(sqlmock.NewRows([]string{"to_jsonb"}).AddRow([]byte(`{"id":"r1","views":0}`)))
	mock.ExpectCommit()

	if _, err := svc.AtomicNumericOp(context.Background(), "proj1", "db1", "t1", "r1", "views", -5, "", nil); err != nil {
		t.Fatalf("AtomicNumericOp returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A user session still needs update permission on the row: the atomic op is
// rejected before any UPDATE runs when the caller lacks it.
func TestAtomicNumericOp_UserWithoutPermissionRejected(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectTableContext(mock, "t1", "db1", "proj1", "counters", false)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT 1 FROM "p_proj1_db1"."counters" WHERE id = $1`)).
		WithArgs("r1").
		WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM permissions`).
		WithArgs("proj1", "table", "t1", "update", "any", "users", "user:u1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT row_security FROM tables WHERE id =`).
		WithArgs("t1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"row_security"}).AddRow(false))

	_, err := svc.AtomicNumericOp(context.Background(), "proj1", "db1", "t1", "r1", "views", 1, "u1", nil)
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// Append is likewise one statement: array_append(col, $1) in a single UPDATE.
func TestAtomicArrayAppend_EmitsSingleAtomicUpdate(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	expectTableContext(mock, "t1", "db1", "proj1", "posts", false)
	expectAnonTx(mock, "db1", "proj1")
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE "posts" SET "tags" = array_append("tags", $1), updated_at = NOW() WHERE id = $2 RETURNING to_jsonb("posts".*)`)).
		WithArgs("new", "r1").
		WillReturnRows(sqlmock.NewRows([]string{"to_jsonb"}).AddRow([]byte(`{"id":"r1","tags":["new"]}`)))
	mock.ExpectCommit()

	row, err := svc.AtomicArrayAppend(context.Background(), "proj1", "db1", "t1", "r1", "tags", "new", "", nil)
	if err != nil {
		t.Fatalf("AtomicArrayAppend returned error: %v", err)
	}
	if row.ID != "r1" {
		t.Fatalf("expected row r1, got %+v", row)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
