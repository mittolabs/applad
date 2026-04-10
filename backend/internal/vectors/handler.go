package vectors

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the vector service HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new vectors Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the AI/vector router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Index management
	r.Post("/indexes", h.createIndex)
	r.Get("/indexes", h.listIndexes)
	r.Get("/indexes/{indexId}", h.getIndex)
	r.Delete("/indexes/{indexId}", h.deleteIndex)

	// Embedding CRUD
	r.Post("/indexes/{indexId}/vectors", h.upsertBatch)
	r.Put("/indexes/{indexId}/embeddings/{docId}", h.upsert)
	r.Delete("/indexes/{indexId}/embeddings/{docId}", h.delete)
	r.Post("/indexes/{indexId}/delete", h.deleteBatch)

	// Similarity search
	r.Post("/indexes/{indexId}/query", h.query)

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
		IndexID        string `json:"indexId"`
		Name           string `json:"name"`
		Dimensions     int    `json:"dimensions"`
		Metric         string `json:"metric"`
		CollectionID   string `json:"collectionId"`
		EmbeddingField string `json:"embeddingField"`
		Model          string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	idx, err := h.svc.CreateIndex(r.Context(), projectID, body.IndexID, body.Name, body.Dimensions, body.Metric,
		body.CollectionID, body.EmbeddingField, body.Model)
	if err != nil {
		apperr.Write(w, http.StatusConflict, "vector_index_exists", err.Error())
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
		indexes = []*VectorIndex{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(indexes), "indexes": indexes})
}

func (h *Handler) getIndex(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	idx, err := h.svc.GetIndex(r.Context(), indexID, projectID)
	if err != nil {
		apperr.NotFound(w, "vector_index")
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

func (h *Handler) upsert(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	docID := chi.URLParam(r, "docId")
	var body struct {
		Vector   []float64              `json:"vector"`
		Metadata map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Vector) == 0 {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "vector is required")
		return
	}
	emb, err := h.svc.Upsert(r.Context(), indexID, projectID, docID, body.Vector, body.Metadata)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, emb)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	indexID := chi.URLParam(r, "indexId")
	docID := chi.URLParam(r, "docId")
	if err := h.svc.Delete(r.Context(), indexID, docID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) upsertBatch(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	var body struct {
		Vectors []struct {
			ID       string                 `json:"id"`
			Values   []float64              `json:"values"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"vectors"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Vectors) == 0 {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "vectors are required")
		return
	}
	results := make([]*Embedding, 0, len(body.Vectors))
	for _, item := range body.Vectors {
		if item.ID == "" || len(item.Values) == 0 {
			continue
		}
		emb, err := h.svc.Upsert(r.Context(), indexID, projectID, item.ID, item.Values, item.Metadata)
		if err != nil {
			apperr.Internal(w, err)
			return
		}
		results = append(results, emb)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(results), "vectors": results})
}

func (h *Handler) deleteBatch(w http.ResponseWriter, r *http.Request) {
	indexID := chi.URLParam(r, "indexId")
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.IDs) == 0 {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "ids are required")
		return
	}
	for _, id := range body.IDs {
		if err := h.svc.Delete(r.Context(), indexID, id); err != nil {
			apperr.Internal(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": len(body.IDs)})
}

func (h *Handler) query(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	indexID := chi.URLParam(r, "indexId")
	var body struct {
		Vector         []float64 `json:"vector"`
		TopK           int       `json:"topK"`
		ScoreThreshold float64   `json:"scoreThreshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Vector) == 0 {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "vector is required")
		return
	}
	if body.TopK == 0 {
		if k, err := strconv.Atoi(r.URL.Query().Get("topK")); err == nil {
			body.TopK = k
		}
	}
	results, err := h.svc.Query(r.Context(), indexID, projectID, body.Vector, body.TopK, body.ScoreThreshold)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if results == nil {
		results = []SimilarityResult{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(results), "hits": results})
}
