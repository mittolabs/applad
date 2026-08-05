package storage

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/mittolabs/applad/internal/model"
)

// ErrForbidden is returned when an authenticated end-user is not permitted to
// act on a file or bucket. Handlers map it to HTTP 403.
var ErrForbidden = errors.New("storage: permission denied")

// storageRoles expands a user id into the role set used for permission matching:
// "any" always, plus "users" and "user:<id>" for a signed-in user. Mirrors the
// databases module so both services enforce permissions identically.
func storageRoles(userID string) []string {
	roles := []string{"any"}
	if userID != "" {
		roles = append(roles, "users", "user:"+userID)
	}
	return roles
}

// permMatch reports whether any of roles is granted action by the inline
// permission strings (e.g. read("user:abc"), delete("any")).
func permMatch(permissions, roles []string, action string) bool {
	set := make(map[string]bool, len(roles))
	for _, r := range roles {
		set[r] = true
	}
	prefix := action + "("
	for _, p := range permissions {
		if strings.HasPrefix(p, prefix) && strings.HasSuffix(p, ")") {
			role := strings.Trim(p[len(prefix):len(p)-1], `"'`)
			if role == "any" || set[role] {
				return true
			}
		}
	}
	return false
}

// authorizeFile decides whether userID may perform action on a file in bucket.
// A server API key (userID == "") always passes. A bucket-level grant covers
// every file in the bucket; when the bucket has file security on, the file's own
// permissions can also grant access. For "create" there is no file yet, so only
// the bucket grant applies.
func (s *Service) authorizeFile(bucket *model.Bucket, file *model.File, userID, action string) error {
	if userID == "" {
		return nil // server API key: full access, mirrors databases
	}
	roles := storageRoles(userID)
	if permMatch(bucket.Permissions, roles, action) {
		return nil
	}
	if bucket.FileSecurity && file != nil && permMatch(file.Permissions, roles, action) {
		return nil
	}
	return ErrForbidden
}

// canReadFile is the read-permission predicate used to filter list results.
func (s *Service) canReadFile(bucket *model.Bucket, file *model.File, userID string) bool {
	return s.authorizeFile(bucket, file, userID, "read") == nil
}

// --- Authorized wrappers (called by the HTTP handlers) ------------------------
//
// Each fetches the bucket (and file), enforces the action for userID, then
// delegates to the underlying operation. Internal callers (e.g. the migration
// engine) keep using the unauthenticated methods and retain full access.

func (s *Service) GetFileWithAuth(ctx context.Context, fileID, bucketID, projectID, userID string) (*model.File, error) {
	bucket, err := s.GetBucket(ctx, bucketID, projectID)
	if err != nil {
		return nil, err
	}
	file, err := s.GetFile(ctx, fileID, bucketID, projectID)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeFile(bucket, file, userID, "read"); err != nil {
		return nil, err
	}
	return file, nil
}

func (s *Service) GetFileContentWithAuth(ctx context.Context, fileID, bucketID, projectID, userID string) ([]byte, string, error) {
	bucket, err := s.GetBucket(ctx, bucketID, projectID)
	if err != nil {
		return nil, "", err
	}
	file, err := s.GetFile(ctx, fileID, bucketID, projectID)
	if err != nil {
		return nil, "", err
	}
	if err := s.authorizeFile(bucket, file, userID, "read"); err != nil {
		return nil, "", err
	}
	return s.GetFileContent(ctx, fileID, bucketID, projectID)
}

func (s *Service) DeleteFileWithAuth(ctx context.Context, fileID, bucketID, projectID, userID string) error {
	bucket, err := s.GetBucket(ctx, bucketID, projectID)
	if err != nil {
		return err
	}
	file, err := s.GetFile(ctx, fileID, bucketID, projectID)
	if err != nil {
		return err
	}
	if err := s.authorizeFile(bucket, file, userID, "delete"); err != nil {
		return err
	}
	return s.DeleteFile(ctx, fileID, bucketID, projectID)
}

func (s *Service) CreateFileStreamWithAuth(ctx context.Context, projectID, bucketID, fileID, name string, content io.Reader, maxBytes int64, mimeType string, permissions []string, userID string) (*model.File, error) {
	if err := s.authorizeCreate(ctx, bucketID, projectID, userID); err != nil {
		return nil, err
	}
	return s.CreateFileStream(ctx, projectID, bucketID, fileID, name, content, maxBytes, mimeType, permissions)
}

func (s *Service) CreateSignedURLWithAuth(ctx context.Context, fileID, bucketID, projectID string, expiresIn int64, userID string) (string, error) {
	bucket, err := s.GetBucket(ctx, bucketID, projectID)
	if err != nil {
		return "", err
	}
	file, err := s.GetFile(ctx, fileID, bucketID, projectID)
	if err != nil {
		return "", err
	}
	if err := s.authorizeFile(bucket, file, userID, "read"); err != nil {
		return "", err
	}
	return s.CreateSignedURL(ctx, fileID, bucketID, projectID, expiresIn)
}

func (s *Service) InitChunkedUploadWithAuth(ctx context.Context, projectID, bucketID, fileID, name, mimeType string, totalSize int64, permissions []string, userID string) (string, error) {
	if err := s.authorizeCreate(ctx, bucketID, projectID, userID); err != nil {
		return "", err
	}
	return s.InitChunkedUpload(ctx, projectID, bucketID, fileID, name, mimeType, totalSize, permissions)
}

func (s *Service) CompleteChunkedUploadWithAuth(ctx context.Context, projectID, bucketID, fileID, userID string) (*model.File, error) {
	if err := s.authorizeCreate(ctx, bucketID, projectID, userID); err != nil {
		return nil, err
	}
	return s.CompleteChunkedUpload(ctx, projectID, bucketID, fileID)
}

func (s *Service) authorizeCreate(ctx context.Context, bucketID, projectID, userID string) error {
	bucket, err := s.GetBucket(ctx, bucketID, projectID)
	if err != nil {
		return err
	}
	return s.authorizeFile(bucket, nil, userID, "create")
}

// ListFilesWithAuth lists files a signed-in user may read. A server API key sees
// everything. When the bucket grants read at the bucket level (and file security
// is off) every file is returned; otherwise the page is filtered to the files
// whose own permissions grant read.
func (s *Service) ListFilesWithAuth(ctx context.Context, projectID, bucketID string, limit, offset int, userID string) ([]*model.File, int, error) {
	files, total, err := s.ListFiles(ctx, projectID, bucketID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if userID == "" {
		return files, total, nil
	}
	bucket, err := s.GetBucket(ctx, bucketID, projectID)
	if err != nil {
		return nil, 0, err
	}
	filtered := files[:0]
	for _, f := range files {
		if s.canReadFile(bucket, f, userID) {
			filtered = append(filtered, f)
		}
	}
	return filtered, total, nil
}
