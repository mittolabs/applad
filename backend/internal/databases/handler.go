package databases

import (
	"encoding/json"
	"io"
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

	// Tables (stored in `collections` MySQL table)
	r.Post("/{databaseId}/tables", h.createCollection)
	r.Get("/{databaseId}/tables", h.listCollections)
	r.Get("/{databaseId}/tables/{tableId}", h.getCollection)
	r.Put("/{databaseId}/tables/{tableId}", h.updateCollection)
	r.Delete("/{databaseId}/tables/{tableId}", h.deleteCollection)

	// Columns (stored in `attributes` MySQL table)
	r.Post("/{databaseId}/tables/{tableId}/columns/string", h.createAttr("string"))
	r.Post("/{databaseId}/tables/{tableId}/columns/integer", h.createAttr("integer"))
	r.Post("/{databaseId}/tables/{tableId}/columns/float", h.createAttr("double"))
	r.Post("/{databaseId}/tables/{tableId}/columns/boolean", h.createAttr("boolean"))
	r.Post("/{databaseId}/tables/{tableId}/columns/email", h.createAttr("email"))
	r.Post("/{databaseId}/tables/{tableId}/columns/url", h.createAttr("url"))
	r.Post("/{databaseId}/tables/{tableId}/columns/datetime", h.createAttr("datetime"))
	r.Post("/{databaseId}/tables/{tableId}/columns/enum", h.createAttr("enum"))
	r.Post("/{databaseId}/tables/{tableId}/columns/point", h.createAttr("point"))
	r.Get("/{databaseId}/tables/{tableId}/columns", h.listAttributes)
	r.Delete("/{databaseId}/tables/{tableId}/columns/{key}", h.deleteAttribute)

	// Relationships
	r.Post("/{databaseId}/tables/{tableId}/columns/relationship", h.createRelationship)
	r.Get("/{databaseId}/tables/{tableId}/relationships", h.listRelationships)
	r.Delete("/{databaseId}/tables/{tableId}/relationships/{relationshipId}", h.deleteRelationship)

	// Indexes
	r.Post("/{databaseId}/tables/{tableId}/indexes", h.createIndex)
	r.Get("/{databaseId}/tables/{tableId}/indexes", h.listIndexes)
	r.Delete("/{databaseId}/tables/{tableId}/indexes/{key}", h.deleteIndex)

	// Transactions
	r.Post("/{databaseId}/transactions", h.executeTransaction)

	// CSV Import
	r.Post("/{databaseId}/tables/{tableId}/import/csv", h.importCSV)
	r.Post("/{databaseId}/tables/{tableId}/import/csv/preview", h.previewCSV)

	// Rows (stored in `documents` MySQL table)
	r.Post("/{databaseId}/tables/{tableId}/rows", h.createDocument)
	r.Get("/{databaseId}/tables/{tableId}/rows", h.listDocuments)
	r.Get("/{databaseId}/tables/{tableId}/rows/{rowId}", h.getDocument)
	r.Patch("/{databaseId}/tables/{tableId}/rows/{rowId}", h.updateDocument)
	r.Delete("/{databaseId}/tables/{tableId}/rows/{rowId}", h.deleteDocument)

	// Permissions
	r.Post("/{databaseId}/tables/{tableId}/permissions", h.setCollectionPermissions)
	r.Get("/{databaseId}/tables/{tableId}/permissions", h.getCollectionPermissions)

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
		TableID          string   `json:"tableId"`
		Name             string   `json:"name"`
		Permissions      []string `json:"permissions"`
		RowSecurity      bool     `json:"rowSecurity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	c, err := h.svc.CreateCollection(r.Context(), projectID, dbID, body.TableID, body.Name, body.Permissions, body.RowSecurity)
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "tables": colls})
}

func (h *Handler) getCollection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
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
	collID := chi.URLParam(r, "tableId")
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
	collID := chi.URLParam(r, "tableId")
	if err := h.svc.DeleteCollection(r.Context(), collID, dbID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createAttr(attrType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collID := chi.URLParam(r, "tableId")
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
	collID := chi.URLParam(r, "tableId")
	attrs, err := h.svc.ListAttributes(r.Context(), collID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(attrs), "columns": attrs})
}

func (h *Handler) deleteAttribute(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "tableId")
	key := chi.URLParam(r, "key")
	if err := h.svc.DeleteAttribute(r.Context(), collID, key); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createIndex(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "tableId")
	var body struct {
		Key     string   `json:"key"`
		Type    string   `json:"type"`
		Columns []string `json:"columns"`
		Orders  []string `json:"orders"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		apperr.BadRequest(w, "key is required")
		return
	}
	idx, err := h.svc.CreateIndex(r.Context(), collID, body.Key, body.Type, body.Columns, body.Orders)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, idx)
}

func (h *Handler) listIndexes(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "tableId")
	idxs, err := h.svc.ListIndexes(r.Context(), collID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(idxs), "indexes": idxs})
}

func (h *Handler) deleteIndex(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "tableId")
	key := chi.URLParam(r, "key")
	if err := h.svc.DeleteIndex(r.Context(), collID, key); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createRelationship(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "tableId")
	var body struct {
		RelatedTableID string `json:"relatedTableId"`
		Type           string `json:"type"` // oneToOne, oneToMany, manyToOne, manyToMany
		TwoWay         bool   `json:"twoWay"`
		Key            string `json:"key"`
		TwoWayKey      string `json:"twoWayKey"`
		OnDelete       string `json:"onDelete"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RelatedTableID == "" || body.Key == "" {
		apperr.BadRequest(w, "relatedTableId and key are required")
		return
	}
	if body.Type == "" {
		body.Type = "oneToMany"
	}
	rel, err := h.svc.CreateRelationship(r.Context(), collID, body.RelatedTableID, body.Type, body.Key, body.TwoWayKey, body.OnDelete, body.TwoWay)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rel)
}

func (h *Handler) listRelationships(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "tableId")
	rels, err := h.svc.ListRelationships(r.Context(), collID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(rels), "relationships": rels})
}

func (h *Handler) deleteRelationship(w http.ResponseWriter, r *http.Request) {
	collID := chi.URLParam(r, "tableId")
	relID := chi.URLParam(r, "relationshipId")
	if err := h.svc.DeleteRelationship(r.Context(), collID, relID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	var body struct {
		RowID       string                 `json:"rowId"`
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
	doc, err := h.svc.CreateDocumentWithAuth(ctx, projectID, dbID, collID, body.RowID, body.Data, body.Permissions, userID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to create documents in this collection")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "rows": docs})
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
	case "geo_near":
		// geo_near("field", lat, lng, maxDistanceKm)
		if len(args) < 4 {
			return nil
		}
		return &Query{Attribute: args[0], Method: method, Values: []interface{}{args[1], args[2], args[3]}}
	case "geo_within":
		// geo_within("field", minLat, maxLat, minLng, maxLng)
		if len(args) < 5 {
			return nil
		}
		return &Query{Attribute: args[0], Method: method, Values: []interface{}{args[1], args[2], args[3], args[4]}}
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
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	docID := chi.URLParam(r, "rowId")
	doc, err := h.svc.GetDocumentWithAuth(ctx, docID, collID, dbID, projectID, userID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to read this document")
			return
		}
		apperr.NotFound(w, "document")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) updateDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	docID := chi.URLParam(r, "rowId")
	var body struct {
		Data        map[string]interface{} `json:"data"`
		Permissions []string               `json:"permissions"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	doc, err := h.svc.UpdateDocumentWithAuth(ctx, docID, collID, dbID, projectID, body.Data, body.Permissions, userID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to update this document")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	docID := chi.URLParam(r, "rowId")
	if err := h.svc.DeleteDocumentWithAuth(ctx, docID, collID, dbID, projectID, userID, nil); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to delete this document")
			return
		}
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- CSV import handlers ---

func (h *Handler) importCSV(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")

	// Parse multipart form — 32 MB max
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apperr.BadRequest(w, "invalid multipart form")
		return
	}

	// Get CSV file
	file, _, err := r.FormFile("file")
	if err != nil {
		apperr.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	csvData, err := io.ReadAll(file)
	if err != nil {
		apperr.BadRequest(w, "failed to read CSV file")
		return
	}

	// Parse optional column mapping from form field
	var columnMapping map[string]string
	if mappingStr := r.FormValue("columnMapping"); mappingStr != "" {
		if err := json.Unmarshal([]byte(mappingStr), &columnMapping); err != nil {
			apperr.BadRequest(w, "invalid columnMapping JSON")
			return
		}
	}

	result, err := h.svc.ImportCSV(r.Context(), projectID, dbID, collID, csvData, columnMapping)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "collection")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) previewCSV(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form — 32 MB max
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apperr.BadRequest(w, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		apperr.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	csvData, err := io.ReadAll(file)
	if err != nil {
		apperr.BadRequest(w, "failed to read CSV file")
		return
	}

	preview, err := h.svc.PreviewCSV(r.Context(), csvData)
	if err != nil {
		apperr.BadRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

// --- permissions handlers ---

func (h *Handler) setCollectionPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	collID := chi.URLParam(r, "tableId")
	var body struct {
		Permissions []Permission `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if body.Permissions == nil {
		body.Permissions = []Permission{}
	}
	if err := h.svc.SetPermissions(ctx, projectID, "collection", collID, body.Permissions); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":       len(body.Permissions),
		"permissions": body.Permissions,
	})
}

func (h *Handler) getCollectionPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	collID := chi.URLParam(r, "tableId")
	perms, err := h.svc.GetPermissions(ctx, projectID, "collection", collID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":       len(perms),
		"permissions": perms,
	})
}

// ── Upsert ──

func (h *Handler) upsertDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	var body struct {
		Data        map[string]interface{} `json:"data"`
		ID          string                 `json:"$id"`
		Permissions []string               `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Invalid JSON body")
		return
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	if body.ID != "" {
		existing, _ := h.svc.GetDocument(ctx, body.ID, collID, dbID, projectID)
		if existing != nil {
			doc, err := h.svc.UpdateDocumentWithAuth(ctx, body.ID, collID, dbID, projectID, body.Data, body.Permissions, userID, nil)
			if err != nil {
				apperr.Internal(w, err)
				return
			}
			writeJSON(w, http.StatusOK, doc)
			return
		}
	}
	doc, err := h.svc.CreateDocumentWithAuth(ctx, projectID, dbID, collID, body.ID, body.Data, body.Permissions, userID, nil)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// ── Bulk operations ──

func (h *Handler) bulkCreateDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	var body struct {
		Documents []map[string]interface{} `json:"documents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Invalid JSON body")
		return
	}
	var created []interface{}
	for _, data := range body.Documents {
		doc, err := h.svc.CreateDocumentWithAuth(ctx, projectID, dbID, collID, "", data, []string{}, userID, nil)
		if err == nil {
			created = append(created, doc)
		}
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"created": len(created), "documents": created})
}

func (h *Handler) bulkUpdateDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	var body struct {
		Documents []struct {
			ID   string                 `json:"$id"`
			Data map[string]interface{} `json:"data"`
		} `json:"documents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Invalid JSON body")
		return
	}
	updated := 0
	for _, item := range body.Documents {
		if _, err := h.svc.UpdateDocumentWithAuth(ctx, item.ID, collID, dbID, projectID, item.Data, []string{}, userID, nil); err == nil {
			updated++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"updated": updated})
}

func (h *Handler) bulkDeleteDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Invalid JSON body")
		return
	}
	deleted := 0
	for _, id := range body.IDs {
		if err := h.svc.DeleteDocumentWithAuth(ctx, id, collID, dbID, projectID, userID, nil); err == nil {
			deleted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": deleted})
}

// ── Atomic operations ──

func (h *Handler) atomicIncrement(w http.ResponseWriter, r *http.Request) {
	h.atomicNumericOp(w, r, 1)
}

func (h *Handler) atomicDecrement(w http.ResponseWriter, r *http.Request) {
	h.atomicNumericOp(w, r, -1)
}

func (h *Handler) atomicNumericOp(w http.ResponseWriter, r *http.Request, sign float64) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	rowID := chi.URLParam(r, "rowId")
	var body struct {
		Field string  `json:"field"`
		Delta float64 `json:"delta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Field == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "field and delta required")
		return
	}
	doc, err := h.svc.GetDocument(ctx, rowID, collID, dbID, projectID)
	if err != nil {
		apperr.NotFound(w, "document")
		return
	}
	data := doc.Data
	cur := 0.0
	if v, ok := data[body.Field]; ok {
		switch n := v.(type) {
		case float64:
			cur = n
		}
	}
	cur += body.Delta * sign
	updated, err := h.svc.UpdateDocumentWithAuth(ctx, rowID, collID, dbID, projectID,
		map[string]interface{}{body.Field: cur}, []string{}, userID, nil)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) atomicAppend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	collID := chi.URLParam(r, "tableId")
	rowID := chi.URLParam(r, "rowId")
	var body struct {
		Field string      `json:"field"`
		Value interface{} `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Field == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "field and value required")
		return
	}
	doc, err := h.svc.GetDocument(ctx, rowID, collID, dbID, projectID)
	if err != nil {
		apperr.NotFound(w, "document")
		return
	}
	var arr []interface{}
	if existing, ok := doc.Data[body.Field].([]interface{}); ok {
		arr = existing
	}
	arr = append(arr, body.Value)
	updated, err := h.svc.UpdateDocumentWithAuth(ctx, rowID, collID, dbID, projectID,
		map[string]interface{}{body.Field: arr}, []string{}, userID, nil)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ── Transactions ──

func (h *Handler) executeTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	var body struct {
		Operations []TransactionOp `json:"operations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if len(body.Operations) == 0 {
		apperr.BadRequest(w, "operations array is required and must not be empty")
		return
	}
	results, err := h.svc.ExecuteTransaction(ctx, projectID, dbID, body.Operations)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "is required") || strings.Contains(err.Error(), "unsupported action") {
			apperr.BadRequest(w, err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":   len(results),
		"results": results,
	})
}
