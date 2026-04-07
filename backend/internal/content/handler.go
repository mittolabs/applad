package content

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the CMS content layer HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new content Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the content router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Content types
	r.Post("/types", h.createType)
	r.Get("/types", h.listTypes)
	r.Get("/types/{typeId}", h.getType)
	r.Put("/types/{typeId}", h.updateType)
	r.Delete("/types/{typeId}", h.deleteType)

	// Entries
	r.Post("/types/{typeId}/entries", h.createEntry)
	r.Get("/types/{typeId}/entries", h.listEntries)
	r.Get("/types/{typeId}/entries/{entryId}", h.getEntry)
	r.Put("/types/{typeId}/entries/{entryId}", h.updateEntry)
	r.Delete("/types/{typeId}/entries/{entryId}", h.deleteEntry)

	// Publish workflow
	r.Patch("/types/{typeId}/entries/{entryId}/publish", h.publish)
	r.Patch("/types/{typeId}/entries/{entryId}/unpublish", h.unpublish)

	// Version history
	r.Get("/types/{typeId}/entries/{entryId}/versions", h.listVersions)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// ── Content types ─────────────────────────────────────────────────────────────

func (h *Handler) createType(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name         string  `json:"name"`
		Slug         string  `json:"slug"`
		Fields       []Field `json:"fields"`
		Versioning   bool    `json:"versioning"`
		Localization bool    `json:"localization"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Slug == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name and slug are required")
		return
	}
	ct, err := h.svc.CreateType(r.Context(), projectID, body.Name, body.Slug, body.Fields, body.Versioning, body.Localization)
	if err != nil {
		apperr.Write(w, http.StatusConflict, "content_type_exists", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ct)
}

func (h *Handler) listTypes(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	types, err := h.svc.ListTypes(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if types == nil {
		types = []*ContentType{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(types), "types": types})
}

func (h *Handler) getType(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	typeID := chi.URLParam(r, "typeId")
	ct, err := h.svc.GetType(r.Context(), typeID, projectID)
	if err != nil {
		apperr.NotFound(w, "content_type")
		return
	}
	writeJSON(w, http.StatusOK, ct)
}

func (h *Handler) updateType(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	typeID := chi.URLParam(r, "typeId")
	var body struct {
		Name   string  `json:"name"`
		Fields []Field `json:"fields"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	ct, err := h.svc.UpdateType(r.Context(), typeID, projectID, body.Name, body.Fields)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ct)
}

func (h *Handler) deleteType(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	typeID := chi.URLParam(r, "typeId")
	if err := h.svc.DeleteType(r.Context(), typeID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Entries ───────────────────────────────────────────────────────────────────

func (h *Handler) createEntry(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	typeID := chi.URLParam(r, "typeId")
	var body struct {
		Slug   string                 `json:"slug"`
		Locale string                 `json:"locale"`
		Data   map[string]interface{} `json:"data"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	authorID := middleware.UserFromContext(r.Context())
	entry, err := h.svc.CreateEntry(r.Context(), typeID, projectID, body.Slug, body.Locale, authorID, body.Data)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h *Handler) listEntries(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	typeID := chi.URLParam(r, "typeId")
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit == 0 {
		limit = 50
	}
	entries, total, err := h.svc.ListEntries(r.Context(), typeID, projectID, q.Get("status"), q.Get("locale"), limit, offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if entries == nil {
		entries = []*Entry{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": total, "limit": limit, "offset": offset, "entries": entries,
	})
}

func (h *Handler) getEntry(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	entryID := chi.URLParam(r, "entryId")
	entry, err := h.svc.GetEntry(r.Context(), entryID, projectID)
	if err != nil {
		apperr.NotFound(w, "content_entry")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) updateEntry(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	entryID := chi.URLParam(r, "entryId")
	var body struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "data is required")
		return
	}
	authorID := middleware.UserFromContext(r.Context())
	entry, err := h.svc.UpdateEntry(r.Context(), entryID, projectID, authorID, body.Data)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	entryID := chi.URLParam(r, "entryId")
	if err := h.svc.DeleteEntry(r.Context(), entryID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	entryID := chi.URLParam(r, "entryId")
	if err := h.svc.PublishEntry(r.Context(), entryID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "published"})
}

func (h *Handler) unpublish(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	entryID := chi.URLParam(r, "entryId")
	if err := h.svc.UnpublishEntry(r.Context(), entryID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "draft"})
}

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	entryID := chi.URLParam(r, "entryId")
	versions, err := h.svc.ListVersions(r.Context(), entryID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if versions == nil {
		versions = []*Version{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(versions), "versions": versions})
}
