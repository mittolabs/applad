package databases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

// Content mode turns a regular table into an editorial collection: rows gain a
// draft/published workflow, a slug, a locale (entries of the same logical item
// share an entry_group), and version snapshots. It is the same storage and the
// same rows API — only the behaviour on top differs.

// RowVersion is a point-in-time snapshot of a row in a content-enabled table.
type RowVersion struct {
	ID        string                 `json:"$id"`
	Version   int                    `json:"version"`
	Data      map[string]interface{} `json:"data"`
	AuthorID  string                 `json:"authorId"`
	CreatedAt time.Time              `json:"$createdAt"`
}

// contentColumns are the system fields added when content mode is enabled.
// They behave like id/created_at: present on every row, not part of the
// user-defined column metadata.
var contentColumns = []string{"status", "slug", "locale", "published_at", "entry_group"}

// SetContentMode enables or disables editorial behaviour for a table. Enabling
// adds the system fields (idempotent); disabling only clears the flag, so no
// content is ever destroyed by toggling it off.
func (s *Service) SetContentMode(ctx context.Context, projectID, tableID string, enabled bool) error {
	// Scoped to the caller's project so one tenant cannot alter another's
	// table — the same hole closed for columns and indexes.
	table, err := s.lookupProjectTable(ctx, tableID, projectID)
	if err != nil {
		return err
	}

	if enabled {
		alter := fmt.Sprintf(`ALTER TABLE %s.%s
			ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'draft',
			ADD COLUMN IF NOT EXISTS slug TEXT,
			ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'en',
			ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ,
			ADD COLUMN IF NOT EXISTS entry_group TEXT`,
			pgIdent(table.Schema), pgIdent(table.Name))
		if _, err := s.db.ExecContext(ctx, alter); err != nil {
			return fmt.Errorf("enable content mode: %w", err)
		}
		// Editorial queries filter on these constantly.
		s.db.ExecContext(ctx, fmt.Sprintf( //nolint:errcheck
			"CREATE INDEX IF NOT EXISTS %s ON %s.%s (status)",
			pgIdent("idx_"+table.Name+"_status"), pgIdent(table.Schema), pgIdent(table.Name)))
		s.db.ExecContext(ctx, fmt.Sprintf( //nolint:errcheck
			"CREATE INDEX IF NOT EXISTS %s ON %s.%s (slug)",
			pgIdent("idx_"+table.Name+"_slug"), pgIdent(table.Schema), pgIdent(table.Name)))
		s.db.ExecContext(ctx, fmt.Sprintf( //nolint:errcheck
			"CREATE INDEX IF NOT EXISTS %s ON %s.%s (entry_group, locale)",
			pgIdent("idx_"+table.Name+"_entry"), pgIdent(table.Schema), pgIdent(table.Name)))
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE tables SET content_enabled = $1, updated_at = NOW() WHERE id = $2", enabled, tableID)
	return err
}

// ContentEnabled reports whether a table is in content mode.
func (s *Service) ContentEnabled(ctx context.Context, tableID string) (bool, error) {
	var enabled bool
	err := s.db.QueryRowContext(ctx,
		"SELECT content_enabled FROM tables WHERE id = $1", tableID).Scan(&enabled)
	return enabled, err
}

// SetRowPublished flips a row between draft and published.
func (s *Service) SetRowPublished(ctx context.Context, projectID, tableID, rowID string, published bool) error {
	table, err := s.lookupProjectTable(ctx, tableID, projectID)
	if err != nil {
		return err
	}
	q := fmt.Sprintf("UPDATE %s.%s SET status='draft', published_at=NULL WHERE id=$1",
		pgIdent(table.Schema), pgIdent(table.Name))
	if published {
		q = fmt.Sprintf("UPDATE %s.%s SET status='published', published_at=NOW() WHERE id=$1",
			pgIdent(table.Schema), pgIdent(table.Name))
	}
	res, err := s.db.ExecContext(ctx, q, rowID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("row not found")
	}
	return nil
}

// RecordRowVersion snapshots a row. No-op for tables that are not in content
// mode, so callers can invoke it unconditionally after a write.
func (s *Service) RecordRowVersion(ctx context.Context, tableID, rowID, authorID string, data map[string]interface{}) {
	enabled, err := s.ContentEnabled(ctx, tableID)
	if err != nil || !enabled {
		return
	}
	var next int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version),0)+1 FROM row_versions WHERE table_id=$1 AND row_id=$2",
		tableID, rowID,
	).Scan(&next); err != nil || next < 1 {
		next = 1
	}
	payload, _ := json.Marshal(data)
	s.db.ExecContext(ctx, //nolint:errcheck
		`INSERT INTO row_versions (id, table_id, row_id, version, data, author_id)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		uid.New(""), tableID, rowID, next, payload, authorID)
}

// ListRowVersions returns a row's snapshots, newest first.
func (s *Service) ListRowVersions(ctx context.Context, tableID, rowID string) ([]RowVersion, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, version, data, author_id, created_at
		   FROM row_versions WHERE table_id=$1 AND row_id=$2 ORDER BY version DESC LIMIT 50`,
		tableID, rowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RowVersion{}
	for rows.Next() {
		var v RowVersion
		var raw []byte
		if err := rows.Scan(&v.ID, &v.Version, &raw, &v.AuthorID, &v.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(raw, &v.Data) //nolint:errcheck
		out = append(out, v)
	}
	return out, nil
}
