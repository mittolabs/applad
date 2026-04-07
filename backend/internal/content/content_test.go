package content

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func newMockDB(t *testing.T) (*db.DB, sqlmock.Sqlmock) {
	t.Helper()
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	return &db.DB{DB: raw}, mock
}

func TestCreateType_Inserts(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO content_types").
		WillReturnResult(sqlmock.NewResult(1, 1))

	fields := []Field{{Key: "title", Label: "Title", Type: "text", Required: true}}
	ct, err := svc.CreateType(context.Background(), "proj1", "Article", "article", fields, true, false)
	if err != nil {
		t.Fatalf("CreateType: %v", err)
	}
	if ct.Slug != "article" {
		t.Errorf("slug = %s", ct.Slug)
	}
	if !ct.Versioning {
		t.Error("versioning should be true")
	}
	if ct.Localization {
		t.Error("localization should be false")
	}
}

func TestCreateEntry_InsertsAndCreatesVersion(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO content_entries").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO content_versions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	entry, err := svc.CreateEntry(context.Background(), "type1", "proj1", "hello-world", "en", "user1", map[string]interface{}{"title": "Hello World"})
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if entry.Status != "draft" {
		t.Errorf("status = %s, want draft", entry.Status)
	}
	if entry.Version != 1 {
		t.Errorf("version = %d, want 1", entry.Version)
	}
	if entry.Locale != "en" {
		t.Errorf("locale = %s", entry.Locale)
	}
}

func TestCreateEntry_DefaultsLocale(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO content_entries").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO content_versions").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	entry, err := svc.CreateEntry(context.Background(), "type1", "proj1", "", "", "", nil)
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if entry.Locale != "en" {
		t.Errorf("default locale = %s, want en", entry.Locale)
	}
}

func TestPublishEntry(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("UPDATE content_entries SET status='published'").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "entry1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.PublishEntry(context.Background(), "entry1", "proj1"); err != nil {
		t.Fatalf("PublishEntry: %v", err)
	}
}

func TestUnpublishEntry(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("UPDATE content_entries SET status='draft'").
		WithArgs(sqlmock.AnyArg(), "entry1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.UnpublishEntry(context.Background(), "entry1", "proj1"); err != nil {
		t.Fatalf("UnpublishEntry: %v", err)
	}
}

func TestDeleteEntry(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM content_entries").
		WithArgs("entry1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteEntry(context.Background(), "entry1", "proj1"); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
}

func TestDeleteType(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM content_types").
		WithArgs("type1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteType(context.Background(), "type1", "proj1"); err != nil {
		t.Fatalf("DeleteType: %v", err)
	}
}

func TestNullStrHelper(t *testing.T) {
	if nullStr("") != nil {
		t.Error("empty string should be nil")
	}
	if nullStr("x") != "x" {
		t.Error("non-empty string should pass through")
	}
}

func TestBoolIntHelper(t *testing.T) {
	if boolInt(true) != 1 || boolInt(false) != 0 {
		t.Error("boolInt mismatch")
	}
}

// Test that the full content lifecycle compiles and the types are coherent.
func TestFieldType_Completeness(t *testing.T) {
	_ = Field{Key: "body", Label: "Body", Type: "richtext", Required: false, Default: nil}
	_ = ContentType{CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = Entry{Status: "published", Version: 3}
	_ = Version{Version: 2}
}
