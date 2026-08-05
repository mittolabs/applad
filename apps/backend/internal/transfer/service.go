package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/auth"
	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/storage"
)

// keyValidator proves a caller may read a source project on this instance by
// presenting a valid API key for it. projects.Service satisfies this.
type keyValidator interface {
	GetKeyBySecret(ctx context.Context, secret string) (*model.APIKey, error)
}

// QueueName is the Redis queue the transfer worker consumes.
const QueueName = "transfer"

// SupportedSources lists the source platforms that can currently execute. The
// list grows as adapters land (Supabase, Appwrite, NHost, Firebase).
var SupportedSources = map[string]bool{
	"applad":   true,
	"supabase": true,
	"appwrite": true,
	"nhost":    true,
	"firebase": true,
}

// Service is the API-facing layer for data migrations: it validates sources,
// stores jobs (with encrypted credentials), enqueues them, and — in the worker —
// executes them by wiring a Source to the Applad Destination.
type Service struct {
	store *Store
	db    *db.DB
	auth  *auth.Service
	dbs   *databases.Service
	stg   *storage.Service
	keys  keyValidator
	queue *queue.Queue
}

func NewService(database *db.DB, a *auth.Service, d *databases.Service, s *storage.Service, keys keyValidator, q *queue.Queue) *Service {
	return &Service{store: NewStore(database), db: database, auth: a, dbs: d, stg: s, keys: keys, queue: q}
}

func defaultGroups(groups []Group) []Group {
	if len(groups) > 0 {
		return groups
	}
	return []Group{GroupAuth, GroupDatabases, GroupStorage}
}

// Validate builds the source from the supplied credentials, connects, and
// returns the count of available resources per group without persisting
// anything. This is the pre-flight step the console shows before "Start".
func (s *Service) Validate(ctx context.Context, sourceType string, groups []Group, creds map[string]any) (map[Group]int, error) {
	if !SupportedSources[sourceType] {
		return nil, fmt.Errorf("transfer: source %q is not yet supported", sourceType)
	}
	credsJSON, _ := json.Marshal(creds)
	src, err := s.buildSource(ctx, sourceType, string(credsJSON))
	if err != nil {
		return nil, err
	}
	defer src.Close()
	return src.Report(ctx, defaultGroups(groups))
}

// Create records a new migration job (encrypting its credentials at rest) and
// enqueues it for the transfer worker.
func (s *Service) Create(ctx context.Context, projectID, sourceType string, groups []Group, options map[string]any, creds map[string]any) (*Migration, error) {
	if !SupportedSources[sourceType] {
		return nil, fmt.Errorf("transfer: source %q is not yet supported", sourceType)
	}
	groups = defaultGroups(groups)
	credsJSON, _ := json.Marshal(creds)

	id, err := s.store.Create(ctx, projectID, sourceType, groups, options, string(credsJSON))
	if err != nil {
		return nil, err
	}
	if s.queue != nil {
		if err := s.queue.Push(ctx, QueueName, queue.Job{
			ID:        id,
			Type:      "data_import",
			Payload:   map[string]interface{}{"migrationId": id, "projectId": projectID},
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			// Enqueue failed: mark the job failed, which also clears the stored
			// credentials so they do not linger un-run.
			_ = s.store.Finish(ctx, id, "failed", "could not enqueue job")
			return nil, fmt.Errorf("transfer: enqueue: %w", err)
		}
	}
	return s.store.Get(ctx, id, projectID)
}

func (s *Service) Get(ctx context.Context, id, projectID string) (*Migration, error) {
	return s.store.Get(ctx, id, projectID)
}

func (s *Service) List(ctx context.Context, projectID string, limit, offset int) ([]*Migration, int, error) {
	return s.store.List(ctx, projectID, limit, offset)
}

func (s *Service) Cancel(ctx context.Context, id, projectID string) error {
	return s.store.Cancel(ctx, id, projectID)
}

// ExportStream drives the same-instance Applad reader for projectID and hands
// every resource (including password credentials) to emit. It powers the
// cross-instance export endpoint: a destination instance calls this on the
// source instance, authenticated by an API key for projectID, to pull a full
// project — the one thing the public API withholds is password hashes, which
// this authenticated, same-project export includes.
func (s *Service) ExportStream(ctx context.Context, projectID string, groups []Group, emit Emit) error {
	src := NewAppladSource(projectID, s.db, s.dbs, s.stg)
	defer src.Close()
	return src.Export(ctx, defaultGroups(groups), emit)
}

// ExportReport returns the per-group resource counts for projectID (the
// cross-instance preflight).
func (s *Service) ExportReport(ctx context.Context, projectID string, groups []Group) (map[Group]int, error) {
	src := NewAppladSource(projectID, s.db, s.dbs, s.stg)
	defer src.Close()
	return src.Report(ctx, defaultGroups(groups))
}

// maxJobDuration bounds a single migration. A hostile source endpoint could
// otherwise stream pages forever (a constant nextPageToken, or always-full
// pages) and pin a worker; the deadline guarantees the job ends and the receipt
// is released regardless of how the remote behaves.
const maxJobDuration = 6 * time.Hour

// RunJob executes a queued migration. Called by the transfer worker. It loads
// the job, decrypts its credentials, builds the source and the Applad
// destination for the target project, and runs the orchestrator.
func (s *Service) RunJob(ctx context.Context, migrationID string) error {
	ctx, cancel := context.WithTimeout(ctx, maxJobDuration)
	defer cancel()
	// Only one worker may run a given migration at a time. The queue's
	// visibility timeout can redeliver a long-running job to a second worker; a
	// Postgres advisory lock (held on a dedicated connection for the duration)
	// makes the redelivery a no-op instead of a concurrent double-import.
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	lockKey := advisoryLockKey(migrationID)
	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		// Another worker holds this migration; let it run.
		return nil
	}
	defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey)

	// The worker has no project scope; read the row unscoped to learn its project.
	var projectID, sourceType, status string
	if err := s.db.QueryRowContext(ctx,
		"SELECT project_id, source_type, status FROM data_migrations WHERE id = $1",
		migrationID).Scan(&projectID, &sourceType, &status); err != nil {
		return fmt.Errorf("transfer: load job: %w", err)
	}
	// Do not re-run a job that already reached a terminal state (a redelivered or
	// duplicate message must not re-import finished data).
	if status == "cancelled" || status == "completed" || status == "failed" {
		return nil
	}
	m, err := s.store.Get(ctx, migrationID, projectID)
	if err != nil {
		return err
	}
	credsJSON, err := s.store.Credentials(ctx, migrationID)
	if err != nil {
		return fmt.Errorf("transfer: read credentials: %w", err)
	}

	src, err := s.buildSource(ctx, sourceType, credsJSON)
	if err != nil {
		return s.store.Finish(ctx, migrationID, "failed", err.Error())
	}
	defer src.Close()

	dst := NewAppladDestination(projectID, s.auth, s.dbs, s.stg)
	defer dst.Close()

	return NewTransfer(s.store, src, dst, migrationID, m.Groups).Run(ctx)
}

// buildSource constructs a Source for sourceType from its credentials JSON.
// New source adapters (supabase, appwrite, nhost, firebase) register a case here.
func (s *Service) buildSource(ctx context.Context, sourceType, credsJSON string) (Source, error) {
	var creds map[string]any
	if credsJSON != "" {
		_ = json.Unmarshal([]byte(credsJSON), &creds)
	}
	switch sourceType {
	case "applad":
		sourceProjectID, _ := creds["sourceProjectId"].(string)
		sourceAPIKey, _ := creds["sourceApiKey"].(string)
		// A remote endpoint switches to the cross-instance reader (cloud <->
		// self-hosted). The remote instance authenticates the API key against its
		// own project, so no local key check applies here.
		if endpoint, _ := creds["endpoint"].(string); strings.TrimSpace(endpoint) != "" {
			return NewRemoteAppladSource(endpoint, sourceProjectID, sourceAPIKey)
		}
		if sourceProjectID == "" || sourceAPIKey == "" {
			return nil, fmt.Errorf("transfer: applad source requires sourceProjectId and sourceApiKey")
		}
		// Authorization: reading a project's data (users, hashes, rows, files)
		// requires holding a valid API key for THAT project. Without this check a
		// caller could name any project as the source and exfiltrate it into
		// their own — a cross-tenant leak.
		if s.keys == nil {
			return nil, fmt.Errorf("transfer: source authorization unavailable")
		}
		key, err := s.keys.GetKeyBySecret(ctx, sourceAPIKey)
		if err != nil || key == nil || key.ProjectID != sourceProjectID {
			return nil, fmt.Errorf("transfer: invalid source API key for project %s", sourceProjectID)
		}
		if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
			return nil, fmt.Errorf("transfer: source API key is expired")
		}
		return NewAppladSource(sourceProjectID, s.db, s.dbs, s.stg), nil
	case "supabase":
		return NewSupabaseSource(creds)
	case "appwrite":
		return NewAppwriteSource(creds)
	case "nhost":
		return NewNhostSource(creds)
	case "firebase":
		return NewFirebaseSource(creds)
	default:
		return nil, fmt.Errorf("transfer: source %q is not yet supported", sourceType)
	}
}

// advisoryLockKey derives a stable 63-bit key for pg_advisory_lock from a
// migration ID (kept positive to sit in a bigint without surprises).
func advisoryLockKey(id string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("data_migration:" + id))
	return int64(h.Sum64() & 0x7fffffffffffffff)
}
