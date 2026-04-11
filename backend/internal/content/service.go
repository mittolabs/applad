// Package content implements a CMS layer with content types, versioned entries,
// draft/publish workflow, and optional localization.
package content

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Field describes a single field within a ContentType.
type Field struct {
	Key      string      `json:"key"`
	Label    string      `json:"label"`
	Type     string      `json:"type"` // text | richtext | number | boolean | date | media | reference | slug | seo
	Required bool        `json:"required"`
	Default  interface{} `json:"default,omitempty"`
	Options  []string    `json:"options,omitempty"`
}

// ContentType is a schema definition for content entries.
type ContentType struct {
	ID           string    `json:"$id"`
	ProjectID    string    `json:"projectId"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Fields       []Field   `json:"fields"`
	Versioning   bool      `json:"versioning"`
	Localization bool      `json:"localization"`
	CreatedAt    time.Time `json:"$createdAt"`
	UpdatedAt    time.Time `json:"$updatedAt"`
}

// Entry is a content entry (instance of a ContentType).
type Entry struct {
	ID          string                 `json:"$id"`
	TypeID      string                 `json:"typeId"`
	ProjectID   string                 `json:"projectId"`
	Slug        string                 `json:"slug,omitempty"`
	Status      string                 `json:"status"` // draft | published | archived
	Locale      string                 `json:"locale"`
	AuthorID    string                 `json:"authorId,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Version     int                    `json:"version"`
	PublishedAt *time.Time             `json:"publishedAt,omitempty"`
	CreatedAt   time.Time              `json:"$createdAt"`
	UpdatedAt   time.Time              `json:"$updatedAt"`
}

// Version is a historical snapshot of an entry's data.
type Version struct {
	ID        string                 `json:"$id"`
	EntryID   string                 `json:"entryId"`
	Version   int                    `json:"version"`
	Data      map[string]interface{} `json:"data"`
	CreatedBy string                 `json:"createdBy,omitempty"`
	CreatedAt time.Time              `json:"$createdAt"`
}

// Service manages content types and entries.
type Service struct {
	db *db.DB
}

// NewService creates a new content Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ── Content Types ─────────────────────────────────────────────────────────────

// CreateType creates a new content type schema.
func (s *Service) CreateType(ctx context.Context, projectID, name, slug string, fields []Field, versioning, localization bool) (*ContentType, error) {
	ct := &ContentType{
		ID: uid.New(""), ProjectID: projectID, Name: name, Slug: slug,
		Fields: fields, Versioning: versioning, Localization: localization,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	fieldsJSON, _ := json.Marshal(fields)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO content_types (id, project_id, name, slug, fields, versioning, localization, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		ct.ID, ct.ProjectID, ct.Name, ct.Slug, fieldsJSON,
		ct.Versioning, ct.Localization,
		ct.CreatedAt, ct.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return nil, fmt.Errorf("content: type with slug %q already exists", slug)
		}
		return nil, fmt.Errorf("content: create type: %w", err)
	}
	return ct, nil
}

// GetType fetches a content type by ID.
func (s *Service) GetType(ctx context.Context, typeID, projectID string) (*ContentType, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, slug, fields, versioning, localization, created_at, updated_at
		 FROM content_types WHERE id = $1 AND project_id = $2`, typeID, projectID)
	return scanType(row)
}

// ListTypes returns all content types for a project.
func (s *Service) ListTypes(ctx context.Context, projectID string) ([]*ContentType, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, slug, fields, versioning, localization, created_at, updated_at
		 FROM content_types WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ContentType
	for rows.Next() {
		ct, err := scanType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, nil
}

// UpdateType patches a content type's fields.
func (s *Service) UpdateType(ctx context.Context, typeID, projectID, name string, fields []Field) (*ContentType, error) {
	ct, err := s.GetType(ctx, typeID, projectID)
	if err != nil {
		return nil, err
	}
	if name != "" {
		ct.Name = name
	}
	if len(fields) > 0 {
		ct.Fields = fields
	}
	fieldsJSON, _ := json.Marshal(ct.Fields)
	_, err = s.db.ExecContext(ctx,
		"UPDATE content_types SET name=$1, fields=$2, updated_at=$3 WHERE id=$4",
		ct.Name, fieldsJSON, time.Now().UTC(), ct.ID)
	return ct, err
}

// DeleteType deletes a content type and all its entries.
func (s *Service) DeleteType(ctx context.Context, typeID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM content_types WHERE id = $1 AND project_id = $2", typeID, projectID)
	return err
}

// ── Entries ───────────────────────────────────────────────────────────────────

// CreateEntry creates a new content entry (starts as draft).
func (s *Service) CreateEntry(ctx context.Context, typeID, projectID, slug, locale, authorID string, data map[string]interface{}) (*Entry, error) {
	if locale == "" {
		locale = "en"
	}
	entry := &Entry{
		ID: uid.New(""), TypeID: typeID, ProjectID: projectID,
		Slug: slug, Status: "draft", Locale: locale, AuthorID: authorID,
		Data: data, Version: 1,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx,
		`INSERT INTO content_entries (id, type_id, project_id, slug, status, locale, author_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		entry.ID, entry.TypeID, entry.ProjectID, nullStr(entry.Slug),
		entry.Status, entry.Locale, nullStr(entry.AuthorID),
		entry.CreatedAt, entry.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("content: create entry: %w", err)
	}

	// Create initial version
	dataJSON, _ := json.Marshal(data)
	_, err = tx.ExecContext(ctx,
		"INSERT INTO content_versions (id, entry_id, version, data, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6)",
		uid.New(""), entry.ID, 1, dataJSON, nullStr(authorID), entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("content: create version: %w", err)
	}
	return entry, tx.Commit()
}

// GetEntry fetches an entry by ID with its latest version data.
func (s *Service) GetEntry(ctx context.Context, entryID, projectID string) (*Entry, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT e.id, e.type_id, e.project_id, COALESCE(e.slug,''), e.status, e.locale, COALESCE(e.author_id,''),
		        COALESCE(v.data,'{}'), COALESCE(v.version,1), e.published_at, e.created_at, e.updated_at
		 FROM content_entries e
		 LEFT JOIN content_versions v ON v.entry_id = e.id AND v.version = (SELECT MAX(version) FROM content_versions WHERE entry_id = e.id)
		 WHERE e.id = $1 AND e.project_id = $2`, entryID, projectID)
	return scanEntry(row)
}

// ListEntries returns entries for a content type with optional status/locale filter.
func (s *Service) ListEntries(ctx context.Context, typeID, projectID, status, locale string, limit, offset int) ([]*Entry, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	// Build dynamic WHERE clause with numbered placeholders.
	args := []interface{}{typeID, projectID}
	where := "e.type_id = $1 AND e.project_id = $2"
	n := 3
	if status != "" {
		where += fmt.Sprintf(" AND e.status = $%d", n)
		args = append(args, status)
		n++
	}
	if locale != "" {
		where += fmt.Sprintf(" AND e.locale = $%d", n)
		args = append(args, locale)
		n++
	}

	var total int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM content_entries e WHERE "+where, args...).Scan(&total) //nolint:errcheck

	listArgs := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		`SELECT e.id, e.type_id, e.project_id, COALESCE(e.slug,''), e.status, e.locale, COALESCE(e.author_id,''),
		        COALESCE(v.data,'{}'), COALESCE(v.version,1), e.published_at, e.created_at, e.updated_at
		 FROM content_entries e
		 LEFT JOIN content_versions v ON v.entry_id = e.id AND v.version = (SELECT MAX(version) FROM content_versions WHERE entry_id = e.id)
		 WHERE `+where+fmt.Sprintf(` ORDER BY e.created_at DESC LIMIT $%d OFFSET $%d`, n, n+1), listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, nil
}

// UpdateEntry creates a new version of an entry's data.
func (s *Service) UpdateEntry(ctx context.Context, entryID, projectID, authorID string, data map[string]interface{}) (*Entry, error) {
	entry, err := s.GetEntry(ctx, entryID, projectID)
	if err != nil {
		return nil, err
	}
	newVer := entry.Version + 1
	dataJSON, _ := json.Marshal(data)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	_, err = tx.ExecContext(ctx,
		"INSERT INTO content_versions (id, entry_id, version, data, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6)",
		uid.New(""), entryID, newVer, dataJSON, nullStr(authorID), time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE content_entries SET updated_at=$1 WHERE id=$2", time.Now().UTC(), entryID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	entry.Version = newVer
	entry.Data = data
	return entry, nil
}

// PublishEntry changes an entry's status to published.
func (s *Service) PublishEntry(ctx context.Context, entryID, projectID string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		"UPDATE content_entries SET status='published', published_at=$1, updated_at=$2 WHERE id=$3 AND project_id=$4",
		now, now, entryID, projectID)
	return err
}

// UnpublishEntry reverts a published entry back to draft.
func (s *Service) UnpublishEntry(ctx context.Context, entryID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE content_entries SET status='draft', updated_at=$1 WHERE id=$2 AND project_id=$3",
		time.Now().UTC(), entryID, projectID)
	return err
}

// DeleteEntry deletes an entry and all its versions.
func (s *Service) DeleteEntry(ctx context.Context, entryID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM content_entries WHERE id=$1 AND project_id=$2", entryID, projectID)
	return err
}

// ListVersions returns all versions for an entry.
func (s *Service) ListVersions(ctx context.Context, entryID, projectID string) ([]*Version, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT v.id, v.entry_id, v.version, v.data, COALESCE(v.created_by,''), v.created_at
		 FROM content_versions v
		 JOIN content_entries e ON e.id = v.entry_id
		 WHERE v.entry_id = $1 AND e.project_id = $2
		 ORDER BY v.version DESC`, entryID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Version
	for rows.Next() {
		v := &Version{}
		var dataRaw []byte
		if err := rows.Scan(&v.ID, &v.EntryID, &v.Version, &dataRaw, &v.CreatedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(dataRaw, &v.Data) //nolint:errcheck
		out = append(out, v)
	}
	return out, nil
}

// ── scanners ──────────────────────────────────────────────────────────────────

func scanType(row interface{ Scan(...interface{}) error }) (*ContentType, error) {
	ct := &ContentType{}
	var fieldsRaw []byte
	if err := row.Scan(&ct.ID, &ct.ProjectID, &ct.Name, &ct.Slug, &fieldsRaw,
		&ct.Versioning, &ct.Localization, &ct.CreatedAt, &ct.UpdatedAt); err != nil {
		return nil, err
	}
	json.Unmarshal(fieldsRaw, &ct.Fields) //nolint:errcheck
	return ct, nil
}

func scanEntry(row interface{ Scan(...interface{}) error }) (*Entry, error) {
	e := &Entry{}
	var dataRaw []byte
	if err := row.Scan(&e.ID, &e.TypeID, &e.ProjectID, &e.Slug, &e.Status, &e.Locale,
		&e.AuthorID, &dataRaw, &e.Version, &e.PublishedAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	if len(dataRaw) > 0 {
		json.Unmarshal(dataRaw, &e.Data) //nolint:errcheck
	}
	return e, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
