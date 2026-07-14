-- +goose Up
CREATE TABLE setting (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- +goose Down
DROP TABLE setting;
