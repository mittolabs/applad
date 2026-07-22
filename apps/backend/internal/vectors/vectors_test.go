package vectors

import (
	"context"
	"math"
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

func TestCosineSimilarity_Identical(t *testing.T) {
	v := []float64{1, 0, 0}
	s := cosineSimilarity(v, v)
	if math.Abs(s-1.0) > 1e-9 {
		t.Errorf("cosine of identical vectors = %f, want 1.0", s)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	s := cosineSimilarity(a, b)
	if math.Abs(s) > 1e-9 {
		t.Errorf("cosine of orthogonal vectors = %f, want 0.0", s)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{-1, 0, 0}
	s := cosineSimilarity(a, b)
	if math.Abs(s+1.0) > 1e-9 {
		t.Errorf("cosine of opposite vectors = %f, want -1.0", s)
	}
}

func TestDotProduct(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{4, 5, 6}
	got := dotProduct(a, b)
	want := 1.0*4 + 2.0*5 + 3.0*6 // 32
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("dot product = %f, want %f", got, want)
	}
}

func TestEuclidean_ZeroDistance(t *testing.T) {
	v := []float64{1, 2, 3}
	d := euclidean(v, v)
	if d != 0 {
		t.Errorf("euclidean distance to self = %f, want 0", d)
	}
}

func TestSimilarity_MismatchedLengths(t *testing.T) {
	s := similarity("cosine", []float64{1, 2}, []float64{1})
	if s != 0 {
		t.Error("mismatched lengths should return 0")
	}
}

func TestSimilarity_EmptyVectors(t *testing.T) {
	s := similarity("cosine", nil, nil)
	if s != 0 {
		t.Error("empty vectors should return 0")
	}
}

func TestCreateIndex_Defaults(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO vector_indexes").
		WillReturnResult(sqlmock.NewResult(1, 1))

	idx, err := svc.CreateIndex(context.Background(), "proj1", "", "embeddings", 0, "", "", "", "")
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	if idx.Dimensions != 1536 {
		t.Errorf("dimensions default = %d, want 1536", idx.Dimensions)
	}
	if idx.Metric != "cosine" {
		t.Errorf("metric default = %s, want cosine", idx.Metric)
	}
	if idx.Model != "text-embedding-3-small" {
		t.Errorf("model default = %s", idx.Model)
	}
}

func TestCreateIndex_UsesProvidedID(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO vector_indexes").
		WillReturnResult(sqlmock.NewResult(1, 1))

	idx, err := svc.CreateIndex(context.Background(), "proj1", "vec_custom", "embeddings", 3, "cosine", "", "", "")
	if err != nil {
		t.Fatalf("CreateIndex: %v", err)
	}
	if idx.ID != "vec_custom" {
		t.Errorf("id = %s, want vec_custom", idx.ID)
	}
}

func TestDeleteIndex(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM vector_indexes").
		WithArgs("idx1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteIndex(context.Background(), "idx1", "proj1"); err != nil {
		t.Fatalf("DeleteIndex: %v", err)
	}
}

func TestUpsertEmbedding(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO vector_embeddings").
		WillReturnResult(sqlmock.NewResult(1, 1))

	vec := []float64{0.1, 0.2, 0.3}
	emb, err := svc.Upsert(context.Background(), "idx1", "proj1", "doc1", vec, nil)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if emb.DocID != "doc1" {
		t.Errorf("docID = %s", emb.DocID)
	}
	if len(emb.Vector) != 3 {
		t.Errorf("vector length = %d, want 3", len(emb.Vector))
	}
}

func TestQuery_ScoreThresholdFilters(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	// Mock metric lookup
	mock.ExpectQuery("SELECT metric FROM vector_indexes").
		WillReturnRows(sqlmock.NewRows([]string{"metric"}).AddRow("cosine"))

	// One embedding that will score 1.0 (identical), one that scores ~0 (orthogonal)
	rows := sqlmock.NewRows([]string{"doc_id", "vector", "metadata"}).
		AddRow("doc1", "[1,0,0]", nil).
		AddRow("doc2", "[0,1,0]", nil)
	mock.ExpectQuery("SELECT doc_id, vector, metadata FROM vector_embeddings").
		WillReturnRows(rows)

	query := []float64{1, 0, 0}
	results, err := svc.Query(context.Background(), "idx1", "proj1", query, 10, 0.5)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	// Only doc1 (score=1.0) should pass threshold 0.5; doc2 (score=0.0) should not
	if len(results) != 1 {
		t.Errorf("results = %d, want 1", len(results))
	}
	if results[0].DocID != "doc1" {
		t.Errorf("top result = %s, want doc1", results[0].DocID)
	}
	if math.Abs(results[0].Score-1.0) > 1e-9 {
		t.Errorf("top score = %f, want 1.0", results[0].Score)
	}
}
