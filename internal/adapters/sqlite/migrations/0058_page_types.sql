-- +goose Up
CREATE TABLE page_types (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE CHECK (length(trim(name)) > 0),
    preset TEXT NOT NULL CHECK (preset IN ('hub', 'entity', 'product', 'store_category', 'catalog', 'technical')),
    template_id TEXT REFERENCES templates (id) ON DELETE SET NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE UNIQUE INDEX page_types_site_name ON page_types (site_id, name);
CREATE INDEX page_types_site_preset ON page_types (site_id, preset, created_at);

ALTER TABLE pages
    ADD COLUMN page_type_source TEXT NOT NULL DEFAULT '' CHECK (page_type_source IN ('', 'auto', 'manual'));

ALTER TABLE pages
    ADD COLUMN page_type_reason TEXT NOT NULL DEFAULT '' CHECK (page_type_reason IN (
        '', 'product', 'store_category', 'shop', 'technical', 'listing', 'default', 'kind', 'user', 'ai', 'import'
    ));

-- +goose Down
ALTER TABLE pages
    DROP COLUMN page_type_reason;

ALTER TABLE pages
    DROP COLUMN page_type_source;

DROP TABLE page_types;
