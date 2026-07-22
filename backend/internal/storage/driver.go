// Package storage — pluggable storage driver (local disk or S3-compatible).
package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Driver is the interface for file storage backends.
type Driver interface {
	Write(ctx context.Context, path string, data []byte) error
	// WriteStream streams r to path and returns the byte count, avoiding the
	// full-file buffer Write requires.
	WriteStream(ctx context.Context, path string, r io.Reader) (int64, error)
	Read(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
	// Path constructs the storage path or object key for a given file.
	Path(projectID, bucketID, fileID string) string
}

// ── Local driver ─────────────────────────────────────────────────────────────

type LocalDriver struct{ root string }

func NewLocalDriver(root string) *LocalDriver { return &LocalDriver{root: root} }

func (d *LocalDriver) Path(projectID, bucketID, fileID string) string {
	return filepath.Join(d.root, projectID, bucketID, fileID)
}

// contain verifies path is still under d.root. Path joins caller-supplied
// segments, so a crafted "../../etc/cron.d/x" ID must fail here even if the
// caller skipped sanitising it.
func (d *LocalDriver) contain(path string) error {
	rel, err := filepath.Rel(d.root, filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("storage: path escapes storage root")
	}
	return nil
}

func (d *LocalDriver) Write(_ context.Context, path string, data []byte) error {
	if err := d.contain(path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func (d *LocalDriver) WriteStream(_ context.Context, path string, r io.Reader) (int64, error) {
	if err := d.contain(path); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return 0, fmt.Errorf("storage: mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, fmt.Errorf("storage: create: %w", err)
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path) //nolint:errcheck
		return n, fmt.Errorf("storage: write: %w", err)
	}
	return n, nil
}

func (d *LocalDriver) Read(_ context.Context, path string) ([]byte, error) {
	if err := d.contain(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("storage: read: %w", err)
	}
	return data, nil
}

func (d *LocalDriver) Delete(_ context.Context, path string) error {
	if err := d.contain(path); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: delete: %w", err)
	}
	return nil
}

// ── S3 driver ─────────────────────────────────────────────────────────────────

// S3Driver stores files in any S3-compatible object store (AWS S3, MinIO,
// Cloudflare R2, etc.) using AWS Signature V4.
type S3Driver struct {
	endpoint   string
	bucket     string
	region     string
	accessKey  string
	secretKey  string
	httpClient *http.Client
}

func NewS3Driver(endpoint, bucket, region, accessKey, secretKey string) *S3Driver {
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://s3.%s.amazonaws.com", region)
	}
	return &S3Driver{
		endpoint:   strings.TrimRight(endpoint, "/"),
		bucket:     bucket,
		region:     region,
		accessKey:  accessKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

func (d *S3Driver) Path(projectID, bucketID, fileID string) string {
	return projectID + "/" + bucketID + "/" + fileID
}

// WriteStream buffers and delegates to Write: a single-part S3 PUT needs a
// Content-Length and payload hash up front, and this driver does not speak
// multipart upload yet.
func (d *S3Driver) WriteStream(ctx context.Context, path string, r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("s3: read stream: %w", err)
	}
	return int64(len(data)), d.Write(ctx, path, data)
}

func (d *S3Driver) Write(ctx context.Context, path string, data []byte) error {
	key := strings.TrimPrefix(path, "/")
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	d.sign(req, data)
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("s3: put %s: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("s3: put %s: %s: %s", key, resp.Status, string(body))
	}
	return nil
}

func (d *S3Driver) Read(ctx context.Context, path string) ([]byte, error) {
	key := strings.TrimPrefix(path, "/")
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	d.sign(req, nil)
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3: get %s: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("storage: file not found")
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("s3: get %s: %s", key, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (d *S3Driver) Delete(ctx context.Context, path string) error {
	key := strings.TrimPrefix(path, "/")
	url := fmt.Sprintf("%s/%s/%s", d.endpoint, d.bucket, key)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	d.sign(req, nil)
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("s3: delete %s: %w", key, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("s3: delete %s: %s", key, resp.Status)
	}
	return nil
}

// sign adds an AWS Signature V4 Authorization header to the request.
func (d *S3Driver) sign(req *http.Request, body []byte) {
	t := time.Now().UTC()
	dateShort := t.Format("20060102")
	dateLong := t.Format("20060102T150405Z")

	payloadHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // sha256("")
	if len(body) > 0 {
		h := sha256.Sum256(body)
		payloadHash = hex.EncodeToString(h[:])
	}

	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", dateLong)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	ct := req.Header.Get("Content-Type")
	var signedHeaders, canonicalHeaders string
	if ct != "" {
		signedHeaders = "content-type;host;x-amz-content-sha256;x-amz-date"
		canonicalHeaders = fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
			ct, req.URL.Host, payloadHash, dateLong)
	} else {
		signedHeaders = "host;x-amz-content-sha256;x-amz-date"
		canonicalHeaders = fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
			req.URL.Host, payloadHash, dateLong)
	}

	canonicalRequest := strings.Join([]string{
		req.Method, req.URL.EscapedPath(), req.URL.RawQuery,
		canonicalHeaders, signedHeaders, payloadHash,
	}, "\n")

	scope := fmt.Sprintf("%s/%s/s3/aws4_request", dateShort, d.region)
	reqHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		dateLong, scope, hex.EncodeToString(reqHash[:]))

	sigKey := s3hmac(s3hmac(s3hmac(s3hmac(
		[]byte("AWS4"+d.secretKey), dateShort),
		d.region), "s3"), "aws4_request")
	sig := hex.EncodeToString(s3hmac(sigKey, stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		d.accessKey, scope, signedHeaders, sig,
	))
}

func s3hmac(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
