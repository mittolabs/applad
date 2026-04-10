// Package credentials manages encrypted credential (secret) storage for projects.
// Credentials are encrypted with AES-256-GCM. Each credential stores a key_version
// so that key rotation re-encrypts data without invalidating existing records.
//
// Key versions:
//   0 = SHA-256(JWT_SECRET)  (legacy — backward compat; not recommended for production)
//   1 = CREDENTIALS_ENCRYPTION_KEY[:32]  (set this in production)
package credentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// ── Types ────────────────────────────────────────────────────────────────────

// Credential is a named, encrypted secret with optional expiry and access control.
type Credential struct {
	ID          string     `json:"$id"`
	ProjectID   string     `json:"projectId"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Description string     `json:"description,omitempty"`
	Data        string     `json:"data,omitempty"` // decrypted; omitted from list responses
	KeyVersion  int        `json:"keyVersion"`
	Protected   bool       `json:"protected"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `json:"$createdAt"`
	UpdatedAt   time.Time  `json:"$updatedAt"`
}

// CredentialSummary is returned from List — no decrypted data.
type CredentialSummary struct {
	ID          string     `json:"$id"`
	ProjectID   string     `json:"projectId"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Description string     `json:"description,omitempty"`
	KeyVersion  int        `json:"keyVersion"`
	Protected   bool       `json:"protected"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time  `json:"$createdAt"`
	UpdatedAt   time.Time  `json:"$updatedAt"`
}

// CredentialAccess is a single entry in the per-credential access log.
type CredentialAccess struct {
	ID           string    `json:"$id"`
	CredentialID string    `json:"credentialId"`
	ProjectID    string    `json:"projectId"`
	Action       string    `json:"action"`
	ActorID      string    `json:"actorId,omitempty"`
	ActorType    string    `json:"actorType,omitempty"`
	IP           string    `json:"ip,omitempty"`
	UserAgent    string    `json:"userAgent,omitempty"`
	AccessedAt   time.Time `json:"accessedAt"`
}

// ── Service ──────────────────────────────────────────────────────────────────

// Service handles credential business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new credentials Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ── Encryption ───────────────────────────────────────────────────────────────

// currentKeyVersion returns the version of the key that should be used for
// new encryptions. Returns 1 if CREDENTIALS_ENCRYPTION_KEY is set, else 0.
func currentKeyVersion() int {
	if os.Getenv("CREDENTIALS_ENCRYPTION_KEY") != "" {
		return 1
	}
	return 0
}

// keyForVersion returns the 32-byte AES key for the given version.
func keyForVersion(v int) ([]byte, error) {
	switch v {
	case 0:
		// Legacy: derived from JWT_SECRET via SHA-256 so any-length secret works.
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("credentials: JWT_SECRET not set")
		}
		slog.Warn("credentials: using JWT_SECRET as encryption key (key_version=0); set CREDENTIALS_ENCRYPTION_KEY for production")
		h := sha256.Sum256([]byte(secret))
		return h[:], nil
	case 1:
		key := os.Getenv("CREDENTIALS_ENCRYPTION_KEY")
		if key == "" {
			return nil, fmt.Errorf("credentials: CREDENTIALS_ENCRYPTION_KEY not set")
		}
		if len(key) < 32 {
			return nil, fmt.Errorf("credentials: CREDENTIALS_ENCRYPTION_KEY must be at least 32 bytes (got %d)", len(key))
		}
		k := make([]byte, 32)
		copy(k, []byte(key))
		return k, nil
	default:
		return nil, fmt.Errorf("credentials: unknown key version %d", v)
	}
}

// encryptWithVersion encrypts plaintext with the key for the given version.
// Returns base64(nonce || ciphertext).
func encryptWithVersion(plaintext string, version int) (string, error) {
	key, err := keyForVersion(version)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("credentials: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("credentials: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("credentials: nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptWithVersion decrypts base64(nonce || ciphertext) using the key for version.
func decryptWithVersion(encoded string, version int) (string, error) {
	key, err := keyForVersion(version)
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("credentials: base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("credentials: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("credentials: gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("credentials: ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("credentials: decrypt: %w", err)
	}
	return string(plaintext), nil
}

// ── CRUD ─────────────────────────────────────────────────────────────────────

// Create stores a new encrypted credential.
func (s *Service) Create(ctx context.Context, projectID, name, credType, description, data string, protected bool, expiresAt *time.Time) (*Credential, error) {
	if name == "" {
		return nil, fmt.Errorf("credentials: name is required")
	}
	if credType == "" {
		return nil, fmt.Errorf("credentials: type is required")
	}
	if data == "" {
		return nil, fmt.Errorf("credentials: data is required")
	}

	version := currentKeyVersion()
	encrypted, err := encryptWithVersion(data, version)
	if err != nil {
		return nil, fmt.Errorf("credentials: encrypt: %w", err)
	}

	id := uid.New("unique()")
	now := time.Now().UTC()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO credentials (id, project_id, name, type, description, data, key_version, protected, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, name, credType, description, encrypted, version, protected, expiresAt, now, now)
	if err != nil {
		return nil, fmt.Errorf("credentials: create: %w", err)
	}

	return &Credential{
		ID: id, ProjectID: projectID, Name: name, Type: credType,
		Description: description, Data: data, KeyVersion: version,
		Protected: protected, ExpiresAt: expiresAt,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// List returns all non-expired credentials for a project without decrypted data.
func (s *Service) List(ctx context.Context, projectID string, limit, offset int) ([]*CredentialSummary, int, error) {
	if limit <= 0 {
		limit = 25
	}

	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM credentials WHERE project_id = ? AND (expires_at IS NULL OR expires_at > NOW())",
		projectID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, type, COALESCE(description,''), key_version, protected, expires_at, created_at, updated_at
		 FROM credentials
		 WHERE project_id = ? AND (expires_at IS NULL OR expires_at > NOW())
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var creds []*CredentialSummary
	for rows.Next() {
		var c CredentialSummary
		var expiresAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Type, &c.Description,
			&c.KeyVersion, &c.Protected, &expiresAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if expiresAt.Valid {
			t := expiresAt.Time
			c.ExpiresAt = &t
		}
		creds = append(creds, &c)
	}
	if creds == nil {
		creds = []*CredentialSummary{}
	}
	return creds, total, nil
}

// Get returns a credential with decrypted data.
// If the credential is protected, isAPIKey must be true — otherwise the data
// is redacted and an error is returned.
func (s *Service) Get(ctx context.Context, id, projectID string, isAPIKey bool) (*Credential, error) {
	var c Credential
	var encrypted, description string
	var expiresAt sql.NullTime
	var protected bool

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, type, COALESCE(description,''), data, key_version, protected, expires_at, created_at, updated_at
		 FROM credentials WHERE id = ? AND project_id = ?`,
		id, projectID).Scan(
		&c.ID, &c.ProjectID, &c.Name, &c.Type, &description,
		&encrypted, &c.KeyVersion, &protected, &expiresAt, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credential not found")
	}
	if err != nil {
		return nil, err
	}
	c.Description = description
	c.Protected = protected
	if expiresAt.Valid {
		if time.Now().UTC().After(expiresAt.Time) {
			return nil, fmt.Errorf("credential expired")
		}
		t := expiresAt.Time
		c.ExpiresAt = &t
	}
	if protected && !isAPIKey {
		return nil, fmt.Errorf("credential requires API key authentication")
	}
	plaintext, err := decryptWithVersion(encrypted, c.KeyVersion)
	if err != nil {
		return nil, fmt.Errorf("credentials: get: %w", err)
	}
	c.Data = plaintext
	return &c, nil
}

// Update re-encrypts and saves a credential with the current key version.
func (s *Service) Update(ctx context.Context, id, projectID, name, credType, description, data string, protected bool, expiresAt *time.Time) (*Credential, error) {
	version := currentKeyVersion()
	encrypted, err := encryptWithVersion(data, version)
	if err != nil {
		return nil, fmt.Errorf("credentials: encrypt: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE credentials
		 SET name=?, type=?, description=?, data=?, key_version=?, protected=?, expires_at=?, updated_at=?
		 WHERE id=? AND project_id=?`,
		name, credType, description, encrypted, version, protected, expiresAt, time.Now().UTC(), id, projectID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id, projectID, true) // actor already verified by caller
}

// Delete removes a credential.
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM credentials WHERE id = ? AND project_id = ?", id, projectID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// ── Key rotation ─────────────────────────────────────────────────────────────

// RotateKeys re-encrypts every credential in the project that is NOT already
// at the current key version. Returns the number of credentials rotated.
func (s *Service) RotateKeys(ctx context.Context, projectID string) (int, error) {
	target := currentKeyVersion()

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, data, key_version FROM credentials
		 WHERE project_id = ? AND key_version != ?`,
		projectID, target)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type row struct {
		id, encrypted string
		version       int
	}
	var toRotate []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.encrypted, &r.version); err != nil {
			return 0, err
		}
		toRotate = append(toRotate, r)
	}

	rotated := 0
	for _, r := range toRotate {
		plaintext, err := decryptWithVersion(r.encrypted, r.version)
		if err != nil {
			slog.Warn("credentials: rotate: decrypt failed, skipping",
				"id", r.id, "from_version", r.version, "error", err)
			continue
		}
		newCiphertext, err := encryptWithVersion(plaintext, target)
		if err != nil {
			return rotated, fmt.Errorf("credentials: rotate: re-encrypt %s: %w", r.id, err)
		}
		_, err = s.db.ExecContext(ctx,
			"UPDATE credentials SET data=?, key_version=?, updated_at=? WHERE id=? AND project_id=?",
			newCiphertext, target, time.Now().UTC(), r.id, projectID)
		if err != nil {
			return rotated, fmt.Errorf("credentials: rotate: update %s: %w", r.id, err)
		}
		rotated++
	}
	return rotated, nil
}

// ── Access log ───────────────────────────────────────────────────────────────

// LogAccess records a credential access event.
func (s *Service) LogAccess(ctx context.Context, credID, projectID, action, actorID, actorType, ip, ua string) {
	id := uid.New("unique()")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO credential_accesses (id, credential_id, project_id, action, actor_id, actor_type, ip, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, credID, projectID, action, actorID, actorType, ip, ua)
	if err != nil {
		slog.Warn("credentials: log access failed", "error", err)
	}
}

// ListAccesses returns the access log for a credential, newest first.
func (s *Service) ListAccesses(ctx context.Context, credID, projectID string, limit, offset int) ([]*CredentialAccess, int, error) {
	if limit <= 0 {
		limit = 50
	}
	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM credential_accesses WHERE credential_id = ? AND project_id = ?",
		credID, projectID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, credential_id, project_id, action,
		        COALESCE(actor_id,''), COALESCE(actor_type,''),
		        COALESCE(ip,''), COALESCE(user_agent,''), accessed_at
		 FROM credential_accesses
		 WHERE credential_id = ? AND project_id = ?
		 ORDER BY accessed_at DESC LIMIT ? OFFSET ?`,
		credID, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accesses []*CredentialAccess
	for rows.Next() {
		var a CredentialAccess
		if err := rows.Scan(&a.ID, &a.CredentialID, &a.ProjectID, &a.Action,
			&a.ActorID, &a.ActorType, &a.IP, &a.UserAgent, &a.AccessedAt); err != nil {
			return nil, 0, err
		}
		accesses = append(accesses, &a)
	}
	if accesses == nil {
		accesses = []*CredentialAccess{}
	}
	return accesses, total, nil
}
