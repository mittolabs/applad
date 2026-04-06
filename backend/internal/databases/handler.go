package databases

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/model"
)

// Handler handles HTTP requests for databases.
type Handler struct {
	svc *Service
}

// NewHandler creates a new databases Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the databases router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Post("/", h.createDatabase)
	r.Get("/", h.listDatabases)
	r.Get("/{databaseId}", h.getDatabase)
	r.Put("/{databaseId}", h.updateDatabase)
	r.Delete("/{databaseId}", h.deleteDatabase)

	r.Post("/{databaseId}/collections", h.createCollection)
	r.Get("/{databaseId}/collections", h.listCollections)
	r.Get("/{databaseId}/collections/{collectionId}", h.getCollection)
	r.Put("/{databaseId}/collections/{collectionId}", h.updateCollection)
	r.Delete("/{databaseId}/collections/{collectionId}", h.deleteCollection)

	r.Post("/{databaseId}/collections/{collectionId}/attributes/string", h.createAttr("string"))
	r.Post("/{databaseId}/collections/{collectionId}/attributes/integer", h.createAttr("integer"))
	r.Post("/{databaseId}/collections/{collectionId}/attributes/float", h.createAttr("double"))
	r.Post("/{databaseId}/collections/{collectionId}/attributes/boolean", h.createAttr("boolean"))
	r.Post("/{databaseId}/collections/{collectionId}/attributes/email", h.createAttr("email"))
	r.Post("/{databaseId}/collections/{collectionId}/attributes/url", h.createAttr("url"))
	r.Post("/{databaseId}/collections/{collectionId}/attributes/datetime", h.createAttr("datetime"))
	r.Post("/{databaseId}/collections/{collectionId}/attributes/enum", h.createAttr("enum"))
	r.Get("/{databaseId}/collections/{collectionId}/attributes", h.listAttributes)
	r.Delete("/{databaseId}/collections/{collectionId}/attributes/{key}", h.deleteAttribute)

	r.Post("/{databaseId}/collections/{collectionId}/indexes", h.createIndex)
	r.Get("/{databaseId}/collections/{collectionId}/indexes", h.listIndexes)
	r.Delete("/{databaseId}/collections/{collectionId}/indexes/{key}", h.deleteIndex)

	r.Post("/{databaseId}/collections/{collectionId}/documents", h.createDocument)
	r.Get("/{databaseId}/collections/{collectionId}/documents", h.listDocuments)
	r.Get("/{databaseId}/collections/{collectionId}/documents/{documentId}", h.getDocument)
	r.Patch("/{databaseId}/collections/{collectionId}/documents/{documentId}", h.updateDocument)
	r.Delete("/{databaseId}/collections/{collectionId}/documents/{documentId}", h.deleteDocument)

	return r
}

func (h *Handler) createDatabase(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		DatabaseID string `json:"databaseId"`
		Name       string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	d, err := h.svc.CreateDatabase(r.Context(), projectID, body.DatabaseID, body.Name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) listDatabases(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbs, total, err := h.svc.ListDatabases(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if dbs == nil {
		dbs = []*model.Database{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "databases": dbs})
}

func (h *Handler) getDatabase(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	d, err := h.svc.GetDatabase(r.Context(), dbID, projectID)
	if err != nil {
		apperr.NotFound(w, "database")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) updateDatabase(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	d, err := h.svc.UpdateDatabase(r.Context(), dbID, projectID, body.Name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) deleteDatabase(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	if err := h.svc.DeleteDatabase(r.Context(), dbID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createCollection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	var body struct {
		CollectionID     string   `json:"collectionId"`
		Name             string   `json:"name"`
		Permissions      []string `json:"permissions"`
		DocumentSecurity bool     `json:"documentSecurity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	c, err := h.svc.CreateCollection(r.Context(), projectID, dbID, body.CollectionID, body.Name, body.Permissions, body.DocumentSecurity)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) listCollections(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	colls, total, err := h.svc.ListCollections(r.Context(), dbID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if colls == nil {
		colls = []*model.Collection{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "collections": colls})
}

func (h *Handler) getCollection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	c, err := h.svc.GetCollection(r.Context(), collID, dbID, projectID)
	if err != nil {
		apperr.NotFound(w, "collection")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) updateCollection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	var body struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		Enabled     *bool    `json:"enabled"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	c, err := h.svc.UpdateCollection(r.Context(), collID, dbID, projectID, body.Name, body.Permissions, enabled)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	if err := h.svc.DeleteCollection(r.Context(), collID, dbID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createAttr(attrType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collID := chi.URLParam(r, "collectionId")
		var body struct {
			Key      string      `json:"key"`
			Required bool        `json:"required"`
			Array    bool        `json:"array"`
			Default  interface{} `json:"default"`
			// type-specific
			Size     int         `json:"size"`
			Min      interface{} `json:"min"`
			Max      interface{} `json:"max"`
			Elements []string    `json:"elements"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
			apperr.BadRequest(w, "key is required")
			return
		}
		options := map[string]interface{}{}
		switch attrType {
		case "string":
			if body.Size > 0 {
				options["size"] = body.Size
			}
		case "integer", "double":
			if body.Min != nil {
				options["min"] = body.Min
			}
			if body.Max != nil {
				options["max"] = body.Max
			}
		case "enum":
			options["elements"] = body.Elements
		}
		attr, err := h.svc.CreateAttribute(r.Context(), collID, body.Key, attrType, body.Required, body.Array, body.Default, options)
		if err != nil {
			apperr.Internal(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, attr)
	}
}

func (h *Handler) listAttributes(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "collectionId")
	attrs, err := h.svc.ListAttributes(r.Context(), collID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(attrs), "attributes": attrs})
}

func (h *Handler) deleteAttribute(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "collectionId")
	key := chi.URLParam(r, "key")
	if err := h.svc.DeleteAttribute(r.Context(), collID, key); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createIndex(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "collectionId")
	var body struct {
		Key        string   `json:"key"`
		Type       string   `json:"type"`
		Attributes []string `json:"attributes"`
		Orders     []string `json:"orders"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		apperr.BadRequest(w, "key is required")
		return
	}
	idx, err := h.svc.CreateIndex(r.Context(), collID, body.Key, body.Type, body.Attributes, body.Orders)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, idx)
}

func (h *Handler) listIndexes(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "collectionId")
	idxs, err := h.svc.ListIndexes(r.Context(), collID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(idxs), "indexes": idxs})
}

func (h *Handler) deleteIndex(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "collectionId")
	key := chi.URLParam(r, "key")
	if err := h.svc.DeleteIndex(r.Context(), collID, key); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	var body struct {
		DocumentID  string                 `json:"documentId"`
		Data        map[string]interface{} `json:"data"`
		Permissions []string               `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Data == nil {
		body.Data = map[string]interface{}{}
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	doc, err := h.svc.CreateDocument(ctx, projectID, dbID, collID, body.DocumentID, body.Data, body.Permissions)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	pg := middleware.ParsePagination(r)

	params := ListParams{
		Limit:       pg.Limit,
		Offset:      pg.Offset,
		OrderAttr:   r.URL.Query().Get("orderAttr"),
		OrderType:   r.URL.Query().Get("orderType"),
		CursorAfter: r.URL.Query().Get("cursorAfter"),
	}

	// Parse queries: ?queries[]=equal("name","John")&queries[]=greaterThan("age",18)
	for _, q := range r.URL.Query()["queries[]"] {
		parsed := parseQueryString(q)
		if parsed != nil {
			params.Queries = append(params.Queries, *parsed)
		}
	}

	docs, total, err := h.svc.ListDocumentsWithQuery(ctx, projectID, dbID, collID, params)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if docs == nil {
		docs = []*model.Document{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "documents": docs})
}

// parseQueryString parses Appwrite-style query strings like:
// equal("name","John") or greaterThan("age",18) or isNull("field")
func parseQueryString(q string) *Query {
	// Find method name (before the opening paren)
	parenIdx := strings.Index(q, "(")
	if parenIdx < 0 {
		return nil
	}
	method := strings.TrimSpace(q[:parenIdx])
	inner := strings.TrimSpace(q[parenIdx+1:])
	if len(inner) > 0 && inner[len(inner)-1] == ')' {
		inner = inner[:len(inner)-1]
	}

	// Parse arguments — split by comma, respecting quotes
	args := splitQueryArgs(inner)

	// Clean up quoted strings
	for i, a := range args {
		a = strings.TrimSpace(a)
		if len(a) >= 2 && a[0] == '"' && a[len(a)-1] == '"' {
			args[i] = a[1 : len(a)-1]
		} else {
			args[i] = a
		}
	}

	switch method {
	case "equal", "notEqual", "contains", "search", "startsWith", "endsWith":
		if len(args) < 2 {
			return nil
		}
		return &Query{Attribute: args[0], Method: method, Values: args[1]}
	case "lessThan", "greaterThan", "lessThanEqual", "greaterThanEqual":
		if len(args) < 2 {
			return nil
		}
		return &Query{Attribute: args[0], Method: method, Values: args[1]}
	case "between":
		if len(args) < 3 {
			return nil
		}
		return &Query{Attribute: args[0], Method: method, Values: []interface{}{args[1], args[2]}}
	case "isNull", "isNotNull":
		if len(args) < 1 {
			return nil
		}
		return &Query{Attribute: args[0], Method: method}
	case "orderAsc":
		// Handled via orderAttr/orderType params — ignore here
		return nil
	case "orderDesc":
		return nil
	default:
		return nil
	}
}

func splitQueryArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	for _, ch := range s {
		switch {
		case ch == '"':
			inQuote = !inQuote
			current.WriteRune(ch)
		case ch == ',' && !inQuote:
			args = append(args, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func (h *Handler) getDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	docID := chi.URLParam(r, "documentId")
	doc, err := h.svc.GetDocument(ctx, docID, collID, dbID, projectID)
	if err != nil {
		apperr.NotFound(w, "document")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) updateDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	docID := chi.URLParam(r, "documentId")
	var body struct {
		Data        map[string]interface{} `json:"data"`
		Permissions []string               `json:"permissions"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	doc, err := h.svc.UpdateDocument(ctx, docID, collID, dbID, projectID, body.Data, body.Permissions)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "collectionId")
	docID := chi.URLParam(r, "documentId")
	if err := h.svc.DeleteDocument(ctx, docID, collID, dbID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
