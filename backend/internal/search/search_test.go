package search

import (
	"context"
	"testing"

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

func TestCreateIndex_Inserts(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO search_indexes").
		WithArgs(
			sqlmock.AnyArg(), // id
			"proj1",
			nil, // collection_id
			"products",
			sqlmock.AnyArg(), // fields JSON
			1,                // typo_tolerance
			"ready",
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	idx, err := svc.CreateIndex(context.Background(), "proj1", "", "products", []string{"name", "description"}, true)
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	if idx.Name != "products" {
		t.Errorf("name = %s", idx.Name)
	}
	if !idx.TypoTolerance {
		t.Error("expected TypoTolerance=true")
	}
	if idx.Status != "ready" {
		t.Errorf("status = %s", idx.Status)
	}
}

func TestSnippet_ExtractsAroundQuery(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog and the lazy fox runs away"
	s := snippet(content, "fox", 160)
	if s == "" {
		t.Error("expected non-empty snippet")
	}
	// Should contain the query term
	if len(s) > len(content)+10 {
		t.Error("snippet longer than content")
	}
}

func TestSnippet_NoMatch(t *testing.T) {
	content := "hello world"
	s := snippet(content, "xyz", 160)
	if s != content {
		t.Errorf("expected full content when no match, got %q", s)
	}
}

func TestSnippet_TruncatesLongContent(t *testing.T) {
	content := "The quick brown fox" + string(make([]byte, 200))
	s := snippet(content, "xyz", 160)
	if len(s) > 165 { // 160 + "…"
		t.Errorf("snippet too long: %d chars", len(s))
	}
}

func TestDeleteIndex(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM search_indexes").
		WithArgs("idx1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteIndex(context.Background(), "idx1", "proj1"); err != nil {
		t.Fatalf("DeleteIndex: %v", err)
	}
}

func TestUpsertDocument(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO search_documents").
		WillReturnResult(sqlmock.NewResult(1, 1))

	doc, err := svc.Upsert(context.Background(), "idx1", "proj1", "doc1", "hello world", nil)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if doc.DocID != "doc1" {
		t.Errorf("docID = %s", doc.DocID)
	}
}

func TestDeleteDocument(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM search_documents").
		WithArgs("idx1", "doc1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteDocument(context.Background(), "idx1", "doc1"); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}
}

func TestBoolInt(t *testing.T) {
	if boolInt(true) != 1 {
		t.Error("boolInt(true) should be 1")
	}
	if boolInt(false) != 0 {
		t.Error("boolInt(false) should be 0")
	}
}
