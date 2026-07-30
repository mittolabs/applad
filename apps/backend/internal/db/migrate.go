package db

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
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
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		checksum   TEXT
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
	// Backfill the checksum column onto ledgers created before it existed.
	// Rows applied before this column simply stay NULL, which the drift check
	// below treats as "unknown, do not warn" so existing installs are never
	// bricked by a false positive.
	if _, err := db.Exec(
		"ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum TEXT"); err != nil {
		return fmt.Errorf("migrate: add checksum column: %w", err)
	}

	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("migrate: glob: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		version := filepath.Base(f)
		content, err := migrationsFS.ReadFile(f)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", f, err)
		}
		sum := checksumOf(content)

		var recorded sql.NullString
		switch err := db.QueryRow(
			"SELECT checksum FROM schema_migrations WHERE version = $1", version,
		).Scan(&recorded); err {
		case nil:
			// Already applied. If a checksum was recorded and the file has
			// since changed, the live schema may no longer match what is on
			// disk. Surface it loudly, but do not hard-fail: an in-place edit
			// (001_init.sql is patched by companion migrations) must not brick
			// a running install. NULL means the row predates checksums — skip.
			if recorded.Valid && recorded.String != "" && recorded.String != sum {
				log.Printf("WARNING: migrate: %s has been modified since it was applied "+
					"(recorded checksum %s, current %s); the live schema may not match this file",
					version, shortSum(recorded.String), shortSum(sum))
			}
			continue
		case sql.ErrNoRows:
			// Not applied yet — fall through and apply it.
		default:
			return fmt.Errorf("migrate: check %s: %w", version, err)
		}

		if err := db.applyMigration(version, splitStatements(string(content)), sum); err != nil {
			return err
		}
		log.Printf("migrate: applied %s", version)
	}
	return nil
}

// applyMigration runs one migration's statements and records it as a single
// atomic unit: everything commits together or nothing does. A partial apply is
// therefore impossible, so a failure leaves the file unrecorded and it re-runs
// cleanly on the next boot rather than wedging every subsequent start.
//
// Each statement runs inside a savepoint so a statement whose error is
// explicitly ignorable (see shouldIgnoreMigrationError) can be rolled back on
// its own without aborting the enclosing transaction, preserving the previous
// per-statement ignore behaviour under the new all-or-nothing wrapper.
func (db *DB) applyMigration(version string, statements []string, checksum string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("migrate: begin %s: %w", version, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for i, stmt := range statements {
		sp := fmt.Sprintf("mig_stmt_%d", i)
		if _, err := tx.Exec("SAVEPOINT " + sp); err != nil {
			return fmt.Errorf("migrate: savepoint %s: %w", version, err)
		}
		if _, err := tx.Exec(stmt); err != nil {
			if shouldIgnoreMigrationError(stmt, err) {
				if _, rbErr := tx.Exec("ROLLBACK TO SAVEPOINT " + sp); rbErr != nil {
					return fmt.Errorf("migrate: rollback savepoint %s: %w", version, rbErr)
				}
				continue
			}
			return fmt.Errorf("migrate: exec %s: %w\nSQL: %s", version, err, stmt)
		}
		if _, err := tx.Exec("RELEASE SAVEPOINT " + sp); err != nil {
			return fmt.Errorf("migrate: release savepoint %s: %w", version, err)
		}
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)", version, checksum,
	); err != nil {
		return fmt.Errorf("migrate: record %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate: commit %s: %w", version, err)
	}
	committed = true
	return nil
}

// checksumOf returns the hex-encoded sha256 of a migration file's bytes.
func checksumOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// shortSum trims a checksum to a readable prefix for log lines.
func shortSum(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// ExtraMigration is a schema change owned by a module compiled into this build
// rather than by core.
type ExtraMigration struct {
	Version string
	SQL     string
}

// MigrateExtras applies module migrations after core's.
//
// Versions are recorded with an "ext:" prefix in the same ledger, so a module
// can number its migrations freely without ever colliding with core's, and a
// glance at schema_migrations still shows everything that has been applied.
func (db *DB) MigrateExtras(ms []ExtraMigration) error {
	if len(ms) == 0 {
		return nil
	}
	if _, err := db.Exec("SELECT pg_advisory_lock($1)", migrateAdvisoryLock); err != nil {
		return fmt.Errorf("migrate extras: acquire lock: %w", err)
	}
	defer db.Exec("SELECT pg_advisory_unlock($1)", migrateAdvisoryLock) //nolint:errcheck

	for _, m := range ms {
		version := "ext:" + m.Version
		var count int
		_ = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
		if count > 0 {
			continue
		}
		// Same all-or-nothing wrapper as core migrations: a module migration
		// that fails part way rolls back whole and re-runs cleanly next boot.
		if err := db.applyMigration(version, splitStatements(m.SQL), checksumOf([]byte(m.SQL))); err != nil {
			return fmt.Errorf("migrate extras: %w", err)
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
