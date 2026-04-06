// Package databases implements Applad's TablesDB service:
// databases, collections, attributes, indexes, documents, queries, and permissions.
package databases

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/uid"
)

// Service handles database/collection/document business logic.
type Service struct {
	db     *db.DB
	events realtime.EventPublisher
}

// NewService creates a new databases Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// SetEventPublisher sets the realtime event publisher for auto-publishing.
func (s *Service) SetEventPublisher(pub realtime.EventPublisher) {
	s.events = pub
}

// --- databases ---

func (s *Service) CreateDatabase(ctx context.Context, projectID, dbID, name string) (*model.Database, error) {
	id := uid.New(dbID)
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO _databases (id, project_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, projectID, name, now, now)
	if err != nil {
		return nil, fmt.Errorf("databases: create: %w", err)
	}
	return &model.Database{ID: id, Name: name, Enabled: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Service) GetDatabase(ctx context.Context, dbID, projectID string) (*model.Database, error) {
	var d model.Database
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, created_at, updated_at FROM _databases WHERE id = ? AND project_id = ?",
		dbID, projectID).Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("database not found")
	}
	if err != nil {
		return nil, err
	}
	d.Enabled = true
	return &d, nil
}

func (s *Service) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, created_at, updated_at FROM _databases WHERE project_id = ? ORDER BY created_at DESC", projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var dbs []*model.Database
	for rows.Next() {
		var d model.Database
		if err := rows.Scan(&d.ID, &d.Name, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		d.Enabled = true
		dbs = append(dbs, &d)
	}
	return dbs, len(dbs), nil
}

func (s *Service) UpdateDatabase(ctx context.Context, dbID, projectID, name string) (*model.Database, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE _databases SET name = ? WHERE id = ? AND project_id = ?", name, dbID, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetDatabase(ctx, dbID, projectID)
}

func (s *Service) DeleteDatabase(ctx context.Context, dbID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM _databases WHERE id = ? AND project_id = ?", dbID, projectID)
	return err
}

// --- collections ---

func (s *Service) CreateCollection(ctx context.Context, projectID, dbID, collID, name string, permissions []string, docSecurity bool) (*model.Collection, error) {
	id := uid.New(collID)
	now := time.Now().UTC()
	permsJSON, _ := json.Marshal(permissions)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO collections (id, database_id, project_id, name, permissions, document_security, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, dbID, projectID, name, permsJSON, docSecurity, now, now)
	if err != nil {
		return nil, fmt.Errorf("collections: create: %w", err)
	}
	return &model.Collection{
		ID: id, DatabaseID: dbID, Name: name, Enabled: true,
		RowSecurity: docSecurity, Permissions: permissions,
		Columns: []model.Column{}, Indexes: []model.Index{},
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Service) GetCollection(ctx context.Context, collID, dbID, projectID string) (*model.Collection, error) {
	var c model.Collection
	var permsJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, database_id, name, document_security, permissions, enabled, created_at, updated_at FROM collections WHERE id = ? AND database_id = ? AND project_id = ?",
		collID, dbID, projectID).Scan(&c.ID, &c.DatabaseID, &c.Name, &c.RowSecurity, &permsJSON, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(permsJSON, &c.Permissions) //nolint:errcheck
	if c.Permissions == nil {
		c.Permissions = []string{}
	}
	attrs, _ := s.ListAttributes(ctx, collID)
	c.Columns = attrs
	idxs, _ := s.ListIndexes(ctx, collID)
	c.Indexes = idxs
	return &c, nil
}

func (s *Service) ListCollections(ctx context.Context, dbID, projectID string) ([]*model.Collection, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, database_id, name, document_security, permissions, enabled, created_at, updated_at FROM collections WHERE database_id = ? AND project_id = ? ORDER BY created_at DESC",
		dbID, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var colls []*model.Collection
	for rows.Next() {
		var c model.Collection
		var permsJSON []byte
		if err := rows.Scan(&c.ID, &c.DatabaseID, &c.Name, &c.RowSecurity, &permsJSON, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(permsJSON, &c.Permissions) //nolint:errcheck
		if c.Permissions == nil {
			c.Permissions = []string{}
		}
		c.Columns = []model.Column{}
		c.Indexes = []model.Index{}
		colls = append(colls, &c)
	}
	return colls, len(colls), nil
}

func (s *Service) UpdateCollection(ctx context.Context, collID, dbID, projectID, name string, permissions []string, enabled bool) (*model.Collection, error) {
	permsJSON, _ := json.Marshal(permissions)
	_, err := s.db.ExecContext(ctx,
		"UPDATE collections SET name = ?, permissions = ?, enabled = ? WHERE id = ? AND database_id = ? AND project_id = ?",
		name, permsJSON, enabled, collID, dbID, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetCollection(ctx, collID, dbID, projectID)
}

func (s *Service) DeleteCollection(ctx context.Context, collID, dbID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM collections WHERE id = ? AND database_id = ? AND project_id = ?", collID, dbID, projectID)
	return err
}

// --- attributes ---

func (s *Service) CreateAttribute(ctx context.Context, collID, key, attrType string, required, array bool, defaultVal interface{}, options map[string]interface{}) (*model.Attribute, error) {
	id := uid.New("unique()")
	optJSON, _ := json.Marshal(options)
	var defStr sql.NullString
	if defaultVal != nil {
		b, _ := json.Marshal(defaultVal)
		defStr = sql.NullString{String: string(b), Valid: true}
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO attributes (id, collection_id, `key`, type, required, array, default_value, options, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'available')",
		id, collID, key, attrType, required, array, defStr, optJSON)
	if err != nil {
		return nil, fmt.Errorf("attributes: create: %w", err)
	}
	attr := &model.Attribute{
		Key: key, Type: attrType, Status: "available",
		Required: required, Array: array, Default: defaultVal,
	}
	return attr, nil
}

func (s *Service) ListAttributes(ctx context.Context, collID string) ([]model.Attribute, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT `key`, type, status, required, array, default_value, options FROM attributes WHERE collection_id = ? ORDER BY created_at ASC",
		collID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var attrs []model.Attribute
	for rows.Next() {
		var a model.Attribute
		var defVal sql.NullString
		var optJSON sql.NullString
		if err := rows.Scan(&a.Key, &a.Type, &a.Status, &a.Required, &a.Array, &defVal, &optJSON); err != nil {
			return nil, err
		}
		if defVal.Valid {
			var v interface{}
			json.Unmarshal([]byte(defVal.String), &v) //nolint:errcheck
			a.Default = v
		}
		if optJSON.Valid {
			json.Unmarshal([]byte(optJSON.String), &a.Options) //nolint:errcheck
		}
		attrs = append(attrs, a)
	}
	if attrs == nil {
		attrs = []model.Attribute{}
	}
	return attrs, nil
}

func (s *Service) DeleteAttribute(ctx context.Context, collID, key string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM attributes WHERE collection_id = ? AND `key` = ?", collID, key)
	return err
}

// --- indexes ---

func (s *Service) CreateIndex(ctx context.Context, collID, key, idxType string, attributes, orders []string) (*model.Index, error) {
	id := uid.New("unique()")
	attrsJSON, _ := json.Marshal(attributes)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO _indexes (id, collection_id, `key`, type, attributes, status) VALUES (?, ?, ?, ?, ?, 'available')",
		id, collID, key, idxType, attrsJSON)
	if err != nil {
		return nil, fmt.Errorf("indexes: create: %w", err)
	}
	if orders == nil {
		orders = []string{}
	}
	return &model.Index{Key: key, Type: idxType, Status: "available", Columns: attributes, Orders: orders}, nil
}

func (s *Service) ListIndexes(ctx context.Context, collID string) ([]model.Index, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT `key`, type, status, attributes FROM _indexes WHERE collection_id = ? ORDER BY created_at ASC",
		collID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var idxs []model.Index
	for rows.Next() {
		var idx model.Index
		var attrsJSON []byte
		if err := rows.Scan(&idx.Key, &idx.Type, &idx.Status, &attrsJSON); err != nil {
			return nil, err
		}
		json.Unmarshal(attrsJSON, &idx.Columns) //nolint:errcheck
		if idx.Columns == nil {
			idx.Columns = []string{}
		}
		idx.Orders = []string{}
		idxs = append(idxs, idx)
	}
	if idxs == nil {
		idxs = []model.Index{}
	}
	return idxs, nil
}

func (s *Service) DeleteIndex(ctx context.Context, collID, key string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM _indexes WHERE collection_id = ? AND `key` = ?", collID, key)
	return err
}

// --- relationships ---

// Relationship represents a link between two tables.
type Relationship struct {
	ID            string `json:"$id"`
	TableID       string `json:"tableId"`
	RelatedTable  string `json:"relatedTableId"`
	Type          string `json:"type"` // oneToOne, oneToMany, manyToOne, manyToMany
	TwoWay        bool   `json:"twoWay"`
	Key           string `json:"key"`
	TwoWayKey     string `json:"twoWayKey,omitempty"`
	OnDelete      string `json:"onDelete"` // setNull, cascade, restrict
}

func (s *Service) CreateRelationship(ctx context.Context, collID, relatedCollID, relType, key, twoWayKey, onDelete string, twoWay bool) (*Relationship, error) {
	id := uid.New("unique()")
	if onDelete == "" {
		onDelete = "setNull"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO collection_relationships (id, collection_id, related_collection, relationship_type, two_way, ` + "`key`" + `, two_way_key, on_delete)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, collID, relatedCollID, relType, twoWay, key, twoWayKey, onDelete)
	if err != nil {
		return nil, fmt.Errorf("relationships: create: %w", err)
	}

	return &Relationship{
		ID: id, TableID: collID, RelatedTable: relatedCollID,
		Type: relType, TwoWay: twoWay, Key: key, TwoWayKey: twoWayKey, OnDelete: onDelete,
	}, nil
}

func (s *Service) ListRelationships(ctx context.Context, collID string) ([]Relationship, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, collection_id, related_collection, relationship_type, two_way, `key`, two_way_key, on_delete FROM collection_relationships WHERE collection_id = ?",
		collID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relationship
	for rows.Next() {
		var r Relationship
		var twoWayKey sql.NullString
		if err := rows.Scan(&r.ID, &r.TableID, &r.RelatedTable, &r.Type, &r.TwoWay, &r.Key, &twoWayKey, &r.OnDelete); err != nil {
			return nil, err
		}
		r.TwoWayKey = twoWayKey.String
		rels = append(rels, r)
	}
	if rels == nil {
		rels = []Relationship{}
	}
	return rels, nil
}

func (s *Service) DeleteRelationship(ctx context.Context, collID, relID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM collection_relationships WHERE id = ? AND collection_id = ?", relID, collID)
	return err
}

// --- permissions ---

// Permission represents a granular RBAC permission entry.
type Permission struct {
	Role   string `json:"role"`
	Action string `json:"action"` // read, create, update, delete
}

// validActions defines the set of allowed permission actions.
var validActions = map[string]bool{
	"read": true, "create": true, "update": true, "delete": true,
}

// checkPermission checks whether any of the given roles have the specified action
// on the resource. Returns true if access is allowed.
func (s *Service) checkPermission(ctx context.Context, projectID, resourceType, resourceID string, roles []string, action string) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}
	placeholders := make([]string, len(roles))
	args := make([]interface{}, 0, len(roles)+4)
	args = append(args, projectID, resourceType, resourceID, action)
	for i, r := range roles {
		placeholders[i] = "?"
		args = append(args, r)
	}
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM permissions WHERE project_id = ? AND resource_type = ? AND resource_id = ? AND action = ? AND role IN (%s)",
		strings.Join(placeholders, ","))
	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// checkDocumentPermission checks document-level permissions stored in the document's
// permissions JSON field. The permissions field stores role strings like "read(\"user:123\")".
func checkDocumentPermission(docPermissions []string, roles []string, action string) bool {
	for _, perm := range docPermissions {
		// Parse permission format: "action(\"role\")" e.g. "read(\"user:abc\")"
		idx := strings.Index(perm, "(")
		if idx < 0 {
			// Simple format: just the role string means all actions
			for _, r := range roles {
				if perm == r {
					return true
				}
			}
			continue
		}
		permAction := perm[:idx]
		if permAction != action {
			continue
		}
		// Extract the role from inside parentheses
		inner := strings.TrimSuffix(perm[idx+1:], ")")
		inner = strings.Trim(inner, "\"")
		for _, r := range roles {
			if inner == r {
				return true
			}
		}
	}
	return false
}

// buildRoles builds the set of roles for a user. This includes the special roles
// "any", "users" (if authenticated), "guests" (if not), and the user-specific role.
func buildRoles(userID string, extraRoles []string) []string {
	roles := []string{"any"}
	if userID == "" {
		roles = append(roles, "guests")
	} else {
		roles = append(roles, "users", "user:"+userID)
	}
	for _, r := range extraRoles {
		roles = append(roles, r)
	}
	return roles
}

// SetPermissions sets the permissions for a resource, replacing any existing ones.
func (s *Service) SetPermissions(ctx context.Context, projectID, resourceType, resourceID string, permissions []Permission) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("permissions: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Delete existing permissions for this resource
	_, err = tx.ExecContext(ctx,
		"DELETE FROM permissions WHERE project_id = ? AND resource_type = ? AND resource_id = ?",
		projectID, resourceType, resourceID)
	if err != nil {
		return fmt.Errorf("permissions: delete existing: %w", err)
	}

	// Insert new permissions
	for _, p := range permissions {
		if !validActions[p.Action] {
			return fmt.Errorf("permissions: invalid action %q", p.Action)
		}
		id := uid.New("unique()")
		_, err = tx.ExecContext(ctx,
			"INSERT INTO permissions (id, project_id, resource_type, resource_id, role, action) VALUES (?, ?, ?, ?, ?, ?)",
			id, projectID, resourceType, resourceID, p.Role, p.Action)
		if err != nil {
			return fmt.Errorf("permissions: insert: %w", err)
		}
	}

	return tx.Commit()
}

// GetPermissions returns all permissions for a resource.
func (s *Service) GetPermissions(ctx context.Context, projectID, resourceType, resourceID string) ([]Permission, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT role, action FROM permissions WHERE project_id = ? AND resource_type = ? AND resource_id = ? ORDER BY created_at ASC",
		projectID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Role, &p.Action); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if perms == nil {
		perms = []Permission{}
	}
	return perms, nil
}

// enforcePermission checks collection-level permissions first, then falls back to
// document-level permissions if the collection has document_security enabled.
// For create operations, only collection-level permissions are checked (no document yet).
func (s *Service) enforcePermission(ctx context.Context, projectID, collID string, userID string, roles []string, action string, docPermissions []string) error {
	allRoles := buildRoles(userID, roles)

	// Check collection-level permissions
	allowed, err := s.checkPermission(ctx, projectID, "collection", collID, allRoles, action)
	if err != nil {
		return fmt.Errorf("permissions: check collection: %w", err)
	}
	if allowed {
		return nil
	}

	// If no collection-level permission, check if document_security is enabled
	var docSecurity bool
	err = s.db.QueryRowContext(ctx,
		"SELECT document_security FROM collections WHERE id = ? AND project_id = ?",
		collID, projectID).Scan(&docSecurity)
	if err != nil {
		// If collection not found, deny access
		return fmt.Errorf("permission denied")
	}

	if docSecurity && len(docPermissions) > 0 {
		if checkDocumentPermission(docPermissions, allRoles, action) {
			return nil
		}
	}

	return fmt.Errorf("permission denied")
}

// --- documents ---

func (s *Service) CreateDocument(ctx context.Context, projectID, dbID, collID, docID string, data map[string]interface{}, permissions []string) (*model.Document, error) {
	return s.CreateDocumentWithAuth(ctx, projectID, dbID, collID, docID, data, permissions, "", nil)
}

// validateDocData checks attribute type constraints (email, url, ip, enum).
func (s *Service) validateDocData(ctx context.Context, collID string, data map[string]interface{}) error {
	attrs, err := s.ListAttributes(ctx, collID)
	if err != nil {
		return nil
	}
	for _, attr := range attrs {
		val, exists := data[attr.Key]
		if !exists {
			if attr.Required {
				return fmt.Errorf("attribute %s is required", attr.Key)
			}
			continue
		}
		strVal := fmt.Sprintf("%v", val)
		if strVal == "" || strVal == "<nil>" {
			continue
		}
		switch attr.Type {
		case "email":
			if !strings.Contains(strVal, "@") || !strings.Contains(strVal, ".") {
				return fmt.Errorf("attribute %s: invalid email", attr.Key)
			}
		case "url":
			if !strings.HasPrefix(strVal, "http://") && !strings.HasPrefix(strVal, "https://") {
				return fmt.Errorf("attribute %s: invalid URL", attr.Key)
			}
		case "ip":
			if net.ParseIP(strVal) == nil {
				return fmt.Errorf("attribute %s: invalid IP address", attr.Key)
			}
		case "point":
			// Point attributes must be objects with "lat" and "lng" numeric fields
			if m, ok := val.(map[string]interface{}); ok {
				lat, latOk := m["lat"]
				lng, lngOk := m["lng"]
				if !latOk || !lngOk {
					return fmt.Errorf("attribute %s: point must have lat and lng fields", attr.Key)
				}
				if _, ok := lat.(float64); !ok {
					return fmt.Errorf("attribute %s: lat must be a number", attr.Key)
				}
				if _, ok := lng.(float64); !ok {
					return fmt.Errorf("attribute %s: lng must be a number", attr.Key)
				}
			} else {
				return fmt.Errorf("attribute %s: point must be an object with lat and lng", attr.Key)
			}
		case "enum":
			if attr.Options != nil {
				if elements, ok := attr.Options["elements"].([]interface{}); ok {
					found := false
					for _, e := range elements {
						if fmt.Sprintf("%v", e) == strVal {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("attribute %s: must be one of %v", attr.Key, elements)
					}
				}
			}
		}
	}
	return nil
}

// CreateDocumentWithAuth creates a document with permission checking.
func (s *Service) CreateDocumentWithAuth(ctx context.Context, projectID, dbID, collID, docID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Document, error) {
	if userID != "" {
		if err := s.enforcePermission(ctx, projectID, collID, userID, roles, "create", nil); err != nil {
			return nil, err
		}
	}
	if err := s.validateDocData(ctx, collID, data); err != nil {
		return nil, err
	}
	id := uid.New(docID)
	now := time.Now().UTC()
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("documents: marshal data: %w", err)
	}
	permsJSON, _ := json.Marshal(permissions)
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO documents (id, collection_id, database_id, project_id, data, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, collID, dbID, projectID, dataJSON, permsJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("documents: create: %w", err)
	}
	doc := &model.Document{
		ID: id, TableID: collID, DatabaseID: dbID,
		Permissions: permissions, Data: data,
		CreatedAt: now, UpdatedAt: now,
	}
	realtime.PublishResourceEvent(s.events, "databases", "documents", "create", projectID, id, doc)
	return doc, nil
}

func (s *Service) GetDocument(ctx context.Context, docID, collID, dbID, projectID string) (*model.Document, error) {
	return s.GetDocumentWithAuth(ctx, docID, collID, dbID, projectID, "", nil)
}

// GetDocumentWithAuth retrieves a document with permission checking.
func (s *Service) GetDocumentWithAuth(ctx context.Context, docID, collID, dbID, projectID, userID string, roles []string) (*model.Document, error) {
	var doc model.Document
	var dataJSON, permsJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, collection_id, database_id, data, permissions, created_at, updated_at FROM documents WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
		docID, collID, dbID, projectID).Scan(&doc.ID, &doc.TableID, &doc.DatabaseID, &dataJSON, &permsJSON, &doc.CreatedAt, &doc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(dataJSON, &doc.Data)        //nolint:errcheck
	json.Unmarshal(permsJSON, &doc.Permissions) //nolint:errcheck
	if doc.Permissions == nil {
		doc.Permissions = []string{}
	}

	// Check read permission
	if userID != "" {
		if err := s.enforcePermission(ctx, projectID, collID, userID, roles, "read", doc.Permissions); err != nil {
			return nil, err
		}
	}

	return &doc, nil
}

// Query represents a single filter condition on document data.
type Query struct {
	Attribute string      // JSON field path (e.g., "name", "age")
	Method    string      // equal, notEqual, lessThan, greaterThan, lessThanEqual, greaterThanEqual, contains, search, startsWith, endsWith, isNull, isNotNull, between
	Values    interface{} // comparison value(s)
}

// ListParams holds all parameters for listing documents.
type ListParams struct {
	Limit      int
	Offset     int
	Queries    []Query
	OrderAttr  string // field to order by
	OrderType  string // ASC or DESC
	CursorAfter string // document ID for cursor-based pagination
}

func (s *Service) ListDocuments(ctx context.Context, projectID, dbID, collID string, limit, offset int) ([]*model.Document, int, error) {
	return s.ListDocumentsWithQuery(ctx, projectID, dbID, collID, ListParams{Limit: limit, Offset: offset})
}

func (s *Service) ListDocumentsWithQuery(ctx context.Context, projectID, dbID, collID string, params ListParams) ([]*model.Document, int, error) {
	if params.Limit <= 0 {
		params.Limit = 25
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	baseWhere := "collection_id = ? AND database_id = ? AND project_id = ?"
	args := []interface{}{collID, dbID, projectID}

	// Build query conditions from filters
	var conditions []string
	var geoOrderBy string // set by geo_near to override default ordering
	for _, q := range params.Queries {
		jsonPath := fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s'))", q.Attribute)
		switch q.Method {
		case "equal":
			conditions = append(conditions, fmt.Sprintf("%s = ?", jsonPath))
			args = append(args, fmt.Sprintf("%v", q.Values))
		case "notEqual":
			conditions = append(conditions, fmt.Sprintf("%s != ?", jsonPath))
			args = append(args, fmt.Sprintf("%v", q.Values))
		case "lessThan":
			conditions = append(conditions, fmt.Sprintf("CAST(%s AS DOUBLE) < ?", jsonPath))
			args = append(args, q.Values)
		case "greaterThan":
			conditions = append(conditions, fmt.Sprintf("CAST(%s AS DOUBLE) > ?", jsonPath))
			args = append(args, q.Values)
		case "lessThanEqual":
			conditions = append(conditions, fmt.Sprintf("CAST(%s AS DOUBLE) <= ?", jsonPath))
			args = append(args, q.Values)
		case "greaterThanEqual":
			conditions = append(conditions, fmt.Sprintf("CAST(%s AS DOUBLE) >= ?", jsonPath))
			args = append(args, q.Values)
		case "contains":
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", jsonPath))
			args = append(args, fmt.Sprintf("%%%v%%", q.Values))
		case "startsWith":
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", jsonPath))
			args = append(args, fmt.Sprintf("%v%%", q.Values))
		case "endsWith":
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", jsonPath))
			args = append(args, fmt.Sprintf("%%%v", q.Values))
		case "search":
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", jsonPath))
			args = append(args, fmt.Sprintf("%%%v%%", q.Values))
		case "isNull":
			conditions = append(conditions, fmt.Sprintf("(JSON_EXTRACT(data, '$.%s') IS NULL OR JSON_TYPE(JSON_EXTRACT(data, '$.%s')) = 'NULL')", q.Attribute, q.Attribute))
		case "isNotNull":
			conditions = append(conditions, fmt.Sprintf("(JSON_EXTRACT(data, '$.%s') IS NOT NULL AND JSON_TYPE(JSON_EXTRACT(data, '$.%s')) != 'NULL')", q.Attribute, q.Attribute))
		case "between":
			if vals, ok := q.Values.([]interface{}); ok && len(vals) == 2 {
				conditions = append(conditions, fmt.Sprintf("CAST(%s AS DOUBLE) BETWEEN ? AND ?", jsonPath))
				args = append(args, vals[0], vals[1])
			}
		case "geo_near":
			// geo_near(field, lat, lng, maxDistanceKm)
			// Uses Haversine formula to filter and sort by distance
			if vals, ok := q.Values.([]interface{}); ok && len(vals) >= 3 {
				latPath := fmt.Sprintf("CAST(JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s.lat')) AS DOUBLE)", q.Attribute)
				lngPath := fmt.Sprintf("CAST(JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s.lng')) AS DOUBLE)", q.Attribute)
				haversine := fmt.Sprintf(
					"(6371 * ACOS(COS(RADIANS(?)) * COS(RADIANS(%s)) * COS(RADIANS(%s) - RADIANS(?)) + SIN(RADIANS(?)) * SIN(RADIANS(%s))))",
					latPath, lngPath, latPath)
				conditions = append(conditions, fmt.Sprintf("%s <= ?", haversine))
				args = append(args, vals[0], vals[1], vals[0], vals[2])
				// Override ordering to sort by distance
				params.OrderAttr = ""
				geoOrderBy = fmt.Sprintf("%s ASC", haversine)
				// Re-add the haversine args for the ORDER BY clause (they're used in the SELECT)
				// Note: MariaDB reuses the WHERE args, the ORDER BY references the same expression
			}
		case "geo_within":
			// geo_within(field, minLat, maxLat, minLng, maxLng)
			// Checks if a point is within a bounding box
			if vals, ok := q.Values.([]interface{}); ok && len(vals) >= 4 {
				latPath := fmt.Sprintf("CAST(JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s.lat')) AS DOUBLE)", q.Attribute)
				lngPath := fmt.Sprintf("CAST(JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s.lng')) AS DOUBLE)", q.Attribute)
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN ? AND ?", latPath))
				args = append(args, vals[0], vals[1])
				conditions = append(conditions, fmt.Sprintf("%s BETWEEN ? AND ?", lngPath))
				args = append(args, vals[2], vals[3])
			}
		}
	}

	// Cursor-based pagination
	if params.CursorAfter != "" {
		conditions = append(conditions, "created_at < (SELECT created_at FROM documents WHERE id = ?)")
		args = append(args, params.CursorAfter)
	}

	where := baseWhere
	if len(conditions) > 0 {
		where += " AND " + strings.Join(conditions, " AND ")
	}

	// Order by
	orderBy := "created_at DESC"
	if geoOrderBy != "" {
		orderBy = geoOrderBy
	} else if params.OrderAttr != "" {
		dir := "ASC"
		if strings.EqualFold(params.OrderType, "DESC") {
			dir = "DESC"
		}
		if params.OrderAttr == "$createdAt" || params.OrderAttr == "created_at" {
			orderBy = "created_at " + dir
		} else if params.OrderAttr == "$updatedAt" || params.OrderAttr == "updated_at" {
			orderBy = "updated_at " + dir
		} else {
			orderBy = fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(data, '$.%s')) %s", params.OrderAttr, dir)
		}
	}

	query := fmt.Sprintf(
		"SELECT id, collection_id, database_id, data, permissions, created_at, updated_at FROM documents WHERE %s ORDER BY %s LIMIT ? OFFSET ?",
		where, orderBy)
	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var docs []*model.Document
	for rows.Next() {
		var doc model.Document
		var dataJSON, permsJSON []byte
		if err := rows.Scan(&doc.ID, &doc.TableID, &doc.DatabaseID, &dataJSON, &permsJSON, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(dataJSON, &doc.Data)        //nolint:errcheck
		json.Unmarshal(permsJSON, &doc.Permissions) //nolint:errcheck
		if doc.Permissions == nil {
			doc.Permissions = []string{}
		}
		docs = append(docs, &doc)
	}

	// Count total matching documents
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM documents WHERE %s", where)
	countArgs := args[:len(args)-2] // exclude LIMIT and OFFSET
	var total int
	s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total) //nolint:errcheck

	return docs, total, nil
}

func (s *Service) UpdateDocument(ctx context.Context, docID, collID, dbID, projectID string, data map[string]interface{}, permissions []string) (*model.Document, error) {
	return s.UpdateDocumentWithAuth(ctx, docID, collID, dbID, projectID, data, permissions, "", nil)
}

// UpdateDocumentWithAuth updates a document with permission checking.
func (s *Service) UpdateDocumentWithAuth(ctx context.Context, docID, collID, dbID, projectID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Document, error) {
	// Check update permission — first fetch existing doc to get its permissions
	if userID != "" {
		existing, err := s.GetDocument(ctx, docID, collID, dbID, projectID)
		if err != nil {
			return nil, err
		}
		if err := s.enforcePermission(ctx, projectID, collID, userID, roles, "update", existing.Permissions); err != nil {
			return nil, err
		}
	}

	if err := s.validateDocData(ctx, collID, data); err != nil {
		return nil, err
	}
	dataJSON, _ := json.Marshal(data)
	permsJSON, _ := json.Marshal(permissions)
	_, err := s.db.ExecContext(ctx,
		"UPDATE documents SET data = ?, permissions = ? WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
		dataJSON, permsJSON, docID, collID, dbID, projectID)
	if err != nil {
		return nil, err
	}
	doc, err := s.GetDocument(ctx, docID, collID, dbID, projectID)
	if err == nil {
		realtime.PublishResourceEvent(s.events, "databases", "documents", "update", projectID, docID, doc)
	}
	return doc, err
}

// --- transactions ---

// TransactionOp represents a single operation within a database transaction.
type TransactionOp struct {
	Action       string                 `json:"action"`       // create, update, delete
	CollectionID string                 `json:"collectionId"`
	DocumentID   string                 `json:"documentId,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Permissions  []string               `json:"permissions,omitempty"`
}

// TransactionResult holds the result of a single operation within a transaction.
type TransactionResult struct {
	Action     string      `json:"action"`
	DocumentID string      `json:"documentId"`
	Result     interface{} `json:"result,omitempty"`
}

// ExecuteTransaction executes a batch of create/update/delete operations atomically.
func (s *Service) ExecuteTransaction(ctx context.Context, projectID, dbID string, ops []TransactionOp) ([]TransactionResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("transaction: begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var results []TransactionResult

	for i, op := range ops {
		if op.CollectionID == "" {
			return nil, fmt.Errorf("transaction: operation %d: collectionId is required", i)
		}
		if op.Permissions == nil {
			op.Permissions = []string{}
		}

		switch op.Action {
		case "create":
			docID := uid.New("")
			now := time.Now().UTC()
			dataJSON, err := json.Marshal(op.Data)
			if err != nil {
				return nil, fmt.Errorf("transaction: operation %d: marshal data: %w", i, err)
			}
			permsJSON, _ := json.Marshal(op.Permissions)
			_, err = tx.ExecContext(ctx,
				"INSERT INTO documents (id, collection_id, database_id, project_id, data, permissions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
				docID, op.CollectionID, dbID, projectID, dataJSON, permsJSON, now, now)
			if err != nil {
				return nil, fmt.Errorf("transaction: operation %d (create): %w", i, err)
			}
			results = append(results, TransactionResult{
				Action:     "create",
				DocumentID: docID,
				Result: map[string]interface{}{
					"$id":        docID,
					"$createdAt": now,
					"$updatedAt": now,
				},
			})

		case "update":
			if op.DocumentID == "" {
				return nil, fmt.Errorf("transaction: operation %d: documentId is required for update", i)
			}
			dataJSON, err := json.Marshal(op.Data)
			if err != nil {
				return nil, fmt.Errorf("transaction: operation %d: marshal data: %w", i, err)
			}
			permsJSON, _ := json.Marshal(op.Permissions)
			res, err := tx.ExecContext(ctx,
				"UPDATE documents SET data = ?, permissions = ?, updated_at = ? WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
				dataJSON, permsJSON, time.Now().UTC(), op.DocumentID, op.CollectionID, dbID, projectID)
			if err != nil {
				return nil, fmt.Errorf("transaction: operation %d (update): %w", i, err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return nil, fmt.Errorf("transaction: operation %d (update): document %s not found", i, op.DocumentID)
			}
			results = append(results, TransactionResult{
				Action:     "update",
				DocumentID: op.DocumentID,
			})

		case "delete":
			if op.DocumentID == "" {
				return nil, fmt.Errorf("transaction: operation %d: documentId is required for delete", i)
			}
			res, err := tx.ExecContext(ctx,
				"DELETE FROM documents WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
				op.DocumentID, op.CollectionID, dbID, projectID)
			if err != nil {
				return nil, fmt.Errorf("transaction: operation %d (delete): %w", i, err)
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return nil, fmt.Errorf("transaction: operation %d (delete): document %s not found", i, op.DocumentID)
			}
			results = append(results, TransactionResult{
				Action:     "delete",
				DocumentID: op.DocumentID,
			})

		default:
			return nil, fmt.Errorf("transaction: operation %d: unsupported action %q (must be create, update, or delete)", i, op.Action)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("transaction: commit: %w", err)
	}

	return results, nil
}

func (s *Service) DeleteDocument(ctx context.Context, docID, collID, dbID, projectID string) error {
	return s.DeleteDocumentWithAuth(ctx, docID, collID, dbID, projectID, "", nil)
}

// DeleteDocumentWithAuth deletes a document with permission checking.
func (s *Service) DeleteDocumentWithAuth(ctx context.Context, docID, collID, dbID, projectID, userID string, roles []string) error {
	// Check delete permission
	if userID != "" {
		existing, err := s.GetDocument(ctx, docID, collID, dbID, projectID)
		if err != nil {
			return err
		}
		if err := s.enforcePermission(ctx, projectID, collID, userID, roles, "delete", existing.Permissions); err != nil {
			return err
		}
	}

	_, err := s.db.ExecContext(ctx,
		"DELETE FROM documents WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
		docID, collID, dbID, projectID)
	if err == nil {
		realtime.PublishResourceEvent(s.events, "databases", "documents", "delete", projectID, docID, map[string]string{"$id": docID})
	}
	return err
}
