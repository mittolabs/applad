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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	appladcrypto "github.com/mittolabs/applad/internal/crypto"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/dek"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/realtime"
	"github.com/mittolabs/applad/internal/uid"
)

var safeSchemaSegment = regexp.MustCompile(`[^a-zA-Z0-9]`)

// Service handles database, table, column, and row operations.
type Service struct {
	db       *db.DB
	events   realtime.EventPublisher
	resolver RoleResolver
	dek      *dek.Service // per-project field-encryption keys; nil disables encrypted columns
}

// RoleResolver returns the extra RLS role tokens a user holds, beyond the
// built-ins (any, users, user:<id>) — for example team memberships as
// "team:<id>". It is resolved SERVER-SIDE from membership on every request and
// must never be sourced from anything the client sends: a client that could
// name its own roles could satisfy any policy. An implementation that errors
// should be treated as "no extra roles" (fail closed), never as a denial of the
// request itself.
type RoleResolver interface {
	RolesForUser(ctx context.Context, projectID, userID string) ([]string, error)
}

type sqlContextExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// NewService creates a new databases service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// SetEventPublisher wires realtime event publishing into the service.
func (s *Service) SetEventPublisher(pub realtime.EventPublisher) {
	s.events = pub
}

// SetRoleResolver wires server-side role resolution (e.g. team memberships) into
// RLS. Without it, only the built-in roles apply and group permissions like
// read("team:X") match no one — which is the safe default, not a silent open.
func (s *Service) SetRoleResolver(r RoleResolver) {
	s.resolver = r
}

// SetDEKService wires per-project field-encryption key management in. Without
// it, creating or writing to an "encrypted" column fails with dek.ErrDisabled
// rather than silently storing plaintext.
func (s *Service) SetDEKService(d *dek.Service) {
	s.dek = d
}

// resolveRoles fills in a caller's group roles when the caller did not pass an
// explicit set. Handlers pass nil so this authoritative, server-derived path
// runs; internal callers that already hold a vetted role list pass it through
// untouched. Errors fail closed to the built-in roles only.
func (s *Service) resolveRoles(ctx context.Context, projectID, userID string, roles []string) []string {
	if roles != nil || s.resolver == nil || userID == "" {
		return roles
	}
	resolved, err := s.resolver.RolesForUser(ctx, projectID, userID)
	if err != nil {
		return nil
	}
	return resolved
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

// sessionClaims are marshaled into the PostgreSQL session config for RLS evaluation.
type sessionClaims struct {
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
			// jsonb_exists rather than the ? operator: the query layer rewrites a
			// literal ? into a bind placeholder, which corrupts the policy SQL.
			clauses = append(clauses, fmt.Sprintf("jsonb_exists(COALESCE(NULLIF(current_setting('request.jwt.claims', true), '')::jsonb -> 'roles', '[]'::jsonb), %s)", pgLiteral(role)))
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

func toSQLType(columnType string, options map[string]interface{}, encrypted bool) string {
	// Ciphertext is an opaque base64 token, never a valid int/bool/timestamp/etc,
	// and always longer than its plaintext — so an encrypted column is always
	// TEXT regardless of logical type, ignoring any size option.
	if encrypted {
		return "TEXT"
	}
	switch columnType {
	case "string":
		if options != nil {
			var size int
			switch v := options["size"].(type) {
			case float64:
				size = int(v)
			case int:
				size = v
			case int64:
				size = int(v)
			}
			if size > 0 {
				return fmt.Sprintf("VARCHAR(%d)", size)
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
	case "richtext", "media":
		// Editorial field types (content mode): rich text body and a media
		// reference (storage file id or URL). Both are text at rest.
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
	schema := schemaName(projectID, id)
	roleAnon := schema + "_anon"
	roleAuth := schema + "_authenticated"

	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", pgIdent(schema))); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	// Create per-database roles for RLS evaluation.
	// applad_user (the connection role) can SET ROLE to these for policy enforcement.
	// A silent failure here means every later query runs as the pooled owner
	// with RLS bypassed, so these must not be best-effort.
	for _, role := range []string{roleAnon, roleAuth} {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf(
			`DO $$ BEGIN
				IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = %s) THEN
					CREATE ROLE %s NOLOGIN;
				END IF;
			END $$`,
			pgLiteral(role), pgIdent(role),
		)); err != nil {
			return nil, fmt.Errorf("create rls role %s: %w", role, err)
		}
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(
		"GRANT USAGE ON SCHEMA %s TO %s, %s",
		pgIdent(schema), pgIdent(roleAnon), pgIdent(roleAuth),
	)); err != nil {
		return nil, fmt.Errorf("grant schema usage to rls roles: %w", err)
	}

	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO databases (id, project_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
		id, projectID, name, now, now,
	); err != nil {
		return nil, fmt.Errorf("create database metadata: %w", err)
	}

	return &model.Database{ID: id, Name: name, Enabled: true, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Service) GetDatabase(ctx context.Context, databaseID, projectID string) (*model.Database, error) {
	var database model.Database
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, name, created_at, updated_at FROM databases WHERE id = $1 AND project_id = $2",
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
		"SELECT id, name, created_at, updated_at FROM databases WHERE project_id = $1 ORDER BY created_at DESC",
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
		"UPDATE databases SET name = $1, updated_at = $2 WHERE id = $3 AND project_id = $4",
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
		"DELETE FROM databases WHERE id = $1 AND project_id = $2",
		databaseID, projectID,
	); err != nil {
		return err
	}
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

	s.db.ExecContext(ctx, fmt.Sprintf( //nolint:errcheck
		"CREATE TRIGGER set_updated_at BEFORE UPDATE ON %s.%s FOR EACH ROW EXECUTE FUNCTION applad_set_updated_at()",
		pgIdent(schema), pgIdent(name),
	))
	s.db.ExecContext(ctx, fmt.Sprintf( //nolint:errcheck
		"CREATE TRIGGER notify_changes AFTER INSERT OR UPDATE OR DELETE ON %s.%s FOR EACH ROW EXECUTE FUNCTION applad_notify_change()",
		pgIdent(schema), pgIdent(name),
	))

	// Grant table access to the per-database RLS roles so policies can evaluate.
	roleAnon := schema + "_anon"
	roleAuth := schema + "_authenticated"
	s.db.ExecContext(ctx, fmt.Sprintf( //nolint:errcheck
		"GRANT SELECT, INSERT, UPDATE, DELETE ON %s.%s TO %s, %s",
		pgIdent(schema), pgIdent(name), pgIdent(roleAnon), pgIdent(roleAuth),
	))

	if rowSecurity {
		s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY", pgIdent(schema), pgIdent(name))) //nolint:errcheck
		s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s.%s FORCE ROW LEVEL SECURITY", pgIdent(schema), pgIdent(name)))  //nolint:errcheck
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO tables (id, database_id, project_id, name, enabled, row_security, permissions, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, TRUE, $5, $6, $7, $8)`,
		id, databaseID, projectID, name, rowSecurity, permissionsJSON, now, now,
	); err != nil {
		return nil, fmt.Errorf("insert table metadata: %w", err)
	}

	// Wire the permissions the caller passed at creation into the metadata that
	// actually drives enforcement. This used to be dropped: the strings were
	// stored on the tables row but never parsed into the permissions table the
	// RLS policies and the create/read checks read from, so a table created with
	// read("team:X") granted nothing. SetPermissions also syncs the policies.
	parsedPerms, err := parsePermissionStrings(permissions)
	if err != nil {
		return nil, err
	}
	// SetPermissions re-syncs the RLS policies for the table, so a row-security
	// table gets its policies here whether or not any table-level grant was given
	// (the per-row clause still applies).
	if err := s.SetPermissions(ctx, projectID, "table", id, parsedPerms); err != nil {
		return nil, fmt.Errorf("apply table permissions: %w", err)
	}

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
		`SELECT id, database_id, name, enabled, row_security, content_enabled, COALESCE(permissions, '[]'), created_at, updated_at
		 FROM tables WHERE id = $1 AND database_id = $2 AND project_id = $3`,
		tableID, databaseID, projectID,
	).Scan(&table.ID, &table.DatabaseID, &table.Name, &table.Enabled, &table.RowSecurity, &table.ContentEnabled, &permissionsJSON, &table.CreatedAt, &table.UpdatedAt); err != nil {
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
		`SELECT id, database_id, name, enabled, row_security, content_enabled, COALESCE(permissions, '[]'), created_at, updated_at
		 FROM tables WHERE database_id = $1 AND project_id = $2 ORDER BY created_at DESC`,
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
		if err := rows.Scan(&table.ID, &table.DatabaseID, &table.Name, &table.Enabled, &table.RowSecurity, &table.ContentEnabled, &permissionsJSON, &table.CreatedAt, &table.UpdatedAt); err != nil {
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

func (s *Service) UpdateTable(ctx context.Context, tableID, databaseID, projectID, name string, permissions []string, enabled *bool, rowSecurity *bool) (*model.Table, error) {
	// Load current table to preserve unchanged fields.
	current, err := s.GetTable(ctx, tableID, databaseID, projectID)
	if err != nil {
		return nil, err
	}
	newEnabled := current.Enabled
	if enabled != nil {
		newEnabled = *enabled
	}
	newRowSecurity := current.RowSecurity
	if rowSecurity != nil {
		newRowSecurity = *rowSecurity
	}
	newName := current.Name
	if name != "" {
		newName = name
	}
	if len(permissions) == 0 {
		permissions = current.Permissions
	}
	permissionsJSON, _ := json.Marshal(permissions)
	if _, err := s.db.ExecContext(ctx,
		"UPDATE tables SET name = $1, enabled = $2, row_security = $3, permissions = $4, updated_at = $5 WHERE id = $6 AND database_id = $7 AND project_id = $8",
		newName, newEnabled, newRowSecurity, permissionsJSON, time.Now().UTC(), tableID, databaseID, projectID,
	); err != nil {
		return nil, err
	}
	if rowSecurity != nil && *rowSecurity != current.RowSecurity {
		tableCtx, err := s.lookupProjectTable(ctx, tableID, projectID)
		if err == nil {
			if *rowSecurity {
				s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY", pgIdent(tableCtx.Schema), pgIdent(tableCtx.Name))) //nolint:errcheck
				s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s.%s FORCE ROW LEVEL SECURITY", pgIdent(tableCtx.Schema), pgIdent(tableCtx.Name)))  //nolint:errcheck
				s.syncRLSPolicies(ctx, tableCtx)                                                                                                    //nolint:errcheck
			} else {
				s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s.%s DISABLE ROW LEVEL SECURITY", pgIdent(tableCtx.Schema), pgIdent(tableCtx.Name))) //nolint:errcheck
			}
		}
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
	if _, err := s.db.ExecContext(ctx, "DELETE FROM tables WHERE id = $1 AND project_id = $2", tableID, projectID); err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateColumn(ctx context.Context, projectID, tableID, key, columnType string, required, array, encrypted bool, defaultValue interface{}, options map[string]interface{}, validation *model.ColumnValidation) (*model.Column, error) {
	if array && encrypted {
		return nil, fmt.Errorf("encrypted array columns are not supported; store a single encrypted JSON value instead")
	}

	tableContext, err := s.lookupProjectTable(ctx, tableID, projectID)
	if err != nil {
		return nil, err
	}

	if encrypted {
		if s.dek == nil {
			return nil, dek.ErrDisabled
		}
		if err := s.dek.EnsureProjectKey(ctx, projectID); err != nil {
			return nil, err
		}
	}

	sqlType := toSQLType(columnType, options, encrypted)
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
	validationJSON := []byte("{}")
	if validation != nil {
		if b, err := json.Marshal(validation); err == nil {
			validationJSON = b
		}
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO columns (id, table_id, key_name, type, required, "array", default_value, options, validation, permissions, status, encrypted, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '["read","write"]', 'available', $10, $11)`,
		uid.New("unique()"), tableID, key, columnType, required, array, defaultJSON, optionsJSON, validationJSON, encrypted, time.Now().UTC(),
	); err != nil {
		return nil, fmt.Errorf("insert column metadata: %w", err)
	}
	if err := s.syncRLSPolicies(ctx, tableContext); err != nil {
		return nil, err
	}

	return &model.Column{Key: key, Type: columnType, Status: "available", Required: required, Array: array, Default: defaultValue, Options: options, Validation: validation, Permissions: []string{"read", "write"}, Encrypted: encrypted}, nil
}

func (s *Service) ListColumns(ctx context.Context, tableID string) ([]model.Column, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key_name, type, status, required, "array", default_value, COALESCE(options, '{}'), COALESCE(validation, '{}'), COALESCE(permissions, '["read","write"]'), encrypted
		 FROM columns WHERE table_id = $1 ORDER BY created_at ASC`,
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
		var validationJSON []byte
		var permsJSON []byte
		if err := rows.Scan(&column.Key, &column.Type, &column.Status, &column.Required, &column.Array, &defaultJSON, &optionsJSON, &validationJSON, &permsJSON, &column.Encrypted); err != nil {
			return nil, err
		}
		if len(defaultJSON) > 0 {
			json.Unmarshal(defaultJSON, &column.Default) //nolint:errcheck
		}
		if len(optionsJSON) > 0 {
			json.Unmarshal(optionsJSON, &column.Options) //nolint:errcheck
		}
		if len(validationJSON) > 2 { // skip empty {}
			var v model.ColumnValidation
			if json.Unmarshal(validationJSON, &v) == nil {
				column.Validation = &v
			}
		}
		if len(permsJSON) > 0 {
			json.Unmarshal(permsJSON, &column.Permissions) //nolint:errcheck
		}
		if column.Permissions == nil {
			column.Permissions = []string{"read", "write"}
		}
		columns = append(columns, column)
	}
	if columns == nil {
		columns = []model.Column{}
	}
	return columns, nil
}

// GetColumnPermissions returns the permissions for a specific column.
func (s *Service) GetColumnPermissions(ctx context.Context, tableID, key string) ([]string, error) {
	var permsJSON []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(permissions, '["read","write"]') FROM columns WHERE table_id = $1 AND key_name = $2`,
		tableID, key,
	).Scan(&permsJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("column not found")
	}
	if err != nil {
		return nil, err
	}
	var perms []string
	json.Unmarshal(permsJSON, &perms) //nolint:errcheck
	if perms == nil {
		perms = []string{"read", "write"}
	}
	return perms, nil
}

// SetColumnPermissions updates the permissions for a specific column.
func (s *Service) SetColumnPermissions(ctx context.Context, tableID, key string, permissions []string) error {
	permsJSON, _ := json.Marshal(permissions)
	res, err := s.db.ExecContext(ctx,
		`UPDATE columns SET permissions = $1 WHERE table_id = $2 AND key_name = $3`,
		permsJSON, tableID, key,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("column not found")
	}
	return nil
}

// readableColumns returns a set of column keys that have "read" permission.
// Columns with no explicit permissions default to allowing read.
func (s *Service) readableColumns(ctx context.Context, tableID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key_name, COALESCE(permissions, '["read","write"]') FROM columns WHERE table_id = $1`,
		tableID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var key string
		var permsJSON []byte
		if err := rows.Scan(&key, &permsJSON); err != nil {
			continue
		}
		var perms []string
		json.Unmarshal(permsJSON, &perms) //nolint:errcheck
		readable := false
		for _, p := range perms {
			if p == "read" {
				readable = true
				break
			}
		}
		result[key] = readable
	}
	return result, nil
}

// writableColumns returns a set of column keys that have "write" permission.
func (s *Service) writableColumns(ctx context.Context, tableID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key_name, COALESCE(permissions, '["read","write"]') FROM columns WHERE table_id = $1`,
		tableID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var key string
		var permsJSON []byte
		if err := rows.Scan(&key, &permsJSON); err != nil {
			continue
		}
		var perms []string
		json.Unmarshal(permsJSON, &perms) //nolint:errcheck
		writable := false
		for _, p := range perms {
			if p == "write" {
				writable = true
				break
			}
		}
		result[key] = writable
	}
	return result, nil
}

// encryptedColumns returns the set of column keys flagged encrypted for a
// table (present and true only — absent means not encrypted). Used by row
// CRUD to know which values need to be sealed/opened, and to reject
// filter/sort/search against ciphertext.
func (s *Service) encryptedColumns(ctx context.Context, tableID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key_name, encrypted FROM columns WHERE table_id = $1`,
		tableID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var key string
		var encrypted bool
		if err := rows.Scan(&key, &encrypted); err != nil {
			continue
		}
		if encrypted {
			result[key] = true
		}
	}
	return result, nil
}

// encryptRowData replaces each encCols-flagged, non-nil value in data with an
// AES-256-GCM-sealed token under the project's field-encryption key, so the
// generic INSERT/UPDATE builders that follow treat it as an opaque string
// destined for a TEXT column. A nil value is left as SQL NULL rather than
// encrypting a JSON "null", so a NOT NULL/required check on the raw column
// behaves the same as it would unencrypted.
func (s *Service) encryptRowData(ctx context.Context, projectID string, data map[string]interface{}, encCols map[string]bool) error {
	if len(encCols) == 0 {
		return nil
	}
	var dekKey []byte
	var dekVersion int
	for key, val := range data {
		if !encCols[key] || val == nil {
			continue
		}
		if dekKey == nil {
			if s.dek == nil {
				return dek.ErrDisabled
			}
			var err error
			dekKey, dekVersion, err = s.dek.Unwrap(ctx, projectID)
			if err != nil {
				return fmt.Errorf("unwrap project encryption key: %w", err)
			}
		}
		plaintext, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("encrypt field %q: %w", key, err)
		}
		token, err := appladcrypto.SealToken("fe", dekVersion, dekKey, plaintext)
		if err != nil {
			return fmt.Errorf("encrypt field %q: %w", key, err)
		}
		data[key] = token
	}
	return nil
}

// decryptFieldValue reverses encryptRowData for a single stored value. The
// DEK version is read from the token itself, so this works for ciphertext
// written under a since-rotated-away project key.
func (s *Service) decryptFieldValue(ctx context.Context, projectID string, value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	token, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("encrypted column value has unexpected type %T", value)
	}
	if s.dek == nil {
		return nil, dek.ErrDisabled
	}
	plaintext, _, err := appladcrypto.OpenToken("fe", func(version int) ([]byte, error) {
		return s.dek.UnwrapVersion(ctx, projectID, version)
	}, token)
	if err != nil {
		return nil, fmt.Errorf("decrypt field: %w", err)
	}
	var out interface{}
	if err := json.Unmarshal(plaintext, &out); err != nil {
		return nil, fmt.Errorf("decrypt field: unmarshal: %w", err)
	}
	return out, nil
}

// rejectEncryptedFilterOrSort refuses a query that would filter, order, or
// search on an encrypted column: the stored value is opaque ciphertext, so
// such an operation would either error confusingly deep in Postgres or —
// worse — silently "succeed" while returning wrong results (e.g. an ILIKE
// that can never match). isNull/isNotNull are exempt: NULL-ness survives
// encryption unchanged, since encryptRowData never encrypts a nil value.
func rejectEncryptedFilterOrSort(queries []Query, orderAttr string, encCols map[string]bool) error {
	if len(encCols) == 0 {
		return nil
	}
	for _, q := range queries {
		if !encCols[q.Field] {
			continue
		}
		if q.Method == "isNull" || q.Method == "isNotNull" {
			continue
		}
		return fmt.Errorf("field %q is encrypted and cannot be filtered, sorted, or searched (opaque ciphertext)", q.Field)
	}
	if orderAttr != "" && encCols[orderAttr] {
		return fmt.Errorf("field %q is encrypted and cannot be used for ordering", orderAttr)
	}
	return nil
}

// applyReadFilter strips non-readable columns from row data.
// If readCols is nil or empty, no filtering is applied.
func applyReadFilter(rows []*model.Row, readCols map[string]bool) []*model.Row {
	if len(readCols) == 0 {
		return rows
	}
	for _, row := range rows {
		if row.Data == nil {
			continue
		}
		for k := range row.Data {
			if allowed, exists := readCols[k]; exists && !allowed {
				delete(row.Data, k)
			}
		}
	}
	return rows
}

// applyWriteFilter strips non-writable columns from data map.
func applyWriteFilter(data map[string]interface{}, writeCols map[string]bool) map[string]interface{} {
	if len(writeCols) == 0 {
		return data
	}
	result := make(map[string]interface{}, len(data))
	for k, v := range data {
		if allowed, exists := writeCols[k]; !exists || allowed {
			result[k] = v
		}
	}
	return result
}

func (s *Service) DeleteColumn(ctx context.Context, projectID, tableID, key string) error {
	tableContext, err := s.lookupProjectTable(ctx, tableID, projectID)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s", pgIdent(tableContext.Schema), pgIdent(tableContext.Name), pgIdent(key)),
	); err != nil {
		return fmt.Errorf("drop column: %w", err)
	}
	s.db.ExecContext(ctx, "DELETE FROM columns WHERE table_id = $1 AND key_name = $2", tableID, key) //nolint:errcheck
	if err := s.syncRLSPolicies(ctx, tableContext); err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateIndex(ctx context.Context, projectID, tableID, key, indexType string, columns, orders []string) (*model.Index, error) {
	tableContext, err := s.lookupProjectTable(ctx, tableID, projectID)
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
		 VALUES ($1, $2, $3, $4, $5, $6, 'available', $7)`,
		uid.New("unique()"), tableID, key, indexType, columnsJSON, ordersJSON, time.Now().UTC(),
	); err != nil {
		return nil, fmt.Errorf("insert index metadata: %w", err)
	}

	return &model.Index{Key: key, Type: indexType, Status: "available", Columns: columns, Orders: orders}, nil
}

func (s *Service) ListIndexes(ctx context.Context, tableID string) ([]model.Index, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT key_name, type, status, columns, orders FROM indexes WHERE table_id = $1 ORDER BY created_at ASC",
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

func (s *Service) DeleteIndex(ctx context.Context, projectID, tableID, key string) error {
	tableContext, err := s.lookupProjectTable(ctx, tableID, projectID)
	if err != nil {
		return err
	}
	s.db.ExecContext(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s.%s", pgIdent(tableContext.Schema), pgIdent(key))) //nolint:errcheck
	s.db.ExecContext(ctx, "DELETE FROM indexes WHERE table_id = $1 AND key_name = $2", tableID, key)             //nolint:errcheck
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

func (s *Service) CreateRelationship(ctx context.Context, projectID, tableID, relatedTableID, relationType, key, twoWayKey, onDelete string, twoWay bool) (*Relationship, error) {
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

	// Both ends must be the caller's own tables: a cross-tenant FK would both
	// alter the victim's table and create a covert read channel.
	leftTable, err := s.lookupProjectTable(ctx, tableID, projectID)
	if err != nil {
		return nil, err
	}
	rightTable, err := s.lookupProjectTable(ctx, relatedTableID, projectID)
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
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, tableID, relatedTableID, relationType, twoWay, key, twoWayKey, onDelete,
	); err != nil {
		return nil, fmt.Errorf("create relationship metadata: %w", err)
	}

	return &Relationship{ID: id, TableID: tableID, RelatedTable: relatedTableID, Type: relationType, TwoWay: twoWay, Key: key, TwoWayKey: twoWayKey, OnDelete: onDelete}, nil
}

func (s *Service) ListRelationships(ctx context.Context, tableID string) ([]Relationship, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, table_id, related_table, relationship_type, two_way, key_name, two_way_key, on_delete
		 FROM table_relationships WHERE table_id = $1`,
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

func (s *Service) DeleteRelationship(ctx context.Context, projectID, tableID, relationshipID string) error {
	leftTable, err := s.lookupProjectTable(ctx, tableID, projectID)
	if err != nil {
		return err
	}
	fkName := "fk_" + relationshipID
	ddl := fmt.Sprintf(`ALTER TABLE %s.%s DROP CONSTRAINT IF EXISTS %s`,
		pgIdent(leftTable.Schema), pgIdent(leftTable.Name), pgIdent(fkName))
	s.db.ExecContext(ctx, ddl) //nolint:errcheck

	_, err = s.db.ExecContext(ctx, "DELETE FROM table_relationships WHERE id = $1 AND table_id = $2", relationshipID, tableID)
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
	// A counter everyone may move without being able to rewrite the row it
	// sits on. Liking a post bumps its like count; it must not also grant the
	// ability to edit the caption, which is what granting "update" would do.
	"increment": true,
}

func (s *Service) checkPermission(ctx context.Context, projectID, resourceType, resourceID string, roles []string, action string) (bool, error) {
	if len(roles) == 0 {
		return false, nil
	}
	placeholders := make([]string, len(roles))
	args := make([]interface{}, 0, len(roles)+4)
	args = append(args, projectID, resourceType, resourceID, action)
	for index, role := range roles {
		placeholders[index] = fmt.Sprintf("$%d", index+5)
		args = append(args, role)
	}
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM permissions WHERE project_id = $1 AND resource_type = $2 AND resource_id = $3 AND action = $4 AND role IN (%s)",
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
		"DELETE FROM permissions WHERE project_id = $1 AND resource_type = $2 AND resource_id = $3",
		projectID, resourceType, resourceID,
	); err != nil {
		return fmt.Errorf("delete permissions: %w", err)
	}
	for _, permission := range permissions {
		if !validActions[permission.Action] {
			return fmt.Errorf("invalid permission action %q", permission.Action)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO permissions (id, project_id, resource_type, resource_id, role, action) VALUES ($1, $2, $3, $4, $5, $6)",
			uid.New("unique()"), projectID, resourceType, resourceID, permission.Role, permission.Action,
		); err != nil {
			return fmt.Errorf("insert permission: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if resourceType == "table" {
		tableContext, err := s.lookupProjectTable(ctx, resourceID, projectID)
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
		"SELECT role, action FROM permissions WHERE project_id = $1 AND resource_type = $2 AND resource_id = $3 ORDER BY created_at ASC",
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
			// The role may be quoted, e.g. delete("user:A"); compare the bare role.
			role = strings.Trim(role, `"'`)
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
		"SELECT row_security FROM tables WHERE id = $1 AND project_id = $2",
		tableID, projectID,
	).Scan(&rowSecurity); err != nil {
		return fmt.Errorf("permission denied")
	}
	if rowSecurity && len(rowPermissions) > 0 && checkRowPermission(rowPermissions, allRoles, action) {
		return nil
	}
	return fmt.Errorf("permission denied")
}

// AuthorizeTableRead resolves a caller's realtime read access to one table by
// name, mirroring enforcePermission's read decision: a table-level read grant
// admits every row (AllowAll); otherwise, a document-security table admits only
// rows whose own _permissions grant read to the caller's roles (RowFiltered),
// which the realtime layer then applies per event. Roles are resolved
// server-side from the caller's identity — never from anything the client sent.
// It implements realtime.ReadAuthorizer. Errors (including a missing table) fail
// closed to a deny at the call site.
func (s *Service) AuthorizeTableRead(ctx context.Context, projectID, databaseID, tableName, userID string) (realtime.TableReadDecision, error) {
	groupRoles := s.resolveRoles(ctx, projectID, userID, nil)
	allRoles := buildRoles(userID, groupRoles)

	var tableID string
	var rowSecurity bool
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, row_security FROM tables WHERE project_id = $1 AND database_id = $2 AND name = $3",
		projectID, databaseID, tableName,
	).Scan(&tableID, &rowSecurity); err != nil {
		if err == sql.ErrNoRows {
			return realtime.TableReadDecision{}, fmt.Errorf("table not found")
		}
		return realtime.TableReadDecision{}, err
	}

	allowed, err := s.checkPermission(ctx, projectID, "table", tableID, allRoles, "read")
	if err != nil {
		return realtime.TableReadDecision{}, err
	}
	if allowed {
		return realtime.TableReadDecision{AllowAll: true}, nil
	}
	if rowSecurity {
		return realtime.TableReadDecision{RowFiltered: true, Roles: allRoles}, nil
	}
	return realtime.TableReadDecision{}, nil
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

type tableContext struct {
	ID          string
	DatabaseID  string
	ProjectID   string
	Name        string
	Schema      string
	RowSecurity bool
}

// rowPermColumn is the hidden column that holds a row's own permissions on a
// document-security table, as {"read":[...],"update":[...],"delete":[...]}. It
// is what makes read("team:X") on a single row mean something: the RLS policies
// consult it per row. The leading underscore keeps it clear of user columns.
const rowPermColumn = "_permissions"

// rowPermExpression is the per-row half of an RLS policy for one action: the row
// is admitted when any role its own permissions grant that action to is among
// the caller's resolved roles. Validated against Postgres before shipping.
func rowPermExpression(action string) string {
	// jsonb_exists rather than the ? operator: the query layer rewrites a literal
	// ? into a bind placeholder, which corrupts the policy SQL.
	return fmt.Sprintf(
		`EXISTS (SELECT 1 FROM jsonb_array_elements_text(COALESCE(%s -> %s, '[]'::jsonb)) AS _grant_role `+
			`WHERE jsonb_exists(COALESCE(NULLIF(current_setting('request.jwt.claims', true), '')::jsonb -> 'roles', '[]'::jsonb), _grant_role))`,
		pgIdent(rowPermColumn), pgLiteral(action))
}

// combinePolicyExprs ORs the non-trivial expressions. An empty or FALSE clause
// contributes nothing; if all are empty the result is empty and the caller skips
// the policy (nobody is granted, which is the safe default).
func combinePolicyExprs(exprs ...string) string {
	kept := make([]string, 0, len(exprs))
	for _, e := range exprs {
		if e != "" && e != "FALSE" {
			kept = append(kept, e)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	if len(kept) == 1 {
		return kept[0]
	}
	return "(" + strings.Join(kept, " OR ") + ")"
}

// parsePermissionString turns `read("team:X")` into ("read", "team:X"). "write"
// is accepted as a shorthand the caller expands. Anything malformed is rejected.
func parsePermissionString(s string) (action, role string, ok bool) {
	s = strings.TrimSpace(s)
	open := strings.Index(s, "(")
	if open <= 0 || !strings.HasSuffix(s, ")") {
		return "", "", false
	}
	action = strings.TrimSpace(s[:open])
	role = strings.TrimSpace(s[open+1 : len(s)-1])
	role = strings.Trim(role, `"'`)
	if action == "" || role == "" {
		return "", "", false
	}
	return action, role, true
}

// parsePermissionStrings expands a list of permission strings into metadata
// Permissions. "write" becomes update + delete, matching the shorthand clients
// use. Unknown actions are rejected so a typo cannot silently grant nothing.
func parsePermissionStrings(perms []string) ([]Permission, error) {
	out := make([]Permission, 0, len(perms))
	for _, p := range perms {
		action, role, ok := parsePermissionString(p)
		if !ok {
			return nil, fmt.Errorf("invalid permission %q", p)
		}
		actions := []string{action}
		if action == "write" {
			actions = []string{"update", "delete"}
		}
		for _, a := range actions {
			if !validActions[a] {
				return nil, fmt.Errorf("invalid permission action %q", a)
			}
			out = append(out, Permission{Action: a, Role: role})
		}
	}
	return out, nil
}

// rowPermissionsJSON builds the normalised {"read":[...],...} a document-security
// row stores, from the permission strings a client sends. Create is table-level,
// so it is ignored here; read/update/delete/increment are what a row can carry.
func rowPermissionsJSON(perms []string) ([]byte, error) {
	parsed, err := parsePermissionStrings(perms)
	if err != nil {
		return nil, err
	}
	grouped := map[string][]string{}
	for _, p := range parsed {
		if p.Action == "create" {
			continue
		}
		grouped[p.Action] = append(grouped[p.Action], p.Role)
	}
	return json.Marshal(grouped)
}

// rowPermissionsToStrings turns a stored {"read":[...]} object back into the
// ["read(\"team:X\")", ...] form the API returns.
func rowPermissionsToStrings(v interface{}) []string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	var out []string
	for action, roles := range m {
		items, ok := roles.([]interface{})
		if !ok {
			continue
		}
		for _, r := range items {
			out = append(out, fmt.Sprintf(`%s(%q)`, action, fmt.Sprintf("%v", r)))
		}
	}
	sort.Strings(out)
	return out
}

func (s *Service) lookupTableContext(ctx context.Context, tableID string) (*tableContext, error) {
	var table tableContext
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, database_id, project_id, name, row_security FROM tables WHERE id = $1",
		tableID,
	).Scan(&table.ID, &table.DatabaseID, &table.ProjectID, &table.Name, &table.RowSecurity); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("table not found")
		}
		return nil, err
	}
	table.Schema = schemaName(table.ProjectID, table.DatabaseID)
	return &table, nil
}

// lookupProjectTable resolves a table only within the caller's project. The
// project predicate is the tenant boundary: resolving by ID alone derives the
// schema from the table's OWN project, letting any authenticated project aim
// DDL at another tenant's table. All request-driven paths must use this.
func (s *Service) lookupProjectTable(ctx context.Context, tableID, projectID string) (*tableContext, error) {
	var table tableContext
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, database_id, project_id, name FROM tables WHERE id = $1 AND project_id = $2",
		tableID, projectID,
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
		"SELECT key_name FROM columns WHERE table_id = $1 AND key_name IN ($2, $3) ORDER BY CASE key_name WHEN 'owner_id' THEN 0 ELSE 1 END LIMIT 1",
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
		"SELECT row_security FROM tables WHERE id = $1 AND project_id = $2",
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

	// The per-row permissions column backs document-level security. Added here
	// (idempotently) so enabling row security on an existing table, or re-syncing
	// one created before this existed, gains it without a migration.
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(
		"ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s JSONB NOT NULL DEFAULT '{}'::jsonb",
		pgIdent(table.Schema), pgIdent(table.Name), pgIdent(rowPermColumn),
	)); err != nil {
		return fmt.Errorf("add row permissions column: %w", err)
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
		// Table-level grant OR the row's own grant. Create is table-level only:
		// a row cannot pre-authorise its own insertion. Read/update/delete also
		// consult the row's permissions, which is what makes read("team:X") on a
		// single document mean something.
		tableExpr := policyRoleExpression(grouped[policy.Action])
		expr := tableExpr
		if policy.Action != "create" {
			expr = combinePolicyExprs(tableExpr, rowPermExpression(policy.Action))
		}
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

func (s *Service) applySessionContext(ctx context.Context, exec sqlContextExecutor, projectID, databaseID, userID string, roles []string) error {
	schema := schemaName(projectID, databaseID)
	pgRole := schema + "_authenticated"
	if userID == "" {
		pgRole = schema + "_anon"
	}

	claims, err := json.Marshal(sessionClaims{
		Role:      pgRole,
		ProjectID: projectID,
		UserID:    userID,
		Roles:     normalizeRoles(userID, roles),
	})
	if err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, "SELECT set_config('applad.project_id', $1, true)", projectID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, "SELECT set_config('applad.database_id', $1, true)", databaseID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, "SELECT set_config('applad.user_id', $1, true)", userID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(claims)); err != nil {
		return err
	}
	// SET LOCAL ROLE enforces RLS policies via the per-database role. It must
	// be a plain statement on this tx: inside a DO block PostgreSQL reverts
	// SET LOCAL at block exit, so the query after it would run as the pooled
	// owner with RLS bypassed. Missing role or failed SET is a hard error for
	// the same reason.
	var roleExists bool
	if err := exec.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", pgRole).Scan(&roleExists); err != nil {
		return fmt.Errorf("check rls role: %w", err)
	}
	if !roleExists {
		return fmt.Errorf("rls role %q does not exist; recreate the database to provision it", pgRole)
	}
	if _, err := exec.ExecContext(ctx, "SET LOCAL ROLE "+pgIdent(pgRole)); err != nil {
		return fmt.Errorf("set local role: %w", err)
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
	if err := s.applySessionContext(ctx, tx, projectID, databaseID, userID, roles); err != nil {
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
		rows, err := exec.QueryContext(ctx, fmt.Sprintf("SELECT to_jsonb(t) FROM %s AS t WHERE id = $1 LIMIT 1", pgIdent(tableName)), rowID)
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
	rows, err := exec.QueryContext(ctx, fmt.Sprintf("SELECT to_jsonb(t) FROM %s AS t ORDER BY created_at DESC LIMIT $1 OFFSET $2", pgIdent(tableName)), limit, offset)
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

// mapToRow assembles a model.Row from a raw to_jsonb-decoded map. When encCols
// is non-empty, flagged columns are decrypted here — the single choke point
// every read path (Create/Update's post-write re-fetch, Get, List, and the
// atomic-op RETURNING paths) funnels through, so this is the only place row
// data needs to know about encryption at all. A decrypt failure fails the
// whole call rather than returning partial or garbage data.
func (s *Service) mapToRow(ctx context.Context, data map[string]interface{}, tableID, databaseID, projectID string, encCols map[string]bool) (*model.Row, error) {
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
		case rowPermColumn:
			// The document-security permissions column, stored as
			// {"read":[...],...}; surface it as the row's permission strings and
			// keep it out of the row's data.
			row.Permissions = append(row.Permissions, rowPermissionsToStrings(value)...)
		default:
			if encCols[key] {
				decoded, err := s.decryptFieldValue(ctx, projectID, value)
				if err != nil {
					return nil, fmt.Errorf("row %v: %w", data["id"], err)
				}
				row.Data[key] = decoded
			} else {
				row.Data[key] = value
			}
		}
	}
	return row, nil
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
		case "between":
			// The client sends between("field","min","max"); the handler parses the
			// two bounds into a []interface{}. Emit an inclusive SQL BETWEEN with a
			// bound param for each end. A malformed value list is skipped rather than
			// silently degrading to an unbounded match.
			lo, hi, ok := betweenBounds(q.Values)
			if !ok {
				continue
			}
			conditions = append(conditions, fmt.Sprintf("%s BETWEEN $%d AND $%d", field, n, n+1))
			args = append(args, lo, hi)
			n += 2
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

// betweenBounds extracts the two bounds a between filter carries. The handler
// parses between("field","min","max") into a []interface{}{min, max}; anything
// else is malformed and reported so the caller can skip the condition.
func betweenBounds(v interface{}) (lo, hi interface{}, ok bool) {
	vals, isSlice := v.([]interface{})
	if !isSlice || len(vals) < 2 {
		return nil, nil, false
	}
	return vals[0], vals[1], true
}

// parseSelectColumns turns a select() projection string like "id, title, body"
// into the bare column names to return. Related-column expansion (author(name))
// is a separate, unimplemented feature: any token naming a relation is dropped
// here so it neither projects a missing column nor errors the query.
func parseSelectColumns(sel string) []string {
	if strings.TrimSpace(sel) == "" {
		return nil
	}
	parts := strings.Split(sel, ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		c := strings.TrimSpace(p)
		if c == "" || strings.ContainsAny(c, "()") {
			continue
		}
		cols = append(cols, c)
	}
	return cols
}

// selectProjection builds the SELECT expression that returns only the requested
// columns as a jsonb object, always including the system columns so the row can
// still be assembled. _permissions is included only for a document-security
// table, since the column does not exist otherwise and would error the query.
func selectProjection(cols []string, rowSecurity bool) string {
	ordered := make([]string, 0, len(cols)+4)
	seen := map[string]bool{}
	add := func(c string) {
		if c == "" || seen[c] {
			return
		}
		seen[c] = true
		ordered = append(ordered, c)
	}
	for _, c := range cols {
		add(c)
	}
	add("id")
	add("created_at")
	add("updated_at")
	if rowSecurity {
		add(rowPermColumn)
	}
	pairs := make([]string, 0, len(ordered))
	for _, c := range ordered {
		pairs = append(pairs, fmt.Sprintf("%s, t.%s", pgLiteral(c), pgIdent(c)))
	}
	return "jsonb_build_object(" + strings.Join(pairs, ", ") + ")"
}

// ValidationErr is returned when row data fails column validation rules.
type ValidationErr struct {
	Field   string
	Rule    string
	Message string
}

func (e *ValidationErr) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

// validateRowData checks each column's validation rules against the provided data.
// Returns the first validation error found, or nil if all rules pass.
func (s *Service) validateRowData(ctx context.Context, tableID string, data map[string]interface{}) *ValidationErr {
	columns, err := s.ListColumns(ctx, tableID)
	if err != nil {
		return nil // don't block on metadata errors
	}
	for _, col := range columns {
		if col.Validation == nil {
			continue
		}
		v := col.Validation
		val, exists := data[col.Key]
		if !exists {
			continue
		}
		customMsg := v.Message

		switch tv := val.(type) {
		case string:
			ruleMsg := func(rule, dflt string) string {
				if customMsg != "" {
					return customMsg
				}
				return dflt
			}
			if v.MinLength != nil && len(tv) < *v.MinLength {
				return &ValidationErr{Field: col.Key, Rule: "minLength", Message: ruleMsg("minLength", fmt.Sprintf("%s must be at least %d characters", col.Key, *v.MinLength))}
			}
			if v.MaxLength != nil && len(tv) > *v.MaxLength {
				return &ValidationErr{Field: col.Key, Rule: "maxLength", Message: ruleMsg("maxLength", fmt.Sprintf("%s must be at most %d characters", col.Key, *v.MaxLength))}
			}
			if v.Pattern != "" {
				re, err := regexp.Compile(v.Pattern)
				if err == nil && !re.MatchString(tv) {
					return &ValidationErr{Field: col.Key, Rule: "pattern", Message: ruleMsg("pattern", fmt.Sprintf("%s does not match required pattern", col.Key))}
				}
			}
		case float64:
			ruleMsg := func(rule, dflt string) string {
				if customMsg != "" {
					return customMsg
				}
				return dflt
			}
			if v.Min != nil && tv < *v.Min {
				return &ValidationErr{Field: col.Key, Rule: "min", Message: ruleMsg("min", fmt.Sprintf("%s must be at least %g", col.Key, *v.Min))}
			}
			if v.Max != nil && tv > *v.Max {
				return &ValidationErr{Field: col.Key, Rule: "max", Message: ruleMsg("max", fmt.Sprintf("%s must be at most %g", col.Key, *v.Max))}
			}
		}
	}
	return nil
}

func (s *Service) CreateRow(ctx context.Context, projectID, databaseID, tableID, rowID string, data map[string]interface{}, permissions []string) (*model.Row, error) {
	return s.CreateRowWithAuth(ctx, projectID, databaseID, tableID, rowID, data, permissions, "", []string{"service"})
}

func (s *Service) CreateRowWithAuth(ctx context.Context, projectID, databaseID, tableID, rowID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Row, error) {
	roles = s.resolveRoles(ctx, projectID, userID, roles)
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

	// Enforce column validation rules before writing.
	if verr := s.validateRowData(ctx, tableID, data); verr != nil {
		return nil, verr
	}

	// Apply column-level write permissions.
	if writeCols, err := s.writableColumns(ctx, tableID); err == nil {
		data = applyWriteFilter(data, writeCols)
	}

	encCols, err := s.encryptedColumns(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("load encrypted columns: %w", err)
	}
	if err := s.encryptRowData(ctx, projectID, data, encCols); err != nil {
		return nil, err
	}

	// Translate the "unique()" sentinel (and any empty/invalid id) into a fresh
	// id. Only checking for "" left the literal string "unique()" as the row id,
	// so the first row took it and every subsequent insert collided on the
	// primary key — a second message in a channel always failed.
	rowID = uid.New(rowID)
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

	// A document-security table carries each row's own permissions. Persist them
	// so read/update/delete RLS can consult them; on tables without row security
	// there is no such column and per-row permissions do not apply.
	if table.RowSecurity && len(permissions) > 0 {
		permJSON, perr := rowPermissionsJSON(permissions)
		if perr != nil {
			return nil, perr
		}
		idents = append(idents, pgIdent(rowPermColumn))
		placeholders = append(placeholders, fmt.Sprintf("$%d::jsonb", len(args)+1))
		args = append(args, string(permJSON))
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
	return s.mapToRow(ctx, rows[0], tableID, databaseID, projectID, encCols)
}

func (s *Service) GetRow(ctx context.Context, rowID, tableID, databaseID, projectID string) (*model.Row, error) {
	return s.GetRowWithAuth(ctx, rowID, tableID, databaseID, projectID, "", []string{"service"})
}

func (s *Service) GetRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID, userID string, roles []string) (*model.Row, error) {
	roles = s.resolveRoles(ctx, projectID, userID, roles)
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
	encCols, err := s.encryptedColumns(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("load encrypted columns: %w", err)
	}
	row, err := s.mapToRow(ctx, rows[0], tableID, databaseID, projectID, encCols)
	if err != nil {
		return nil, err
	}
	if userID != "" {
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "read", row.Permissions); err != nil {
			return nil, err
		}
	}

	// Apply column-level read permissions.
	if readCols, err := s.readableColumns(ctx, tableID); err == nil {
		applyReadFilter([]*model.Row{row}, readCols)
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
	roles = s.resolveRoles(ctx, projectID, userID, roles)
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, 0, err
	}

	// A client caller (a user session, not a server API key) must hold table-level
	// read to list a table, exactly as Get/Create enforce it. The list path used to
	// lean entirely on RLS, but RLS is only enabled for document-security tables:
	// a table with row security OFF keeps an unconditional GRANT SELECT, so without
	// this check a user lacking table-level read would be handed every row while a
	// single-row Get was correctly denied. For a document-security table RLS filters
	// per row, so the blanket gate is skipped — a caller with only row-level read
	// must still list the rows they are permitted to read, matching how Get admits a
	// row via its own permissions. Server API keys (userID == "") keep full access.
	if userID != "" && !table.RowSecurity {
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "read", nil); err != nil {
			return nil, 0, err
		}
	}

	if params.Limit <= 0 {
		params.Limit = 25
	}

	encCols, err := s.encryptedColumns(ctx, tableID)
	if err != nil {
		return nil, 0, fmt.Errorf("load encrypted columns: %w", err)
	}
	if err := rejectEncryptedFilterOrSort(params.Queries, params.OrderAttr, encCols); err != nil {
		return nil, 0, err
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

	// Resolve the ordering column and direction once; both offset and keyset
	// pagination order by the same thing. Default is newest-first by created_at.
	orderCol, dir := "created_at", "DESC"
	if params.OrderAttr != "" {
		orderCol = params.OrderAttr
		dir = "ASC"
		if strings.EqualFold(params.OrderType, "desc") {
			dir = "DESC"
		}
	}

	// The data query starts from the same WHERE the count used, then grows a
	// keyset predicate and paging args of its own.
	dataArgs := append([]interface{}{}, whereArgs...)
	dataWhere := whereClause
	useKeyset := params.CursorAfter != ""

	orderClause := "ORDER BY created_at DESC"
	if useKeyset {
		// Keyset pagination needs a total order, so tie-break on id in the same
		// direction as the primary sort. The cursor row is located by id and the
		// page is everything ordered after it: > for ascending, < for descending.
		orderClause = fmt.Sprintf("ORDER BY %s %s, id %s", pgIdent(orderCol), dir, dir)
		cmp := ">"
		if dir == "DESC" {
			cmp = "<"
		}
		cursorN := len(dataArgs) + 1
		cond := fmt.Sprintf("(%s, id) %s ((SELECT %s FROM %s WHERE id = $%d), $%d)",
			pgIdent(orderCol), cmp, pgIdent(orderCol), pgIdent(table.Name), cursorN, cursorN)
		if dataWhere == "" {
			dataWhere = "WHERE " + cond
		} else {
			dataWhere = dataWhere + " AND " + cond
		}
		dataArgs = append(dataArgs, params.CursorAfter)
	} else if params.OrderAttr != "" {
		orderClause = fmt.Sprintf("ORDER BY %s %s", pgIdent(orderCol), dir)
	}

	// Project only the requested columns when select() was used; otherwise the
	// whole row. System columns are always kept so the row can be assembled.
	projection := "to_jsonb(t)"
	if cols := parseSelectColumns(params.Select); len(cols) > 0 {
		projection = selectProjection(cols, table.RowSecurity)
	}

	// Append the paging args after the WHERE/keyset args. Keyset paging walks
	// from the cursor and does not offset; offset paging is left unchanged.
	limitN := len(dataArgs) + 1
	dataArgs = append(dataArgs, params.Limit)
	offsetClause := ""
	if !useKeyset {
		offsetClause = fmt.Sprintf("OFFSET $%d", len(dataArgs)+1)
		dataArgs = append(dataArgs, params.Offset)
	}
	dataQuery := fmt.Sprintf("SELECT %s FROM %s AS t %s %s LIMIT $%d %s",
		projection, pgIdent(table.Name), dataWhere, orderClause, limitN, offsetClause)

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
		mapped, err := s.mapToRow(ctx, item, tableID, databaseID, projectID, encCols)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, mapped)
	}
	if err := sqlRows.Err(); err != nil {
		return nil, 0, err
	}

	// Apply column-level read permissions.
	if readCols, err := s.readableColumns(ctx, tableID); err == nil {
		result = applyReadFilter(result, readCols)
	}

	return result, total, nil
}

func (s *Service) UpdateRow(ctx context.Context, rowID, tableID, databaseID, projectID string, data map[string]interface{}, permissions []string) (*model.Row, error) {
	return s.UpdateRowWithAuth(ctx, rowID, tableID, databaseID, projectID, data, permissions, "", []string{"service"})
}

func (s *Service) UpdateRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID string, data map[string]interface{}, permissions []string, userID string, roles []string) (*model.Row, error) {
	roles = s.resolveRoles(ctx, projectID, userID, roles)
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	if userID != "" {
		perms, exists, err := s.existingRowPermissions(ctx, table, rowID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("row not found")
		}
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "update", perms); err != nil {
			return nil, err
		}
	}
	if data == nil {
		data = map[string]interface{}{}
	}

	// Enforce column validation rules before writing.
	if verr := s.validateRowData(ctx, tableID, data); verr != nil {
		return nil, verr
	}

	// Apply column-level write permissions.
	if writeCols, err := s.writableColumns(ctx, tableID); err == nil {
		data = applyWriteFilter(data, writeCols)
	}

	encCols, err := s.encryptedColumns(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("load encrypted columns: %w", err)
	}
	if err := s.encryptRowData(ctx, projectID, data, encCols); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		if key != "$permissions" && key != "id" && key != "created_at" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	assignments := make([]string, 0, len(keys)+2)
	args := make([]interface{}, 0, len(keys)+2)
	for i, key := range keys {
		assignments = append(assignments, fmt.Sprintf("%s = $%d", pgIdent(key), i+1))
		args = append(args, data[key])
	}
	// Let a row's own permissions be changed on a document-security table.
	if table.RowSecurity && len(permissions) > 0 {
		permJSON, perr := rowPermissionsJSON(permissions)
		if perr != nil {
			return nil, perr
		}
		assignments = append(assignments, fmt.Sprintf("%s = $%d::jsonb", pgIdent(rowPermColumn), len(args)+1))
		args = append(args, string(permJSON))
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
	return s.mapToRow(ctx, rows[0], tableID, databaseID, projectID, encCols)
}

func (s *Service) DeleteRow(ctx context.Context, rowID, tableID, databaseID, projectID string) error {
	return s.DeleteRowWithAuth(ctx, rowID, tableID, databaseID, projectID, "", []string{"service"})
}

// existingRowPermissions reads a row's own permission strings (and whether it
// exists) WITHOUT RLS, for the pre-write permission check. It must bypass RLS on
// purpose: on a document-security table the caller may lack read access to a row
// they are nonetheless trying to update or delete, and we still need to consult
// that row's permissions to decide. It runs on the pooled connection, which is
// the schema owner and not the sandboxed per-request role, so no policy applies.
func (s *Service) existingRowPermissions(ctx context.Context, table *tableContext, rowID string) (perms []string, exists bool, err error) {
	if table.RowSecurity {
		var raw []byte
		err = s.db.QueryRowContext(ctx, fmt.Sprintf(
			"SELECT %s FROM %s.%s WHERE id = $1", pgIdent(rowPermColumn), pgIdent(table.Schema), pgIdent(table.Name),
		), rowID).Scan(&raw)
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		var m map[string]interface{}
		_ = json.Unmarshal(raw, &m)
		return rowPermissionsToStrings(m), true, nil
	}
	var one int
	err = s.db.QueryRowContext(ctx, fmt.Sprintf(
		"SELECT 1 FROM %s.%s WHERE id = $1", pgIdent(table.Schema), pgIdent(table.Name),
	), rowID).Scan(&one)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return nil, true, nil
}

func (s *Service) DeleteRowWithAuth(ctx context.Context, rowID, tableID, databaseID, projectID, userID string, roles []string) error {
	roles = s.resolveRoles(ctx, projectID, userID, roles)
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return err
	}
	if userID != "" {
		perms, exists, err := s.existingRowPermissions(ctx, table, rowID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("row not found")
		}
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "delete", perms); err != nil {
			return err
		}
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

// authorizeRowUpdate resolves a table (enforcing the project boundary the way
// lookupProjectTable does) and, for a user session (userID != ""), checks the
// row's update permission exactly as UpdateRowWithAuth. It is the shared auth
// prelude for the single-statement atomic writes below. A server API key
// (userID == "") skips the app-level check and keeps full access.
func (s *Service) authorizeRowUpdate(ctx context.Context, projectID, tableID, rowID, userID string, roles []string) (*tableContext, error) {
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	if table.ProjectID != projectID {
		return nil, fmt.Errorf("row not found")
	}
	if userID != "" {
		perms, exists, err := s.existingRowPermissions(ctx, table, rowID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("row not found")
		}
		if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "update", perms); err != nil {
			return nil, err
		}
	}
	return table, nil
}

// authorizeRowIncrement resolves access to an atomic numeric operation.
//
// Append deliberately still requires "update": adding an arbitrary value to an
// array is not the same narrow, bounded act as moving a counter, and there is
// no case yet that wants it opened up.
//
// These used to require "update", which conflated two very different grants: a
// social app cannot let anyone like a post without also letting them rewrite
// it. A row may now carry increment("users") on its own. "update" still
// implies it — being allowed to rewrite a row obviously includes moving a
// number in it — so nothing that worked before stops working.
func (s *Service) authorizeRowIncrement(ctx context.Context, projectID, tableID, rowID, userID string, roles []string) (*tableContext, error) {
	table, err := s.lookupTableContext(ctx, tableID)
	if err != nil {
		return nil, err
	}
	if table.ProjectID != projectID {
		return nil, fmt.Errorf("row not found")
	}
	if userID == "" {
		return table, nil
	}
	perms, exists, err := s.existingRowPermissions(ctx, table, rowID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("row not found")
	}
	if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "increment", perms); err == nil {
		return table, nil
	}
	if err := s.enforcePermission(ctx, projectID, tableID, userID, roles, "update", perms); err != nil {
		return nil, err
	}
	return table, nil
}

// numericBounds returns the min/max bounds configured on a column, or nil when
// unset, so an atomic increment can clamp its result within the same statement.
func (s *Service) numericBounds(ctx context.Context, tableID, field string) (min, max *float64) {
	var raw []byte
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(validation, '{}') FROM columns WHERE table_id = $1 AND key_name = $2`,
		tableID, field,
	).Scan(&raw); err != nil {
		return nil, nil
	}
	var v model.ColumnValidation
	if json.Unmarshal(raw, &v) != nil {
		return nil, nil
	}
	return v.Min, v.Max
}

// AtomicNumericOp adds delta to a numeric field in a single UPDATE statement, so
// concurrent increments cannot lose each other's writes the way the previous
// read-in-one-tx then write-in-another pair did. The row's update permission is
// enforced first for a user session and the write runs under the caller's RLS
// role; a server API key keeps full access. Any min/max bound configured on the
// column clamps the result within the same statement.
func (s *Service) AtomicNumericOp(ctx context.Context, projectID, databaseID, tableID, rowID, field string, delta float64, userID string, roles []string) (*model.Row, error) {
	roles = s.resolveRoles(ctx, projectID, userID, roles)
	table, err := s.authorizeRowIncrement(ctx, projectID, tableID, rowID, userID, roles)
	if err != nil {
		return nil, err
	}

	// Bounds come from trusted column metadata but are still bound as parameters
	// rather than interpolated into the statement.
	args := []interface{}{delta}
	valueExpr := fmt.Sprintf("COALESCE(%s, 0) + $1", pgIdent(field))
	if lower, upper := s.numericBounds(ctx, tableID, field); lower != nil || upper != nil {
		if lower != nil {
			args = append(args, *lower)
			valueExpr = fmt.Sprintf("GREATEST(%s, $%d)", valueExpr, len(args))
		}
		if upper != nil {
			args = append(args, *upper)
			valueExpr = fmt.Sprintf("LEAST(%s, $%d)", valueExpr, len(args))
		}
	}
	args = append(args, rowID)

	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	query := fmt.Sprintf(
		"UPDATE %s SET %s = %s, updated_at = NOW() WHERE id = $%d RETURNING to_jsonb(%s.*)",
		pgIdent(table.Name), pgIdent(field), valueExpr, len(args), pgIdent(table.Name))
	row, err := scanReturnedRow(ctx, tx, query, args...)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	encCols, err := s.encryptedColumns(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("load encrypted columns: %w", err)
	}
	return s.mapToRow(ctx, row, tableID, databaseID, projectID, encCols)
}

// AtomicArrayAppend appends value to an array field in a single UPDATE, so
// concurrent appends cannot clobber each other. array_append(NULL, v) yields
// {v}, so a not-yet-set column becomes a single-element array. It enforces the
// row's update permission for a user session, like AtomicNumericOp.
func (s *Service) AtomicArrayAppend(ctx context.Context, projectID, databaseID, tableID, rowID, field string, value interface{}, userID string, roles []string) (*model.Row, error) {
	roles = s.resolveRoles(ctx, projectID, userID, roles)
	table, err := s.authorizeRowUpdate(ctx, projectID, tableID, rowID, userID, roles)
	if err != nil {
		return nil, err
	}

	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	query := fmt.Sprintf(
		"UPDATE %s SET %s = array_append(%s, $1), updated_at = NOW() WHERE id = $2 RETURNING to_jsonb(%s.*)",
		pgIdent(table.Name), pgIdent(field), pgIdent(field), pgIdent(table.Name))
	row, err := scanReturnedRow(ctx, tx, query, value, rowID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	encCols, err := s.encryptedColumns(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("load encrypted columns: %w", err)
	}
	return s.mapToRow(ctx, row, tableID, databaseID, projectID, encCols)
}

// scanReturnedRow runs an UPDATE ... RETURNING to_jsonb(t.*) and decodes the one
// row it returns. No row (RLS filtered it, or the id is gone) is "row not found".
func scanReturnedRow(ctx context.Context, tx *db.Tx, query string, args ...interface{}) (map[string]interface{}, error) {
	var raw []byte
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("row not found")
		}
		return nil, fmt.Errorf("atomic update: %w", err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("atomic update: %w", err)
	}
	return data, nil
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

// txOpAction maps a transaction operation's method to the permission action it
// requires, matching how the single-row handlers gate each verb.
func txOpAction(method string) string {
	switch strings.ToUpper(method) {
	case "POST", "CREATE":
		return "create"
	case "PATCH", "UPDATE":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "read"
	}
}

func (s *Service) ExecuteTransaction(ctx context.Context, projectID, databaseID, userID string, roles []string, operations []TransactionOp) ([]TransactionResult, error) {
	// Roles are resolved server-side from the caller's identity, never taken from
	// the request. A user session (userID != "") runs under the _authenticated
	// RLS role and is permission-checked per op below; a server API key
	// (userID == "") keeps full access, exactly as the single-row WithAuth paths.
	roles = s.resolveRoles(ctx, projectID, userID, roles)
	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	results := make([]TransactionResult, 0, len(operations))
	for _, operation := range operations {
		// lookupTableContext is resolved by id, so enforce the project boundary
		// here — the same tenant guard lookupProjectTable applies — while also
		// getting the row-security flag existingRowPermissions needs.
		table, err := s.lookupTableContext(ctx, operation.TableID)
		if err != nil {
			return nil, err
		}
		if table.ProjectID != projectID {
			return nil, fmt.Errorf("table not found")
		}
		// Enforce per-op permission for a user session before touching the table.
		// Non-row-security tables carry unconditional DML grants, so without this
		// a user could read/modify/delete every row via a transaction op. A user
		// lacking permission on any op fails the whole transaction (the tx is
		// rolled back by the deferred Rollback), all-or-nothing. A server API key
		// (userID == "") keeps full access.
		if userID != "" {
			action := txOpAction(operation.Method)
			switch action {
			case "update", "delete":
				// Consult the row's own permissions, exactly as the single-row
				// WithAuth methods do, so a document-security row grant still admits
				// the write.
				perms, exists, perr := s.existingRowPermissions(ctx, table, operation.RowID)
				if perr != nil {
					return nil, perr
				}
				if !exists {
					return nil, fmt.Errorf("row not found")
				}
				if err := s.enforcePermission(ctx, projectID, operation.TableID, userID, roles, action, perms); err != nil {
					return nil, err
				}
			case "read":
				// A read/list op on a document-security table is filtered per row by
				// RLS, so skip the blanket table-level gate there — matching
				// ListRowsWithAuth. On a non-RLS table there is no RLS, so table-level
				// read is required.
				if !table.RowSecurity {
					if err := s.enforcePermission(ctx, projectID, operation.TableID, userID, roles, action, nil); err != nil {
						return nil, err
					}
				}
			default: // create
				if err := s.enforcePermission(ctx, projectID, operation.TableID, userID, roles, action, nil); err != nil {
					return nil, err
				}
			}
		}
		result := TransactionResult{Method: operation.Method}
		switch strings.ToUpper(operation.Method) {
		case "POST", "CREATE":
			rowID := uid.New(operation.RowID) // "unique()"/empty/invalid -> fresh id
			data := map[string]interface{}{"id": rowID}
			for key, value := range operation.Data {
				data[key] = value
			}
			encCols, err := s.encryptedColumns(ctx, operation.TableID)
			if err != nil {
				return nil, fmt.Errorf("load encrypted columns: %w", err)
			}
			if err := s.encryptRowData(ctx, projectID, data, encCols); err != nil {
				return nil, err
			}
			keys := make([]string, 0, len(data))
			for key := range data {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			placeholders := make([]string, 0, len(keys))
			args := make([]interface{}, 0, len(keys))
			idents := make([]string, 0, len(keys))
			for i, key := range keys {
				idents = append(idents, pgIdent(key))
				placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
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
			result.Body, err = s.mapToRow(ctx, rows[0], operation.TableID, databaseID, projectID, encCols)
			if err != nil {
				return nil, err
			}
		case "PATCH", "UPDATE":
			encCols, err := s.encryptedColumns(ctx, operation.TableID)
			if err != nil {
				return nil, fmt.Errorf("load encrypted columns: %w", err)
			}
			if err := s.encryptRowData(ctx, projectID, operation.Data, encCols); err != nil {
				return nil, err
			}
			keys := make([]string, 0, len(operation.Data))
			for key := range operation.Data {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			assignments := make([]string, 0, len(keys)+1)
			args := make([]interface{}, 0, len(keys)+1)
			for i, key := range keys {
				assignments = append(assignments, fmt.Sprintf("%s = $%d", pgIdent(key), i+1))
				args = append(args, operation.Data[key])
			}
			assignments = append(assignments, "updated_at = NOW()")
			args = append(args, operation.RowID)
			query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", pgIdent(table.Name), strings.Join(assignments, ", "), len(keys)+1)
			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return nil, err
			}
			rows, err := queryRowsAsJSON(ctx, tx, table.Name, operation.RowID, 1, 0)
			if err != nil || len(rows) == 0 {
				return nil, fmt.Errorf("transaction update fetch failed")
			}
			result.Status = http.StatusOK
			result.Body, err = s.mapToRow(ctx, rows[0], operation.TableID, databaseID, projectID, encCols)
			if err != nil {
				return nil, err
			}
		case "DELETE":
			if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = $1", pgIdent(table.Name)), operation.RowID); err != nil {
				return nil, err
			}
			result.Status = http.StatusNoContent
		default:
			rows, err := queryRowsAsJSON(ctx, tx, table.Name, "", 25, 0)
			if err != nil {
				return nil, err
			}
			encCols, err := s.encryptedColumns(ctx, operation.TableID)
			if err != nil {
				return nil, fmt.Errorf("load encrypted columns: %w", err)
			}
			mapped := make([]*model.Row, 0, len(rows))
			for _, item := range rows {
				row, err := s.mapToRow(ctx, item, operation.TableID, databaseID, projectID, encCols)
				if err != nil {
					return nil, err
				}
				mapped = append(mapped, row)
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

// sessionControlPattern blocks SET/RESET: SET ROLE / RESET ROLE / RESET ALL
// would drop or restore the SET LOCAL ROLE sandbox mid-transaction.
var sessionControlPattern = regexp.MustCompile(`(?i)^\s*(set|reset)\b`)

// stripLeadingSQLComments removes leading whitespace and SQL comments so the
// keyword guards can't be dodged with a comment prefix ("/* */ DROP ...").
func stripLeadingSQLComments(statement string) string {
	for {
		statement = strings.TrimSpace(statement)
		switch {
		case strings.HasPrefix(statement, "--"):
			if idx := strings.IndexByte(statement, '\n'); idx >= 0 {
				statement = statement[idx+1:]
				continue
			}
			return ""
		case strings.HasPrefix(statement, "/*"):
			depth := 0
			i := 0
			for i < len(statement) {
				if strings.HasPrefix(statement[i:], "/*") {
					depth++
					i += 2
				} else if strings.HasPrefix(statement[i:], "*/") {
					depth--
					i += 2
					if depth == 0 {
						break
					}
				} else {
					i++
				}
			}
			if depth != 0 {
				return "" // unterminated comment: nothing executable follows
			}
			statement = statement[i:]
		default:
			return statement
		}
	}
}

// hasMultipleStatements reports whether statement contains a ';' separator
// outside string literals, quoted identifiers, dollar quotes, and comments,
// with anything after it. A lone trailing ';' is fine. Backslashes are NOT
// treated as escapes: doing so would hide a real separator in
// standard-conforming strings, so E'...' strings can only false-positive.
func hasMultipleStatements(statement string) bool {
	i := 0
	for i < len(statement) {
		c := statement[i]
		switch {
		case c == '\'': // string literal, '' escapes
			i++
			for i < len(statement) {
				if statement[i] == '\'' {
					if i+1 < len(statement) && statement[i+1] == '\'' {
						i += 2
						continue
					}
					break
				}
				i++
			}
			i++
		case c == '"': // quoted identifier, "" escapes
			i++
			for i < len(statement) {
				if statement[i] == '"' {
					if i+1 < len(statement) && statement[i+1] == '"' {
						i += 2
						continue
					}
					break
				}
				i++
			}
			i++
		case c == '$': // dollar-quoted string $tag$...$tag$
			j := i + 1
			for j < len(statement) && (statement[j] == '_' || isAlnumByte(statement[j])) {
				j++
			}
			if j < len(statement) && statement[j] == '$' {
				tag := statement[i : j+1]
				end := strings.Index(statement[j+1:], tag)
				if end < 0 {
					return false // unterminated: nothing executable follows
				}
				i = j + 1 + end + len(tag)
			} else {
				i++
			}
		case c == '-' && i+1 < len(statement) && statement[i+1] == '-':
			nl := strings.IndexByte(statement[i:], '\n')
			if nl < 0 {
				return false
			}
			i += nl + 1
		case c == '/' && i+1 < len(statement) && statement[i+1] == '*':
			depth := 1
			i += 2
			for i < len(statement) && depth > 0 {
				if strings.HasPrefix(statement[i:], "/*") {
					depth++
					i += 2
				} else if strings.HasPrefix(statement[i:], "*/") {
					depth--
					i += 2
				} else {
					i++
				}
			}
		case c == ';':
			// Separator only if executable content follows.
			return stripLeadingSQLComments(statement[i+1:]) != ""
		default:
			i++
		}
	}
	return false
}

func isAlnumByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func (s *Service) ExecuteSQL(ctx context.Context, projectID, databaseID, userID string, roles []string, statement string, writeAllowed bool) (*SQLExecutionResult, error) {
	roles = s.resolveRoles(ctx, projectID, userID, roles)
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return nil, fmt.Errorf("sql statement is required")
	}
	// One statement per request: a second statement after ';' would dodge the
	// prefix-anchored guards below, and only the pgx driver happens to reject
	// multi-statement strings today.
	if hasMultipleStatements(statement) {
		return nil, fmt.Errorf("multiple SQL statements are not allowed; execute one statement at a time")
	}
	head := stripLeadingSQLComments(statement)
	if blockedSQLPattern.MatchString(head) {
		return nil, fmt.Errorf("DDL statements are not allowed in the SQL editor")
	}
	if sessionControlPattern.MatchString(head) {
		return nil, fmt.Errorf("SET/RESET statements are not allowed in the SQL editor")
	}
	start := time.Now()
	tx, _, err := s.prepareDirectAccessTx(ctx, projectID, databaseID, userID, roles, !writeAllowed)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	result := &SQLExecutionResult{Statement: statement, WriteAllowed: writeAllowed}
	if querySQLPattern.MatchString(head) {
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
	tableContext, err := s.lookupProjectTable(ctx, tableID, projectID)
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

	claims, err := json.Marshal(sessionClaims{
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
