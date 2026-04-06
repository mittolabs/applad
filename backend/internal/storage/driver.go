package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Driver is the interface for file storage backends.
type Driver interface {
	Write(ctx context.Context, path string, content []byte) error
	Read(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
}

// LocalDriver stores files on the local filesystem.
type LocalDriver struct {
	basePath string
}

// NewLocalDriver creates a local filesystem storage driver.
func NewLocalDriver(basePath string) *LocalDriver {
	return &LocalDriver{basePath: basePath}
}

func (d *LocalDriver) Write(_ context.Context, path string, content []byte) error {
	fullPath := filepath.Join(d.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("local: mkdir: %w", err)
	}
	return os.WriteFile(fullPath, content, 0644)
}

func (d *LocalDriver) Read(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.basePath, path))
}

func (d *LocalDriver) Delete(_ context.Context, path string) error {
	return os.Remove(filepath.Join(d.basePath, path))
}

// S3Driver stores files in an S3-compatible object store.
type S3Driver struct {
	endpoint  string
	bucket    string
	accessKey string
	secretKey string
	region    string
	client    *http.Client
}

// S3Config holds S3 connection settings.
type S3Config struct {
	Endpoint  string // e.g. "https://s3.amazonaws.com" or MinIO URL
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

// NewS3Driver creates an S3-compatible storage driver.
func NewS3Driver(cfg S3Config) *S3Driver {
	return &S3Driver{
		endpoint:  cfg.Endpoint,
		bucket:    cfg.Bucket,
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		region:    cfg.Region,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *S3Driver) objectURL(path string) string {
	return fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, path)
}

func (d *S3Driver) Write(ctx context.Context, path string, content []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, d.objectURL(path), bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	d.sign(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3: put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3: put returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (d *S3Driver) Read(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.objectURL(path), nil)
	if err != nil {
		return nil, err
	}
	d.sign(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3: get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("s3: object not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("s3: get returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (d *S3Driver) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, d.objectURL(path), nil)
	if err != nil {
		return err
	}
	d.sign(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3: delete: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// sign adds basic S3 auth headers. For production, use AWS SigV4.
func (d *S3Driver) sign(req *http.Request) {
	if d.accessKey != "" {
		// Simple auth header for MinIO/S3-compatible stores
		req.SetBasicAuth(d.accessKey, d.secretKey)
	}
}
