package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate applies any pending migrations, tracked by goose in the
// goose_db_version table.
func Migrate(conn *sql.DB) error {
	goose.SetBaseFS(migrationFiles)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	if err := goose.Up(conn, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
