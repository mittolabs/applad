// Package databases implements Applad's TablesDB service:
// databases, collections, attributes, indexes, documents, queries, and permissions.
package databases

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/uid"
)

// Service handles database/collection/document business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new databases Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
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
		DocumentSecurity: docSecurity, Permissions: permissions,
		Attributes: []model.Attribute{}, Indexes: []model.Index{},
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Service) GetCollection(ctx context.Context, collID, dbID, projectID string) (*model.Collection, error) {
	var c model.Collection
	var permsJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, database_id, name, document_security, permissions, enabled, created_at, updated_at FROM collections WHERE id = ? AND database_id = ? AND project_id = ?",
		collID, dbID, projectID).Scan(&c.ID, &c.DatabaseID, &c.Name, &c.DocumentSecurity, &permsJSON, &c.Enabled, &c.CreatedAt, &c.UpdatedAt)
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
	c.Attributes = attrs
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
		if err := rows.Scan(&c.ID, &c.DatabaseID, &c.Name, &c.DocumentSecurity, &permsJSON, &c.Enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(permsJSON, &c.Permissions) //nolint:errcheck
		if c.Permissions == nil {
			c.Permissions = []string{}
		}
		c.Attributes = []model.Attribute{}
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
		"SELECT `key`, type, status, required, array, default_value FROM attributes WHERE collection_id = ? ORDER BY created_at ASC",
		collID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var attrs []model.Attribute
	for rows.Next() {
		var a model.Attribute
		var defVal sql.NullString
		if err := rows.Scan(&a.Key, &a.Type, &a.Status, &a.Required, &a.Array, &defVal); err != nil {
			return nil, err
		}
		if defVal.Valid {
			var v interface{}
			json.Unmarshal([]byte(defVal.String), &v) //nolint:errcheck
			a.Default = v
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
	return &model.Index{Key: key, Type: idxType, Status: "available", Attributes: attributes, Orders: orders}, nil
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
		json.Unmarshal(attrsJSON, &idx.Attributes) //nolint:errcheck
		if idx.Attributes == nil {
			idx.Attributes = []string{}
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

// --- documents ---

func (s *Service) CreateDocument(ctx context.Context, projectID, dbID, collID, docID string, data map[string]interface{}, permissions []string) (*model.Document, error) {
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
	return &model.Document{
		ID: id, CollectionID: collID, DatabaseID: dbID,
		Permissions: permissions, Data: data,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Service) GetDocument(ctx context.Context, docID, collID, dbID, projectID string) (*model.Document, error) {
	var doc model.Document
	var dataJSON, permsJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, collection_id, database_id, data, permissions, created_at, updated_at FROM documents WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
		docID, collID, dbID, projectID).Scan(&doc.ID, &doc.CollectionID, &doc.DatabaseID, &dataJSON, &permsJSON, &doc.CreatedAt, &doc.UpdatedAt)
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
	return &doc, nil
}

func (s *Service) ListDocuments(ctx context.Context, projectID, dbID, collID string, limit, offset int) ([]*model.Document, int, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, collection_id, database_id, data, permissions, created_at, updated_at FROM documents WHERE collection_id = ? AND database_id = ? AND project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		collID, dbID, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var docs []*model.Document
	for rows.Next() {
		var doc model.Document
		var dataJSON, permsJSON []byte
		if err := rows.Scan(&doc.ID, &doc.CollectionID, &doc.DatabaseID, &dataJSON, &permsJSON, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(dataJSON, &doc.Data)        //nolint:errcheck
		json.Unmarshal(permsJSON, &doc.Permissions) //nolint:errcheck
		if doc.Permissions == nil {
			doc.Permissions = []string{}
		}
		docs = append(docs, &doc)
	}
	var total int
	s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM documents WHERE collection_id = ? AND database_id = ? AND project_id = ?",
		collID, dbID, projectID).Scan(&total) //nolint:errcheck
	return docs, total, nil
}

func (s *Service) UpdateDocument(ctx context.Context, docID, collID, dbID, projectID string, data map[string]interface{}, permissions []string) (*model.Document, error) {
	dataJSON, _ := json.Marshal(data)
	permsJSON, _ := json.Marshal(permissions)
	_, err := s.db.ExecContext(ctx,
		"UPDATE documents SET data = ?, permissions = ? WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
		dataJSON, permsJSON, docID, collID, dbID, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetDocument(ctx, docID, collID, dbID, projectID)
}

func (s *Service) DeleteDocument(ctx context.Context, docID, collID, dbID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM documents WHERE id = ? AND collection_id = ? AND database_id = ? AND project_id = ?",
		docID, collID, dbID, projectID)
	return err
}
