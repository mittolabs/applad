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

	// Tables
	r.Post("/{databaseId}/tables", h.createTable)
	r.Get("/{databaseId}/tables", h.listTables)
	r.Get("/{databaseId}/tables/{tableId}", h.getTable)
	r.Put("/{databaseId}/tables/{tableId}", h.updateTable)
	r.Delete("/{databaseId}/tables/{tableId}", h.deleteTable)

	// Columns
	r.Post("/{databaseId}/tables/{tableId}/columns/string", h.createColumn("string"))
	r.Post("/{databaseId}/tables/{tableId}/columns/integer", h.createColumn("integer"))
	r.Post("/{databaseId}/tables/{tableId}/columns/float", h.createColumn("double"))
	r.Post("/{databaseId}/tables/{tableId}/columns/boolean", h.createColumn("boolean"))
	r.Post("/{databaseId}/tables/{tableId}/columns/email", h.createColumn("email"))
	r.Post("/{databaseId}/tables/{tableId}/columns/url", h.createColumn("url"))
	r.Post("/{databaseId}/tables/{tableId}/columns/datetime", h.createColumn("datetime"))
	r.Post("/{databaseId}/tables/{tableId}/columns/enum", h.createColumn("enum"))
	// Editorial field types (content mode).
	r.Post("/{databaseId}/tables/{tableId}/columns/richtext", h.createColumn("richtext"))
	r.Post("/{databaseId}/tables/{tableId}/columns/media", h.createColumn("media"))
	r.Post("/{databaseId}/tables/{tableId}/columns/point", h.createColumn("point"))
	r.Get("/{databaseId}/tables/{tableId}/columns", h.listColumns)
	r.Delete("/{databaseId}/tables/{tableId}/columns/{key}", h.deleteColumn)
	r.Get("/{databaseId}/tables/{tableId}/columns/{key}/permissions", h.getColumnPermissions)
	r.Post("/{databaseId}/tables/{tableId}/columns/{key}/permissions", h.setColumnPermissions)

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
	r.Post("/{databaseId}/sql", h.executeSQL)

	// CSV Import
	r.Post("/{databaseId}/tables/{tableId}/import/csv", h.importCSV)
	r.Post("/{databaseId}/tables/{tableId}/import/csv/preview", h.previewCSV)

	// Rows
	r.Post("/{databaseId}/tables/{tableId}/rows", h.createRow)
	r.Get("/{databaseId}/tables/{tableId}/rows", h.listRows)
	r.Get("/{databaseId}/tables/{tableId}/rows/{rowId}", h.getRow)
	r.Put("/{databaseId}/tables/{tableId}/rows/{rowId}", h.upsertRow)
	r.Patch("/{databaseId}/tables/{tableId}/rows/{rowId}", h.updateRow)
	r.Delete("/{databaseId}/tables/{tableId}/rows/{rowId}", h.deleteRow)

	// Bulk / atomic row operations
	// Content mode: editorial behaviour on top of the same table and rows API.
	r.Post("/{databaseId}/tables/{tableId}/content", h.enableContentMode)
	r.Delete("/{databaseId}/tables/{tableId}/content", h.disableContentMode)
	r.Post("/{databaseId}/tables/{tableId}/rows/{rowId}/publish", h.setPublished(true))
	r.Post("/{databaseId}/tables/{tableId}/rows/{rowId}/unpublish", h.setPublished(false))
	r.Get("/{databaseId}/tables/{tableId}/rows/{rowId}/versions", h.listRowVersions)

	r.Post("/{databaseId}/tables/{tableId}/rows/bulk", h.bulkCreateRows)
	r.Patch("/{databaseId}/tables/{tableId}/rows/bulk", h.bulkUpdateRows)
	r.Delete("/{databaseId}/tables/{tableId}/rows/bulk", h.bulkDeleteRows)
	r.Post("/{databaseId}/tables/{tableId}/rows/{rowId}/increment", h.atomicIncrement)
	r.Post("/{databaseId}/tables/{tableId}/rows/{rowId}/decrement", h.atomicDecrement)
	r.Post("/{databaseId}/tables/{tableId}/rows/{rowId}/append", h.atomicAppend)

	// Permissions
	r.Post("/{databaseId}/tables/{tableId}/permissions", h.setTablePermissions)
	r.Get("/{databaseId}/tables/{tableId}/permissions", h.getTablePermissions)

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

func (h *Handler) createTable(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	var body struct {
		TableID     string   `json:"tableId"`
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		RowSecurity bool     `json:"rowSecurity"`
		// documentSecurity is the SDKs' name for the same thing; accept both so a
		// client that sends one is not silently ignored.
		DocumentSecurity bool `json:"documentSecurity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	rowSecurity := body.RowSecurity || body.DocumentSecurity
	table, err := h.svc.CreateTable(r.Context(), projectID, dbID, body.TableID, body.Name, body.Permissions, rowSecurity)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, table)
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	tables, total, err := h.svc.ListTables(r.Context(), dbID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if tables == nil {
		tables = []*model.Table{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "tables": tables})
}

func (h *Handler) getTable(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	table, err := h.svc.GetTable(r.Context(), tableID, dbID, projectID)
	if err != nil {
		apperr.NotFound(w, "table")
		return
	}
	writeJSON(w, http.StatusOK, table)
}

func (h *Handler) updateTable(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	var body struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		Enabled     *bool    `json:"enabled"`
		RowSecurity *bool    `json:"rowSecurity"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	table, err := h.svc.UpdateTable(r.Context(), tableID, dbID, projectID, body.Name, body.Permissions, body.Enabled, body.RowSecurity)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, table)
}

func (h *Handler) deleteTable(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	if err := h.svc.DeleteTable(r.Context(), tableID, dbID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createColumn(attrType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := middleware.ProjectFromContext(r.Context())
		tableID := chi.URLParam(r, "tableId")
		var body struct {
			Key        string                  `json:"key"`
			Required   bool                    `json:"required"`
			Array      bool                    `json:"array"`
			Default    interface{}             `json:"default"`
			Validation *model.ColumnValidation `json:"validation"`
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
		column, err := h.svc.CreateColumn(r.Context(), projectID, tableID, body.Key, attrType, body.Required, body.Array, body.Default, options, body.Validation)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				apperr.NotFound(w, "table")
				return
			}
			apperr.Internal(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, column)
	}
}

func (h *Handler) listColumns(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	columns, err := h.svc.ListColumns(r.Context(), tableID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(columns), "columns": columns})
}

func (h *Handler) deleteColumn(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	tableID := chi.URLParam(r, "tableId")
	key := chi.URLParam(r, "key")
	if err := h.svc.DeleteColumn(r.Context(), projectID, tableID, key); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "table")
			return
		}
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getColumnPermissions(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	key := chi.URLParam(r, "key")
	perms, err := h.svc.GetColumnPermissions(r.Context(), tableID, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "column")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"$permissions": perms})
}

func (h *Handler) setColumnPermissions(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	key := chi.URLParam(r, "key")
	var body struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	// Validate: only "read" and "write" are valid column permissions.
	for _, p := range body.Permissions {
		if p != "read" && p != "write" {
			apperr.BadRequest(w, "column permissions must be \"read\" or \"write\"")
			return
		}
	}
	if err := h.svc.SetColumnPermissions(r.Context(), tableID, key, body.Permissions); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "column")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"$permissions": body.Permissions})
}

func (h *Handler) createIndex(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
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
	idx, err := h.svc.CreateIndex(r.Context(), projectID, collID, body.Key, body.Type, body.Columns, body.Orders)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "table")
			return
		}
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
	projectID := middleware.ProjectFromContext(r.Context())
	collID := chi.URLParam(r, "tableId")
	key := chi.URLParam(r, "key")
	if err := h.svc.DeleteIndex(r.Context(), projectID, collID, key); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "table")
			return
		}
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createRelationship(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
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
	rel, err := h.svc.CreateRelationship(r.Context(), projectID, collID, body.RelatedTableID, body.Type, body.Key, body.TwoWayKey, body.OnDelete, body.TwoWay)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "table")
			return
		}
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
	projectID := middleware.ProjectFromContext(r.Context())
	collID := chi.URLParam(r, "tableId")
	relID := chi.URLParam(r, "relationshipId")
	if err := h.svc.DeleteRelationship(r.Context(), projectID, collID, relID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "table")
			return
		}
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createRow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
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
	row, err := h.svc.CreateRowWithAuth(ctx, projectID, dbID, tableID, body.RowID, body.Data, body.Permissions, userID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to create rows in this table")
			return
		}
		if verr, ok := err.(*ValidationErr); ok {
			apperr.ValidationError(w, []apperr.ValidationFieldError{{Field: verr.Field, Rule: verr.Rule, Message: verr.Message}})
			return
		}
		apperr.Internal(w, err)
		return
	}
	// No-op unless the table is in content mode.
	h.svc.RecordRowVersion(ctx, tableID, row.ID, userID, body.Data)
	writeJSON(w, http.StatusCreated, row)
}

// ── Content mode ─────────────────────────────────────────────────────────────

func (h *Handler) enableContentMode(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetContentMode(r.Context(), middleware.ProjectFromContext(r.Context()), chi.URLParam(r, "tableId"), true); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"contentEnabled": true})
}

func (h *Handler) disableContentMode(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetContentMode(r.Context(), middleware.ProjectFromContext(r.Context()), chi.URLParam(r, "tableId"), false); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"contentEnabled": false})
}

func (h *Handler) setPublished(published bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rowID := chi.URLParam(r, "rowId")
		if err := h.svc.SetRowPublished(r.Context(), middleware.ProjectFromContext(r.Context()), chi.URLParam(r, "tableId"), rowID, published); err != nil {
			apperr.NotFound(w, "row")
			return
		}
		status := "draft"
		if published {
			status = "published"
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"$id": rowID, "status": status})
	}
}

func (h *Handler) listRowVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := h.svc.ListRowVersions(r.Context(), chi.URLParam(r, "tableId"), chi.URLParam(r, "rowId"))
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(versions), "versions": versions})
}

func (h *Handler) listRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	pg := middleware.ParsePagination(r)

	params := ListParams{
		Limit:       pg.Limit,
		Offset:      pg.Offset,
		OrderAttr:   r.URL.Query().Get("orderAttr"),
		OrderType:   r.URL.Query().Get("orderType"),
		Select:      r.URL.Query().Get("select"),
		CursorAfter: r.URL.Query().Get("cursorAfter"),
	}

	// Parse queries: ?queries[]=equal("name","John")&greaterThan("age",18)
	for _, q := range r.URL.Query()["queries[]"] {
		parsed := parseQueryString(q)
		if parsed != nil {
			params.Queries = append(params.Queries, *parsed)
		}
	}

	// Editorial shorthands for content-enabled tables: ?status=published&locale=en
	if status := r.URL.Query().Get("status"); status != "" {
		params.Queries = append(params.Queries, Query{Field: "status", Method: "equal", Values: status})
	}
	if locale := r.URL.Query().Get("locale"); locale != "" {
		params.Queries = append(params.Queries, Query{Field: "locale", Method: "equal", Values: locale})
	}

	rows, total, err := h.svc.ListRowsWithAuth(ctx, projectID, dbID, tableID, userID, nil, params)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to read rows in this table")
			return
		}
		apperr.Internal(w, err)
		return
	}
	if rows == nil {
		rows = []*model.Row{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "rows": rows})
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
		return &Query{Field: args[0], Method: method, Values: args[1]}
	case "lessThan", "greaterThan", "lessThanEqual", "greaterThanEqual":
		if len(args) < 2 {
			return nil
		}
		return &Query{Field: args[0], Method: method, Values: args[1]}
	case "between":
		if len(args) < 3 {
			return nil
		}
		return &Query{Field: args[0], Method: method, Values: []interface{}{args[1], args[2]}}
	case "geo_near":
		// geo_near("field", lat, lng, maxDistanceKm)
		if len(args) < 4 {
			return nil
		}
		return &Query{Field: args[0], Method: method, Values: []interface{}{args[1], args[2], args[3]}}
	case "geo_within":
		// geo_within("field", minLat, maxLat, minLng, maxLng)
		if len(args) < 5 {
			return nil
		}
		return &Query{Field: args[0], Method: method, Values: []interface{}{args[1], args[2], args[3], args[4]}}
	case "isNull", "isNotNull":
		if len(args) < 1 {
			return nil
		}
		return &Query{Field: args[0], Method: method}
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

func (h *Handler) getRow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	rowID := chi.URLParam(r, "rowId")
	row, err := h.svc.GetRowWithAuth(ctx, rowID, tableID, dbID, projectID, userID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to read this row")
			return
		}
		apperr.NotFound(w, "row")
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) updateRow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	rowID := chi.URLParam(r, "rowId")
	var body struct {
		Data        map[string]interface{} `json:"data"`
		Permissions []string               `json:"permissions"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if body.Permissions == nil {
		body.Permissions = []string{}
	}
	row, err := h.svc.UpdateRowWithAuth(ctx, rowID, tableID, dbID, projectID, body.Data, body.Permissions, userID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to update this row")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "row")
			return
		}
		if verr, ok := err.(*ValidationErr); ok {
			apperr.ValidationError(w, []apperr.ValidationFieldError{{Field: verr.Field, Rule: verr.Rule, Message: verr.Message}})
			return
		}
		apperr.Internal(w, err)
		return
	}
	// No-op unless the table is in content mode.
	h.svc.RecordRowVersion(ctx, tableID, rowID, userID, body.Data)
	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) deleteRow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	rowID := chi.URLParam(r, "rowId")
	if err := h.svc.DeleteRowWithAuth(ctx, rowID, tableID, dbID, projectID, userID, nil); err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to delete this row")
			return
		}
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "row")
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
	tableID := chi.URLParam(r, "tableId")

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

	result, err := h.svc.ImportCSV(r.Context(), projectID, dbID, tableID, csvData, columnMapping)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "table")
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

func (h *Handler) setTablePermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	tableID := chi.URLParam(r, "tableId")
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
	if err := h.svc.SetPermissions(ctx, projectID, "table", tableID, body.Permissions); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":       len(body.Permissions),
		"permissions": body.Permissions,
	})
}

func (h *Handler) getTablePermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	tableID := chi.URLParam(r, "tableId")
	perms, err := h.svc.GetPermissions(ctx, projectID, "table", tableID)
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

func (h *Handler) upsertRow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
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
		existing, _ := h.svc.GetRow(ctx, body.ID, tableID, dbID, projectID)
		if existing != nil {
			row, err := h.svc.UpdateRowWithAuth(ctx, body.ID, tableID, dbID, projectID, body.Data, body.Permissions, userID, nil)
			if err != nil {
				apperr.Internal(w, err)
				return
			}
			writeJSON(w, http.StatusOK, row)
			return
		}
	}
	row, err := h.svc.CreateRowWithAuth(ctx, projectID, dbID, tableID, body.ID, body.Data, body.Permissions, userID, nil)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

// ── Bulk operations ──

func (h *Handler) bulkCreateRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	var body struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Invalid JSON body")
		return
	}
	var created []interface{}
	for _, data := range body.Rows {
		row, err := h.svc.CreateRowWithAuth(ctx, projectID, dbID, tableID, "", data, []string{}, userID, nil)
		if err == nil {
			created = append(created, row)
		}
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"created": len(created), "rows": created})
}

func (h *Handler) bulkUpdateRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	var body struct {
		Rows []struct {
			ID   string                 `json:"$id"`
			Data map[string]interface{} `json:"data"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Invalid JSON body")
		return
	}
	updated := 0
	for _, item := range body.Rows {
		if _, err := h.svc.UpdateRowWithAuth(ctx, item.ID, tableID, dbID, projectID, item.Data, []string{}, userID, nil); err == nil {
			updated++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"updated": updated})
}

func (h *Handler) bulkDeleteRows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Invalid JSON body")
		return
	}
	deleted := 0
	for _, id := range body.IDs {
		if err := h.svc.DeleteRowWithAuth(ctx, id, tableID, dbID, projectID, userID, nil); err == nil {
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
	tableID := chi.URLParam(r, "tableId")
	rowID := chi.URLParam(r, "rowId")
	var body struct {
		Field string  `json:"field"`
		Delta float64 `json:"delta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Field == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "field and delta required")
		return
	}
	updated, err := h.svc.AtomicNumericOp(ctx, projectID, dbID, tableID, rowID, body.Field, body.Delta*sign, userID, nil)
	if err != nil {
		writeAtomicError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// writeAtomicError maps a service error from an atomic op to the right status:
// a permission failure is 403, a missing row is 404, anything else is 500.
func writeAtomicError(w http.ResponseWriter, err error) {
	switch {
	case strings.Contains(err.Error(), "permission denied"):
		apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to update this row")
	case strings.Contains(err.Error(), "not found"):
		apperr.NotFound(w, "row")
	default:
		apperr.Internal(w, err)
	}
}

func (h *Handler) atomicAppend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	dbID := chi.URLParam(r, "databaseId")
	tableID := chi.URLParam(r, "tableId")
	rowID := chi.URLParam(r, "rowId")
	var body struct {
		Field string      `json:"field"`
		Value interface{} `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Field == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "field and value required")
		return
	}
	updated, err := h.svc.AtomicArrayAppend(ctx, projectID, dbID, tableID, rowID, body.Field, body.Value, userID, nil)
	if err != nil {
		writeAtomicError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ── Transactions ──

func (h *Handler) executeTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
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
	// Thread the caller's identity so each op is permission-checked for a user
	// session; a server API key (userID == "") keeps full access. Roles are
	// resolved server-side in the service, never taken from the request.
	results, err := h.svc.ExecuteTransaction(ctx, projectID, dbID, userID, nil, body.Operations)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You do not have permission to perform one of the operations in this transaction")
			return
		}
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

func (h *Handler) executeSQL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// The raw /sql endpoint runs under the schema's _authenticated role, whose
	// GRANTs on a non-document-security table are unconditional — it applies no
	// app-level table permission check. That makes it an operator/admin escape
	// hatch, not a client API: a plain end-user session must not reach it, or it
	// could SELECT/UPDATE/DELETE every row of any non-RLS table regardless of the
	// table's grants. Restrict it to a server API key or a console admin.
	if !middleware.IsAPIKey(ctx) && !middleware.IsConsoleAdmin(ctx) {
		apperr.Write(w, http.StatusForbidden, "permission_denied", "The SQL editor requires a server API key or console admin.")
		return
	}
	projectID := middleware.ProjectFromContext(ctx)
	userID := middleware.UserFromContext(ctx)
	databaseID := chi.URLParam(r, "databaseId")
	var body struct {
		Statement    string `json:"statement"`
		WriteAllowed bool   `json:"writeAllowed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Statement) == "" {
		apperr.BadRequest(w, "statement is required")
		return
	}
	// Roles come from the authenticated session only. A roles field in the
	// body let callers mint any RLS role ("admin") and satisfy any policy.
	result, err := h.svc.ExecuteSQL(ctx, projectID, databaseID, userID, nil, body.Statement, body.WriteAllowed)
	if err != nil {
		apperr.Write(w, http.StatusBadRequest, "sql_execution_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
