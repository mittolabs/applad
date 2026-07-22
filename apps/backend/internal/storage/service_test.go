package storage

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
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
