// Package storage implements Applad's storage service:
// buckets, file upload (chunked), file retrieval, image transformations, encryption, and antivirus.
package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "image/gif" // register GIF decoder

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/uid"
)

// Service handles storage business logic.
type Service struct {
	db          *db.DB
	storagePath string
	clamavAddr  string
	events      realtime.EventPublisher
}

// NewService creates a new storage Service.
func NewService(database *db.DB, storagePath string) *Service {
	return &Service{db: database, storagePath: storagePath}
}

// SetEventPublisher sets the realtime event publisher.
func (s *Service) SetEventPublisher(pub realtime.EventPublisher) {
	s.events = pub
}

// SetClamAV configures antivirus scanning.
func (s *Service) SetClamAV(addr string) {
	s.clamavAddr = addr
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
	// Antivirus scan if configured
	if s.clamavAddr != "" {
		result, err := ScanWithClamAV(s.clamavAddr, content)
		if err != nil {
			return nil, fmt.Errorf("storage: antivirus scan failed: %w", err)
		}
		if !result.Clean {
			return nil, fmt.Errorf("storage: file rejected by antivirus: %s", result.Message)
		}
	}

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

	f := &model.File{
		ID: id, BucketID: bucketID, Name: name,
		MimeType: mimeType, SizeOriginal: int64(len(content)),
		Signature: sig, Permissions: permissions,
		CreatedAt: now, UpdatedAt: now,
	}
	realtime.PublishResourceEvent(s.events, "storage", "files", "create", projectID, id, f)
	return f, nil
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

// --- chunked uploads ---

// InitChunkedUpload creates a file record and temp directory for chunks.
func (s *Service) InitChunkedUpload(ctx context.Context, projectID, bucketID, fileID, name, mimeType string, totalSize int64, permissions []string) (string, error) {
	id := uid.New(fileID)

	// Create temp chunk directory
	chunkDir := filepath.Join(s.storagePath, "_chunks", id)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return "", fmt.Errorf("storage: create chunk dir: %w", err)
	}

	// Create a pending file record
	now := time.Now().UTC()
	permsJSON, _ := json.Marshal(permissions)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO files (id, bucket_id, project_id, name, mime_type, size, permissions, path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, bucketID, projectID, name, mimeType, totalSize, permsJSON, chunkDir, now, now)
	if err != nil {
		return "", fmt.Errorf("storage: init chunked upload: %w", err)
	}
	return id, nil
}

// UploadChunk writes a single chunk to disk.
func (s *Service) UploadChunk(_ context.Context, projectID, bucketID, fileID string, index int, data []byte) error {
	chunkDir := filepath.Join(s.storagePath, "_chunks", fileID)
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%06d", index))
	return os.WriteFile(chunkPath, data, 0644)
}

// CompleteChunkedUpload assembles chunks into the final file.
func (s *Service) CompleteChunkedUpload(ctx context.Context, projectID, bucketID, fileID string) (*model.File, error) {
	chunkDir := filepath.Join(s.storagePath, "_chunks", fileID)

	entries, err := os.ReadDir(chunkDir)
	if err != nil {
		return nil, fmt.Errorf("storage: read chunks: %w", err)
	}

	// Assemble chunks in order
	dir := filepath.Join(s.storagePath, projectID, bucketID)
	os.MkdirAll(dir, 0755)
	finalPath := filepath.Join(dir, fileID)

	out, err := os.Create(finalPath)
	if err != nil {
		return nil, fmt.Errorf("storage: create final file: %w", err)
	}

	var totalSize int64
	for _, entry := range entries {
		chunk, err := os.ReadFile(filepath.Join(chunkDir, entry.Name()))
		if err != nil {
			out.Close()
			return nil, fmt.Errorf("storage: read chunk %s: %w", entry.Name(), err)
		}
		n, _ := out.Write(chunk)
		totalSize += int64(n)
	}
	out.Close()

	// Clean up chunks
	os.RemoveAll(chunkDir)

	// Update file record with final path and size
	_, err = s.db.ExecContext(ctx,
		"UPDATE files SET path = ?, size = ? WHERE id = ? AND bucket_id = ? AND project_id = ?",
		finalPath, totalSize, fileID, bucketID, projectID)
	if err != nil {
		return nil, err
	}

	return s.GetFile(ctx, fileID, bucketID, projectID)
}

// TransformImage resizes and converts an image.
func (s *Service) TransformImage(content []byte, mimeType string, width, height, quality int, outputFormat string) ([]byte, string, error) {
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("not an image")
	}

	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	// Resize if requested
	if width > 0 || height > 0 {
		img = resizeImage(img, width, height)
	}

	// Determine output format
	if outputFormat == "" {
		if strings.Contains(mimeType, "png") {
			outputFormat = "png"
		} else {
			outputFormat = "jpg"
		}
	}

	if quality <= 0 {
		quality = 85
	}

	var buf bytes.Buffer
	var outMime string

	switch outputFormat {
	case "png":
		err = png.Encode(&buf, img)
		outMime = "image/png"
	default: // jpg/jpeg
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		outMime = "image/jpeg"
	}
	if err != nil {
		return nil, "", fmt.Errorf("encode image: %w", err)
	}

	return buf.Bytes(), outMime, nil
}

// resizeImage scales an image to fit within the given dimensions, preserving aspect ratio.
func resizeImage(src image.Image, targetW, targetH int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if targetW <= 0 {
		targetW = srcW * targetH / srcH
	}
	if targetH <= 0 {
		targetH = srcH * targetW / srcW
	}

	// Simple nearest-neighbor resize using Go stdlib
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := x * srcW / targetW
			srcY := y * srcH / targetH
			dst.Set(x, y, src.At(srcX+bounds.Min.X, srcY+bounds.Min.Y))
		}
	}
	return dst
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
	realtime.PublishResourceEvent(s.events, "storage", "files", "delete", projectID, fileID, map[string]string{"$id": fileID})
	return nil
}
