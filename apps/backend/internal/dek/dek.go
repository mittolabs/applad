// Package dek manages per-project data-encryption keys ("DEKs") for
// field-level encryption at rest. Each project gets its own 32-byte AES-256
// key, generated once and stored wrapped ("enveloped") by the instance-wide
// MASTER_ENCRYPTION_KEY in the project_encryption_keys table — the server
// never persists a DEK in the clear. internal/databases and internal/storage
// both depend on this package rather than each managing their own key
// material.
//
// This is server-side envelope encryption, not end-to-end encryption: the
// backend holds the master key and can decrypt on behalf of an authorized
// request. It protects data if the database, disk, or a backup is
// compromised without the master key alongside it.
package dek

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	appladcrypto "github.com/mittolabs/applad/internal/crypto"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

const (
	tokenPrefix = "dek"
	cacheTTL    = 5 * time.Minute
)

// ErrDisabled is returned by every operation when no master key is
// configured. Field/bucket encryption is opt-in per column/bucket, so an
// instance that never uses it boots fine without MASTER_ENCRYPTION_KEY —
// this error only surfaces when something actually tries to use the feature.
var ErrDisabled = fmt.Errorf("dek: field encryption is not configured on this instance — set MASTER_ENCRYPTION_KEY")

type cachedDEK struct {
	key       []byte
	version   int
	expiresAt time.Time
}

// Service manages per-project DEKs: provisioning, wrap/unwrap, and rotation.
// DEKs are cached in-process only (never in Redis or on disk) for a short TTL
// once unwrapped, bounding how long plaintext key material sits in memory.
type Service struct {
	db         *db.DB
	masterKey  []byte // nil when disabled (Enabled() reports false)
	kekVersion int

	mu           sync.Mutex
	cacheActive  map[string]cachedDEK // projectID -> active DEK
	cacheVersion map[string]cachedDEK // "projectID:vN" -> that DEK version
}

// NewService constructs a dek.Service. masterKeyRaw is the raw
// MASTER_ENCRYPTION_KEY env value; pass "" to construct a disabled service
// (every method but Enabled then returns ErrDisabled). A non-empty value that
// fails validation is returned as an error — callers should treat that as
// fatal at boot, since a present-but-broken key is worse than an absent one.
func NewService(database *db.DB, masterKeyRaw string) (*Service, error) {
	s := &Service{
		db:           database,
		cacheActive:  make(map[string]cachedDEK),
		cacheVersion: make(map[string]cachedDEK),
	}
	if masterKeyRaw == "" {
		return s, nil
	}
	key, err := ParseMasterKey(masterKeyRaw)
	if err != nil {
		return nil, err
	}
	s.masterKey = key
	s.kekVersion = 1
	return s, nil
}

// Enabled reports whether a master key is configured.
func (s *Service) Enabled() bool { return s != nil && s.masterKey != nil }

// ParseMasterKey validates and normalizes a MASTER_ENCRYPTION_KEY value into a
// 32-byte AES-256 key. A hex-encoded value decoding to at least 32 bytes is
// preferred (matches `openssl rand -hex 32`, and carries full entropy per
// byte); otherwise the raw string must be at least 32 bytes and its first 32
// bytes are used directly, the same fallback CREDENTIALS_ENCRYPTION_KEY
// already uses elsewhere in this codebase.
func ParseMasterKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) >= 32 {
		return decoded[:32], nil
	}
	if len(trimmed) < 32 {
		return nil, fmt.Errorf("dek: MASTER_ENCRYPTION_KEY must be at least 32 bytes, or a hex string decoding to at least 32 bytes (got %d bytes)", len(trimmed))
	}
	key := make([]byte, 32)
	copy(key, []byte(trimmed))
	return key, nil
}

// EnsureProjectKey creates an active DEK for projectID if none exists yet.
// Idempotent and safe under concurrent callers: a race to create the first
// key relies on the (project_id, key_version) unique constraint to let only
// one insert win.
func (s *Service) EnsureProjectKey(ctx context.Context, projectID string) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_encryption_keys WHERE project_id = $1 AND status = 'active')`,
		projectID).Scan(&exists); err != nil {
		return fmt.Errorf("dek: check existing key: %w", err)
	}
	if exists {
		return nil
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Errorf("dek: generate dek: %w", err)
	}
	wrapped, err := appladcrypto.SealToken(tokenPrefix, s.kekVersion, s.masterKey, raw)
	if err != nil {
		return fmt.Errorf("dek: wrap dek: %w", err)
	}
	id := uid.New("unique()")
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO project_encryption_keys (id, project_id, key_version, kek_version, wrapped_dek, status)
		 VALUES ($1, $2, 1, $3, $4, 'active')
		 ON CONFLICT (project_id, key_version) DO NOTHING`,
		id, projectID, s.kekVersion, wrapped)
	if err != nil {
		return fmt.Errorf("dek: insert dek: %w", err)
	}
	return nil
}

// Unwrap returns the active DEK and its key_version for projectID, unwrapping
// from project_encryption_keys and caching briefly in-process. Returns an
// error if the project has no active key yet — callers that write encrypted
// data must call EnsureProjectKey first.
func (s *Service) Unwrap(ctx context.Context, projectID string) ([]byte, int, error) {
	if !s.Enabled() {
		return nil, 0, ErrDisabled
	}
	if c, ok := s.getCached(s.cacheActive, projectID); ok {
		return c.key, c.version, nil
	}

	var version int
	var wrapped string
	err := s.db.QueryRowContext(ctx,
		`SELECT key_version, wrapped_dek FROM project_encryption_keys WHERE project_id = $1 AND status = 'active'`,
		projectID).Scan(&version, &wrapped)
	if err == sql.ErrNoRows {
		return nil, 0, fmt.Errorf("dek: no active encryption key for project %s", projectID)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("dek: query active key: %w", err)
	}
	key, err := s.unwrapToken(wrapped)
	if err != nil {
		return nil, 0, err
	}
	s.setCached(s.cacheActive, projectID, cachedDEK{key: key, version: version, expiresAt: time.Now().Add(cacheTTL)})
	return key, version, nil
}

// UnwrapVersion returns a specific (possibly retired) DEK version, used to
// decrypt ciphertext written before a rotation.
func (s *Service) UnwrapVersion(ctx context.Context, projectID string, version int) ([]byte, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	cacheKey := fmt.Sprintf("%s:v%d", projectID, version)
	if c, ok := s.getCached(s.cacheVersion, cacheKey); ok {
		return c.key, nil
	}

	var wrapped string
	err := s.db.QueryRowContext(ctx,
		`SELECT wrapped_dek FROM project_encryption_keys WHERE project_id = $1 AND key_version = $2`,
		projectID, version).Scan(&wrapped)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dek: no encryption key version %d for project %s", version, projectID)
	}
	if err != nil {
		return nil, fmt.Errorf("dek: query key version: %w", err)
	}
	key, err := s.unwrapToken(wrapped)
	if err != nil {
		return nil, err
	}
	s.setCached(s.cacheVersion, cacheKey, cachedDEK{key: key, expiresAt: time.Now().Add(cacheTTL)})
	return key, nil
}

// RotateProjectKey issues a new active DEK for projectID, retiring the
// current one. Existing ciphertext under the old DEK remains readable via
// UnwrapVersion — v1 performs no live re-encryption of already-written data;
// that is a separate, explicitly deferred maintenance job.
func (s *Service) RotateProjectKey(ctx context.Context, projectID string) (int, error) {
	if !s.Enabled() {
		return 0, ErrDisabled
	}
	var currentVersion int
	err := s.db.QueryRowContext(ctx,
		`SELECT key_version FROM project_encryption_keys WHERE project_id = $1 AND status = 'active'`,
		projectID).Scan(&currentVersion)
	if err == sql.ErrNoRows {
		if err := s.EnsureProjectKey(ctx, projectID); err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("dek: query active key: %w", err)
	}

	newVersion := currentVersion + 1
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return 0, fmt.Errorf("dek: generate dek: %w", err)
	}
	wrapped, err := appladcrypto.SealToken(tokenPrefix, s.kekVersion, s.masterKey, raw)
	if err != nil {
		return 0, fmt.Errorf("dek: wrap dek: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("dek: begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`UPDATE project_encryption_keys SET status = 'retired', updated_at = NOW() WHERE project_id = $1 AND key_version = $2`,
		projectID, currentVersion); err != nil {
		return 0, fmt.Errorf("dek: retire key: %w", err)
	}
	id := uid.New("unique()")
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO project_encryption_keys (id, project_id, key_version, kek_version, wrapped_dek, status)
		 VALUES ($1, $2, $3, $4, $5, 'active')`,
		id, projectID, newVersion, s.kekVersion, wrapped); err != nil {
		return 0, fmt.Errorf("dek: insert new key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("dek: commit: %w", err)
	}

	s.mu.Lock()
	delete(s.cacheActive, projectID)
	s.mu.Unlock()

	return newVersion, nil
}

// RewrapAll re-wraps every project's DEK under the service's current master
// key, assuming existing wrapped_dek tokens were wrapped under
// previousMasterKey. Used when rotating MASTER_ENCRYPTION_KEY itself; cheap
// (O(projects), not O(data)) since only the small wrapped DEK moves, never
// project data itself.
func (s *Service) RewrapAll(ctx context.Context, previousMasterKey []byte) (int, error) {
	if !s.Enabled() {
		return 0, ErrDisabled
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, wrapped_dek FROM project_encryption_keys`)
	if err != nil {
		return 0, fmt.Errorf("dek: list keys: %w", err)
	}
	type row struct{ id, wrapped string }
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.wrapped); err != nil {
			rows.Close()
			return 0, fmt.Errorf("dek: scan: %w", err)
		}
		all = append(all, r)
	}
	rows.Close()

	resolvePrevious := func(int) ([]byte, error) { return previousMasterKey, nil }

	count := 0
	for _, r := range all {
		plaintext, _, err := appladcrypto.OpenToken(tokenPrefix, resolvePrevious, r.wrapped)
		if err != nil {
			return count, fmt.Errorf("dek: unwrap %s during rewrap: %w", r.id, err)
		}
		newWrapped, err := appladcrypto.SealToken(tokenPrefix, s.kekVersion, s.masterKey, plaintext)
		if err != nil {
			return count, fmt.Errorf("dek: rewrap %s: %w", r.id, err)
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE project_encryption_keys SET wrapped_dek = $1, kek_version = $2, updated_at = NOW() WHERE id = $3`,
			newWrapped, s.kekVersion, r.id); err != nil {
			return count, fmt.Errorf("dek: update %s: %w", r.id, err)
		}
		count++
	}

	s.mu.Lock()
	s.cacheActive = make(map[string]cachedDEK)
	s.cacheVersion = make(map[string]cachedDEK)
	s.mu.Unlock()

	return count, nil
}

func (s *Service) unwrapToken(wrapped string) ([]byte, error) {
	plaintext, _, err := appladcrypto.OpenToken(tokenPrefix, s.resolveKEK, wrapped)
	if err != nil {
		return nil, fmt.Errorf("dek: unwrap: %w", err)
	}
	return plaintext, nil
}

func (s *Service) resolveKEK(version int) ([]byte, error) {
	if version == s.kekVersion {
		return s.masterKey, nil
	}
	return nil, fmt.Errorf("dek: unknown master key version %d (has the key been rotated without RewrapAll?)", version)
}

func (s *Service) getCached(cache map[string]cachedDEK, key string) (cachedDEK, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := cache[key]
	if !ok || time.Now().After(c.expiresAt) {
		return cachedDEK{}, false
	}
	return c, true
}

func (s *Service) setCached(cache map[string]cachedDEK, key string, value cachedDEK) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cache[key] = value
}
