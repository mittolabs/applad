package transfer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/credentials"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// GroupCount is the per-group progress tally surfaced to the console.
type GroupCount struct {
	Total   int `json:"total"`
	Done    int `json:"done"`
	Error   int `json:"error"`
	Warning int `json:"warning"`
	Skip    int `json:"skip"`
}

// Migration is one import job. Credentials are never included here; they live
// encrypted in the row and are only decrypted for the worker via Credentials().
type Migration struct {
	ID         string                `json:"id"`
	ProjectID  string                `json:"projectId"`
	SourceType string                `json:"sourceType"`
	Status     string                `json:"status"`
	Groups     []Group               `json:"groups"`
	Options    map[string]any        `json:"options"`
	Counts     map[string]GroupCount `json:"counts"`
	Error      string                `json:"error,omitempty"`
	CreatedAt  time.Time             `json:"createdAt"`
	UpdatedAt  time.Time             `json:"updatedAt"`
	StartedAt  *time.Time            `json:"startedAt,omitempty"`
	FinishedAt *time.Time            `json:"finishedAt,omitempty"`
}

// Store is the persistence layer for data migrations.
type Store struct {
	db *db.DB
}

func NewStore(database *db.DB) *Store { return &Store{db: database} }

// Create inserts a new pending migration, encrypting the source credentials at
// rest with the credential vault key. Returns the generated ID.
func (s *Store) Create(ctx context.Context, projectID, sourceType string, groups []Group, options map[string]any, credsJSON string) (string, error) {
	id := uid.New("")
	groupsJSON, _ := json.Marshal(groups)
	if options == nil {
		options = map[string]any{}
	}
	optsJSON, _ := json.Marshal(options)

	var encCreds sql.NullString
	if credsJSON != "" {
		enc, err := credentials.EncryptSecret(credsJSON)
		if err != nil {
			return "", fmt.Errorf("transfer: encrypt credentials: %w", err)
		}
		encCreds = sql.NullString{String: enc, Valid: true}
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO data_migrations (id, project_id, source_type, status, groups, options, credentials, counts)
		 VALUES ($1, $2, $3, 'pending', $4, $5, $6, '{}')`,
		id, projectID, sourceType, groupsJSON, optsJSON, encCreds)
	if err != nil {
		return "", fmt.Errorf("transfer: create migration: %w", err)
	}
	return id, nil
}

// Get returns a migration scoped to its project. Credentials are never returned.
func (s *Store) Get(ctx context.Context, id, projectID string) (*Migration, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, source_type, status, groups, options, counts,
		        COALESCE(error,''), created_at, updated_at, started_at, finished_at
		 FROM data_migrations WHERE id = $1 AND project_id = $2`, id, projectID)
	return scanMigration(row)
}

// List returns a project's migrations, newest first.
func (s *Store) List(ctx context.Context, projectID string, limit, offset int) ([]*Migration, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM data_migrations WHERE project_id = $1", projectID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, source_type, status, groups, options, counts,
		        COALESCE(error,''), created_at, updated_at, started_at, finished_at
		 FROM data_migrations WHERE project_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Migration
	for rows.Next() {
		m, err := scanMigration(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanMigration(row scanner) (*Migration, error) {
	var m Migration
	var groupsJSON, optsJSON, countsJSON []byte
	var startedAt, finishedAt sql.NullTime
	if err := row.Scan(&m.ID, &m.ProjectID, &m.SourceType, &m.Status,
		&groupsJSON, &optsJSON, &countsJSON, &m.Error,
		&m.CreatedAt, &m.UpdatedAt, &startedAt, &finishedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(groupsJSON, &m.Groups)
	_ = json.Unmarshal(optsJSON, &m.Options)
	_ = json.Unmarshal(countsJSON, &m.Counts)
	if m.Counts == nil {
		m.Counts = map[string]GroupCount{}
	}
	if startedAt.Valid {
		m.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		m.FinishedAt = &finishedAt.Time
	}
	return &m, nil
}

// Credentials decrypts and returns the stored source credentials JSON for the
// worker. Returns "" if none were stored or they have been cleared.
func (s *Store) Credentials(ctx context.Context, id string) (string, error) {
	var enc sql.NullString
	if err := s.db.QueryRowContext(ctx,
		"SELECT credentials FROM data_migrations WHERE id = $1", id).Scan(&enc); err != nil {
		return "", err
	}
	if !enc.Valid || enc.String == "" {
		return "", nil
	}
	return credentials.DecryptSecret(enc.String)
}

// MarkRunning transitions a job to running and stamps started_at.
func (s *Store) MarkRunning(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE data_migrations SET status='running', started_at=NOW(), updated_at=NOW() WHERE id=$1", id)
	return err
}

// Finish sets the terminal status, stores any error, and clears the stored
// credentials so secrets do not linger past the job.
func (s *Store) Finish(ctx context.Context, id, status, errMsg string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE data_migrations
		 SET status=$2, error=NULLIF($3,''), credentials=NULL, finished_at=NOW(), updated_at=NOW()
		 WHERE id=$1`, id, status, errMsg)
	return err
}

// SetCounts persists the current per-group progress tally.
func (s *Store) SetCounts(ctx context.Context, id string, counts map[string]GroupCount) error {
	b, _ := json.Marshal(counts)
	_, err := s.db.ExecContext(ctx,
		"UPDATE data_migrations SET counts=$2, updated_at=NOW() WHERE id=$1", id, b)
	return err
}

// Status returns the current status string (used to detect cancellation).
func (s *Store) Status(ctx context.Context, id string) (string, error) {
	var st string
	err := s.db.QueryRowContext(ctx, "SELECT status FROM data_migrations WHERE id=$1", id).Scan(&st)
	return st, err
}

// Cancel requests cancellation of a pending/running job.
func (s *Store) Cancel(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE data_migrations SET status='cancelled', updated_at=NOW()
		 WHERE id=$1 AND project_id=$2 AND status IN ('pending','running')`, id, projectID)
	return err
}

// RecordResource upserts the per-resource status row (idempotent on the logical
// resource). Successful bulk rows are not persisted individually by the caller to
// keep the table bounded; failures and non-bulk resources are.
func (s *Store) RecordResource(ctx context.Context, migrationID string, grp Group, resourceType, sourceID, destID, status, message string) error {
	id := uid.New("")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO data_migration_resources (id, migration_id, grp, resource_type, source_id, dest_id, status, message)
		 VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),$7,NULLIF($8,''))
		 ON CONFLICT (migration_id, grp, resource_type, source_id)
		 DO UPDATE SET dest_id=EXCLUDED.dest_id, status=EXCLUDED.status, message=EXCLUDED.message, updated_at=NOW()`,
		id, migrationID, string(grp), resourceType, sourceID, destID, status, message)
	return err
}
