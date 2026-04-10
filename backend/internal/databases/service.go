// Package databases implements PostgreSQL-backed schema orchestration and row CRUD.
package databases

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/uid"
)

var safeSchemaSegment = regexp.MustCompile(`[^a-zA-Z0-9]`)

// Service handles database, table, column, and row operations.
type Service struct {
	db           *db.DB
	events       realtime.EventPublisher
	postgrestURL string
	jwtSecret    string
}

type sqlContextExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// NewService creates a new databases service.
// Optional args: postgrestURL, jwtSecret.
func NewService(database *db.DB, args ...string) *Service {
	postgrestURL := "http://postgrest:3000"
	jwtSecret := ""
	if len(args) > 0 && args[0] != "" {
		postgrestURL = args[0]
	}
	if len(args) > 1 {
		jwtSecret = args[1]
	}
	return &Service{db: database, postgrestURL: postgrestURL, jwtSecret: jwtSecret}
}

// SetEventPublisher wires realtime event publishing into the service.
func (s *Service) SetEventPublisher(pub realtime.EventPublisher) {
	s.events = pub
}

func schemaName(projectID, databaseID string) string {
	return "p_" + safeSchemaSegment.ReplaceAllString(projectID, "_") + "_" + safeSchemaSegment.ReplaceAllString(databaseID, "_")
}

func pgIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func pgLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func (s *Service) notifyPostgREST(ctx context.Context) {
	s.db.ExecContext(ctx, "NOTIFY pgrst, 'reload schema'") //nolint:errcheck
}

type postgrestClaims struct {
	jwt.RegisteredClaims
	Role      string   `json:"role"`
	ProjectID string   `json:"project_id"`
	UserID    string   `json:"user_id,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}

func normalizeRoles(userID string, roles []string) []string {
	allRoles := buildRoles(userID, roles)
	return uniqueSortedRoles(allRoles)
}

func uniqueSortedRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(roles))
	normalized := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		normalized = append(normalized, role)
	}
	sort.Strings(normalized)
	return normalized
}

func (s *Service) signedPostgRESTJWT(projectID, userID string, roles []string) (string, error) {
	if strings.TrimSpace(s.jwtSecret) == "" {
		return "", nil
	}
	claims := postgrestClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(5 * time.Minute)),
		},
		Role:      "applad_user",
		ProjectID: projectID,
		UserID:    userID,
		Roles:     normalizeRoles(userID, roles),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) postgrestHeaders(projectID, userID string, roles []string, extra map[string]string) (map[string]string, error) {
	headers := map[string]string{}
	for key, value := range extra {
		headers[key] = value
	}
	token, err := s.signedPostgRESTJWT(projectID, userID, roles)
	if err != nil {
		return nil, err
	}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	return headers, nil
}

func policyRoleExpression(roles []string) string {
	if len(roles) == 0 {
		return "FALSE"
	}
	clauses := make([]string, 0, len(roles))
	for _, role := range uniqueSortedRoles(roles) {
		switch {
		case role == "any":
			return "TRUE"
		case role == "users":
			clauses = append(clauses, "NULLIF(current_setting('applad.user_id', true), '') IS NOT NULL")
		case strings.HasPrefix(role, "user:"):
			clauses = append(clauses, fmt.Sprintf("current_setting('applad.user_id', true) = %s", pgLiteral(strings.TrimPrefix(role, "user:"))))
		default:
			clauses = append(clauses, fmt.Sprintf("(COALESCE(NULLIF(current_setting('request.jwt.claims', true), '')::jsonb -> 'roles', '[]'::jsonb) ? %s)", pgLiteral(role)))
		}
	}
	if len(clauses) == 0 {
		return "FALSE"
	}
	return "(" + strings.Join(clauses, " OR ") + ")"
}

func ownerAccessExpression(ownerColumn string) string {
	if ownerColumn == "" {
		return ""
	}
	return fmt.Sprintf("(NULLIF(current_setting('applad.user_id', true), '') IS NOT NULL AND %s = current_setting('applad.user_id', true))", pgIdent(ownerColumn))
}

func createPolicySQL(schema, table, policyName, scope, command, usingExpr, checkExpr string) string {
	parts := []string{fmt.Sprintf("CREATE POLICY %s ON %s.%s", pgIdent(policyName), pgIdent(schema), pgIdent(table))}
	if scope != "" {
		parts = append(parts, scope)
	}
	if command != "" {
		parts = append(parts, "FOR "+command)
	}
	if usingExpr != "" {
		parts = append(parts, "USING ("+usingExpr+")")
	}
	if checkExpr != "" {
		parts = append(parts, "WITH CHECK ("+checkExpr+")")
	}
	return strings.Join(parts, " ")
}

func toSQLType(columnType string, options map[string]interface{}) string {
	switch columnType {
	case "string":
		if options != nil {
			if size, ok := options["size"].(float64); ok && size > 0 {
				return fmt.Sprintf("VARCHAR(%d)", int(size))
			}
		}
		return "TEXT"
	case "integer":
		return "BIGINT"
	case "float", "double":
		return "DOUBLE PRECISION"
	case "boolean":
		return "BOOLEAN"
	case "datetime":
		return "TIMESTAMPTZ"
	case "email", "url", "enum":
		return "TEXT"
	case "ip":
		return "INET"
	case "point":
		return "POINT"
	default:
		return "TEXT"
	}
}

func (s *Service) CreateDatabase(ctx context.Context, projectID, databaseID, name string) (*model.Database, error) {
	id := uid.New(databaseID)
	now := time.Now().UTC()

	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pgIdent(schemaName(projectID, id)))); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO databases (id, project_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id, projectID, name, now, now,
	); err != nil {
		return nil, fmt.Errorf("create database metadata: %w", err)
	}

	s.notifyPostgREST(ctx)
	return &model.Database{ID: id, Name: name, Enabled: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Service) GetDatabase(ctx context.Context, databaseID, projectID string) (*model.Database, error) {
	var database model.Database
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, name, created_at, updated_at FROM databases WHERE id = ? AND project_id = ?",
		databaseID, projectID,
	).Scan(&database.ID, &database.Name, &database.CreatedAt, &database.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("database not found")
		}
		return nil, err
	}
	database.Enabled = true
	return &database, nil
}

func (s *Service) ListDatabases(ctx context.Context, projectID string) ([]*model.Database, int, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, created_at, updated_at FROM databases WHERE project_id = ? ORDER BY created_at DESC",
		projectID,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var databases []*model.Database
	for rows.Next() {
		var database model.Database
		if err := rows.Scan(&database.ID, &database.Name, &database.CreatedAt, &database.UpdatedAt); err != nil {
			return nil, 0, err
		}
		database.Enabled = true
		databases = append(databases, &database)
	}
	if databases == nil {
		databases = []*model.Database{}
	}
	return databases, len(databases), nil
}

func (s *Service) UpdateDatabase(ctx context.Context, databaseID, projectID, name string) (*model.Database, error) {
	if _, err := s.db.ExecContext(ctx,
		"UPDATE databases SET name = ?, updated_at = ? WHERE id = ? AND project_id = ?",
		name, time.Now().UTC(), databaseID, projectID,
	); err != nil {
		return nil, err
	}
	return s.GetDatabase(ctx, databaseID, projectID)
}

func (s *Service) DeleteDatabase(ctx context.Context, databaseID, projectID string) error {
	if _, err := s.GetDatabase(ctx, databaseID, projectID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", pgIdent(schemaName(projectID, databaseID))),
	); err != nil {
		return fmt.Errorf("drop schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		"DELETE FROM databases WHERE id = ? AND project_id = ?",
		databaseID, projectID,
	); err != nil {
		return err
	}
	s.notifyPostgREST(ctx)
	return nil
}

func (s *Service) CreateTable(ctx context.Context, projectID, databaseID, tableID, name string, permissions []string, rowSecurity bool) (*model.Table, error) {
	id := uid.New(tableID)
	now := time.Now().UTC()
	permissionsJSON, _ := json.Marshal(permissions)
	schema := schemaName(projectID, databaseID)

	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id TEXT NOT NULL DEFAULT gen_random_uuid()::text PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`, pgIdent(schema), pgIdent(name))
	if _, err := s.db.ExecContext(ctx, createTableSQL); err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}

	triggerSQL := fmt.Sprintf(
		"CREATE TRIGGER set_updated_at BEFORE UPDATE ON %s.%s FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at()",
		pgIdent(schema), pgIdent(name),
	)
	s.db.ExecContext(ctx, triggerSQL) //nolint:errcheck
	notifySQL := fmt.Sprintf(
		"CREATE TRIGGER notify_changes AFTER INSERT OR UPDATE OR DELETE ON %s.%s FOR EACH ROW EXECUTE FUNCTION applad_notify_change()",
		pgIdent(schema), pgIdent(name),
	)
	s.db.ExecContext(ctx, notifySQL) //nolint:errcheck

	if rowSecurity {
		s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY", pgIdent(schema), pgIdent(name))) //nolint:errcheck
		s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s.%s FORCE ROW LEVEL SECURITY", pgIdent(schema), pgIdent(name)))  //nolint:errcheck
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO tables (id, database_id, project_id, name, enabled, row_security, permissions, created_at, updated_at)
		 VALUES (?, ?, ?, ?, TRUE, ?, ?, ?, ?)`,
		id, databaseID, projectID, name, rowSecurity, permissionsJSON, now, now,
	); err != nil {
		return nil, fmt.Errorf("insert table metadata: %w", err)
	}
	if rowSecurity {
		tableContext, err := s.lookupTableContext(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.syncRLSPolicies(ctx, tableContext); err != nil {
			return nil, err
		}
	}

	s.notifyPostgREST(ctx)
	return &model.Table{
		ID: id, DatabaseID: databaseID, Name: name, Enabled: true,
		RowSecurity: rowSecurity, Permissions: permissions,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Service) GetTable(ctx context.Context, tableID, databaseID, projectID string) (*model.Table, error) {
	var table model.Table
	var permissionsJSON []byte
	if err := s.db.QueryRowContext(ctx,
		`SELECT id, database_id, name, enabled, row_security, COALESCE(permissions, '[]'), created_at, updated_at
		 FROM tables WHERE id = ? AND database_id = ? AND project_id = ?`,
		tableID, databaseID, projectID,
	).Scan(&table.ID, &table.DatabaseID, &table.Name, &table.Enabled, &table.RowSecurity, &permissionsJSON, &table.CreatedAt, &table.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("table not found")
		}
		return nil, err
	}
	json.Unmarshal(permissionsJSON, &table.Permissions) //nolint:errcheck
	table.Columns, _ = s.ListColumns(ctx, table.ID)
	table.Indexes, _ = s.ListIndexes(ctx, table.ID)
	return &table, nil
}

func (s *Service) ListTables(ctx context.Context, databaseID, projectID string) ([]*model.Table, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, database_id, name, enabled, row_security, COALESCE(permissions, '[]'), created_at, updated_at
		 FROM tables WHERE database_id = ? AND project_id = ? ORDER BY created_at DESC`,
		databaseID, projectID,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tables []*model.Table
	for rows.Next() {
		var table model.Table
		var permissionsJSON []byte
		if err := rows.Scan(&table.ID, &table.DatabaseID, &table.Name, &table.Enabled, &table.RowSecurity, &permissionsJSON, &table.CreatedAt, &table.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(permissionsJSON, &table.Permissions) //nolint:errcheck
		tables = append(tables, &table)
	}
	if tables == nil {
		tables = []*model.Table{}
	}
	return tables, len(tables), nil
}

func (s *Service) UpdateTable(ctx context.Context, tableID, databaseID, projectID, name string, permissions []string, enabled bool) (*model.Table, error) {
	permissionsJSON, _ := json.Marshal(permissions)
	if _, err := s.db.ExecContext(ctx,
		"UPDATE tables SET name = ?, enabled = ?, permissions = ?, updated_at = ? WHERE id = ? AND database_id = ? AND project_id = ?",
		name, enabled, permissionsJSON, time.Now().UTC(), tableID, databaseID, projectID,
	); err != nil {
		return nil, err
	}
	return s.GetTable(ctx, tableID, databaseID, projectID)
}

func (s *Service) DeleteTable(ctx context.Context, tableID, databaseID, projectID string) error {
	table, err := s.GetTable(ctx, tableID, databaseID, projectID)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf("DROP TABLE IF EXISTS %s.%s CASCADE", pgIdent(schemaName(projectID, databaseID)), pgIdent(table.Name)),
	); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM tables WHERE id = ? AND project_id = ?", tableID, projectID); err != nil {
		return err
	}
	s.notifyPostgREST(ctx)
	return nil
}

func (s *Service) CreateColumn(ctx context.Context, tableID, key, columnType string, required, array bool, defaultValue interface{}, options map[string]interface{}) (*model.Column, error) {
	tableContext, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}

	sqlType := toSQLType(columnType, options)
	if array {
		sqlType += "[]"
	}
	ddl := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s %s", pgIdent(tableContext.Schema), pgIdent(tableContext.Name), pgIdent(key), sqlType)
	if required && defaultValue == nil {
		ddl += " NOT NULL"
	}
	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return nil, fmt.Errorf("add column: %w", err)
	}

	optionsJSON, _ := json.Marshal(options)
	var defaultJSON []byte
	if defaultValue != nil {
		defaultJSON, _ = json.Marshal(defaultValue)
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO columns (id, table_id, key_name, type, required, "array", default_value, options, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'available', ?)`,
		uid.New("unique()"), tableID, key, columnType, required, array, defaultJSON, optionsJSON, time.Now().UTC(),
	); err != nil {
		return nil, fmt.Errorf("insert column metadata: %w", err)
	}
	if err := s.syncRLSPolicies(ctx, tableContext); err != nil {
		return nil, err
	}

	s.notifyPostgREST(ctx)
	return &model.Column{Key: key, Type: columnType, Status: "available", Required: required, Array: array, Default: defaultValue, Options: options}, nil
}

func (s *Service) ListColumns(ctx context.Context, tableID string) ([]model.Column, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key_name, type, status, required, "array", default_value, COALESCE(options, '{}')
		 FROM columns WHERE table_id = ? ORDER BY created_at ASC`,
		tableID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []model.Column
	for rows.Next() {
		var column model.Column
		var defaultJSON []byte
		var optionsJSON []byte
		if err := rows.Scan(&column.Key, &column.Type, &column.Status, &column.Required, &column.Array, &defaultJSON, &optionsJSON); err != nil {
			return nil, err
		}
		if len(defaultJSON) > 0 {
			json.Unmarshal(defaultJSON, &column.Default) //nolint:errcheck
		}
		if len(optionsJSON) > 0 {
			json.Unmarshal(optionsJSON, &column.Options) //nolint:errcheck
		}
		columns = append(columns, column)
	}
	if columns == nil {
		columns = []model.Column{}
	}
	return columns, nil
}

func (s *Service) DeleteColumn(ctx context.Context, tableID, key string) error {
	tableContext, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s", pgIdent(tableContext.Schema), pgIdent(tableContext.Name), pgIdent(key)),
	); err != nil {
		return fmt.Errorf("drop column: %w", err)
	}
	s.db.ExecContext(ctx, "DELETE FROM columns WHERE table_id = ? AND key_name = ?", tableID, key) //nolint:errcheck
	if err := s.syncRLSPolicies(ctx, tableContext); err != nil {
		return err
	}
	s.notifyPostgREST(ctx)
	return nil
}

func (s *Service) CreateIndex(ctx context.Context, tableID, key, indexType string, columns, orders []string) (*model.Index, error) {
	tableContext, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, pgIdent(column))
	}

	var ddl string
	switch indexType {
	case "unique":
		ddl = fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s.%s (%s)", pgIdent(key), pgIdent(tableContext.Schema), pgIdent(tableContext.Name), strings.Join(quotedColumns, ", "))
	case "fulltext":
		if len(columns) == 1 {
			ddl = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s.%s USING GIN (to_tsvector('english', %s))", pgIdent(key), pgIdent(tableContext.Schema), pgIdent(tableContext.Name), pgIdent(columns[0]))
		} else {
			ddl = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s.%s (%s)", pgIdent(key), pgIdent(tableContext.Schema), pgIdent(tableContext.Name), strings.Join(quotedColumns, ", "))
		}
	default:
		ddl = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s.%s (%s)", pgIdent(key), pgIdent(tableContext.Schema), pgIdent(tableContext.Name), strings.Join(quotedColumns, ", "))
	}

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}

	columnsJSON, _ := json.Marshal(columns)
	ordersJSON, _ := json.Marshal(orders)
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO indexes (id, table_id, key_name, type, columns, orders, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'available', ?)`,
		uid.New("unique()"), tableID, key, indexType, columnsJSON, ordersJSON, time.Now().UTC(),
	); err != nil {
		return nil, fmt.Errorf("insert index metadata: %w", err)
	}

	s.notifyPostgREST(ctx)
	return &model.Index{Key: key, Type: indexType, Status: "available", Columns: columns, Orders: orders}, nil
}

func (s *Service) ListIndexes(ctx context.Context, tableID string) ([]model.Index, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT key_name, type, status, columns, orders FROM indexes WHERE table_id = ? ORDER BY created_at ASC",
		tableID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []model.Index
	for rows.Next() {
		var index model.Index
		var columnsJSON []byte
		var ordersJSON []byte
		if err := rows.Scan(&index.Key, &index.Type, &index.Status, &columnsJSON, &ordersJSON); err != nil {
			return nil, err
		}
		json.Unmarshal(columnsJSON, &index.Columns) //nolint:errcheck
		json.Unmarshal(ordersJSON, &index.Orders)   //nolint:errcheck
		indexes = append(indexes, index)
	}
	if indexes == nil {
		indexes = []model.Index{}
	}
	return indexes, nil
}

func (s *Service) DeleteIndex(ctx context.Context, tableID, key string) error {
	tableContext, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return err
	}
	s.db.ExecContext(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s.%s", pgIdent(tableContext.Schema), pgIdent(key))) //nolint:errcheck
	s.db.ExecContext(ctx, "DELETE FROM indexes WHERE table_id = ? AND key_name = ?", tableID, key)               //nolint:errcheck
	s.notifyPostgREST(ctx)
	return nil
}

// Relationship represents a foreign key between tables.
type Relationship struct {
	ID           string `json:"$id"`
	TableID      string `json:"tableId"`
	RelatedTable string `json:"relatedTableId"`
	Type         string `json:"type"`
	TwoWay       bool   `json:"twoWay"`
	Key          string `json:"key"`
	TwoWayKey    string `json:"twoWayKey,omitempty"`
	OnDelete     string `json:"onDelete"`
}

func (s *Service) CreateRelationship(ctx context.Context, tableID, relatedTableID, relationType, key, twoWayKey, onDelete string, twoWay bool) (*Relationship, error) {
	id := uid.New("unique()")
	if onDelete == "" {
		onDelete = "setNull"
	}
	pgOnDelete := "SET NULL"
	switch onDelete {
	case "cascade":
		pgOnDelete = "CASCADE"
	case "restrict":
		pgOnDelete = "RESTRICT"
	}

	leftTable, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	rightTable, err := s.lookupTableContext(ctx, relatedTableID)
	if err != nil {
		return nil, err
	}

	fkName := "fk_" + id
	ddl := fmt.Sprintf(
		`ALTER TABLE %s.%s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(id) ON DELETE %s`,
		pgIdent(leftTable.Schema), pgIdent(leftTable.Name), pgIdent(fkName),
		pgIdent(key), pgIdent(rightTable.Schema), pgIdent(rightTable.Name), pgOnDelete,
	)
	s.db.ExecContext(ctx, ddl) //nolint:errcheck

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO table_relationships (id, table_id, related_table, relationship_type, two_way, key_name, two_way_key, on_delete)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tableID, relatedTableID, relationType, twoWay, key, twoWayKey, onDelete,
	); err != nil {
		return nil, fmt.Errorf("create relationship metadata: %w", err)
	}

	s.notifyPostgREST(ctx)
	return &Relationship{ID: id, TableID: tableID, RelatedTable: relatedTableID, Type: relationType, TwoWay: twoWay, Key: key, TwoWayKey: twoWayKey, OnDelete: onDelete}, nil
}

func (s *Service) ListRelationships(ctx context.Context, tableID string) ([]Relationship, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, table_id, related_table, relationship_type, two_way, key_name, two_way_key, on_delete
		 FROM table_relationships WHERE table_id = ?`,
		tableID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []Relationship
	for rows.Next() {
		var relationship Relationship
		var twoWayKey sql.NullString
		if err := rows.Scan(&relationship.ID, &relationship.TableID, &relationship.RelatedTable, &relationship.Type, &relationship.TwoWay, &relationship.Key, &twoWayKey, &relationship.OnDelete); err != nil {
			return nil, err
		}
		relationship.TwoWayKey = twoWayKey.String
		relationships = append(relationships, relationship)
	}
	if relationships == nil {
		relationships = []Relationship{}
	}
	return relationships, nil
}

func (s *Service) DeleteRelationship(ctx context.Context, tableID, relationshipID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM table_relationships WHERE id = ? AND table_id = ?", relationshipID, tableID)
	return err
}

// Permission represents a granular permission row.
type Permission struct {
	Role   string `json:"role"`
	Action string `json:"action"`
}

var validActions = map[string]bool{
	"read":   true,
	"create": true,
	"update": true,
	"delete": true,
}

func (s *Service) checkPermission(ctx context.Context, projectID, resourceType, resourceID string, roles []string, action string) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}
	placeholders := make([]string, len(roles))
	args := make([]interface{}, 0, len(roles)+4)
	args = append(args, projectID, resourceType, resourceID, action)
	for index, role := range roles {
		placeholders[index] = "?"
		args = append(args, role)
	}
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM permissions WHERE project_id = ? AND resource_type = ? AND resource_id = ? AND action = ? AND role IN (%s)",
		strings.Join(placeholders, ","),
	)
	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) SetPermissions(ctx context.Context, projectID, resourceType, resourceID string, permissions []Permission) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin permission transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM permissions WHERE project_id = ? AND resource_type = ? AND resource_id = ?",
		projectID, resourceType, resourceID,
	); err != nil {
		return fmt.Errorf("delete permissions: %w", err)
	}
	for _, permission := range permissions {
		if !validActions[permission.Action] {
			return fmt.Errorf("invalid permission action %q", permission.Action)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO permissions (id, project_id, resource_type, resource_id, role, action) VALUES (?, ?, ?, ?, ?, ?)",
			uid.New("unique()"), projectID, resourceType, resourceID, permission.Role, permission.Action,
		); err != nil {
			return fmt.Errorf("insert permission: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if resourceType == "table" {
		tableContext, err := s.lookupTableContext(ctx, resourceID)
		if err != nil {
			return err
		}
		if err := s.syncRLSPolicies(ctx, tableContext); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetPermissions(ctx context.Context, projectID, resourceType, resourceID string) ([]Permission, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT role, action FROM permissions WHERE project_id = ? AND resource_type = ? AND resource_id = ? ORDER BY created_at ASC",
		projectID, resourceType, resourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var permission Permission
		if err := rows.Scan(&permission.Role, &permission.Action); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if permissions == nil {
		permissions = []Permission{}
	}
	return permissions, nil
}

func buildRoles(userID string, roles []string) []string {
	allRoles := make([]string, 0, len(roles)+3)
	allRoles = append(allRoles, "any")
	if userID != "" {
		allRoles = append(allRoles, "users", "user:"+userID)
	}
	allRoles = append(allRoles, roles...)
	return allRoles
}

func checkRowPermission(permissions []string, roles []string, action string) bool {
	roleSet := make(map[string]bool, len(roles))
	for _, role := range roles {
		roleSet[role] = true
	}
	prefix := action + "("
	for _, permission := range permissions {
		if strings.HasPrefix(permission, prefix) && strings.HasSuffix(permission, ")") {
			role := permission[len(prefix) : len(permission)-1]
			if role == "any" || roleSet[role] {
				return true
			}
		}
	}
	return false
}

func (s *Service) enforcePermission(ctx context.Context, projectID, tableID, userID string, roles []string, action string, rowPermissions []string) error {
	allRoles := buildRoles(userID, roles)
	allowed, err := s.checkPermission(ctx, projectID, "table", tableID, allRoles, action)
	if err != nil {
		return fmt.Errorf("check table permission: %w", err)
	}
	if allowed {
		return nil
	}

	var rowSecurity bool
	if err := s.db.QueryRowContext(ctx,
		"SELECT row_security FROM tables WHERE id = ? AND project_id = ?",
		tableID, projectID,
	).Scan(&rowSecurity); err != nil {
		return fmt.Errorf("permission denied")
	}
	if rowSecurity && len(rowPermissions) > 0 && checkRowPermission(rowPermissions, allRoles, action) {
		return nil
	}
	return fmt.Errorf("permission denied")
}

// ListParams defines filtering, ordering, and pagination for row queries.
type ListParams struct {
	Limit       int
	Offset      int
	OrderAttr   string
	OrderType   string
	Select      string
	CursorAfter string
	Queries     []Query
}

// Query represents one filter condition.
type Query struct {
	Field  string
	Method string
	Values interface{}
}

func queryToPostgREST(query Query) (string, string) {
	valueString := func(value interface{}) string { return fmt.Sprintf("%v", value) }
	switch query.Method {
	case "equal":
		return query.Field, "eq." + valueString(query.Values)
	case "notEqual":
		return query.Field, "neq." + valueString(query.Values)
	case "lessThan":
		return query.Field, "lt." + valueString(query.Values)
	case "lessThanEqual":
		return query.Field, "lte." + valueString(query.Values)
	case "greaterThan":
		return query.Field, "gt." + valueString(query.Values)
	case "greaterThanEqual":
		return query.Field, "gte." + valueString(query.Values)
	case "contains":
		return query.Field, "like.*" + valueString(query.Values) + "*"
	case "startsWith":
		return query.Field, "like." + valueString(query.Values) + "*"
	case "endsWith":
		return query.Field, "like.*" + valueString(query.Values)
	case "search":
		return query.Field, "fts." + valueString(query.Values)
	case "isNull":
		return query.Field, "is.null"
	case "isNotNull":
		return query.Field, "not.is.null"
	default:
		return query.Field, "eq." + valueString(query.Values)
	}
}

type tableContext struct {
	ID         string
	DatabaseID string
	ProjectID  string
	Name       string
	Schema     string
}

func (s *Service) lookupTableContext(ctx context.Context, tableID string) (*tableContext, error) {
	var table tableContext
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, database_id, project_id, name FROM tables WHERE id = ?",
		tableID,
	).Scan(&table.ID, &table.DatabaseID, &table.ProjectID, &table.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("table not found")
		}
		return nil, err
	}
	table.Schema = schemaName(table.ProjectID, table.DatabaseID)
	return &table, nil
}

func (s *Service) ownerColumn(ctx context.Context, tableID string) (string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT key_name FROM columns WHERE table_id = ? AND key_name IN (?, ?) ORDER BY CASE key_name WHEN 'owner_id' THEN 0 ELSE 1 END LIMIT 1",
		tableID, "owner_id", "user_id",
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return "", err
		}
		return key, nil
	}
	return "", nil
}

func (s *Service) syncRLSPolicies(ctx context.Context, table *tableContext) error {
	var rowSecurity bool
	if err := s.db.QueryRowContext(ctx,
		"SELECT row_security FROM tables WHERE id = ? AND project_id = ?",
		table.ID, table.ProjectID,
	).Scan(&rowSecurity); err != nil {
		return fmt.Errorf("load row security: %w", err)
	}

	policyNames := []string{
		"applad_project_isolation",
		"applad_read_access",
		"applad_create_access",
		"applad_update_access",
		"applad_delete_access",
		"applad_owner_rows",
	}
	for _, policyName := range policyNames {
		if _, err := s.db.ExecContext(ctx,
			fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s.%s", pgIdent(policyName), pgIdent(table.Schema), pgIdent(table.Name)),
		); err != nil {
			return fmt.Errorf("drop policy %s: %w", policyName, err)
		}
	}

	if !rowSecurity {
		return nil
	}
	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY", pgIdent(table.Schema), pgIdent(table.Name)),
	); err != nil {
		return fmt.Errorf("enable row level security: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf("ALTER TABLE %s.%s FORCE ROW LEVEL SECURITY", pgIdent(table.Schema), pgIdent(table.Name)),
	); err != nil {
		return fmt.Errorf("force row level security: %w", err)
	}

	isolationExpr := fmt.Sprintf("current_setting('applad.project_id', true) = %s", pgLiteral(table.ProjectID))
	if _, err := s.db.ExecContext(ctx,
		createPolicySQL(table.Schema, table.Name, "applad_project_isolation", "AS RESTRICTIVE", "ALL", isolationExpr, isolationExpr),
	); err != nil {
		return fmt.Errorf("create project isolation policy: %w", err)
	}

	permissions, err := s.GetPermissions(ctx, table.ProjectID, "table", table.ID)
	if err != nil {
		return fmt.Errorf("load permissions: %w", err)
	}
	grouped := map[string][]string{}
	for _, permission := range permissions {
		if validActions[permission.Action] {
			grouped[permission.Action] = append(grouped[permission.Action], permission.Role)
		}
	}

	actionPolicies := []struct {
		Action   string
		Name     string
		Command  string
		UseUsing bool
		UseCheck bool
	}{
		{Action: "read", Name: "applad_read_access", Command: "SELECT", UseUsing: true},
		{Action: "create", Name: "applad_create_access", Command: "INSERT", UseCheck: true},
		{Action: "update", Name: "applad_update_access", Command: "UPDATE", UseUsing: true, UseCheck: true},
		{Action: "delete", Name: "applad_delete_access", Command: "DELETE", UseUsing: true},
	}
	for _, policy := range actionPolicies {
		expr := policyRoleExpression(grouped[policy.Action])
		if expr == "" || expr == "FALSE" {
			continue
		}
		usingExpr := ""
		checkExpr := ""
		if policy.UseUsing {
			usingExpr = expr
		}
		if policy.UseCheck {
			checkExpr = expr
		}
		if _, err := s.db.ExecContext(ctx,
			createPolicySQL(table.Schema, table.Name, policy.Name, "", policy.Command, usingExpr, checkExpr),
		); err != nil {
			return fmt.Errorf("create %s policy: %w", policy.Action, err)
		}
	}

	ownerColumn, err := s.ownerColumn(ctx, table.ID)
	if err != nil {
		return fmt.Errorf("load owner column: %w", err)
	}
	if ownerExpr := ownerAccessExpression(ownerColumn); ownerExpr != "" {
		if _, err := s.db.ExecContext(ctx,
			createPolicySQL(table.Schema, table.Name, "applad_owner_rows", "", "ALL", ownerExpr, ownerExpr),
		); err != nil {
			return fmt.Errorf("create owner policy: %w", err)
		}
	}

	return nil
}

func (s *Service) applySessionContext(ctx context.Context, exec sqlContextExecutor, projectID, userID string, roles []string) error {
	claims, err := json.Marshal(postgrestClaims{
		Role:      "applad_user",
		ProjectID: projectID,
		UserID:    userID,
		Roles:     normalizeRoles(userID, roles),
	})
	if err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, "SELECT set_config('applad.project_id', ?, true)", projectID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, "SELECT set_config('applad.user_id', ?, true)", userID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, "SELECT set_config('request.jwt.claims', ?, true)", string(claims)); err != nil {
		return err
	}
	return nil
}

func (s *Service) prepareDirectAccessTx(ctx context.Context, projectID, databaseID, userID string, roles []string, readOnly bool) (*db.Tx, string, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		return nil, "", err
	}
	schema := schemaName(projectID, databaseID)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL search_path = %s, public", pgIdent(schema))); err != nil {
		tx.Rollback() //nolint:errcheck
		return nil, "", err
	}
	if _, err := tx.ExecContext(ctx, "SET LOCAL statement_timeout = '30s'"); err != nil {
		tx.Rollback() //nolint:errcheck
		return nil, "", err
	}
	if err := s.applySessionContext(ctx, tx, projectID, userID, roles); err != nil {
		tx.Rollback() //nolint:errcheck
		return nil, "", err
	}
	return tx, schema, nil
}

func scanJSONMaps(rows *sql.Rows) ([]map[string]interface{}, []string, error) {
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		scans := make([]interface{}, len(columns))
		for index := range values {
			scans[index] = &values[index]
		}
		if err := rows.Scan(scans...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]interface{}, len(columns))
		for index, column := range columns {
			value := values[index]
			if bytesValue, ok := value.([]byte); ok {
				row[column] = string(bytesValue)
			} else {
				row[column] = value
			}
		}
		result = append(result, row)
	}
	return result, columns, rows.Err()
}

func queryRowsAsJSON(ctx context.Context, exec sqlContextExecutor, tableName, rowID string, limit, offset int) ([]map[string]interface{}, error) {
	if rowID != "" {
		rows, err := exec.QueryContext(ctx, fmt.Sprintf("SELECT to_jsonb(t) FROM %s AS t WHERE id = ? LIMIT 1", pgIdent(tableName)), rowID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := make([]map[string]interface{}, 0, 1)
		for rows.Next() {
			var raw []byte
			if err := rows.Scan(&raw); err != nil {
				return nil, err
			}
			var item map[string]interface{}
			if err := json.Unmarshal(raw, &item); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
		return result, rows.Err()
	}
	rows, err := exec.QueryContext(ctx, fmt.Sprintf("SELECT to_jsonb(t) FROM %s AS t ORDER BY created_at DESC LIMIT ? OFFSET ?", pgIdent(tableName)), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item map[string]interface{}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func mapToRow(data map[string]interface{}, tableID, databaseID string) *model.Row {
	row := &model.Row{TableID: tableID, DatabaseID: databaseID, Data: map[string]interface{}{}}
	for key, value := range data {
		switch key {
		case "id":
			row.ID = fmt.Sprintf("%v", value)
		case "created_at":
			if valueString, ok := value.(string); ok {
				row.CreatedAt, _ = time.Parse(time.RFC3339, valueString)
			}
		case "updated_at":
			if valueString, ok := value.(string); ok {
				row.UpdatedAt, _ = time.Parse(time.RFC3339, valueString)
			}
		case "$permissions", "permissions":
			if items, ok := value.([]interface{}); ok {
				for _, item := range items {
					row.Permissions = append(row.Permissions, fmt.Sprintf("%v", item))
				}
			}
		default:
			row.Data[key] = value
		}
	}
	return row
}

// queryToWhereSQL converts a slice of Query filters into a SQL WHERE clause and
// positional argument list. Arguments are numbered starting from $1.
func queryToWhereSQL(queries []Query) (string, []interface{}) {
	if len(queries) == 0 {
		return "", nil
	}
	conditions := make([]string, 0, len(queries))
	args := make([]interface{}, 0, len(queries))
	n := 1
	for _, q := range queries {
		field := pgIdent(q.Field)
		valStr := fmt.Sprintf("%v", q.Values)
		switch q.Method {
		case "equal":
			conditions = append(conditions, fmt.Sprintf("%s = $%d", field, n))
			args = append(args, q.Values)
			n++
		case "notEqual":
			conditions = append(conditions, fmt.Sprintf("%s != $%d", field, n))
			args = append(args, q.Values)
			n++
		case "lessThan":
			conditions = append(conditions, fmt.Sprintf("%s < $%d", field, n))
			args = append(args, q.Values)
			n++
		case "lessThanEqual":
			conditions = append(conditions, fmt.Sprintf("%s <= $%d", field, n))
			args = append(args, q.Values)
			n++
		case "greaterThan":
			conditions = append(conditions, fmt.Sprintf("%s > $%d", field, n))
			args = append(args, q.Values)
			n++
		case "greaterThanEqual":
			conditions = append(conditions, fmt.Sprintf("%s >= $%d", field, n))
			args = append(args, q.Values)
			n++
		case "contains":
			conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", field, n))
			args = append(args, "%"+valStr+"%")
			n++
		case "startsWith":
			conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", field, n))
			args = append(args, valStr+"%")
			n++
		case "endsWith":
			conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", field, n))
			args = append(args, "%"+valStr)
			n++
		case "search":
			conditions = append(conditions, fmt.Sprintf("to_tsvector('english', %s::text) @@ plainto_tsquery('english', $%d)", field, n))
			args = append(args, valStr)
			n++
		case "isNull":
			conditions = append(conditions, fmt.Sprintf("%s IS NULL", field))
		case "isNotNull":
			conditions = append(conditions, fmt.Sprintf("%s IS NOT NULL", field))
		default:
			conditions = append(conditions, fmt.Sprintf("%s = $%d", field, n))
			args = append(args, q.Values)
			n++
		}
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func (s *Service) CreateRow(ctx context.Context, projectID, databaseID, tableID, rowID string, data map[string]interface{}, permissions []string) (*model.Row, error) {
	return s.CreateRowWithAuth(ctx, projectID, databaseID, tableID, rowID, data, permissions, "", []string{"service"})
}

func (s *Service) CreateRowWithAuth(ctx context.Context, projectID, databaseID, tableID, rowID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Row, error) {
	if userID != "" {
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "create", nil); err != nil {
			return nil, err
		}
	}
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	if rowID == "" {
		rowID = uid.New("")
	}
	data["id"] = rowID

	// Filter out special/unknown fields before building the INSERT.
	keys := make([]string, 0, len(data))
	for key := range data {
		if key != "$permissions" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	idents := make([]string, 0, len(keys))
	placeholders := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys))
	for i, key := range keys {
		idents = append(idents, pgIdent(key))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, data[key])
	}

	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		pgIdent(table.Name), strings.Join(idents, ", "), strings.Join(placeholders, ", "))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("create row: %w", err)
	}
	rows, err := queryRowsAsJSON(ctx, tx, table.Name, rowID, 1, 0)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("create row: failed to fetch after insert")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return mapToRow(rows[0], tableID, databaseID), nil
}

func (s *Service) GetRow(ctx context.Context, rowID, tableID, databaseID, projectID string) (*model.Row, error) {
	return s.GetRowWithAuth(ctx, rowID, tableID, databaseID, projectID, "", []string{"service"})
}

func (s *Service) GetRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID, userID string, roles []string) (*model.Row, error) {
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := queryRowsAsJSON(ctx, tx, table.Name, rowID, 1, 0)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("row not found")
	}
	row := mapToRow(rows[0], tableID, databaseID)
	if userID != "" {
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "read", row.Permissions); err != nil {
			return nil, err
		}
	}
	return row, nil
}

func (s *Service) ListRows(ctx context.Context, projectID, databaseID, tableID string, limit, offset int) ([]*model.Row, int, error) {
	return s.ListRowsWithAuth(ctx, projectID, databaseID, tableID, "", []string{"service"}, ListParams{Limit: limit, Offset: offset})
}

func (s *Service) ListRowsWithQuery(ctx context.Context, projectID, databaseID, tableID string, params ListParams) ([]*model.Row, int, error) {
	return s.ListRowsWithAuth(ctx, projectID, databaseID, tableID, "", []string{"service"}, params)
}

func (s *Service) ListRowsWithAuth(ctx context.Context, projectID, databaseID, tableID, userID string, roles []string, params ListParams) ([]*model.Row, int, error) {
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, 0, err
	}
	if params.Limit <= 0 {
		params.Limit = 25
	}

	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, true)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback() //nolint:errcheck

	whereClause, whereArgs := queryToWhereSQL(params.Queries)

	// Total count.
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", pgIdent(table.Name), whereClause)
	if err := tx.QueryRowContext(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// ORDER BY
	orderClause := "ORDER BY created_at DESC"
	if params.OrderAttr != "" {
		dir := "ASC"
		if strings.EqualFold(params.OrderType, "desc") {
			dir = "DESC"
		}
		orderClause = fmt.Sprintf("ORDER BY %s %s", pgIdent(params.OrderAttr), dir)
	}

	// Append LIMIT / OFFSET args after the WHERE args.
	limitN := len(whereArgs) + 1
	offsetN := limitN + 1
	dataArgs := append(append([]interface{}{}, whereArgs...), params.Limit, params.Offset)
	dataQuery := fmt.Sprintf("SELECT to_jsonb(t) FROM %s AS t %s %s LIMIT $%d OFFSET $%d",
		pgIdent(table.Name), whereClause, orderClause, limitN, offsetN)

	sqlRows, err := tx.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer sqlRows.Close()

	result := make([]*model.Row, 0)
	for sqlRows.Next() {
		var raw []byte
		if err := sqlRows.Scan(&raw); err != nil {
			return nil, 0, err
		}
		var item map[string]interface{}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, 0, err
		}
		result = append(result, mapToRow(item, tableID, databaseID))
	}
	if err := sqlRows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

func (s *Service) UpdateRow(ctx context.Context, rowID, tableID, databaseID, projectID string, data map[string]interface{}, permissions []string) (*model.Row, error) {
	return s.UpdateRowWithAuth(ctx, rowID, tableID, databaseID, projectID, data, permissions, "", []string{"service"})
}

func (s *Service) UpdateRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Row, error) {
	if userID != "" {
		existingRow, err := s.GetRow(ctx, rowID, tableID, databaseID, projectID)
		if err != nil {
			return nil, err
		}
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "update", existingRow.Permissions); err != nil {
			return nil, err
		}
	}
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]interface{}{}
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		if key != "$permissions" && key != "id" && key != "created_at" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	assignments := make([]string, 0, len(keys)+1)
	args := make([]interface{}, 0, len(keys)+1)
	for i, key := range keys {
		assignments = append(assignments, fmt.Sprintf("%s = $%d", pgIdent(key), i+1))
		args = append(args, data[key])
	}
	assignments = append(assignments, "updated_at = NOW()")
	args = append(args, rowID)
	idN := len(args)

	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d",
		pgIdent(table.Name), strings.Join(assignments, ", "), idN)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("update row: %w", err)
	}
	rows, err := queryRowsAsJSON(ctx, tx, table.Name, rowID, 1, 0)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("update row: failed to fetch after update")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return mapToRow(rows[0], tableID, databaseID), nil
}

func (s *Service) DeleteRow(ctx context.Context, rowID, tableID, databaseID, projectID string) error {
	return s.DeleteRowWithAuth(ctx, rowID, tableID, databaseID, projectID, "", []string{"service"})
}

func (s *Service) DeleteRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID, userID string, roles []string) error {
	if userID != "" {
		existingRow, err := s.GetRow(ctx, rowID, tableID, databaseID, projectID)
		if err != nil {
			return err
		}
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "delete", existingRow.Permissions); err != nil {
			return err
		}
	}
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return err
	}
	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, false)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = $1", pgIdent(table.Name)), rowID); err != nil {
		return fmt.Errorf("delete row: %w", err)
	}
	return tx.Commit()
}

// TransactionOp defines a single operation in an atomic request.
type TransactionOp struct {
	Method  string                 `json:"method"`
	TableID string                 `json:"tableId"`
	RowID   string                 `json:"rowId"`
	Data    map[string]interface{} `json:"data"`
	Queries []Query                `json:"queries"`
}

// TransactionResult is the result of one operation.
type TransactionResult struct {
	Method string      `json:"method"`
	Status int         `json:"status"`
	Body   interface{} `json:"body"`
}

func (s *Service) ExecuteTransaction(ctx context.Context, projectID, databaseID string, operations []TransactionOp) ([]TransactionResult, error) {
	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, "", []string{"service"}, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	results := make([]TransactionResult, 0, len(operations))
	for _, operation := range operations {
		table, err := s.lookupTableContext(ctx, operation.TableID)
		if err != nil {
			return nil, err
		}
		result := TransactionResult{Method: operation.Method}
		switch strings.ToUpper(operation.Method) {
		case "POST", "CREATE":
			rowID := operation.RowID
			if rowID == "" {
				rowID = uid.New("")
			}
			data := map[string]interface{}{"id": rowID}
			for key, value := range operation.Data {
				data[key] = value
			}
			keys := make([]string, 0, len(data))
			for key := range data {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			placeholders := make([]string, 0, len(keys))
			args := make([]interface{}, 0, len(keys))
			idents := make([]string, 0, len(keys))
			for _, key := range keys {
				idents = append(idents, pgIdent(key))
				placeholders = append(placeholders, "?")
				args = append(args, data[key])
			}
			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", pgIdent(table.Name), strings.Join(idents, ", "), strings.Join(placeholders, ", "))
			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return nil, err
			}
			rows, err := queryRowsAsJSON(ctx, tx, table.Name, rowID, 1, 0)
			if err != nil || len(rows) == 0 {
				return nil, fmt.Errorf("transaction create fetch failed")
			}
			result.Status = http.StatusCreated
			result.Body = mapToRow(rows[0], operation.TableID, databaseID)
		case "PATCH", "UPDATE":
			keys := make([]string, 0, len(operation.Data))
			for key := range operation.Data {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			assignments := make([]string, 0, len(keys)+1)
			args := make([]interface{}, 0, len(keys)+1)
			for _, key := range keys {
				assignments = append(assignments, fmt.Sprintf("%s = ?", pgIdent(key)))
				args = append(args, operation.Data[key])
			}
			assignments = append(assignments, "updated_at = NOW()")
			args = append(args, operation.RowID)
			query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", pgIdent(table.Name), strings.Join(assignments, ", "))
			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return nil, err
			}
			rows, err := queryRowsAsJSON(ctx, tx, table.Name, operation.RowID, 1, 0)
			if err != nil || len(rows) == 0 {
				return nil, fmt.Errorf("transaction update fetch failed")
			}
			result.Status = http.StatusOK
			result.Body = mapToRow(rows[0], operation.TableID, databaseID)
		case "DELETE":
			if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", pgIdent(table.Name)), operation.RowID); err != nil {
				return nil, err
			}
			result.Status = http.StatusNoContent
		default:
			rows, err := queryRowsAsJSON(ctx, tx, table.Name, "", 25, 0)
			if err != nil {
				return nil, err
			}
			mapped := make([]*model.Row, 0, len(rows))
			for _, item := range rows {
				mapped = append(mapped, mapToRow(item, operation.TableID, databaseID))
			}
			result.Status = http.StatusOK
			result.Body = map[string]interface{}{"total": len(mapped), "rows": mapped}
		}
		results = append(results, result)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return results, nil
}

type SQLExecutionResult struct {
	Columns      []string                 `json:"columns,omitempty"`
	Rows         []map[string]interface{} `json:"rows,omitempty"`
	RowCount     int64                    `json:"rowCount"`
	ExecutionMs  int64                    `json:"executionMs"`
	Statement    string                   `json:"statement"`
	WriteAllowed bool                     `json:"writeAllowed"`
}

var blockedSQLPattern = regexp.MustCompile(`(?i)^\s*(create|alter|drop|truncate|comment|reindex|grant|revoke)\b`)
var querySQLPattern = regexp.MustCompile(`(?i)^\s*(select|with|show|explain|values)\b`)

func (s *Service) ExecuteSQL(ctx context.Context, projectID, databaseID, userID string, roles []string, statement string, writeAllowed bool) (*SQLExecutionResult, error) {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return nil, fmt.Errorf("sql statement is required")
	}
	if blockedSQLPattern.MatchString(statement) {
		return nil, fmt.Errorf("DDL statements are not allowed in the SQL editor")
	}
	start := time.Now()
	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, !writeAllowed)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	result := &SQLExecutionResult{Statement: statement, WriteAllowed: writeAllowed}
	if querySQLPattern.MatchString(statement) {
		rows, err := tx.QueryContext(ctx, statement)
		if err != nil {
			return nil, err
		}
		mapped, columns, err := scanJSONMaps(rows)
		if err != nil {
			return nil, err
		}
		result.Columns = columns
		result.Rows = mapped
		result.RowCount = int64(len(mapped))
	} else {
		res, err := tx.ExecContext(ctx, statement)
		if err != nil {
			return nil, err
		}
		affected, _ := res.RowsAffected()
		result.RowCount = affected
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	result.ExecutionMs = time.Since(start).Milliseconds()
	return result, nil
}

// CSVPreview contains the first few parsed rows for inspection.
type CSVPreview struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Total   int                      `json:"total"`
}

// CSVImportResult summarizes a CSV import.
type CSVImportResult struct {
	Imported int      `json:"imported"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

func (s *Service) PreviewCSV(_ context.Context, csvData []byte) (*CSVPreview, error) {
	reader := csv.NewReader(bytes.NewReader(csvData))
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}

	previewRows := make([]map[string]interface{}, 0, 5)
	for index := 0; index < 5; index++ {
		record, err := reader.Read()
		if err != nil {
			break
		}
		row := make(map[string]interface{}, len(headers))
		for columnIndex, header := range headers {
			if columnIndex < len(record) {
				row[header] = record[columnIndex]
			}
		}
		previewRows = append(previewRows, row)
	}

	return &CSVPreview{Columns: headers, Rows: previewRows, Total: len(previewRows)}, nil
}

func (s *Service) ImportCSV(ctx context.Context, projectID, databaseID, tableID string, csvData []byte, columnMapping map[string]string) (*CSVImportResult, error) {
	tableContext, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	columns, err := s.ListColumns(ctx, tableID)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(bytes.NewReader(csvData))
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}

	result := &CSVImportResult{Errors: []string{}}
	allowedColumns := map[string]struct{}{
		"id":         {},
		"created_at": {},
		"updated_at": {},
	}
	for _, column := range columns {
		allowedColumns[column.Key] = struct{}{}
	}

	selectedIndexes := make([]int, 0, len(headers))
	targetColumns := make([]string, 0, len(headers))
	seenColumns := make(map[string]struct{}, len(headers))
	for index, header := range headers {
		mapped := strings.TrimSpace(header)
		if value, ok := columnMapping[header]; ok {
			mapped = strings.TrimSpace(value)
		}
		if mapped == "" {
			continue
		}
		if _, ok := allowedColumns[mapped]; !ok {
			return &CSVImportResult{
				Failed: 1,
				Errors: []string{fmt.Sprintf("unknown column %q in CSV import", mapped)},
			}, nil
		}
		if _, exists := seenColumns[mapped]; exists {
			return &CSVImportResult{
				Failed: 1,
				Errors: []string{fmt.Sprintf("duplicate mapped column %q in CSV import", mapped)},
			}, nil
		}
		seenColumns[mapped] = struct{}{}
		selectedIndexes = append(selectedIndexes, index)
		targetColumns = append(targetColumns, mapped)
	}
	if len(targetColumns) == 0 {
		return &CSVImportResult{
			Failed: 1,
			Errors: []string{"CSV import did not match any table columns"},
		}, nil
	}

	var copyBuffer bytes.Buffer
	writer := csv.NewWriter(&copyBuffer)
	if err := writer.Write(targetColumns); err != nil {
		return nil, fmt.Errorf("prepare copy header: %w", err)
	}
	validRows := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		copyRow := make([]string, len(selectedIndexes))
		for targetIndex, sourceIndex := range selectedIndexes {
			if sourceIndex < len(record) {
				copyRow[targetIndex] = record[sourceIndex]
			}
		}
		if err := writer.Write(copyRow); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		validRows++
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("prepare copy data: %w", err)
	}
	if validRows == 0 {
		return result, nil
	}

	claims, err := json.Marshal(postgrestClaims{
		Role:      "applad_user",
		ProjectID: projectID,
		Roles:     normalizeRoles("", []string{"service"}),
	})
	if err != nil {
		return nil, err
	}

	conn, err := s.db.DB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire copy connection: %w", err)
	}
	defer conn.Close()

	err = conn.Raw(func(driverConn any) error {
		pgConn := driverConn.(*stdlib.Conn).Conn()
		tx, err := pgConn.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path = %s, public", pgIdent(tableContext.Schema))); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '30s'"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('applad.project_id', $1, true)", projectID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('applad.user_id', $1, true)", ""); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(claims)); err != nil {
			return err
		}

		copySQL := fmt.Sprintf(
			"COPY %s.%s (%s) FROM STDIN WITH (FORMAT csv, HEADER true)",
			pgIdent(tableContext.Schema),
			pgIdent(tableContext.Name),
			strings.Join(func() []string {
				quoted := make([]string, 0, len(targetColumns))
				for _, column := range targetColumns {
					quoted = append(quoted, pgIdent(column))
				}
				return quoted
			}(), ", "),
		)
		commandTag, err := tx.Conn().PgConn().CopyFrom(ctx, bytes.NewReader(copyBuffer.Bytes()), copySQL)
		if err != nil {
			return err
		}
		result.Imported += int(commandTag.RowsAffected())
		return tx.Commit(ctx)
	})
	if err != nil {
		result.Failed += validRows
		result.Errors = append(result.Errors, err.Error())
		return result, nil
	}

	return result, nil
}
