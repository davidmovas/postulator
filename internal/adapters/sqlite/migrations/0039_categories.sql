-- +goose Up
CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL,
    parent_id TEXT REFERENCES categories (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (parent_id IS NULL OR parent_id <> id)
) STRICT;

CREATE UNIQUE INDEX categories_site_parent_key ON categories (site_id, coalesce(parent_id, ''), name_key);
CREATE INDEX categories_site_parent ON categories (site_id, parent_id);

-- +goose Down
DROP INDEX categories_site_parent;
DROP INDEX categories_site_parent_key;
DROP TABLE categories;
