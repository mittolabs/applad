package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/model"
)

// Handler handles HTTP requests for storage.
type Handler struct {
	svc *Service
}

// NewHandler creates a new storage Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the storage router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/buckets", h.createBucket)
	r.Get("/buckets", h.listBuckets)
	r.Get("/buckets/{bucketId}", h.getBucket)
	r.Put("/buckets/{bucketId}", h.updateBucket)
	r.Delete("/buckets/{bucketId}", h.deleteBucket)
	r.Post("/buckets/{bucketId}/files", h.createFile)
	r.Get("/buckets/{bucketId}/files", h.listFiles)
	r.Get("/buckets/{bucketId}/files/{fileId}", h.getFile)
	r.Get("/buckets/{bucketId}/files/{fileId}/download", h.downloadFile)
	r.Get("/buckets/{bucketId}/files/{fileId}/view", h.viewFile)
	r.Get("/buckets/{bucketId}/files/{fileId}/preview", h.previewFile)
	r.Post("/buckets/{bucketId}/files/chunked", h.initChunkedUpload)
	r.Patch("/buckets/{bucketId}/files/{fileId}/chunks", h.uploadChunk)
	r.Post("/buckets/{bucketId}/files/{fileId}/chunks/complete", h.completeChunkedUpload)
	r.Delete("/buckets/{bucketId}/files/{fileId}", h.deleteFile)
	return r
}

func (h *Handler) createBucket(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		BucketID            string   `json:"bucketId"`
		Name                string   `json:"name"`
		Permissions         []string `json:"permissions"`
		FileSizeLimit       int64    `json:"maximumFileSize"`
		AllowedMimeTypes    []string `json:"allowedFileExtensions"`
		Compression         string   `json:"compression"`
		Encryption          bool     `json:"encryption"`
		Antivirus           bool     `json:"antivirus"`
		FileSecurity        bool     `json:"fileSecurity"`
		ImageTransformations *bool   `json:"imageTransformations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	if body.AllowedMimeTypes == nil {
		body.AllowedMimeTypes = []string{}
	}
	imgTransform := true
	if body.ImageTransformations != nil {
		imgTransform = *body.ImageTransformations
	}
	b, err := h.svc.CreateBucket(r.Context(), projectID, body.BucketID, body.Name, body.Permissions, body.FileSizeLimit, body.AllowedMimeTypes, body.Compression, body.Encryption, body.Antivirus, body.FileSecurity, imgTransform)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (h *Handler) listBuckets(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	buckets, total, err := h.svc.ListBuckets(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if buckets == nil {
		buckets = []*model.Bucket{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "buckets": buckets})
}

func (h *Handler) getBucket(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	bucketID := chi.URLParam(r, "bucketId")
	b, err := h.svc.GetBucket(r.Context(), bucketID, projectID)
	if err != nil {
		apperr.NotFound(w, "bucket")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *Handler) updateBucket(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	bucketID := chi.URLParam(r, "bucketId")
	var body struct {
		Name                 string   `json:"name"`
		Permissions          []string `json:"permissions"`
		FileSizeLimit        int64    `json:"maximumFileSize"`
		Enabled              *bool    `json:"enabled"`
		AllowedMimeTypes     []string `json:"allowedFileExtensions"`
		Compression          string   `json:"compression"`
		Encryption           *bool    `json:"encryption"`
		Antivirus            *bool    `json:"antivirus"`
		FileSecurity         *bool    `json:"fileSecurity"`
		ImageTransformations *bool    `json:"imageTransformations"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	// Fetch current bucket to use as defaults for unset fields
	current, err := h.svc.GetBucket(r.Context(), bucketID, projectID)
	if err != nil {
		apperr.NotFound(w, "bucket")
		return
	}
	if body.Name == "" {
		body.Name = current.Name
	}
	if body.Permissions == nil {
		body.Permissions = current.Permissions
	}
	if body.FileSizeLimit == 0 {
		body.FileSizeLimit = current.FileSizeLimit
	}
	if body.AllowedMimeTypes == nil {
		body.AllowedMimeTypes = current.AllowedFileExtensions
	}
	if body.Compression == "" {
		body.Compression = current.Compression
	}
	enabled := current.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	encryption := current.Encryption
	if body.Encryption != nil {
		encryption = *body.Encryption
	}
	antivirus := current.Antivirus
	if body.Antivirus != nil {
		antivirus = *body.Antivirus
	}
	fileSecurity := current.FileSecurity
	if body.FileSecurity != nil {
		fileSecurity = *body.FileSecurity
	}
	imgTransform := current.ImageTransformations
	if body.ImageTransformations != nil {
		imgTransform = *body.ImageTransformations
	}

	b, err := h.svc.UpdateBucket(r.Context(), bucketID, projectID, body.Name, body.Permissions, body.FileSizeLimit, body.AllowedMimeTypes, body.Compression, encryption, antivirus, fileSecurity, imgTransform, enabled)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *Handler) deleteBucket(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	bucketID := chi.URLParam(r, "bucketId")
	if err := h.svc.DeleteBucket(r.Context(), bucketID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apperr.BadRequest(w, "invalid multipart form")
		return
	}

	fileID := r.FormValue("fileId")
	permsStr := r.FormValue("permissions")
	var permissions []string
	if permsStr != "" {
		json.Unmarshal([]byte(permsStr), &permissions) //nolint:errcheck
	}
	if permissions == nil {
		permissions = []string{}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apperr.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		apperr.Internal(w, fmt.Errorf("read upload: %w", err))
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	f, err := h.svc.CreateFile(ctx, projectID, bucketID, fileID, header.Filename, content, mimeType, permissions)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")
	pg := middleware.ParsePagination(r)
	files, total, err := h.svc.ListFiles(ctx, projectID, bucketID, pg.Limit, pg.Offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if files == nil {
		files = []*model.File{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "files": files})
}

func (h *Handler) getFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")
	fileID := chi.URLParam(r, "fileId")
	f, err := h.svc.GetFile(ctx, fileID, bucketID, projectID)
	if err != nil {
		apperr.NotFound(w, "file")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request) {
	h.serveFile(w, r, true)
}

func (h *Handler) viewFile(w http.ResponseWriter, r *http.Request) {
	h.serveFile(w, r, false)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, download bool) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")
	fileID := chi.URLParam(r, "fileId")
	content, mimeType, err := h.svc.GetFileContent(ctx, fileID, bucketID, projectID)
	if err != nil {
		apperr.NotFound(w, "file")
		return
	}
	w.Header().Set("Content-Type", mimeType)
	if download {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileID))
	} else {
		w.Header().Set("Content-Disposition", "inline")
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	w.Write(content) //nolint:errcheck
}

func (h *Handler) previewFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")
	fileID := chi.URLParam(r, "fileId")

	content, mimeType, err := h.svc.GetFileContent(ctx, fileID, bucketID, projectID)
	if err != nil {
		apperr.NotFound(w, "file")
		return
	}

	// Parse transformation params
	width, _ := strconv.Atoi(r.URL.Query().Get("width"))
	height, _ := strconv.Atoi(r.URL.Query().Get("height"))
	quality, _ := strconv.Atoi(r.URL.Query().Get("quality"))
	output := r.URL.Query().Get("output") // png, jpg, webp
	gravity := r.URL.Query().Get("gravity")
	borderWidth, _ := strconv.Atoi(r.URL.Query().Get("borderWidth"))
	borderColor := r.URL.Query().Get("borderColor")
	borderRadius, _ := strconv.Atoi(r.URL.Query().Get("borderRadius"))
	opacity, _ := strconv.Atoi(r.URL.Query().Get("opacity"))
	rotation, _ := strconv.Atoi(r.URL.Query().Get("rotation"))
	background := r.URL.Query().Get("background")

	hasTransform := width > 0 || height > 0 || output != "" || gravity != "" ||
		borderWidth > 0 || borderRadius > 0 || (opacity > 0 && opacity < 100) ||
		rotation > 0 || background != ""

	if !hasTransform {
		// No transformations, serve as-is
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Disposition", "inline")
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.Write(content)
		return
	}

	// Apply image transformations
	opts := ImageTransformOpts{
		Width:        width,
		Height:       height,
		Quality:      quality,
		OutputFormat: output,
		Gravity:      gravity,
		BorderWidth:  borderWidth,
		BorderColor:  borderColor,
		BorderRadius: borderRadius,
		Opacity:      opacity,
		Rotation:     rotation,
		Background:   background,
	}
	transformed, newMime, err := h.svc.TransformImageAdvanced(content, mimeType, opts)
	if err != nil {
		// If transformation fails (not an image), serve original
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Disposition", "inline")
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.Write(content)
		return
	}

	w.Header().Set("Content-Type", newMime)
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Content-Length", strconv.Itoa(len(transformed)))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(transformed)
}

// --- Chunked uploads ---

func (h *Handler) initChunkedUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")

	var body struct {
		FileID   string   `json:"fileId"`
		Name     string   `json:"name"`
		MimeType string   `json:"mimeType"`
		Size     int64    `json:"size"`
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.MimeType == "" {
		body.MimeType = "application/octet-stream"
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}

	fileID, err := h.svc.InitChunkedUpload(ctx, projectID, bucketID, body.FileID, body.Name, body.MimeType, body.Size, body.Permissions)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"fileId":    fileID,
		"uploadUrl": fmt.Sprintf("/v1/storage/buckets/%s/files/%s/chunks", bucketID, fileID),
	})
}

func (h *Handler) uploadChunk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")
	fileID := chi.URLParam(r, "fileId")

	// Parse Content-Range header: bytes start-end/total
	rangeHeader := r.Header.Get("Content-Range")
	chunkIndex, _ := strconv.Atoi(r.URL.Query().Get("chunk"))

	chunk, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB per chunk
	if err != nil {
		apperr.BadRequest(w, "failed to read chunk")
		return
	}

	if err := h.svc.UploadChunk(ctx, projectID, bucketID, fileID, chunkIndex, chunk); err != nil {
		apperr.Internal(w, err)
		return
	}

	_ = rangeHeader
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"chunk":    chunkIndex,
		"received": len(chunk),
	})
}

func (h *Handler) completeChunkedUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")
	fileID := chi.URLParam(r, "fileId")

	f, err := h.svc.CompleteChunkedUpload(ctx, projectID, bucketID, fileID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	bucketID := chi.URLParam(r, "bucketId")
	fileID := chi.URLParam(r, "fileId")
	if err := h.svc.DeleteFile(ctx, fileID, bucketID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
