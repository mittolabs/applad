package transfer

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mittolabs/applad/internal/auth"
)

var (
	pgHostRe = regexp.MustCompile(`^[A-Za-z0-9.\-]+$`)
	pgPortRe = regexp.MustCompile(`^[0-9]+$`)
	pgDBRe   = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)
)

// buildPGDSN assembles a Postgres URL and validates the structural parts so a
// crafted host/port/database cannot inject extra DSN options (e.g. flipping
// sslmode). user and password are URL-escaped; the rest must be plain.
func buildPGDSN(host, port, user, password, database string) (string, error) {
	if !pgHostRe.MatchString(host) {
		return "", fmt.Errorf("invalid database host")
	}
	if !pgPortRe.MatchString(port) {
		return "", fmt.Errorf("invalid database port")
	}
	if !pgDBRe.MatchString(database) {
		return "", fmt.Errorf("invalid database name")
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		url.QueryEscape(user), url.QueryEscape(password), host, port, database), nil
}

// pgSource is the shared reader for Postgres-backed platforms (Supabase, NHost).
// It maps the tables in one schema (usually "public") to a single Applad
// database, and reads accounts through a platform-specific auth query. Storage
// is layered on by the concrete source (Supabase adds an object lister).
type pgSource struct {
	name       string
	db         *sql.DB
	dataSchema string // schema whose tables become the imported database
	// authSQL selects columns in this exact order:
	//   id, email, phone, name, password_hash, email_verified
	// Password hashes here are bcrypt for both Supabase and NHost.
	authSQL string
	// storage, when set, exports the platform's files after databases.
	storage func(ctx context.Context, emit Emit) (int, error)
}

func (s *pgSource) Name() string { return s.name }

func (s *pgSource) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// pgQuote double-quotes a Postgres identifier, escaping embedded quotes. Table
// and column names come from information_schema (real identifiers), but quoting
// keeps a name with a special character or reserved word safe in a SELECT.
func pgQuote(id string) string {
	return `"` + strings.ReplaceAll(id, `"`, `""`) + `"`
}

func (s *pgSource) Report(ctx context.Context, groups []Group) (map[Group]int, error) {
	out := map[Group]int{}
	for _, g := range groups {
		switch g {
		case GroupAuth:
			var n int
			if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM ("+s.authSQL+") AS _u").Scan(&n); err != nil {
				return nil, fmt.Errorf("%s: count users: %w", s.name, err)
			}
			out[GroupAuth] = n
		case GroupDatabases:
			tables, err := s.listTables(ctx)
			if err != nil {
				return nil, err
			}
			total := 1 // the schema maps to one database
			for _, t := range tables {
				cols, _ := s.listColumns(ctx, t)
				total += 1 + len(cols)
				var rc int
				if err := s.db.QueryRowContext(ctx,
					"SELECT count(*) FROM "+pgQuote(s.dataSchema)+"."+pgQuote(t)).Scan(&rc); err == nil {
					total += rc
				}
			}
			out[GroupDatabases] = total
		case GroupStorage:
			// Reported lazily as files stream; a precise count would list every
			// object up front. Left at 0 so the bar fills as files import.
			out[GroupStorage] = 0
		}
	}
	return out, nil
}

func (s *pgSource) Export(ctx context.Context, groups []Group, emit Emit) error {
	for _, g := range groups {
		var err error
		switch g {
		case GroupAuth:
			err = s.exportUsers(ctx, emit)
		case GroupDatabases:
			err = s.exportTables(ctx, emit)
		case GroupStorage:
			if s.storage != nil {
				_, err = s.storage(ctx, emit)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *pgSource) exportUsers(ctx context.Context, emit Emit) error {
	rows, err := s.db.QueryContext(ctx, s.authSQL)
	if err != nil {
		return fmt.Errorf("%s: query users: %w", s.name, err)
	}
	defer rows.Close()
	batch := make([]Resource, 0, 200)
	for rows.Next() {
		var id, email, phone, name, hash sql.NullString
		var verified sql.NullBool
		if err := rows.Scan(&id, &email, &phone, &name, &hash, &verified); err != nil {
			return err
		}
		batch = append(batch, User{
			ID:            id.String,
			Email:         email.String,
			Phone:         phone.String,
			Name:          name.String,
			PasswordHash:  hash.String,
			PasswordAlgo:  AlgoBcryptFor(hash.String), // Supabase/NHost store bcrypt
			EmailVerified: verified.Bool,
		})
		if len(batch) >= 200 {
			if err := emit(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		return emit(ctx, batch)
	}
	return nil
}

func (s *pgSource) exportTables(ctx context.Context, emit Emit) error {
	dbID := s.dataSchema
	if err := emit(ctx, []Resource{Database{ID: dbID, Name: dbID}}); err != nil {
		return err
	}
	tables, err := s.listTables(ctx)
	if err != nil {
		return err
	}
	for _, t := range tables {
		if err := emit(ctx, []Resource{Table{DatabaseID: dbID, ID: t, Name: t}}); err != nil {
			return err
		}
		cols, err := s.listColumns(ctx, t)
		if err != nil {
			return err
		}
		for _, c := range cols {
			if err := emit(ctx, []Resource{Column{
				DatabaseID: dbID, TableID: t, Key: c.name, Type: mapColumnType(c.dataType),
				Required: c.notNull, Array: c.isArray,
			}}); err != nil {
				return err
			}
		}
		if err := s.exportRows(ctx, dbID, t, cols, emit); err != nil {
			return err
		}
	}
	return nil
}

func (s *pgSource) exportRows(ctx context.Context, dbID, table string, cols []pgColumn, emit Emit) error {
	const page = 500
	offset := 0
	for {
		q := "SELECT * FROM " + pgQuote(s.dataSchema) + "." + pgQuote(table) +
			fmt.Sprintf(" LIMIT %d OFFSET %d", page, offset)
		rows, err := s.db.QueryContext(ctx, q)
		if err != nil {
			return fmt.Errorf("%s: read rows of %s: %w", s.name, table, err)
		}
		names, _ := rows.Columns()
		batch := make([]Resource, 0, page)
		n := 0
		for rows.Next() {
			cells := make([]any, len(names))
			ptrs := make([]any, len(names))
			for i := range cells {
				ptrs[i] = &cells[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return err
			}
			data := map[string]any{}
			var id string
			for i, name := range names {
				v := normalizeCell(cells[i])
				if name == "id" && v != nil {
					id = fmt.Sprintf("%v", v)
					continue // id becomes the row ID, not a data column
				}
				data[name] = v
			}
			batch = append(batch, Row{DatabaseID: dbID, TableID: table, ID: id, Data: data})
			n++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		if err := emit(ctx, batch); err != nil {
			return err
		}
		if n < page {
			return nil
		}
		offset += page
	}
}

type pgColumn struct {
	name     string
	dataType string
	notNull  bool
	isArray  bool
}

func (s *pgSource) listTables(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		 ORDER BY table_name`, s.dataSchema)
	if err != nil {
		return nil, fmt.Errorf("%s: list tables: %w", s.name, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *pgSource) listColumns(ctx context.Context, table string) ([]pgColumn, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT column_name, data_type, is_nullable
		 FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY ordinal_position`, s.dataSchema, table)
	if err != nil {
		return nil, fmt.Errorf("%s: list columns: %w", s.name, err)
	}
	defer rows.Close()
	var out []pgColumn
	for rows.Next() {
		var name, dataType, nullable string
		if err := rows.Scan(&name, &dataType, &nullable); err != nil {
			return nil, err
		}
		if name == "id" {
			continue // id is carried as the row identifier, not a column
		}
		out = append(out, pgColumn{
			name:     name,
			dataType: dataType,
			notNull:  nullable == "NO",
			isArray:  dataType == "ARRAY",
		})
	}
	return out, rows.Err()
}

// normalizeCell converts a scanned Postgres value into something JSON-friendly
// for a row's Data map: []byte becomes a string, everything else passes through.
func normalizeCell(v any) any {
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return v
	}
}

// AlgoBcryptFor returns the bcrypt algo id for a non-empty hash, or "" for an
// empty one (an account with no local password, e.g. OAuth-only).
func AlgoBcryptFor(hash string) string {
	if strings.TrimSpace(hash) == "" {
		return ""
	}
	return auth.AlgoBcrypt
}
