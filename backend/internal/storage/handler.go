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
	r.Delete("/buckets/{bucketId}/files/{fileId}", h.deleteFile)
	return r
}

func (h *Handler) createBucket(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		BucketID         string   `json:"bucketId"`
		Name             string   `json:"name"`
		Permissions      []string `json:"permissions"`
		FileSizeLimit    int64    `json:"maximumFileSize"`
		AllowedMimeTypes []string `json:"allowedFileExtensions"`
		Compression      string   `json:"compression"`
		Encryption       bool     `json:"encryption"`
		Antivirus        bool     `json:"antivirus"`
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
	b, err := h.svc.CreateBucket(r.Context(), projectID, body.BucketID, body.Name, body.Permissions, body.FileSizeLimit, body.AllowedMimeTypes, body.Compression, body.Encryption, body.Antivirus)
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
		Name          string   `json:"name"`
		Permissions   []string `json:"permissions"`
		FileSizeLimit int64    `json:"maximumFileSize"`
		Enabled       *bool    `json:"enabled"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	b, err := h.svc.UpdateBucket(r.Context(), bucketID, projectID, body.Name, body.Permissions, body.FileSizeLimit, enabled)
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
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	files, total, err := h.svc.ListFiles(ctx, projectID, bucketID, limit, offset)
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
