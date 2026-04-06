package databases

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func newMockService(t *testing.T) (*Service, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	database := &db.DB{DB: mockDB}
	svc := NewService(database)
	return svc, mock, mockDB
}

// --- database CRUD ---

func TestCreateDatabase_Success(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO _databases").
		WithArgs(sqlmock.AnyArg(), "proj1", "My DB", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	d, err := svc.CreateDatabase(context.Background(), "proj1", "unique()", "My DB")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Name != "My DB" {
		t.Errorf("expected name 'My DB', got %q", d.Name)
	}
	if !d.Enabled {
		t.Error("expected Enabled=true")
	}
	if d.ID == "" {
		t.Error("expected non-empty ID")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetDatabase_NotFound(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectQuery("SELECT id, name, created_at, updated_at FROM _databases").
		WithArgs("missing-id", "proj1").
		WillReturnError(sql.ErrNoRows)

	_, err := svc.GetDatabase(context.Background(), "missing-id", "proj1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListDatabases_ReturnsAll(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
		AddRow("db1", "First", now, now).
		AddRow("db2", "Second", now, now).
		AddRow("db3", "Third", now, now)

	mock.ExpectQuery("SELECT id, name, created_at, updated_at FROM _databases").
		WithArgs("proj1").
		WillReturnRows(rows)

	dbs, count, err := svc.ListDatabases(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}
	if len(dbs) != 3 {
		t.Errorf("expected 3 databases, got %d", len(dbs))
	}
	for _, d := range dbs {
		if !d.Enabled {
			t.Errorf("expected Enabled=true for db %s", d.ID)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- collection CRUD ---

func TestCreateCollection_Success(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO collections").
		WithArgs(sqlmock.AnyArg(), "db1", "proj1", "users", sqlmock.AnyArg(), false, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	c, err := svc.CreateCollection(context.Background(), "proj1", "db1", "unique()", "users", []string{"read", "write"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "users" {
		t.Errorf("expected name 'users', got %q", c.Name)
	}
	if !c.Enabled {
		t.Error("expected Enabled=true")
	}
	if len(c.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(c.Permissions))
	}
	if len(c.Columns) != 0 {
		t.Errorf("expected 0 columns, got %d", len(c.Columns))
	}
	if len(c.Indexes) != 0 {
		t.Errorf("expected 0 indexes, got %d", len(c.Indexes))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetCollection_LoadsColumnsAndIndexes(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	now := time.Now().UTC()
	permsJSON, _ := json.Marshal([]string{"read"})

	// Collection query
	mock.ExpectQuery("SELECT id, database_id, name, document_security, permissions, enabled, created_at, updated_at FROM collections").
		WithArgs("coll1", "db1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "database_id", "name", "document_security", "permissions", "enabled", "created_at", "updated_at"}).
			AddRow("coll1", "db1", "users", false, permsJSON, true, now, now))

	// Attributes query
	mock.ExpectQuery("SELECT .+ FROM attributes WHERE collection_id").
		WithArgs("coll1").
		WillReturnRows(sqlmock.NewRows([]string{"key", "type", "status", "required", "array", "default_value", "options"}).
			AddRow("name", "string", "available", true, false, nil, nil).
			AddRow("age", "integer", "available", false, false, nil, nil))

	// Indexes query
	idxAttrsJSON, _ := json.Marshal([]string{"name"})
	mock.ExpectQuery("SELECT .+ FROM _indexes WHERE collection_id").
		WithArgs("coll1").
		WillReturnRows(sqlmock.NewRows([]string{"key", "type", "status", "attributes"}).
			AddRow("name_idx", "key", "available", idxAttrsJSON))

	c, err := svc.GetCollection(context.Background(), "coll1", "db1", "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "users" {
		t.Errorf("expected name 'users', got %q", c.Name)
	}
	if len(c.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(c.Columns))
	}
	if len(c.Indexes) != 1 {
		t.Errorf("expected 1 index, got %d", len(c.Indexes))
	}
	if c.Indexes[0].Key != "name_idx" {
		t.Errorf("expected index key 'name_idx', got %q", c.Indexes[0].Key)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- document CRUD ---

func TestCreateRow_Success(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO documents").
		WithArgs(sqlmock.AnyArg(), "coll1", "db1", "proj1", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	data := map[string]interface{}{"name": "John", "age": float64(30)}
	doc, err := svc.CreateDocument(context.Background(), "proj1", "db1", "coll1", "unique()", data, []string{"read"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Data["name"] != "John" {
		t.Errorf("expected data name='John', got %v", doc.Data["name"])
	}
	if doc.TableID != "coll1" {
		t.Errorf("expected TableID='coll1', got %q", doc.TableID)
	}
	if doc.ID == "" {
		t.Error("expected non-empty ID")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetRow_NotFound(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectQuery("SELECT id, collection_id, database_id, data, permissions, created_at, updated_at FROM documents").
		WithArgs("missing", "coll1", "db1", "proj1").
		WillReturnError(sql.ErrNoRows)

	_, err := svc.GetDocument(context.Background(), "missing", "coll1", "db1", "proj1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- query operator tests ---

// newQueryMockService creates a service with regexp query matching for flexible SQL assertions.
func newQueryMockService(t *testing.T) (*Service, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	database := &db.DB{DB: mockDB}
	svc := NewService(database)
	return svc, mock, mockDB
}

func docRows() *sqlmock.Rows {
	now := time.Now().UTC()
	dataJSON := []byte(`{"name":"John","age":30}`)
	permsJSON := []byte(`["read"]`)
	return sqlmock.NewRows([]string{"id", "collection_id", "database_id", "data", "permissions", "created_at", "updated_at"}).
		AddRow("doc1", "coll1", "db1", dataJSON, permsJSON, now, now)
}

func TestListDocumentsWithQuery_EqualOperator(t *testing.T) {
	svc, mock, mockDB := newQueryMockService(t)
	defer mockDB.Close()

	// Expect SELECT with JSON_EXTRACT for equal operator
	mock.ExpectQuery(`JSON_EXTRACT.*\$\.name`).
		WillReturnRows(docRows())
	// COUNT query
	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	params := ListParams{
		Limit: 10,
		Queries: []Query{
			{Attribute: "name", Method: "equal", Values: "John"},
		},
	}
	docs, total, err := svc.ListDocumentsWithQuery(context.Background(), "proj1", "db1", "coll1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListDocumentsWithQuery_GreaterThan(t *testing.T) {
	svc, mock, mockDB := newQueryMockService(t)
	defer mockDB.Close()

	// Expect CAST and > for greaterThan operator
	mock.ExpectQuery(`CAST.*AS DOUBLE.*>`).
		WillReturnRows(docRows())
	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	params := ListParams{
		Limit: 10,
		Queries: []Query{
			{Attribute: "age", Method: "greaterThan", Values: 18},
		},
	}
	docs, _, err := svc.ListDocumentsWithQuery(context.Background(), "proj1", "db1", "coll1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListDocumentsWithQuery_Contains(t *testing.T) {
	svc, mock, mockDB := newQueryMockService(t)
	defer mockDB.Close()

	// Expect LIKE for contains operator
	mock.ExpectQuery(`LIKE`).
		WillReturnRows(docRows())
	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	params := ListParams{
		Limit: 10,
		Queries: []Query{
			{Attribute: "name", Method: "contains", Values: "oh"},
		},
	}
	docs, _, err := svc.ListDocumentsWithQuery(context.Background(), "proj1", "db1", "coll1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListDocumentsWithQuery_IsNull(t *testing.T) {
	svc, mock, mockDB := newQueryMockService(t)
	defer mockDB.Close()

	// Expect IS NULL check
	mock.ExpectQuery(`IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "collection_id", "database_id", "data", "permissions", "created_at", "updated_at"}))
	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	params := ListParams{
		Limit: 10,
		Queries: []Query{
			{Attribute: "phone", Method: "isNull"},
		},
	}
	docs, total, err := svc.ListDocumentsWithQuery(context.Background(), "proj1", "db1", "coll1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListDocumentsWithQuery_OrderBy(t *testing.T) {
	svc, mock, mockDB := newQueryMockService(t)
	defer mockDB.Close()

	// Expect ORDER BY with JSON_EXTRACT for custom field
	mock.ExpectQuery(`ORDER BY JSON_UNQUOTE.*\$\.name.*ASC`).
		WillReturnRows(docRows())
	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	params := ListParams{
		Limit:     10,
		OrderAttr: "name",
		OrderType: "ASC",
	}
	docs, _, err := svc.ListDocumentsWithQuery(context.Background(), "proj1", "db1", "coll1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListDocumentsWithQuery_CursorAfter(t *testing.T) {
	svc, mock, mockDB := newQueryMockService(t)
	defer mockDB.Close()

	// Expect subquery for cursor-based pagination
	mock.ExpectQuery(`created_at < \(SELECT created_at FROM documents WHERE id`).
		WillReturnRows(docRows())
	mock.ExpectQuery(`SELECT COUNT`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	params := ListParams{
		Limit:       10,
		CursorAfter: "prev-doc-id",
	}
	docs, _, err := svc.ListDocumentsWithQuery(context.Background(), "proj1", "db1", "coll1", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- relationships ---

func TestCreateRelationship_Success(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectExec("INSERT INTO collection_relationships").
		WithArgs(sqlmock.AnyArg(), "coll1", "coll2", "oneToMany", true, "posts", "author", "cascade").
		WillReturnResult(sqlmock.NewResult(1, 1))

	rel, err := svc.CreateRelationship(context.Background(), "coll1", "coll2", "oneToMany", "posts", "author", "cascade", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.TableID != "coll1" {
		t.Errorf("expected TableID='coll1', got %q", rel.TableID)
	}
	if rel.RelatedTable != "coll2" {
		t.Errorf("expected RelatedTable='coll2', got %q", rel.RelatedTable)
	}
	if rel.Type != "oneToMany" {
		t.Errorf("expected Type='oneToMany', got %q", rel.Type)
	}
	if !rel.TwoWay {
		t.Error("expected TwoWay=true")
	}
	if rel.Key != "posts" {
		t.Errorf("expected Key='posts', got %q", rel.Key)
	}
	if rel.TwoWayKey != "author" {
		t.Errorf("expected TwoWayKey='author', got %q", rel.TwoWayKey)
	}
	if rel.OnDelete != "cascade" {
		t.Errorf("expected OnDelete='cascade', got %q", rel.OnDelete)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListRelationships_Empty(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectQuery("SELECT id, collection_id, related_collection").
		WithArgs("coll1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "collection_id", "related_collection", "relationship_type", "two_way", "key", "two_way_key", "on_delete"}))

	rels, err := svc.ListRelationships(context.Background(), "coll1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rels == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(rels) != 0 {
		t.Errorf("expected 0 relationships, got %d", len(rels))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- update / delete document ---

func TestUpdateRow_Success(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	now := time.Now().UTC()
	updatedData := map[string]interface{}{"name": "Jane", "age": float64(25)}
	dataJSON, _ := json.Marshal(updatedData)
	permsJSON, _ := json.Marshal([]string{"read", "write"})

	// UPDATE
	mock.ExpectExec("UPDATE documents SET data").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "doc1", "coll1", "db1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// GetDocument after update
	mock.ExpectQuery("SELECT id, collection_id, database_id, data, permissions, created_at, updated_at FROM documents").
		WithArgs("doc1", "coll1", "db1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "collection_id", "database_id", "data", "permissions", "created_at", "updated_at"}).
			AddRow("doc1", "coll1", "db1", dataJSON, permsJSON, now, now))

	doc, err := svc.UpdateDocument(context.Background(), "doc1", "coll1", "db1", "proj1", updatedData, []string{"read", "write"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Data["name"] != "Jane" {
		t.Errorf("expected data name='Jane', got %v", doc.Data["name"])
	}
	if doc.ID != "doc1" {
		t.Errorf("expected ID='doc1', got %q", doc.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDeleteRow_Success(t *testing.T) {
	svc, mock, mockDB := newMockService(t)
	defer mockDB.Close()

	mock.ExpectExec("DELETE FROM documents WHERE").
		WithArgs("doc1", "coll1", "db1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.DeleteDocument(context.Background(), "doc1", "coll1", "db1", "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
