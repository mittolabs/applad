package observe

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func TestListErrors_SearchFiltersByTerm(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()
	svc := NewService(&db.DB{DB: mockDB})

	// The search term must reach the query as an ILIKE clause with a wrapped
	// %term% argument, alongside the project id and the limit.
	mock.ExpectQuery(regexp.QuoteMeta("ILIKE")).
		WithArgs("proj1", "%boom%", 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "project_id", "title", "error_type", "level", "status", "fingerprint",
			"stack_trace", "breadcrumbs", "user_context", "request_ctx", "runtime_ctx", "tags",
			"environment", "release", "count", "affected_users", "priority", "assignee",
			"first_seen", "last_seen",
		}))

	if _, err := svc.ListErrors(context.Background(), "proj1", "", "", "boom", 0); err != nil {
		t.Fatalf("ListErrors: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestListErrors_NoSearchNoIlike(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()
	svc := NewService(&db.DB{DB: mockDB})

	// Without a search term, only project id + limit are bound and no ILIKE is
	// emitted (the query matcher would fail if ILIKE were present with 2 args).
	mock.ExpectQuery("FROM observe_errors").
		WithArgs("proj1", 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "project_id", "title", "error_type", "level", "status", "fingerprint",
			"stack_trace", "breadcrumbs", "user_context", "request_ctx", "runtime_ctx", "tags",
			"environment", "release", "count", "affected_users", "priority", "assignee",
			"first_seen", "last_seen",
		}))

	if _, err := svc.ListErrors(context.Background(), "proj1", "", "", "", 0); err != nil {
		t.Fatalf("ListErrors: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
