// Package vectors provides a managed vector store with embedding storage,
// cosine/dot-product similarity search, and RAG pipeline helpers.
package vectors

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// VectorIndex defines a named vector space.
type VectorIndex struct {
	ID             string    `json:"$id"`
	ProjectID      string    `json:"projectId"`
	Name           string    `json:"name"`
	Dimensions     int       `json:"dimensions"`
	Metric         string    `json:"metric"` // cosine | dot | euclidean
	CollectionID   string    `json:"collectionId,omitempty"`
	EmbeddingField string    `json:"embeddingField,omitempty"`
	Model          string    `json:"model"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"$createdAt"`
	UpdatedAt      time.Time `json:"$updatedAt"`
}

// Embedding is a stored vector alongside the document it represents.
type Embedding struct {
	ID        string                 `json:"$id"`
	IndexID   string                 `json:"indexId"`
	DocID     string                 `json:"documentId"`
	Vector    []float64              `json:"vector"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"$createdAt"`
}

// SimilarityResult is a ranked similarity-search hit.
type SimilarityResult struct {
	DocID    string                 `json:"documentId"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Service manages vector indexes and embeddings.
type Service struct {
	db *db.DB
}

// NewService creates a new vectors Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ── Index management ──────────────────────────────────────────────────────────

// CreateIndex creates a new vector index.
func (s *Service) CreateIndex(ctx context.Context, projectID, name string, dimensions int, metric, collectionID, embeddingField, model string) (*VectorIndex, error) {
	if dimensions <= 0 {
		dimensions = 1536
	}
	if metric == "" {
		metric = "cosine"
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	idx := &VectorIndex{
		ID: uid.New(""), ProjectID: projectID, Name: name, Dimensions: dimensions,
		Metric: metric, CollectionID: collectionID, EmbeddingField: embeddingField,
		Model: model, Status: "ready",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vector_indexes (id, project_id, name, dimensions, metric, collection_id, embedding_field, model, status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		idx.ID, idx.ProjectID, idx.Name, idx.Dimensions, idx.Metric,
		nullStr(idx.CollectionID), nullStr(idx.EmbeddingField), idx.Model, idx.Status,
		idx.CreatedAt, idx.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, fmt.Errorf("vectors: index %q already exists", name)
		}
		return nil, fmt.Errorf("vectors: create index: %w", err)
	}
	return idx, nil
}

// GetIndex fetches a vector index by ID.
func (s *Service) GetIndex(ctx context.Context, indexID, projectID string) (*VectorIndex, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, dimensions, metric,
		        COALESCE(collection_id,''), COALESCE(embedding_field,''), model, status, created_at, updated_at
		 FROM vector_indexes WHERE id = ? AND project_id = ?`, indexID, projectID)
	return scanIndex(row)
}

// ListIndexes returns all vector indexes for a project.
func (s *Service) ListIndexes(ctx context.Context, projectID string) ([]*VectorIndex, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, dimensions, metric,
		        COALESCE(collection_id,''), COALESCE(embedding_field,''), model, status, created_at, updated_at
		 FROM vector_indexes WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*VectorIndex
	for rows.Next() {
		idx, err := scanIndex(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, idx)
	}
	return out, nil
}

// DeleteIndex deletes a vector index and all its embeddings.
func (s *Service) DeleteIndex(ctx context.Context, indexID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM vector_indexes WHERE id = ? AND project_id = ?", indexID, projectID)
	return err
}

// ── Embeddings ────────────────────────────────────────────────────────────────

// Upsert stores or updates an embedding for a document.
func (s *Service) Upsert(ctx context.Context, indexID, projectID, docID string, vector []float64, metadata map[string]interface{}) (*Embedding, error) {
	emb := &Embedding{
		ID: uid.New(""), IndexID: indexID, DocID: docID,
		Vector: vector, Metadata: metadata, CreatedAt: time.Now().UTC(),
	}
	vecJSON, _ := json.Marshal(vector)
	metaJSON, _ := json.Marshal(metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vector_embeddings (id, index_id, project_id, doc_id, vector, metadata, created_at)
		 VALUES (?,?,?,?,?,?,?)
		 ON DUPLICATE KEY UPDATE vector=VALUES(vector), metadata=VALUES(metadata), created_at=VALUES(created_at)`,
		emb.ID, indexID, projectID, docID, string(vecJSON), nullBytes(metaJSON), emb.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("vectors: upsert: %w", err)
	}
	return emb, nil
}

// Delete removes an embedding by document ID.
func (s *Service) Delete(ctx context.Context, indexID, docID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM vector_embeddings WHERE index_id = ? AND doc_id = ?", indexID, docID)
	return err
}

// Query performs an approximate nearest-neighbour search using in-process cosine similarity.
// For production scale, this should be replaced with a dedicated vector DB; for
// self-hosted deployments the full-scan approach is acceptable up to ~100k vectors.
func (s *Service) Query(ctx context.Context, indexID, projectID string, queryVector []float64, topK int, scoreThreshold float64) ([]SimilarityResult, error) {
	if topK <= 0 || topK > 1000 {
		topK = 10
	}

	// Load index metric
	metric := "cosine"
	s.db.QueryRowContext(ctx, "SELECT metric FROM vector_indexes WHERE id=?", indexID).Scan(&metric) //nolint:errcheck

	rows, err := s.db.QueryContext(ctx,
		"SELECT doc_id, vector, metadata FROM vector_embeddings WHERE index_id = ? AND project_id = ?",
		indexID, projectID)
	if err != nil {
		return nil, fmt.Errorf("vectors: query: %w", err)
	}
	defer rows.Close()

	type candidate struct {
		docID    string
		score    float64
		metadata map[string]interface{}
	}
	var candidates []candidate
	for rows.Next() {
		var docID, vecRaw string
		var metaRaw []byte
		if err := rows.Scan(&docID, &vecRaw, &metaRaw); err != nil {
			return nil, err
		}
		var vec []float64
		if err := json.Unmarshal([]byte(vecRaw), &vec); err != nil {
			continue
		}
		score := similarity(metric, queryVector, vec)
		if score < scoreThreshold {
			continue
		}
		var meta map[string]interface{}
		if len(metaRaw) > 0 {
			json.Unmarshal(metaRaw, &meta) //nolint:errcheck
		}
		candidates = append(candidates, candidate{docID, score, meta})
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	out := make([]SimilarityResult, len(candidates))
	for i, c := range candidates {
		out[i] = SimilarityResult{DocID: c.docID, Score: c.score, Metadata: c.metadata}
	}
	return out, nil
}

// ── similarity metrics ────────────────────────────────────────────────────────

func similarity(metric string, a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	switch metric {
	case "dot":
		return dotProduct(a, b)
	case "euclidean":
		return 1.0 / (1.0 + euclidean(a, b))
	default: // cosine
		return cosineSimilarity(a, b)
	}
}

func cosineSimilarity(a, b []float64) float64 {
	dot := dotProduct(a, b)
	normA := vecNorm(a)
	normB := vecNorm(b)
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (normA * normB)
}

func dotProduct(a, b []float64) float64 {
	var s float64
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func vecNorm(v []float64) float64 {
	return math.Sqrt(dotProduct(v, v))
}

func euclidean(a, b []float64) float64 {
	var s float64
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return math.Sqrt(s)
}

// ── scanners ──────────────────────────────────────────────────────────────────

func scanIndex(row interface{ Scan(...interface{}) error }) (*VectorIndex, error) {
	idx := &VectorIndex{}
	if err := row.Scan(&idx.ID, &idx.ProjectID, &idx.Name, &idx.Dimensions, &idx.Metric,
		&idx.CollectionID, &idx.EmbeddingField, &idx.Model, &idx.Status,
		&idx.CreatedAt, &idx.UpdatedAt); err != nil {
		return nil, err
	}
	return idx, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
