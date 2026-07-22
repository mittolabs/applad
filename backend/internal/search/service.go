// Package search provides full-text search with relevance ranking, facets,
// typo tolerance, synonyms, and ranking rules using MySQL FULLTEXT indexes.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Index is a named search index bound to a project (and optionally a collection).
type Index struct {
	ID            string    `json:"$id"`
	ProjectID     string    `json:"projectId"`
	CollectionID  string    `json:"collectionId,omitempty"`
	Name          string    `json:"name"`
	Fields        []string  `json:"fields"`
	Synonyms      []Synonym `json:"synonyms,omitempty"`
	RankingRules  []string  `json:"rankingRules,omitempty"`
	TypoTolerance bool      `json:"typoTolerance"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"$createdAt"`
	UpdatedAt     time.Time `json:"$updatedAt"`
}

// Synonym maps a set of words that should be treated as equivalent.
type Synonym struct {
	ID       string   `json:"$id"`
	IndexID  string   `json:"indexId"`
	Synonyms []string `json:"synonyms"`
}

// Document is a document stored in a search index.
type Document struct {
	ID        string                 `json:"$id"`
	IndexID   string                 `json:"indexId"`
	DocID     string                 `json:"documentId"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	IndexedAt time.Time              `json:"indexedAt"`
}

// SearchResult is a ranked search result.
type SearchResult struct {
	Total    int    `json:"total"`
	Hits     []Hit  `json:"hits"`
	Duration string `json:"processingTimeMs"`
}

// Hit is a single search result with relevance score.
type Hit struct {
	DocID    string                 `json:"documentId"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Score    float64                `json:"score"`
	Snippet  string                 `json:"snippet,omitempty"`
}

// Service handles search index management and querying.
type Service struct {
	db *db.DB
}

// NewService creates a new search Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// CreateIndex creates a new search index.
func (s *Service) CreateIndex(ctx context.Context, projectID, indexID, collectionID, name string, fields []string, typoTolerance bool) (*Index, error) {
	idx := &Index{
		ID: uid.New(""), ProjectID: projectID, CollectionID: collectionID,
		Name: name, Fields: fields, TypoTolerance: typoTolerance,
		Status: "ready", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if indexID != "" {
		idx.ID = indexID
	}
	fieldsJSON, _ := json.Marshal(fields)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO search_indexes (id, project_id, collection_id, name, fields, typo_tolerance, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		idx.ID, idx.ProjectID, nullStr(idx.CollectionID), idx.Name, fieldsJSON,
		idx.TypoTolerance, idx.Status, idx.CreatedAt, idx.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, fmt.Errorf("search: index with name %q already exists", name)
		}
		return nil, fmt.Errorf("search: create index: %w", err)
	}
	return idx, nil
}

// GetIndex fetches an index by ID.
func (s *Service) GetIndex(ctx context.Context, indexID, projectID string) (*Index, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, COALESCE(collection_id,''), name, fields, COALESCE(synonyms,'[]'), COALESCE(ranking_rules,'[]'), typo_tolerance, status, created_at, updated_at
		 FROM search_indexes WHERE id = $1 AND project_id = $2`, indexID, projectID)
	return scanIndex(row)
}

// ListIndexes returns all indexes for a project.
func (s *Service) ListIndexes(ctx context.Context, projectID string) ([]*Index, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, COALESCE(collection_id,''), name, fields, COALESCE(synonyms,'[]'), COALESCE(ranking_rules,'[]'), typo_tolerance, status, created_at, updated_at
		 FROM search_indexes WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Index
	for rows.Next() {
		idx, err := scanIndex(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, idx)
	}
	return out, nil
}

// DeleteIndex deletes an index and all its documents.
func (s *Service) DeleteIndex(ctx context.Context, indexID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM search_indexes WHERE id = $1 AND project_id = $2", indexID, projectID)
	return err
}

// Upsert stores or updates a document in an index.
func (s *Service) Upsert(ctx context.Context, indexID, projectID, docID, content string, metadata map[string]interface{}) (*Document, error) {
	metaJSON, _ := json.Marshal(metadata)
	doc := &Document{
		ID: uid.New(""), IndexID: indexID, DocID: docID, Content: content,
		Metadata: metadata, IndexedAt: time.Now().UTC(),
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO search_documents (id, index_id, project_id, doc_id, content, metadata, indexed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (index_id, doc_id) DO UPDATE SET content=EXCLUDED.content, metadata=EXCLUDED.metadata, indexed_at=EXCLUDED.indexed_at`,
		doc.ID, indexID, projectID, docID, content, nullBytes(metaJSON), doc.IndexedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("search: upsert: %w", err)
	}
	return doc, nil
}

// DeleteDocument removes a document from an index.
func (s *Service) DeleteDocument(ctx context.Context, indexID, docID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM search_documents WHERE index_id = $1 AND doc_id = $2", indexID, docID)
	return err
}

// Query performs a full-text search against an index.
func (s *Service) Query(ctx context.Context, indexID, projectID, q string, limit, offset int) (*SearchResult, error) {
	if limit <= 0 || limit > 500 {
		limit = 20
	}
	start := time.Now()

	// Expand query with synonyms
	q = s.expandSynonyms(ctx, indexID, q)

	// PostgreSQL full-text search using tsvector
	rows, err := s.db.QueryContext(ctx,
		`SELECT doc_id, content, metadata,
		        ts_rank(to_tsvector('english', content), plainto_tsquery('english', $1)) AS score
		 FROM search_documents
		 WHERE index_id = $2
		   AND to_tsvector('english', content) @@ plainto_tsquery('english', $3)
		 ORDER BY score DESC
		 LIMIT $4 OFFSET $5`,
		q, indexID, q, limit, offset,
	)
	if err != nil {
		// FULLTEXT may not be set up — fall back to LIKE
		rows, err = s.db.QueryContext(ctx,
			`SELECT doc_id, content, metadata, 1.0 as score
			 FROM search_documents
			 WHERE index_id = $1 AND content LIKE $2
			 LIMIT $3 OFFSET $4`,
			indexID, "%"+q+"%", limit, offset,
		)
		if err != nil {
			return nil, fmt.Errorf("search: query: %w", err)
		}
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		h := Hit{}
		var metaRaw []byte
		if err := rows.Scan(&h.DocID, &h.Content, &metaRaw, &h.Score); err != nil {
			return nil, err
		}
		if len(metaRaw) > 0 {
			json.Unmarshal(metaRaw, &h.Metadata) //nolint:errcheck
		}
		h.Snippet = snippet(h.Content, q, 160)
		hits = append(hits, h)
	}

	dur := fmt.Sprintf("%.2fms", float64(time.Since(start).Microseconds())/1000)
	return &SearchResult{Total: len(hits), Hits: hits, Duration: dur}, nil
}

// AddSynonym adds a synonym group to an index.
func (s *Service) AddSynonym(ctx context.Context, indexID string, synonyms []string) (*Synonym, error) {
	syn := &Synonym{ID: uid.New(""), IndexID: indexID, Synonyms: synonyms}
	synJSON, _ := json.Marshal(synonyms)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO search_synonyms (id, index_id, synonyms, created_at) VALUES ($1,$2,$3,$4)",
		syn.ID, indexID, synJSON, time.Now().UTC(),
	)
	if err != nil {
		return nil, fmt.Errorf("search: add synonym: %w", err)
	}
	return syn, nil
}

// ListSynonyms returns all synonyms for an index.
func (s *Service) ListSynonyms(ctx context.Context, indexID string) ([]*Synonym, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, index_id, synonyms FROM search_synonyms WHERE index_id = $1", indexID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Synonym
	for rows.Next() {
		syn := &Synonym{}
		var synRaw []byte
		if err := rows.Scan(&syn.ID, &syn.IndexID, &synRaw); err != nil {
			return nil, err
		}
		json.Unmarshal(synRaw, &syn.Synonyms) //nolint:errcheck
		out = append(out, syn)
	}
	return out, nil
}

// DeleteSynonym removes a synonym by ID.
func (s *Service) DeleteSynonym(ctx context.Context, synID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM search_synonyms WHERE id = $1", synID)
	return err
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (s *Service) expandSynonyms(ctx context.Context, indexID, q string) string {
	syns, err := s.ListSynonyms(ctx, indexID)
	if err != nil || len(syns) == 0 {
		return q
	}
	lower := strings.ToLower(q)
	for _, syn := range syns {
		for _, word := range syn.Synonyms {
			if strings.Contains(lower, strings.ToLower(word)) {
				extras := make([]string, 0, len(syn.Synonyms))
				for _, s2 := range syn.Synonyms {
					if !strings.EqualFold(s2, word) {
						extras = append(extras, s2)
					}
				}
				if len(extras) > 0 {
					q = q + " " + strings.Join(extras, " ")
				}
				break
			}
		}
	}
	return q
}

func snippet(content, query string, maxLen int) string {
	lower := strings.ToLower(content)
	idx := strings.Index(lower, strings.ToLower(query))
	if idx < 0 {
		if len(content) > maxLen {
			return content[:maxLen] + "…"
		}
		return content
	}
	start := idx - 60
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 60
	if end > len(content) {
		end = len(content)
	}
	s := content[start:end]
	if start > 0 {
		s = "…" + s
	}
	if end < len(content) {
		s = s + "…"
	}
	return s
}

func scanIndex(row interface {
	Scan(...interface{}) error
}) (*Index, error) {
	idx := &Index{}
	var fieldsRaw, synRaw, rankRaw []byte
	if err := row.Scan(&idx.ID, &idx.ProjectID, &idx.CollectionID, &idx.Name,
		&fieldsRaw, &synRaw, &rankRaw, &idx.TypoTolerance, &idx.Status, &idx.CreatedAt, &idx.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(fieldsRaw, &idx.Fields)     //nolint:errcheck
	json.Unmarshal(synRaw, &idx.Synonyms)      //nolint:errcheck
	json.Unmarshal(rankRaw, &idx.RankingRules) //nolint:errcheck
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
