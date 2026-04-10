// Package storage implements Applad's storage service:
// buckets, file upload (chunked), file retrieval, image transformations, encryption, and antivirus.
package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
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
	storagePath string // kept for local-driver chunk temp path
	driver      Driver
	clamavAddr  string
	jwtSecret   string
	events      realtime.EventPublisher
}

// NewService creates a new storage Service using the local filesystem driver.
func NewService(database *db.DB, storagePath string, jwtSecret ...string) *Service {
	svc := &Service{
		db:          database,
		storagePath: storagePath,
		driver:      NewLocalDriver(storagePath),
	}
	if len(jwtSecret) > 0 {
		svc.jwtSecret = jwtSecret[0]
	}
	return svc
}

// SetDriver replaces the storage backend (call before first use).
func (s *Service) SetDriver(d Driver) { s.driver = d }

// SetEventPublisher sets the realtime event publisher.
func (s *Service) SetEventPublisher(pub realtime.EventPublisher) {
	s.events = pub
}

// SetClamAV configures antivirus scanning.
func (s *Service) SetClamAV(addr string) {
	s.clamavAddr = addr
}

// --- buckets ---

func (s *Service) CreateBucket(ctx context.Context, projectID, bucketID, name string, permissions []string, fileSizeLimit int64, allowedMimeTypes []string, compression string, encryption, antivirus, fileSecurity, imageTransformations bool) (*model.Bucket, error) {
	id := uid.New(bucketID)
	now := time.Now().UTC()
	permsJSON, _ := json.Marshal(permissions)
	mimeJSON, _ := json.Marshal(allowedMimeTypes)
	if compression == "" {
		compression = "none"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO buckets (id, project_id, name, permissions, file_size_limit, allowed_mime_types, compression, encryption, antivirus, file_security, image_transformations, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, name, permsJSON, fileSizeLimit, mimeJSON, compression, encryption, antivirus, fileSecurity, imageTransformations, now, now)
	if err != nil {
		return nil, fmt.Errorf("storage: create bucket: %w", err)
	}
	return &model.Bucket{
		ID: id, Name: name, Enabled: true,
		Permissions: permissions, FileSizeLimit: fileSizeLimit,
		AllowedFileExtensions: allowedMimeTypes,
		Compression: compression, Encryption: encryption, Antivirus: antivirus,
		FileSecurity: fileSecurity, ImageTransformations: imageTransformations,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Service) GetBucket(ctx context.Context, bucketID, projectID string) (*model.Bucket, error) {
	var b model.Bucket
	var permsJSON, mimeJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, permissions, file_size_limit, allowed_mime_types, compression, encryption, antivirus, file_security, image_transformations, enabled, created_at, updated_at FROM buckets WHERE id = ? AND project_id = ?",
		bucketID, projectID).Scan(&b.ID, &b.Name, &permsJSON, &b.FileSizeLimit, &mimeJSON, &b.Compression, &b.Encryption, &b.Antivirus, &b.FileSecurity, &b.ImageTransformations, &b.Enabled, &b.CreatedAt, &b.UpdatedAt)
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
		"SELECT id, name, permissions, file_size_limit, allowed_mime_types, compression, encryption, antivirus, file_security, image_transformations, enabled, created_at, updated_at FROM buckets WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var buckets []*model.Bucket
	for rows.Next() {
		var b model.Bucket
		var permsJSON, mimeJSON []byte
		if err := rows.Scan(&b.ID, &b.Name, &permsJSON, &b.FileSizeLimit, &mimeJSON, &b.Compression, &b.Encryption, &b.Antivirus, &b.FileSecurity, &b.ImageTransformations, &b.Enabled, &b.CreatedAt, &b.UpdatedAt); err != nil {
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

func (s *Service) UpdateBucket(ctx context.Context, bucketID, projectID, name string, permissions []string, fileSizeLimit int64, allowedMimeTypes []string, compression string, encryption, antivirus, fileSecurity, imageTransformations, enabled bool) (*model.Bucket, error) {
	permsJSON, _ := json.Marshal(permissions)
	mimeJSON, _ := json.Marshal(allowedMimeTypes)
	_, err := s.db.ExecContext(ctx,
		`UPDATE buckets SET name = ?, permissions = ?, file_size_limit = ?, allowed_mime_types = ?,
		 compression = ?, encryption = ?, antivirus = ?, file_security = ?, image_transformations = ?,
		 enabled = ? WHERE id = ? AND project_id = ?`,
		name, permsJSON, fileSizeLimit, mimeJSON,
		compression, encryption, antivirus, fileSecurity, imageTransformations,
		enabled, bucketID, projectID)
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
	// Sanitize filename: strip directory traversal components and enforce length.
	name = filepath.Base(name)
	if name == "." || name == "/" {
		name = "file"
	}
	if len(name) > 255 {
		name = name[:255]
	}

	// Enforce bucket-level quota and MIME restrictions
	bucket, err := s.GetBucket(ctx, bucketID, projectID)
	if err != nil {
		return nil, fmt.Errorf("storage: bucket not found")
	}
	if bucket.FileSizeLimit > 0 && int64(len(content)) > bucket.FileSizeLimit {
		return nil, fmt.Errorf("storage: file size %d exceeds bucket limit of %d bytes", len(content), bucket.FileSizeLimit)
	}
	// AllowedFileExtensions can hold extensions ("jpg") or MIME types ("image/jpeg").
	if len(bucket.AllowedFileExtensions) > 0 {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
		allowed := false
		for _, allowed_ := range bucket.AllowedFileExtensions {
			a := strings.ToLower(strings.TrimPrefix(allowed_, "."))
			if a == ext || strings.EqualFold(a, mimeType) || a == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("storage: file type %q not allowed in this bucket", ext)
		}
	}

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

	// Write via pluggable driver (local disk or S3)
	storagePath := s.driver.Path(projectID, bucketID, id)
	if err := s.driver.Write(ctx, storagePath, content); err != nil {
		return nil, fmt.Errorf("storage: write file: %w", err)
	}

	sig := fmt.Sprintf("%x", md5.Sum(content))
	permsJSON, _ := json.Marshal(permissions)

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO files (id, bucket_id, project_id, name, mime_type, size, permissions, path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, bucketID, projectID, name, mimeType, int64(len(content)), permsJSON, storagePath, now, now)
	if err != nil {
		s.driver.Delete(ctx, storagePath) //nolint:errcheck
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
	content, err := s.driver.Read(ctx, path)
	if err != nil {
		return nil, "", err
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

	// Store chunks under storagePath/_chunks/<id> for local assembly.
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

// UploadChunk writes a single chunk to the chunk staging directory.
func (s *Service) UploadChunk(_ context.Context, projectID, bucketID, fileID string, index int, data []byte) error {
	chunkDir := filepath.Join(s.storagePath, "_chunks", fileID)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return err
	}
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%06d", index))
	return os.WriteFile(chunkPath, data, 0644)
}

// CompleteChunkedUpload assembles chunks and uploads via the configured driver.
func (s *Service) CompleteChunkedUpload(ctx context.Context, projectID, bucketID, fileID string) (*model.File, error) {
	chunkDir := filepath.Join(s.storagePath, "_chunks", fileID)

	entries, err := os.ReadDir(chunkDir)
	if err != nil {
		return nil, fmt.Errorf("storage: read chunks: %w", err)
	}

	// Assemble all chunks in-memory
	var buf bytes.Buffer
	for _, entry := range entries {
		chunk, err := os.ReadFile(filepath.Join(chunkDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("storage: read chunk %s: %w", entry.Name(), err)
		}
		buf.Write(chunk)
	}
	assembled := buf.Bytes()

	// Clean up temp chunks
	os.RemoveAll(chunkDir) //nolint:errcheck

	// Upload assembled file via driver
	finalPath := s.driver.Path(projectID, bucketID, fileID)
	if err := s.driver.Write(ctx, finalPath, assembled); err != nil {
		return nil, fmt.Errorf("storage: write assembled file: %w", err)
	}

	// Update file record with final path and size
	_, err = s.db.ExecContext(ctx,
		"UPDATE files SET path = ?, size = ? WHERE id = ? AND bucket_id = ? AND project_id = ?",
		finalPath, int64(len(assembled)), fileID, bucketID, projectID)
	if err != nil {
		return nil, err
	}

	return s.GetFile(ctx, fileID, bucketID, projectID)
}

// ImageTransformOpts holds all image transformation parameters.
type ImageTransformOpts struct {
	Width        int
	Height       int
	Quality      int
	OutputFormat string
	Gravity      string // center, top, bottom, left, right, top-left, top-right, bottom-left, bottom-right
	BorderWidth  int
	BorderColor  string // hex color e.g. "ff0000"
	BorderRadius int
	Opacity      int    // 0-100
	Rotation     int    // 0, 90, 180, 270
	Background   string // hex color for background fill
}

// TransformImage resizes, rotates, borders, and converts an image.
func (s *Service) TransformImage(content []byte, mimeType string, width, height, quality int, outputFormat string) ([]byte, string, error) {
	return s.TransformImageAdvanced(content, mimeType, ImageTransformOpts{
		Width:        width,
		Height:       height,
		Quality:      quality,
		OutputFormat: outputFormat,
	})
}

// TransformImageAdvanced applies the full set of image transformations.
// Order: resize (with gravity crop) -> rotate -> border radius -> border -> opacity -> background fill -> encode.
func (s *Service) TransformImageAdvanced(content []byte, mimeType string, opts ImageTransformOpts) ([]byte, string, error) {
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("not an image")
	}

	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	// 1. Resize (with gravity-based cropping)
	if opts.Width > 0 || opts.Height > 0 {
		if opts.Gravity != "" && opts.Gravity != "center" && opts.Width > 0 && opts.Height > 0 {
			img = cropToGravity(img, opts.Width, opts.Height, opts.Gravity)
		}
		img = resizeImage(img, opts.Width, opts.Height)
	}

	// 2. Rotate
	if opts.Rotation == 90 || opts.Rotation == 180 || opts.Rotation == 270 {
		img = rotateImage(img, opts.Rotation)
	}

	// 3. Border radius (round corners)
	if opts.BorderRadius > 0 {
		img = applyBorderRadius(img, opts.BorderRadius)
	}

	// 4. Border
	if opts.BorderWidth > 0 {
		borderCol := parseHexColor(opts.BorderColor)
		img = applyBorder(img, opts.BorderWidth, borderCol)
	}

	// 5. Opacity
	if opts.Opacity > 0 && opts.Opacity < 100 {
		img = applyOpacity(img, opts.Opacity)
	}

	// 6. Background fill (useful after rotation or border-radius with transparency)
	if opts.Background != "" {
		bgCol := parseHexColor(opts.Background)
		img = applyBackground(img, bgCol)
	}

	// Determine output format
	outFmt := opts.OutputFormat
	if outFmt == "" {
		if strings.Contains(mimeType, "png") {
			outFmt = "png"
		} else {
			outFmt = "jpg"
		}
	}

	q := opts.Quality
	if q <= 0 {
		q = 85
	}

	var buf bytes.Buffer
	var outMime string

	switch outFmt {
	case "png":
		err = png.Encode(&buf, img)
		outMime = "image/png"
	default: // jpg/jpeg
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: q})
		outMime = "image/jpeg"
	}
	if err != nil {
		return nil, "", fmt.Errorf("encode image: %w", err)
	}

	return buf.Bytes(), outMime, nil
}

// parseHexColor parses a hex color string (with or without #) into a color.RGBA.
func parseHexColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		// Expand shorthand: "f00" -> "ff0000"
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return color.RGBA{0, 0, 0, 255}
	}
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}

// rotateImage rotates an image by 90, 180, or 270 degrees.
func rotateImage(src image.Image, degrees int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	switch degrees {
	case 90:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(h-1-y, x, src.At(x+bounds.Min.X, y+bounds.Min.Y))
			}
		}
		return dst
	case 180:
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(w-1-x, h-1-y, src.At(x+bounds.Min.X, y+bounds.Min.Y))
			}
		}
		return dst
	case 270:
		dst := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.Set(y, w-1-x, src.At(x+bounds.Min.X, y+bounds.Min.Y))
			}
		}
		return dst
	}
	return src
}

// applyBorder draws a border of the given width and color around the image.
func applyBorder(src image.Image, borderWidth int, borderColor color.RGBA) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	newW := w + 2*borderWidth
	newH := h + 2*borderWidth

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	// Fill entire image with border color
	draw.Draw(dst, dst.Bounds(), &image.Uniform{borderColor}, image.Point{}, draw.Src)
	// Draw the original image centered
	draw.Draw(dst, image.Rect(borderWidth, borderWidth, borderWidth+w, borderWidth+h), src, bounds.Min, draw.Src)
	return dst
}

// applyBorderRadius rounds the corners of the image by making pixels outside
// the rounded rectangle transparent.
func applyBorderRadius(src image.Image, radius int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)

	if radius > w/2 {
		radius = w / 2
	}
	if radius > h/2 {
		radius = h / 2
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !insideRoundedRect(x, y, w, h, radius) {
				dst.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return dst
}

// insideRoundedRect checks whether (x,y) is inside a rounded rectangle.
func insideRoundedRect(x, y, w, h, r int) bool {
	// Check the four corner regions
	corners := [][2]int{
		{r, r},         // top-left
		{w - r, r},     // top-right
		{r, h - r},     // bottom-left
		{w - r, h - r}, // bottom-right
	}
	for _, c := range corners {
		cx, cy := c[0], c[1]
		// Determine if (x,y) is in this corner's quadrant
		inCorner := false
		switch {
		case x < r && y < r: // top-left
			inCorner = (cx == r && cy == r)
		case x >= w-r && y < r: // top-right
			inCorner = (cx == w-r && cy == r)
		case x < r && y >= h-r: // bottom-left
			inCorner = (cx == r && cy == h-r)
		case x >= w-r && y >= h-r: // bottom-right
			inCorner = (cx == w-r && cy == h-r)
		}
		if inCorner {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if math.Sqrt(dx*dx+dy*dy) > float64(r) {
				return false
			}
		}
	}
	return true
}

// applyOpacity blends the image with transparency (0 = fully transparent, 100 = opaque).
func applyOpacity(src image.Image, opacity int) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	alpha := float64(opacity) / 100.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := src.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			newA := uint8(float64(a>>8) * alpha)
			dst.SetRGBA(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), newA})
		}
	}
	return dst
}

// applyBackground composites the source image over a solid background color.
func applyBackground(src image.Image, bg color.RGBA) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, &image.Uniform{bg}, image.Point{}, draw.Src)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Over)
	return dst
}

// cropToGravity crops the source image based on gravity before resizing.
func cropToGravity(src image.Image, targetW, targetH int, gravity string) image.Image {
	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	// Calculate the crop rectangle that matches the target aspect ratio
	targetRatio := float64(targetW) / float64(targetH)
	srcRatio := float64(srcW) / float64(srcH)

	var cropW, cropH int
	if srcRatio > targetRatio {
		// Source is wider — crop width
		cropH = srcH
		cropW = int(float64(cropH) * targetRatio)
	} else {
		// Source is taller — crop height
		cropW = srcW
		cropH = int(float64(cropW) / targetRatio)
	}

	// Determine crop origin based on gravity
	var x0, y0 int
	switch gravity {
	case "top":
		x0 = (srcW - cropW) / 2
		y0 = 0
	case "bottom":
		x0 = (srcW - cropW) / 2
		y0 = srcH - cropH
	case "left":
		x0 = 0
		y0 = (srcH - cropH) / 2
	case "right":
		x0 = srcW - cropW
		y0 = (srcH - cropH) / 2
	case "top-left":
		x0 = 0
		y0 = 0
	case "top-right":
		x0 = srcW - cropW
		y0 = 0
	case "bottom-left":
		x0 = 0
		y0 = srcH - cropH
	case "bottom-right":
		x0 = srcW - cropW
		y0 = srcH - cropH
	default: // center
		x0 = (srcW - cropW) / 2
		y0 = (srcH - cropH) / 2
	}

	x0 += bounds.Min.X
	y0 += bounds.Min.Y

	cropped := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	draw.Draw(cropped, cropped.Bounds(), src, image.Pt(x0, y0), draw.Src)
	return cropped
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

// signedURLToken produces a compact HMAC-SHA256 token encoding bucketID+fileID+expiresAt.
// Format (base64url-encoded JSON): {"b":bucketID,"f":fileID,"e":unixExpiry}
func (s *Service) signedURLToken(bucketID, fileID string, expiresAt int64) string {
	secret := s.jwtSecret
	if secret == "" {
		secret = "applad-storage-default"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"b": bucketID,
		"f": fileID,
		"e": expiresAt,
	})
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := mac.Sum(nil)
	combined := append(payload, '.')
	combined = append(combined, sig...)
	return base64.RawURLEncoding.EncodeToString(combined)
}

// CreateSignedURL returns a time-limited token for serving a private file.
// expiresIn is the TTL in seconds (default 3600 if zero).
func (s *Service) CreateSignedURL(ctx context.Context, fileID, bucketID, projectID string, expiresIn int64) (string, error) {
	if _, err := s.GetFile(ctx, fileID, bucketID, projectID); err != nil {
		return "", fmt.Errorf("file not found")
	}
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	token := s.signedURLToken(bucketID, fileID, time.Now().Unix()+expiresIn)
	return token, nil
}

// VerifySignedToken validates a token and returns (bucketID, fileID, ok).
func (s *Service) VerifySignedToken(token string) (bucketID, fileID string, ok bool) {
	secret := s.jwtSecret
	if secret == "" {
		secret = "applad-storage-default"
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return
	}
	dotIdx := -1
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx < 0 {
		return
	}
	payload := raw[:dotIdx]
	sig := raw[dotIdx+1:]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return
	}
	exp, _ := claims["e"].(float64)
	if time.Now().Unix() > int64(exp) {
		return // expired
	}
	bucketID, _ = claims["b"].(string)
	fileID, _ = claims["f"].(string)
	ok = bucketID != "" && fileID != ""
	return
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
	s.driver.Delete(ctx, path) //nolint:errcheck
	realtime.PublishResourceEvent(s.events, "storage", "files", "delete", projectID, fileID, map[string]string{"$id": fileID})
	return nil
}
