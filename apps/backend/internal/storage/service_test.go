package storage

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	appladcrypto "github.com/mittolabs/applad/internal/crypto"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/dek"
)

// helper: create a service backed by sqlmock + temp directory.
func setup(t *testing.T) (*Service, sqlmock.Sqlmock, string) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })
	database := &db.DB{DB: mockDB}
	tmpDir := t.TempDir()
	svc := NewService(database, tmpDir)
	return svc, mock, tmpDir
}

// makePNG creates a minimal width x height PNG in memory.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with a non-transparent colour so the image is valid.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// bucketRowsFor builds a single-bucket result row with the given at-rest flags.
func bucketRowsFor(compression string, encryption, antivirus bool) *sqlmock.Rows {
	now := time.Now().UTC()
	return sqlmock.NewRows([]string{
		"id", "name", "permissions", "file_size_limit", "allowed_mime_types",
		"compression", "encryption", "antivirus", "file_security", "image_transformations",
		"enabled", "created_at", "updated_at",
	}).AddRow("b1", "Test Bucket", []byte(`[]`), int64(0), []byte(`[]`),
		compression, encryption, antivirus, false, true, true, now, now)
}

// fakeClamd starts a TCP listener that speaks just enough of clamd's INSTREAM
// protocol to hand back a canned verdict, and returns its address.
func fakeClamd(t *testing.T, verdict string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				c.SetReadDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 4096)
				_, _ = c.Read(buf) // drain the client's stream; reply is all we need
				c.Write([]byte(verdict + "\x00"))
			}(conn)
		}
	}()
	return ln.Addr().String()
}

// --- Bucket tests ---

func TestCreateBucket_Success(t *testing.T) {
	svc, mock, _ := setup(t)

	mock.ExpectExec("INSERT INTO buckets").
		WithArgs(
			sqlmock.AnyArg(), // id
			"proj1",          // project_id
			"My Bucket",      // name
			sqlmock.AnyArg(), // permissions JSON
			int64(10485760),  // file_size_limit
			sqlmock.AnyArg(), // allowed_mime_types JSON
			"none",           // compression
			false,            // encryption
			false,            // antivirus
			false,            // file_security
			true,             // image_transformations
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	bucket, err := svc.CreateBucket(context.Background(), "proj1", "unique()", "My Bucket",
		[]string{"read"}, 10485760, []string{"image/png"}, "", false, false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bucket.Name != "My Bucket" {
		t.Errorf("expected name 'My Bucket', got %q", bucket.Name)
	}
	if bucket.Compression != "none" {
		t.Errorf("expected compression 'none', got %q", bucket.Compression)
	}
	if bucket.FileSizeLimit != 10485760 {
		t.Errorf("expected file size limit 10485760, got %d", bucket.FileSizeLimit)
	}
	if len(bucket.Permissions) != 1 || bucket.Permissions[0] != "read" {
		t.Errorf("unexpected permissions: %v", bucket.Permissions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetBucket_NotFound(t *testing.T) {
	svc, mock, _ := setup(t)

	mock.ExpectQuery("SELECT .+ FROM buckets WHERE").
		WithArgs("nonexistent", "proj1").
		WillReturnRows(sqlmock.NewRows(nil))

	_, err := svc.GetBucket(context.Background(), "nonexistent", "proj1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "bucket not found" {
		t.Errorf("expected 'bucket not found', got %q", err.Error())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListBuckets_ReturnsAll(t *testing.T) {
	svc, mock, _ := setup(t)

	rows := sqlmock.NewRows([]string{
		"id", "name", "permissions", "file_size_limit", "allowed_mime_types",
		"compression", "encryption", "antivirus", "file_security", "image_transformations", "enabled", "created_at", "updated_at",
	}).
		AddRow("b1", "Bucket 1", []byte(`["read"]`), int64(1000), []byte(`["image/png"]`), "none", false, false, false, true, true, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)).
		AddRow("b2", "Bucket 2", []byte(`[]`), int64(2000), []byte(`[]`), "gzip", true, false, false, true, true, time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))

	mock.ExpectQuery("SELECT .+ FROM buckets WHERE").
		WithArgs("proj1").
		WillReturnRows(rows)

	buckets, count, err := svc.ListBuckets(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Name != "Bucket 1" {
		t.Errorf("expected 'Bucket 1', got %q", buckets[0].Name)
	}
	if buckets[1].Compression != "gzip" {
		t.Errorf("expected 'gzip', got %q", buckets[1].Compression)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- File tests ---

func TestCreateFile_WritesToDisk(t *testing.T) {
	svc, mock, tmpDir := setup(t)

	now := time.Now().UTC()
	bucketRows := sqlmock.NewRows([]string{
		"id", "name", "permissions", "file_size_limit", "allowed_mime_types",
		"compression", "encryption", "antivirus", "file_security", "image_transformations",
		"enabled", "created_at", "updated_at",
	}).AddRow("bucket1", "Test Bucket", []byte(`[]`), int64(0), []byte(`[]`),
		"", false, false, false, false, true, now, now)
	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("bucket1", "proj1").
		WillReturnRows(bucketRows)

	mock.ExpectExec("INSERT INTO files").
		WithArgs(
			sqlmock.AnyArg(), // id
			"bucket1",        // bucket_id
			"proj1",          // project_id
			"test.txt",       // name
			"text/plain",     // mime_type
			int64(11),        // size
			sqlmock.AnyArg(), // permissions JSON
			sqlmock.AnyArg(), // path
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	content := []byte("hello world")
	f, err := svc.CreateFile(context.Background(), "proj1", "bucket1", "unique()", "test.txt", content, "text/plain", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file was written to disk.
	diskPath := filepath.Join(tmpDir, "proj1", "bucket1", f.ID)
	data, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("file not written to disk: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("disk content mismatch: got %q, want %q", data, content)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestCreateFile_StoresCorrectSize(t *testing.T) {
	svc, mock, _ := setup(t)

	now := time.Now().UTC()
	bucketRows := sqlmock.NewRows([]string{
		"id", "name", "permissions", "file_size_limit", "allowed_mime_types",
		"compression", "encryption", "antivirus", "file_security", "image_transformations",
		"enabled", "created_at", "updated_at",
	}).AddRow("b1", "Test Bucket", []byte(`[]`), int64(0), []byte(`[]`),
		"", false, false, false, false, true, now, now)
	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRows)

	mock.ExpectExec("INSERT INTO files").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	content := make([]byte, 42)
	f, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "data.bin", content, "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.SizeOriginal != 42 {
		t.Errorf("expected SizeOriginal=42, got %d", f.SizeOriginal)
	}
}

func TestGetFileContent_ReadsFromDisk(t *testing.T) {
	svc, mock, tmpDir := setup(t)

	// Manually create a file on disk.
	dir := filepath.Join(tmpDir, "proj1", "b1")
	os.MkdirAll(dir, 0755)
	filePath := filepath.Join(dir, "file123")
	expected := []byte("disk content here")
	os.WriteFile(filePath, expected, 0644)

	rows := sqlmock.NewRows([]string{"path", "mime_type"}).
		AddRow(filePath, "text/plain")
	mock.ExpectQuery("SELECT path, mime_type FROM files WHERE").
		WithArgs("file123", "b1", "proj1").
		WillReturnRows(rows)

	content, mimeType, err := svc.GetFileContent(context.Background(), "file123", "b1", "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(content, expected) {
		t.Errorf("content mismatch: got %q, want %q", content, expected)
	}
	if mimeType != "text/plain" {
		t.Errorf("expected mime 'text/plain', got %q", mimeType)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDeleteFile_RemovesFromDisk(t *testing.T) {
	svc, mock, tmpDir := setup(t)

	// Create a file on disk.
	dir := filepath.Join(tmpDir, "proj1", "b1")
	os.MkdirAll(dir, 0755)
	filePath := filepath.Join(dir, "todelete")
	os.WriteFile(filePath, []byte("bye"), 0644)

	// Mock SELECT path.
	mock.ExpectQuery("SELECT path FROM files WHERE").
		WithArgs("todelete", "b1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(filePath))

	// Mock DELETE.
	mock.ExpectExec("DELETE FROM files WHERE").
		WithArgs("todelete", "b1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.DeleteFile(context.Background(), "todelete", "b1", "proj1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file is gone.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("expected file to be removed from disk")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- At-rest transform tests (compression + encryption) ---

func expectInsertFiles(mock sqlmock.Sqlmock) {
	mock.ExpectExec("INSERT INTO files").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestCreateFile_CompressionRoundTrip(t *testing.T) {
	svc, mock, tmpDir := setup(t)

	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("gzip", false, false))
	expectInsertFiles(mock)

	original := bytes.Repeat([]byte("compress me! "), 1000)
	f, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "data.txt", original, "text/plain", nil)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if f.SizeOriginal != int64(len(original)) {
		t.Errorf("expected original size %d, got %d", len(original), f.SizeOriginal)
	}

	// On disk the bytes are gzip-compressed and the path carries the .gz marker.
	diskPath := filepath.Join(tmpDir, "proj1", "b1", f.ID) + ".gz"
	raw, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("compressed file not written at %s: %v", diskPath, err)
	}
	if len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		t.Error("stored bytes are not gzip-compressed")
	}
	if len(raw) >= len(original) {
		t.Errorf("expected compression to shrink %d bytes, stored %d", len(original), len(raw))
	}

	// The read path transparently decompresses back to the original bytes.
	mock.ExpectQuery("SELECT path, mime_type FROM files WHERE").
		WithArgs(f.ID, "b1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"path", "mime_type"}).AddRow(diskPath, "text/plain"))

	got, mime, err := svc.GetFileContent(context.Background(), f.ID, "b1", "proj1")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Error("decompressed content does not match original")
	}
	if mime != "text/plain" {
		t.Errorf("mime mismatch: %q", mime)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestCreateFile_EncryptionRoundTrip(t *testing.T) {
	svc, mock, tmpDir := setup(t)
	if err := svc.SetEncryptionKey(strings.Repeat("ab", 32)); err != nil {
		t.Fatalf("SetEncryptionKey: %v", err)
	}

	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("", true, false))
	expectInsertFiles(mock)

	original := []byte("top secret payload — do not store in the clear")
	f, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "secret.txt", original, "text/plain", nil)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	// On disk the bytes are ciphertext and the path carries the .enc marker.
	diskPath := filepath.Join(tmpDir, "proj1", "b1", f.ID) + ".enc"
	raw, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("encrypted file not written at %s: %v", diskPath, err)
	}
	if bytes.Contains(raw, original) {
		t.Error("plaintext is present in the stored bytes")
	}

	mock.ExpectQuery("SELECT path, mime_type FROM files WHERE").
		WithArgs(f.ID, "b1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"path", "mime_type"}).AddRow(diskPath, "text/plain"))

	got, _, err := svc.GetFileContent(context.Background(), f.ID, "b1", "proj1")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Error("decrypted content does not match original")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestCreateFile_CompressAndEncryptRoundTrip(t *testing.T) {
	svc, mock, tmpDir := setup(t)
	if err := svc.SetEncryptionKey(strings.Repeat("cd", 16)); err != nil { // 16 bytes -> AES-128
		t.Fatalf("SetEncryptionKey: %v", err)
	}

	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("gzip", true, false))
	expectInsertFiles(mock)

	original := bytes.Repeat([]byte("both transforms applied. "), 500)
	f, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "both.txt", original, "text/plain", nil)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	diskPath := filepath.Join(tmpDir, "proj1", "b1", f.ID) + ".gz.enc"
	if _, err := os.Stat(diskPath); err != nil {
		t.Fatalf("expected file at %s: %v", diskPath, err)
	}

	mock.ExpectQuery("SELECT path, mime_type FROM files WHERE").
		WithArgs(f.ID, "b1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"path", "mime_type"}).AddRow(diskPath, "text/plain"))

	got, _, err := svc.GetFileContent(context.Background(), f.ID, "b1", "proj1")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Error("round-tripped content does not match original")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestCreateFile_EncryptionRequiresKey(t *testing.T) {
	svc, mock, _ := setup(t)

	// Encryption flag on but no key configured: the upload must be rejected,
	// never stored in the clear. No INSERT is expected.
	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("", true, false))

	_, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "x.txt", []byte("hi"), "text/plain", nil)
	if err == nil {
		t.Fatal("expected error when encryption is on but no key is configured")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestCreateFile_PerProjectDEKEncryptionRoundTrip covers the new envelope
// scheme: once a dek.Service is wired in, an encrypted bucket writes under
// the project's own key (not the legacy global STORAGE_ENCRYPTION_KEY), the
// stored path records the DEK version (".enc.vN"), and GetFileContent
// unwraps that exact version to decrypt — the version stays correct even
// though it comes from a project key wrapped under the master key rather
// than a flat instance-wide secret.
func TestCreateFile_PerProjectDEKEncryptionRoundTrip(t *testing.T) {
	svc, mock, tmpDir := setup(t)

	masterKeyHex := strings.Repeat("11", 32) // 64 hex chars -> 32 bytes
	database := &db.DB{DB: svc.db.DB}
	dekSvc, err := dek.NewService(database, masterKeyHex)
	if err != nil {
		t.Fatalf("dek.NewService: %v", err)
	}
	svc.SetDEKService(dekSvc)

	masterKey, err := dek.ParseMasterKey(masterKeyHex)
	if err != nil {
		t.Fatalf("ParseMasterKey: %v", err)
	}
	rawDEK := bytes.Repeat([]byte{0x42}, 32)
	wrapped, err := appladcrypto.SealToken("dek", 1, masterKey, rawDEK)
	if err != nil {
		t.Fatalf("SealToken: %v", err)
	}

	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("", true, false))
	mock.ExpectQuery("SELECT key_version, wrapped_dek").
		WithArgs("proj1").
		WillReturnRows(sqlmock.NewRows([]string{"key_version", "wrapped_dek"}).AddRow(1, wrapped))
	expectInsertFiles(mock)

	original := []byte("top secret payload — do not store in the clear")
	f, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "secret.txt", original, "text/plain", nil)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}

	// On disk the path carries the versioned marker, not the bare legacy ".enc".
	diskPath := filepath.Join(tmpDir, "proj1", "b1", f.ID) + ".enc.v1"
	raw, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("encrypted file not written at %s: %v", diskPath, err)
	}
	if bytes.Contains(raw, original) {
		t.Error("plaintext is present in the stored bytes")
	}

	// Reading it back unwraps DEK version 1 specifically (a fresh query, since
	// UnwrapVersion's cache is keyed separately from Unwrap's).
	mock.ExpectQuery("SELECT path, mime_type").
		WithArgs(f.ID, "b1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"path", "mime_type"}).AddRow(diskPath, "text/plain"))
	mock.ExpectQuery("SELECT wrapped_dek FROM project_encryption_keys").
		WithArgs("proj1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"wrapped_dek"}).AddRow(wrapped))

	got, mimeType, err := svc.GetFileContent(context.Background(), f.ID, "b1", "proj1")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("got %q, want %q", got, original)
	}
	if mimeType != "text/plain" {
		t.Fatalf("unexpected mime type %q", mimeType)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestGetFileContent_LegacyGlobalKeyStillDecodes proves upgrading an instance
// to a per-project dek.Service does not strand files already encrypted under
// the old global STORAGE_ENCRYPTION_KEY: a bare ".enc" suffix (no version
// marker) still decodes via the legacy key, never attempting to unwrap a
// project DEK that file was never encrypted with.
func TestGetFileContent_LegacyGlobalKeyStillDecodes(t *testing.T) {
	svc, mock, tmpDir := setup(t)
	if err := svc.SetEncryptionKey(strings.Repeat("ab", 32)); err != nil {
		t.Fatalf("SetEncryptionKey: %v", err)
	}
	// A dek.Service is also configured (as it would be on an upgraded
	// instance), but must not be consulted for a legacy-suffixed file.
	dekSvc, err := dek.NewService(&db.DB{DB: svc.db.DB}, strings.Repeat("11", 32))
	if err != nil {
		t.Fatalf("dek.NewService: %v", err)
	}
	svc.SetDEKService(dekSvc)

	original := []byte("legacy encrypted before per-project DEKs existed")
	sealed, err := svc.encrypt(original)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	diskPath := filepath.Join(tmpDir, "proj1", "b1", "f1") + ".enc"
	if err := os.MkdirAll(filepath.Dir(diskPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(diskPath, sealed, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	mock.ExpectQuery("SELECT path, mime_type").
		WithArgs("f1", "b1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"path", "mime_type"}).AddRow(diskPath, "text/plain"))

	got, _, err := svc.GetFileContent(context.Background(), "f1", "b1", "proj1")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("got %q, want %q", got, original)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (no project-DEK query should have run): %v", err)
	}
}

func TestSetEncryptionKey_Invalid(t *testing.T) {
	svc, _, _ := setup(t)
	if err := svc.SetEncryptionKey("not-hex!!"); err == nil {
		t.Error("expected error for non-hex key")
	}
	if err := svc.SetEncryptionKey("abcd"); err == nil {
		t.Error("expected error for wrong-length key (2 bytes)")
	}
	if err := svc.SetEncryptionKey(strings.Repeat("ab", 32)); err != nil {
		t.Errorf("expected 32-byte key to be accepted, got %v", err)
	}
}

// --- Antivirus wiring tests ---

func TestCreateFile_AntivirusClean(t *testing.T) {
	svc, mock, _ := setup(t)
	svc.SetClamAV(fakeClamd(t, "stream: OK"))

	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("", false, true))
	expectInsertFiles(mock)

	_, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "ok.txt", []byte("clean file"), "text/plain", nil)
	if err != nil {
		t.Fatalf("clean upload should succeed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestCreateFile_AntivirusInfected(t *testing.T) {
	svc, mock, _ := setup(t)
	svc.SetClamAV(fakeClamd(t, "stream: Eicar-Test-Signature FOUND"))

	// Rejected before any file record is written, so no INSERT is expected.
	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("", false, true))

	_, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "bad.txt", []byte("x5O!P%@AP"), "text/plain", nil)
	if err == nil || !strings.Contains(err.Error(), "rejected by antivirus") {
		t.Fatalf("expected antivirus rejection, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestCreateFile_AntivirusSkippedWhenBucketFlagOff(t *testing.T) {
	svc, mock, _ := setup(t)
	// A scanner is configured, but the bucket does not opt in. If the per-bucket
	// gate were ignored, this dead address would make the scan error and fail
	// the upload; instead the upload must succeed with no scan attempted.
	svc.SetClamAV("127.0.0.1:1")

	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs("b1", "proj1").
		WillReturnRows(bucketRowsFor("", false, false))
	expectInsertFiles(mock)

	_, err := svc.CreateFile(context.Background(), "proj1", "b1", "unique()", "ok.txt", []byte("data"), "text/plain", nil)
	if err != nil {
		t.Fatalf("upload should skip scanning and succeed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- Image transformation tests ---

func TestTransformImage_Resize(t *testing.T) {
	svc, _, _ := setup(t)

	pngData := makePNG(t, 100, 100)

	resized, mime, err := svc.TransformImage(pngData, "image/png", 50, 0, 0, "png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("expected mime 'image/png', got %q", mime)
	}

	// Decode and verify dimensions.
	img, _, err := image.Decode(bytes.NewReader(resized))
	if err != nil {
		t.Fatalf("failed to decode resized image: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 50 {
		t.Errorf("expected width 50, got %d", bounds.Dx())
	}
	if bounds.Dy() != 50 {
		t.Errorf("expected height 50 (aspect preserved), got %d", bounds.Dy())
	}
}

func TestTransformImage_FormatConversion(t *testing.T) {
	svc, _, _ := setup(t)

	pngData := makePNG(t, 10, 10)

	result, mime, err := svc.TransformImage(pngData, "image/png", 0, 0, 80, "jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("expected mime 'image/jpeg', got %q", mime)
	}
	// JPEG magic bytes: 0xFF 0xD8
	if len(result) < 2 || result[0] != 0xFF || result[1] != 0xD8 {
		t.Error("output does not start with JPEG magic bytes (0xFF 0xD8)")
	}
}

func TestTransformImage_NotAnImage(t *testing.T) {
	svc, _, _ := setup(t)

	_, _, err := svc.TransformImage([]byte("not an image"), "text/plain", 50, 50, 0, "png")
	if err == nil {
		t.Fatal("expected error for non-image content, got nil")
	}
	if err.Error() != "not an image" {
		t.Errorf("expected 'not an image' error, got %q", err.Error())
	}
}

// --- Chunked upload tests ---

func TestInitChunkedUpload_CreatesDir(t *testing.T) {
	svc, mock, tmpDir := setup(t)

	mock.ExpectExec("INSERT INTO files").
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	id, err := svc.InitChunkedUpload(context.Background(), "proj1", "b1", "unique()", "bigfile.zip", "application/zip", 999999, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify chunk directory was created.
	chunkDir := filepath.Join(tmpDir, "_chunks", id)
	info, err := os.Stat(chunkDir)
	if err != nil {
		t.Fatalf("chunk dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected chunk path to be a directory")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestCompleteChunkedUpload_AssemblesChunks(t *testing.T) {
	svc, mock, tmpDir := setup(t)

	fileID := "chunkedfile1"
	projectID := "proj1"
	bucketID := "b1"

	// Create chunk directory and chunk files manually.
	chunkDir := filepath.Join(tmpDir, "_chunks", fileID)
	os.MkdirAll(chunkDir, 0755)
	os.WriteFile(filepath.Join(chunkDir, "000000"), []byte("AAA"), 0644)
	os.WriteFile(filepath.Join(chunkDir, "000001"), []byte("BBB"), 0644)
	os.WriteFile(filepath.Join(chunkDir, "000002"), []byte("CCC"), 0644)

	// CompleteChunkedUpload now reads the bucket to apply at-rest transforms.
	mock.ExpectQuery("SELECT id, name, permissions").
		WithArgs(bucketID, projectID).
		WillReturnRows(bucketRowsFor("", false, false))

	// Mock UPDATE for setting final path and size.
	mock.ExpectExec("UPDATE files SET path").
		WithArgs(
			sqlmock.AnyArg(), // path
			int64(9),         // total size = 3+3+3
			fileID, bucketID, projectID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock the GetFile query that CompleteChunkedUpload calls at the end.
	mock.ExpectQuery("SELECT .+ FROM files WHERE").
		WithArgs(fileID, bucketID, projectID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "bucket_id", "name", "mime_type", "size", "permissions", "created_at", "updated_at",
		}).AddRow(fileID, bucketID, "bigfile.zip", "application/zip", int64(9), []byte(`[]`), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)))

	f, err := svc.CompleteChunkedUpload(context.Background(), projectID, bucketID, fileID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.SizeOriginal != 9 {
		t.Errorf("expected size 9, got %d", f.SizeOriginal)
	}

	// Verify the assembled file content.
	finalPath := filepath.Join(tmpDir, projectID, bucketID, fileID)
	data, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("final file not found: %v", err)
	}
	if string(data) != "AAABBBCCC" {
		t.Errorf("expected 'AAABBBCCC', got %q", string(data))
	}

	// Verify chunk directory was cleaned up.
	if _, err := os.Stat(chunkDir); !os.IsNotExist(err) {
		t.Error("expected chunk directory to be removed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
