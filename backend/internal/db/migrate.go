package db

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs all pending SQL migrations from the embedded migrations directory.
func (db *DB) Migrate() error {
	// Bootstrap the tracking table before anything else so the version check
	// below never hits a "table doesn't exist" error on a fresh database.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    VARCHAR(32) NOT NULL PRIMARY KEY,
		applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("migrate: bootstrap: %w", err)
	}

	files, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("migrate: glob: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		version := filepath.Base(f)
		var count int
		_ = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if count > 0 {
			continue
		}
		content, err := migrationsFS.ReadFile(f)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", f, err)
		}
		for _, stmt := range splitStatements(string(content)) {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("migrate: exec %s: %w\nSQL: %s", version, err, stmt)
			}
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			return fmt.Errorf("migrate: record %s: %w", version, err)
		}
		log.Printf("migrate: applied %s", version)
	}
	return nil
}

func splitStatements(sql string) []string {
	var stmts []string
	for _, s := range strings.Split(sql, ";") {
		s = strings.TrimSpace(s)
		if s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}
