package db

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrateAdvisoryLock is a stable arbitrary integer used as the PostgreSQL
// advisory lock key for serialising concurrent migration runs.
// Value chosen to be unlikely to collide with application advisory locks.
const migrateAdvisoryLock = 7_369_327_832

// Migrate runs all pending SQL migrations from the embedded migrations directory.
// It acquires a PostgreSQL session-level advisory lock so that only one API
// instance runs migrations at a time (safe for multi-replica Kubernetes rollouts).
func (db *DB) Migrate() error {
	// Acquire advisory lock — blocks until previous migrator releases it.
	if _, err := db.Exec("SELECT pg_advisory_lock($1)", migrateAdvisoryLock); err != nil {
		return fmt.Errorf("migrate: acquire lock: %w", err)
	}
	defer db.Exec("SELECT pg_advisory_unlock($1)", migrateAdvisoryLock) //nolint:errcheck

	// Bootstrap the tracking table before anything else so the version check
	// below never hits a "table doesn't exist" error on a fresh database.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    VARCHAR(128) NOT NULL PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("migrate: bootstrap: %w", err)
	}
	// Widen an existing ledger. The column started at 32 characters, which a
	// descriptive migration filename outgrew — and the failure landed on
	// recording the migration, after its statements had already run, which is
	// the worst place to discover a limit like this.
	if _, err := db.Exec(
		"ALTER TABLE schema_migrations ALTER COLUMN version TYPE VARCHAR(128)"); err != nil {
		return fmt.Errorf("migrate: widen ledger: %w", err)
	}

	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("migrate: glob: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		version := filepath.Base(f)
		var count int
		_ = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if count > 0 {
			continue
		}
		content, err := migrationsFS.ReadFile(f)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", f, err)
		}
		for _, stmt := range splitStatements(string(content)) {
			if _, err := db.Exec(stmt); err != nil {
				if shouldIgnoreMigrationError(stmt, err) {
					continue
				}
				return fmt.Errorf("migrate: exec %s: %w\nSQL: %s", version, err, stmt)
			}
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("migrate: record %s: %w", version, err)
		}
		log.Printf("migrate: applied %s", version)
	}
	return nil
}

func splitStatements(sql string) []string {
	var (
		stmts         []string
		current       strings.Builder
		inSingleQuote bool
		inDoubleQuote bool
		lineComment   bool
		blockComment  bool
		dollarTag     string
	)

	flush := func() {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
		current.Reset()
	}

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		if lineComment {
			current.WriteByte(ch)
			if ch == '\n' {
				lineComment = false
			}
			continue
		}

		if blockComment {
			current.WriteByte(ch)
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				current.WriteByte(sql[i+1])
				i++
				blockComment = false
			}
			continue
		}

		if dollarTag != "" {
			current.WriteByte(ch)
			if ch == '$' && strings.HasPrefix(sql[i:], dollarTag) {
				for j := 1; j < len(dollarTag); j++ {
					current.WriteByte(sql[i+j])
				}
				i += len(dollarTag) - 1
				dollarTag = ""
			}
			continue
		}

		if !inSingleQuote && !inDoubleQuote {
			if ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
				current.WriteString("--")
				i++
				lineComment = true
				continue
			}
			if ch == '/' && i+1 < len(sql) && sql[i+1] == '*' {
				current.WriteString("/*")
				i++
				blockComment = true
				continue
			}
			if ch == '$' {
				if tag, ok := readDollarTag(sql[i:]); ok {
					current.WriteString(tag)
					i += len(tag) - 1
					dollarTag = tag
					continue
				}
			}
		}

		current.WriteByte(ch)

		if ch == '\'' && !inDoubleQuote {
			if inSingleQuote && i+1 < len(sql) && sql[i+1] == '\'' {
				current.WriteByte(sql[i+1])
				i++
				continue
			}
			inSingleQuote = !inSingleQuote
			continue
		}

		if ch == '"' && !inSingleQuote {
			if inDoubleQuote && i+1 < len(sql) && sql[i+1] == '"' {
				current.WriteByte(sql[i+1])
				i++
				continue
			}
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if ch == ';' && !inSingleQuote && !inDoubleQuote {
			flush()
		}
	}

	flush()
	return stmts
}

func readDollarTag(sql string) (string, bool) {
	if len(sql) < 2 || sql[0] != '$' {
		return "", false
	}
	for i := 1; i < len(sql); i++ {
		if sql[i] == '$' {
			return sql[:i+1], true
		}
		if !unicode.IsLetter(rune(sql[i])) && !unicode.IsDigit(rune(sql[i])) && sql[i] != '_' {
			return "", false
		}
	}
	return "", false
}

func shouldIgnoreMigrationError(stmt string, err error) bool {
	message := strings.ToLower(err.Error())
	statement := strings.ToLower(strings.TrimSpace(stmt))
	if strings.HasPrefix(statement, "create trigger set_updated_at") && strings.Contains(message, "already exists") {
		return true
	}
	return false
}
