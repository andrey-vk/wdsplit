-- +goose Up
-- Adapters shift from compiled-Go-per-format dispatch to a sandboxed JS
-- execution model: every non-manual adapter now carries its own source
-- code and is run identically regardless of origin. kind narrows from
-- one value per upstream format to just 'manual' | 'js' (enforced in Go,
-- not a DB CHECK, to avoid a disruptive table rebuild for a two-value
-- constraint SQLite can't add via a plain ALTER TABLE).
ALTER TABLE adapter ADD COLUMN source_code TEXT;
ALTER TABLE adapter ADD COLUMN is_builtin INTEGER NOT NULL DEFAULT 0;
ALTER TABLE adapter ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
-- Set only on a forked (non-builtin) adapter: which builtin it was
-- forked from, and that builtin's version at fork time. Compared against
-- the builtin's current version to detect "template has an update"
-- without needing to retain old source snapshots — see project memory.
ALTER TABLE adapter ADD COLUMN template_source_id INTEGER REFERENCES adapter(id);
ALTER TABLE adapter ADD COLUMN template_version_at_fork INTEGER;

-- +goose Down
ALTER TABLE adapter DROP COLUMN template_version_at_fork;
ALTER TABLE adapter DROP COLUMN template_source_id;
ALTER TABLE adapter DROP COLUMN version;
ALTER TABLE adapter DROP COLUMN is_builtin;
ALTER TABLE adapter DROP COLUMN source_code;
