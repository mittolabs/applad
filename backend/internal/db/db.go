package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps *sql.DB with helper methods.
type DB struct {
	*sql.DB
}

// Connect opens a PostgreSQL connection pool and verifies connectivity.
// maxOpen and maxIdle configure the pool; pass 0 to use defaults (25/10).
func Connect(dsn string, maxOpen, maxIdle int) (*DB, error) {
	d, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	if maxOpen <= 0 {
		maxOpen = 25
	}
	if maxIdle <= 0 {
		maxIdle = 10
	}
	d.SetMaxOpenConns(maxOpen)
	d.SetMaxIdleConns(maxIdle)
	d.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return &DB{d}, nil
}
