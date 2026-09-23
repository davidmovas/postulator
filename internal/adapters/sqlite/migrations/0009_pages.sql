-- +goose Up
CREATE TABLE pages (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    wp_type TEXT NOT NULL CHECK (wp_type IN ('page', 'post', 'product', 'product_cat')),
    wp_id INTEGER,
    title TEXT NOT NULL DEFAULT '',
    h1 TEXT NOT NULL DEFAULT '',
    meta_title TEXT NOT NULL DEFAULT '',
    meta_description TEXT NOT NULL DEFAULT '',
    canonical TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('planned', 'exists', 'published', 'archived')),
    entity_id TEXT REFERENCES entities (id) ON DELETE SET NULL,
    template_id TEXT REFERENCES templates (id) ON DELETE SET NULL,
    content_hash TEXT NOT NULL DEFAULT '',
    wp_modified_at TEXT,
    last_synced_at TEXT,
    drift INTEGER NOT NULL DEFAULT 0 CHECK (drift IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (site_id, path)
) STRICT;

CREATE INDEX pages_site_created_at ON pages (site_id, created_at, id);
CREATE INDEX pages_site_entity ON pages (site_id, entity_id);
CREATE INDEX pages_entity ON pages (entity_id);
CREATE INDEX pages_parent ON pages (parent_page_id);
CREATE INDEX pages_template ON pages (template_id);

CREATE TABLE page_links (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    to_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    to_url TEXT NOT NULL,
    anchor_text TEXT NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('generated', 'observed')),
    observed_at TEXT NOT NULL
) STRICT;

CREATE INDEX page_links_from ON page_links (from_page_id, observed_at, id);
CREATE INDEX page_links_to ON page_links (to_page_id);
CREATE INDEX page_links_site ON page_links (site_id);

-- +goose Down
DROP INDEX page_links_site;
DROP INDEX page_links_to;
DROP INDEX page_links_from;
DROP TABLE page_links;
DROP INDEX pages_template;
DROP INDEX pages_parent;
DROP INDEX pages_entity;
DROP INDEX pages_site_entity;
DROP INDEX pages_site_created_at;
DROP TABLE pages;
