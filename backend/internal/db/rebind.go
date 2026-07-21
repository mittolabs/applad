package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// rebind converts MySQL-style ? placeholders to PostgreSQL-style $1, $2, ... placeholders.
func rebind(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	// A query already written with $N placeholders is positional, so any ? in
	// it belongs to a JSONB operator — ?, ?| and ?& all test for keys.
	// Rewriting those produces a query that is silently wrong rather than one
	// that fails loudly, so they are left alone.
	if strings.Contains(query, "$1") {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 10)
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			fmt.Fprintf(&b, "$%d", n)
			n++
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

// QueryContext overrides sql.DB.QueryContext with automatic ? → $N rebinding.
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.DB.QueryContext(ctx, rebind(query), args...)
}

// QueryRowContext overrides sql.DB.QueryRowContext with automatic ? → $N rebinding.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.DB.QueryRowContext(ctx, rebind(query), args...)
}

// ExecContext overrides sql.DB.ExecContext with automatic ? → $N rebinding.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.DB.ExecContext(ctx, rebind(query), args...)
}

// Query overrides sql.DB.Query with automatic ? → $N rebinding.
func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(rebind(query), args...)
}

// QueryRow overrides sql.DB.QueryRow with automatic ? → $N rebinding.
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.DB.QueryRow(rebind(query), args...)
}

// Exec overrides sql.DB.Exec with automatic ? → $N rebinding.
func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.DB.Exec(rebind(query), args...)
}

// Tx wraps *sql.Tx and provides automatic ? → $N rebinding.
type Tx struct {
	*sql.Tx
}

// Begin starts a transaction and returns a rebinding-aware Tx.
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

// BeginTx starts a transaction with options and returns a rebinding-aware Tx.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

// QueryContext on Tx with automatic ? → $N rebinding.
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.Tx.QueryContext(ctx, rebind(query), args...)
}

// QueryRowContext on Tx with automatic ? → $N rebinding.
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.Tx.QueryRowContext(ctx, rebind(query), args...)
}

// ExecContext on Tx with automatic ? → $N rebinding.
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.Tx.ExecContext(ctx, rebind(query), args...)
}

// Query on Tx with automatic ? → $N rebinding.
func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.Tx.Query(rebind(query), args...)
}

// QueryRow on Tx with automatic ? → $N rebinding.
func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.Tx.QueryRow(rebind(query), args...)
}

// Exec on Tx with automatic ? → $N rebinding.
func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.Tx.Exec(rebind(query), args...)
}
