package settings

import (
	"context"
	"database/sql"
	"log/slog"
)

// Store persists setting values as key/value pairs.
type Store interface {
	// GetAll returns every stored setting, keyed by DBKey. Called once at
	// startup so New can resolve all settings from a single query.
	GetAll(ctx context.Context) (map[string]string, error)
	Save(ctx context.Context, key, value string) error
}

// SQLStore is a Store backed by the setting table.
type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM setting`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			slog.Warn("close rows", "error", cerr)
		}
	}()

	values := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}

func (s *SQLStore) Save(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO setting (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}
