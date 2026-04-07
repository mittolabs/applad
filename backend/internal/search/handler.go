package search

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the search HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new search Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the search router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Index management
	r.Post("/indexes", h.createIndex)
	r.Get("/indexes", h.listIndexes)
	r.Get("/indexes/{indexId}", h.getIndex)
	r.Delete("/indexes/{indexId}", h.deleteIndex)

	// Document management
	r.Put("/indexes/{indexId}/documents/{docId}", h.upsertDocument)
	r.Delete("/indexes/{indexId}/documents/{docId}", h.deleteDocument)

	// Query
	r.Post("/indexes/{indexId}/query", h.query)
	r.Get("/indexes/{indexId}/query", h.queryGET)

	// Synonyms
	r.Post("/indexes/{indexId}/synonyms", h.addSynonym)
	r.Get("/indexes/{indexId}/synonyms", h.listSynonyms)
	r.Delete("/indexes/{indexId}/synonyms/{synId}", h.deleteSynonym)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) createIndex(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		CollectionID  string   `json:"collectionId"`
		Name          string   `json:"name"`
		Fields        []string `json:"fields"`
		TypoTolerance bool     `json:"typoTolerance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	if len(body.Fields) == 0 {
		body.Fields = []string{"content"}
	}
	idx, err := h.svc.CreateIndex(r.Context(), projectID, body.CollectionID, body.Name, body.Fields, body.TypoTolerance)
	if err != nil {
		apperr.Write(w, http.StatusConflict, "search_index_exists", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, idx)
}

func (h *Handler) listIndexes(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexes, err := h.svc.ListIndexes(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if indexes == nil {
		indexes = []*Index{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(indexes), "indexes": indexes})
}

func (h *Handler) getIndex(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	idx, err := h.svc.GetIndex(r.Context(), indexID, projectID)
	if err != nil {
		apperr.NotFound(w, "search_index")
		return
	}
	writeJSON(w, http.StatusOK, idx)
}

func (h *Handler) deleteIndex(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	if err := h.svc.DeleteIndex(r.Context(), indexID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) upsertDocument(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	docID := chi.URLParam(r, "docId")
	var body struct {
		Content  string                 `json:"content"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Content == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "content is required")
		return
	}
	doc, err := h.svc.Upsert(r.Context(), indexID, projectID, docID, body.Content, body.Metadata)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	indexID := chi.URLParam(r, "indexId")
	docID := chi.URLParam(r, "docId")
	if err := h.svc.DeleteDocument(r.Context(), indexID, docID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) query(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	var body struct {
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Query == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "query is required")
		return
	}
	result, err := h.svc.Query(r.Context(), indexID, projectID, body.Query, body.Limit, body.Offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) queryGET(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	q := r.URL.Query().Get("q")
	if q == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "q is required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	result, err := h.svc.Query(r.Context(), indexID, projectID, q, limit, offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) addSynonym(w http.ResponseWriter, r *http.Request) {
	indexID := chi.URLParam(r, "indexId")
	var body struct {
		Synonyms []string `json:"synonyms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Synonyms) < 2 {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "at least 2 synonyms required")
		return
	}
	syn, err := h.svc.AddSynonym(r.Context(), indexID, body.Synonyms)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, syn)
}

func (h *Handler) listSynonyms(w http.ResponseWriter, r *http.Request) {
	indexID := chi.URLParam(r, "indexId")
	syns, err := h.svc.ListSynonyms(r.Context(), indexID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if syns == nil {
		syns = []*Synonym{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(syns), "synonyms": syns})
}

func (h *Handler) deleteSynonym(w http.ResponseWriter, r *http.Request) {
	synID := chi.URLParam(r, "synId")
	if err := h.svc.DeleteSynonym(r.Context(), synID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
