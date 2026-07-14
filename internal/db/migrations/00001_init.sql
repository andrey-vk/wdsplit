-- +goose Up
CREATE TABLE category (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE service (
    id          INTEGER PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES category(id),
    name        TEXT NOT NULL,
    UNIQUE (category_id, name)
);

CREATE TABLE catalog_mode (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE adapter (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL
);

CREATE TABLE router (
    id                      INTEGER PRIMARY KEY,
    name                    TEXT NOT NULL,
    active_catalog_mode_id  INTEGER REFERENCES catalog_mode(id),
    api_host                TEXT NOT NULL,
    api_port                INTEGER NOT NULL DEFAULT 8728,
    api_username            TEXT NOT NULL,
    api_password_encrypted  BLOB NOT NULL,
    api_password_nonce      BLOB NOT NULL,
    key_version             INTEGER NOT NULL DEFAULT 1,
    web_auth_mode           TEXT NOT NULL DEFAULT 'login',
    web_login               TEXT,
    web_password_hash       TEXT,
    created_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Feeds can be admin-owned (owner_router_id NULL, gated by catalog_mode
-- membership) or a single router's private manual feed (owner_router_id
-- set, always part of that router's catalog, skips mode gating).
CREATE TABLE feed (
    id              INTEGER PRIMARY KEY,
    adapter_id      INTEGER NOT NULL REFERENCES adapter(id),
    name            TEXT NOT NULL,
    url             TEXT,
    owner_router_id INTEGER REFERENCES router(id),
    enabled         INTEGER NOT NULL DEFAULT 1,
    last_synced_at  TIMESTAMP,
    last_sync_error TEXT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feed_owner_router ON feed(owner_router_id);

-- service_id lives on the entry, not the feed: one feed can populate
-- multiple services (e.g. a bundled sing-box rule-set file).
CREATE TABLE catalog_entry (
    id         INTEGER PRIMARY KEY,
    feed_id    INTEGER NOT NULL REFERENCES feed(id) ON DELETE CASCADE,
    service_id INTEGER NOT NULL REFERENCES service(id),
    type       TEXT NOT NULL CHECK (type IN ('domain', 'domain_suffix', 'domain_keyword', 'cidr')),
    value      TEXT NOT NULL,
    UNIQUE (feed_id, service_id, type, value)
);

CREATE INDEX idx_catalog_entry_service ON catalog_entry(service_id);

CREATE TABLE feed_catalog_mode (
    feed_id         INTEGER NOT NULL REFERENCES feed(id) ON DELETE CASCADE,
    catalog_mode_id INTEGER NOT NULL REFERENCES catalog_mode(id) ON DELETE CASCADE,
    PRIMARY KEY (feed_id, catalog_mode_id)
);

-- Exactly one of category_id / service_id is set per row: a router selects
-- either a whole category (cascades to every service in it, including ones
-- added by later syncs) or specific services within a category — never
-- both for the same category. Enforced here at the row level; the
-- no-overlap-within-a-category rule is enforced by the application.
CREATE TABLE router_selection (
    id              INTEGER PRIMARY KEY,
    router_id       INTEGER NOT NULL REFERENCES router(id) ON DELETE CASCADE,
    catalog_mode_id INTEGER NOT NULL REFERENCES catalog_mode(id),
    category_id     INTEGER REFERENCES category(id),
    service_id      INTEGER REFERENCES service(id),
    enabled         INTEGER NOT NULL DEFAULT 1,
    CHECK (
        (category_id IS NOT NULL AND service_id IS NULL) OR
        (category_id IS NULL AND service_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_router_selection_category
    ON router_selection(router_id, catalog_mode_id, category_id)
    WHERE category_id IS NOT NULL;

CREATE UNIQUE INDEX idx_router_selection_service
    ON router_selection(router_id, catalog_mode_id, service_id)
    WHERE service_id IS NOT NULL;

-- +goose Down
DROP TABLE router_selection;
DROP TABLE feed_catalog_mode;
DROP TABLE catalog_entry;
DROP TABLE feed;
DROP TABLE router;
DROP TABLE adapter;
DROP TABLE catalog_mode;
DROP TABLE service;
DROP TABLE category;
