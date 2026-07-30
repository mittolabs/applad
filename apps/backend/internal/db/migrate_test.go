package db

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// newMockDB returns a DB backed by go-sqlmock with exact (non-regexp) query
// matching so the SAVEPOINT / RELEASE / ROLLBACK bookkeeping can be asserted
// verbatim.
func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()
	raw, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	return &DB{raw}, mock
}

// A migration whose statement fails must roll the whole file back: no INSERT
// into schema_migrations, and the transaction is rolled back — so the file is
// not marked applied and re-runs cleanly next boot.
func TestApplyMigration_RollsBackOnFailure(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	boom := errors.New("syntax error at or near \"OOPS\"")

	mock.ExpectBegin()
	// First statement succeeds.
	mock.ExpectExec("SAVEPOINT mig_stmt_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE a (id INT)").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("RELEASE SAVEPOINT mig_stmt_0").WillReturnResult(sqlmock.NewResult(0, 0))
	// Second statement fails with a non-ignorable error.
	mock.ExpectExec("SAVEPOINT mig_stmt_1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("OOPS").WillReturnError(boom)
	// The whole file is rolled back — crucially, no INSERT is expected.
	mock.ExpectRollback()

	stmts := []string{"CREATE TABLE a (id INT)", "OOPS"}
	if err := db.applyMigration("999_test.sql", stmts, "deadbeef"); err == nil {
		t.Fatal("expected applyMigration to return an error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// The happy path commits every statement and records the row with its checksum.
func TestApplyMigration_CommitsAndRecordsChecksum(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("SAVEPOINT mig_stmt_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE a (id INT)").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("RELEASE SAVEPOINT mig_stmt_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)").
		WithArgs("999_test.sql", "abc123").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := db.applyMigration("999_test.sql", []string{"CREATE TABLE a (id INT)"}, "abc123"); err != nil {
		t.Fatalf("applyMigration: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// An ignorable error (a duplicate set_updated_at trigger) is rolled back to its
// savepoint and the migration continues and commits, matching the prior
// per-statement ignore behaviour under the new atomic wrapper.
func TestApplyMigration_IgnorableErrorContinues(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	dup := errors.New("trigger \"set_updated_at\" for relation \"tests\" already exists")

	mock.ExpectBegin()
	mock.ExpectExec("SAVEPOINT mig_stmt_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TRIGGER set_updated_at BEFORE UPDATE ON tests").WillReturnError(dup)
	mock.ExpectExec("ROLLBACK TO SAVEPOINT mig_stmt_0").WillReturnResult(sqlmock.NewResult(0, 0))
	// Next statement still runs.
	mock.ExpectExec("SAVEPOINT mig_stmt_1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE b (id INT)").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("RELEASE SAVEPOINT mig_stmt_1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)").
		WithArgs("999_test.sql", "sum").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	stmts := []string{"CREATE TRIGGER set_updated_at BEFORE UPDATE ON tests", "CREATE TABLE b (id INT)"}
	if err := db.applyMigration("999_test.sql", stmts, "sum"); err != nil {
		t.Fatalf("applyMigration: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// checksumOf is stable and content-sensitive — the property the drift warning
// relies on.
func TestChecksumOf(t *testing.T) {
	a := checksumOf([]byte("CREATE TABLE x (id INT);"))
	again := checksumOf([]byte("CREATE TABLE x (id INT);"))
	b := checksumOf([]byte("CREATE TABLE x (id BIGINT);"))
	if a != again {
		t.Fatal("checksumOf not stable for identical content")
	}
	if a == b {
		t.Fatal("checksumOf collided on differing content")
	}
	if len(a) != 64 {
		t.Fatalf("expected 64-hex-char sha256, got %d chars", len(a))
	}
}
