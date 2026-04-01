package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB wraps *sql.DB with helper methods.
type DB struct {
	*sql.DB
}

// Connect opens a MariaDB connection pool and verifies connectivity.
func Connect(dsn string) (*DB, error) {
	d, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	d.SetMaxOpenConns(25)
	d.SetMaxIdleConns(10)
	d.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return &DB{d}, nil
}
