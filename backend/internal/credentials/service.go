// Package credentials manages encrypted credential storage for projects.
package credentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Credential represents a stored credential with encrypted data.
type Credential struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Data      string    `json:"data,omitempty"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
}

// CredentialSummary is returned from List — excludes decrypted data.
type CredentialSummary struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
}

// Service handles credential business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new credentials Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// encryptionKey derives a 32-byte key from JWT_SECRET.
func encryptionKey() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("credentials: JWT_SECRET not set")
	}
	key := []byte(secret)
	if len(key) < 32 {
		return nil, fmt.Errorf("credentials: JWT_SECRET must be at least 32 bytes")
	}
	return key[:32], nil
}

// encrypt encrypts plaintext using AES-256-GCM and returns base64(nonce+ciphertext).
func encrypt(plaintext string) (string, error) {
	key, err := encryptionKey()
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

// decrypt decodes base64(nonce+ciphertext) and decrypts using AES-256-GCM.
func decrypt(encoded string) (string, error) {
	key, err := encryptionKey()
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

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("credentials: decrypt: %w", err)
	}
	return string(plaintext), nil
}

// Create creates a new credential with encrypted data.
func (s *Service) Create(ctx context.Context, projectID, name, credType, data string) (*Credential, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	encrypted, err := encrypt(data)
	if err != nil {
		return nil, fmt.Errorf("credentials: create: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO credentials (id, project_id, name, type, data, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, name, credType, encrypted, now, now)
	if err != nil {
		return nil, fmt.Errorf("credentials: create: %w", err)
	}

	return &Credential{
		ID: id, ProjectID: projectID, Name: name, Type: credType,
		Data: data, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// List returns all credentials for a project without decrypted data.
func (s *Service) List(ctx context.Context, projectID string) ([]*CredentialSummary, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, type, created_at, updated_at
		 FROM credentials WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var creds []*CredentialSummary
	for rows.Next() {
		var c CredentialSummary
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Type, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		creds = append(creds, &c)
	}
	return creds, len(creds), nil
}

// Get returns a credential by ID with decrypted data.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Credential, error) {
	var c Credential
	var encrypted string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, type, data, created_at, updated_at
		 FROM credentials WHERE id = ? AND project_id = ?`, id, projectID,
	).Scan(&c.ID, &c.ProjectID, &c.Name, &c.Type, &encrypted, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credential not found")
	}
	if err != nil {
		return nil, err
	}

	plaintext, err := decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("credentials: get: %w", err)
	}
	c.Data = plaintext
	return &c, nil
}

// Update updates a credential's name, type, and/or data.
func (s *Service) Update(ctx context.Context, id, projectID, name, credType, data string) (*Credential, error) {
	encrypted, err := encrypt(data)
	if err != nil {
		return nil, fmt.Errorf("credentials: update: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE credentials SET name=?, type=?, data=?, updated_at=?
		 WHERE id=? AND project_id=?`,
		name, credType, encrypted, time.Now().UTC(), id, projectID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id, projectID)
}

// Delete removes a credential.
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM credentials WHERE id = ? AND project_id = ?", id, projectID)
	return err
}
