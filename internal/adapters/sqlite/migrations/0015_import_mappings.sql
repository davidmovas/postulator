-- +goose Up
CREATE TABLE import_mappings (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    columns TEXT NOT NULL DEFAULT '{}',
    options TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (site_id, name)
) STRICT;

CREATE INDEX import_mappings_site_name ON import_mappings (site_id, name, id);

-- +goose Down
DROP INDEX import_mappings_site_name;
DROP TABLE import_mappings;
