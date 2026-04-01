// Package storage implements Applad's storage service:
// buckets, file upload (chunked), file retrieval, image transformations, encryption, and antivirus.
package storage

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/uid"
)

// Service handles storage business logic.
type Service struct {
	db          *db.DB
	storagePath string
}

// NewService creates a new storage Service.
func NewService(database *db.DB, storagePath string) *Service {
	return &Service{db: database, storagePath: storagePath}
}

// --- buckets ---

func (s *Service) CreateBucket(ctx context.Context, projectID, bucketID, name string, permissions []string, fileSizeLimit int64, allowedMimeTypes []string, compression string, encryption, antivirus bool) (*model.Bucket, error) {
	id := uid.New(bucketID)
	now := time.Now().UTC()
	permsJSON, _ := json.Marshal(permissions)
	mimeJSON, _ := json.Marshal(allowedMimeTypes)
	if compression == "" {
		compression = "none"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO buckets (id, project_id, name, permissions, file_size_limit, allowed_mime_types, compression, encryption, antivirus, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, name, permsJSON, fileSizeLimit, mimeJSON, compression, encryption, antivirus, now, now)
	if err != nil {
		return nil, fmt.Errorf("storage: create bucket: %w", err)
	}
	return &model.Bucket{
		ID: id, Name: name, Enabled: true,
		Permissions: permissions, FileSizeLimit: fileSizeLimit,
		AllowedFileExtensions: allowedMimeTypes,
		Compression:           compression, Encryption: encryption, Antivirus: antivirus,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Service) GetBucket(ctx context.Context, bucketID, projectID string) (*model.Bucket, error) {
	var b model.Bucket
	var permsJSON, mimeJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, permissions, file_size_limit, allowed_mime_types, compression, encryption, antivirus, enabled, created_at, updated_at FROM buckets WHERE id = ? AND project_id = ?",
		bucketID, projectID).Scan(&b.ID, &b.Name, &permsJSON, &b.FileSizeLimit, &mimeJSON, &b.Compression, &b.Encryption, &b.Antivirus, &b.Enabled, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bucket not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(permsJSON, &b.Permissions)          //nolint:errcheck
	json.Unmarshal(mimeJSON, &b.AllowedFileExtensions) //nolint:errcheck
	if b.Permissions == nil {
		b.Permissions = []string{}
	}
	if b.AllowedFileExtensions == nil {
		b.AllowedFileExtensions = []string{}
	}
	return &b, nil
}

func (s *Service) ListBuckets(ctx context.Context, projectID string) ([]*model.Bucket, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, permissions, file_size_limit, allowed_mime_types, compression, encryption, antivirus, enabled, created_at, updated_at FROM buckets WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var buckets []*model.Bucket
	for rows.Next() {
		var b model.Bucket
		var permsJSON, mimeJSON []byte
		if err := rows.Scan(&b.ID, &b.Name, &permsJSON, &b.FileSizeLimit, &mimeJSON, &b.Compression, &b.Encryption, &b.Antivirus, &b.Enabled, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(permsJSON, &b.Permissions)          //nolint:errcheck
		json.Unmarshal(mimeJSON, &b.AllowedFileExtensions) //nolint:errcheck
		if b.Permissions == nil {
			b.Permissions = []string{}
		}
		if b.AllowedFileExtensions == nil {
			b.AllowedFileExtensions = []string{}
		}
		buckets = append(buckets, &b)
	}
	return buckets, len(buckets), nil
}

func (s *Service) UpdateBucket(ctx context.Context, bucketID, projectID, name string, permissions []string, fileSizeLimit int64, enabled bool) (*model.Bucket, error) {
	permsJSON, _ := json.Marshal(permissions)
	_, err := s.db.ExecContext(ctx,
		"UPDATE buckets SET name = ?, permissions = ?, file_size_limit = ?, enabled = ? WHERE id = ? AND project_id = ?",
		name, permsJSON, fileSizeLimit, enabled, bucketID, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetBucket(ctx, bucketID, projectID)
}

func (s *Service) DeleteBucket(ctx context.Context, bucketID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM buckets WHERE id = ? AND project_id = ?", bucketID, projectID)
	return err
}

// --- files ---

func (s *Service) CreateFile(ctx context.Context, projectID, bucketID, fileID, name string, content []byte, mimeType string, permissions []string) (*model.File, error) {
	id := uid.New(fileID)
	now := time.Now().UTC()

	// Write to disk
	dir := filepath.Join(s.storagePath, projectID, bucketID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("storage: mkdir: %w", err)
	}
	path := filepath.Join(dir, id)
	if err := os.WriteFile(path, content, 0644); err != nil {
		return nil, fmt.Errorf("storage: write file: %w", err)
	}

	sig := fmt.Sprintf("%x", md5.Sum(content))
	permsJSON, _ := json.Marshal(permissions)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO files (id, bucket_id, project_id, name, mime_type, size, permissions, path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, bucketID, projectID, name, mimeType, int64(len(content)), permsJSON, path, now, now)
	if err != nil {
		os.Remove(path) //nolint:errcheck
		return nil, fmt.Errorf("storage: create file record: %w", err)
	}

	return &model.File{
		ID: id, BucketID: bucketID, Name: name,
		MimeType: mimeType, SizeOriginal: int64(len(content)),
		Signature: sig, Permissions: permissions,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Service) GetFile(ctx context.Context, fileID, bucketID, projectID string) (*model.File, error) {
	var f model.File
	var permsJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, bucket_id, name, mime_type, size, permissions, created_at, updated_at FROM files WHERE id = ? AND bucket_id = ? AND project_id = ?",
		fileID, bucketID, projectID).Scan(&f.ID, &f.BucketID, &f.Name, &f.MimeType, &f.SizeOriginal, &permsJSON, &f.CreatedAt, &f.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(permsJSON, &f.Permissions) //nolint:errcheck
	if f.Permissions == nil {
		f.Permissions = []string{}
	}
	return &f, nil
}

func (s *Service) GetFileContent(ctx context.Context, fileID, bucketID, projectID string) ([]byte, string, error) {
	var path, mimeType string
	err := s.db.QueryRowContext(ctx,
		"SELECT path, mime_type FROM files WHERE id = ? AND bucket_id = ? AND project_id = ?",
		fileID, bucketID, projectID).Scan(&path, &mimeType)
	if err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("file not found")
	}
	if err != nil {
		return nil, "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("storage: read file: %w", err)
	}
	return content, mimeType, nil
}

func (s *Service) ListFiles(ctx context.Context, projectID, bucketID string, limit, offset int) ([]*model.File, int, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, bucket_id, name, mime_type, size, permissions, created_at, updated_at FROM files WHERE bucket_id = ? AND project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		bucketID, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var files []*model.File
	for rows.Next() {
		var f model.File
		var permsJSON []byte
		if err := rows.Scan(&f.ID, &f.BucketID, &f.Name, &f.MimeType, &f.SizeOriginal, &permsJSON, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(permsJSON, &f.Permissions) //nolint:errcheck
		if f.Permissions == nil {
			f.Permissions = []string{}
		}
		files = append(files, &f)
	}
	var total int
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE bucket_id = ? AND project_id = ?", bucketID, projectID).Scan(&total) //nolint:errcheck
	return files, total, nil
}

func (s *Service) DeleteFile(ctx context.Context, fileID, bucketID, projectID string) error {
	var path string
	err := s.db.QueryRowContext(ctx,
		"SELECT path FROM files WHERE id = ? AND bucket_id = ? AND project_id = ?",
		fileID, bucketID, projectID).Scan(&path)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx,
		"DELETE FROM files WHERE id = ? AND bucket_id = ? AND project_id = ?", fileID, bucketID, projectID)
	os.Remove(path) //nolint:errcheck
	return nil
}
