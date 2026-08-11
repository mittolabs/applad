package chat

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

// fixedTime returns a deterministic timestamp for tests that need to scan a
// TIMESTAMPTZ column but don't assert on its exact value.
func fixedTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func newMockService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })
	database := &db.DB{DB: mockDB}
	return NewService(database), mock
}

func TestIsConversationMember(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("conv1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	ok, err := svc.IsConversationMember(context.Background(), "proj1", "conv1", "user1")
	if err != nil {
		t.Fatalf("IsConversationMember: %v", err)
	}
	if !ok {
		t.Fatal("expected membership to be true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestDeviceOwnedByUser(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("dev1", "proj1", "user1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	ok, err := svc.deviceOwnedByUser(context.Background(), "proj1", "dev1", "user1")
	if err != nil {
		t.Fatalf("deviceOwnedByUser: %v", err)
	}
	if ok {
		t.Fatal("expected ownership to be false")
	}
}
