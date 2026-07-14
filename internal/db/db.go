// Package db manages the SQLite connection and schema migrations for wdsplit.
package db

import (
	"database/sql"
	"fmt"

	// Registers the "sqlite" driver used by sql.Open below.
	_ "modernc.org/sqlite"
)

// Open opens the SQLite database at path and applies any pending migrations.
func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, closeAndWrap(conn, "ping database", err)
	}

	if err := Migrate(conn); err != nil {
		return nil, closeAndWrap(conn, "migrate database", err)
	}

	return conn, nil
}

func closeAndWrap(conn *sql.DB, op string, cause error) error {
	if cerr := conn.Close(); cerr != nil {
		return fmt.Errorf("%s: %w (close error: %v)", op, cause, cerr)
	}
	return fmt.Errorf("%s: %w", op, cause)
}
