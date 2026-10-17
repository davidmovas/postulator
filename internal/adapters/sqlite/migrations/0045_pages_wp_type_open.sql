-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE pages_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    wp_type TEXT NOT NULL CHECK (wp_type <> ''),
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
    wp_link TEXT NOT NULL DEFAULT '',
    wp_slug TEXT NOT NULL DEFAULT '',
    wp_status TEXT NOT NULL DEFAULT '',
    wp_title TEXT NOT NULL DEFAULT '',
    wp_h1 TEXT NOT NULL DEFAULT '',
    keywords TEXT NOT NULL DEFAULT '[]',
    notes TEXT NOT NULL DEFAULT '[]',
    planned_path TEXT NOT NULL DEFAULT '',
    UNIQUE (site_id, path)
) STRICT;

INSERT INTO pages_rebuilt (id, site_id, path, slug, parent_page_id, wp_type, wp_id, title, h1, meta_title,
    meta_description, canonical, status, entity_id, template_id, content_hash, wp_modified_at, last_synced_at,
    drift, created_at, updated_at, wp_link, wp_slug, wp_status, wp_title, wp_h1, keywords, notes, planned_path)
SELECT id, site_id, path, slug, parent_page_id, wp_type, wp_id, title, h1, meta_title,
    meta_description, canonical, status, entity_id, template_id, content_hash, wp_modified_at, last_synced_at,
    drift, created_at, updated_at, wp_link, wp_slug, wp_status, wp_title, wp_h1, keywords, notes, planned_path
FROM pages;

DROP INDEX pages_template;
DROP INDEX pages_parent;
DROP INDEX pages_entity;
DROP INDEX pages_site_entity;
DROP INDEX pages_site_created_at;
DROP TABLE pages;

ALTER TABLE pages_rebuilt RENAME TO pages;

CREATE INDEX pages_site_created_at ON pages (site_id, created_at, id);
CREATE INDEX pages_site_entity ON pages (site_id, entity_id);
CREATE INDEX pages_entity ON pages (entity_id);
CREATE INDEX pages_parent ON pages (parent_page_id);
CREATE INDEX pages_template ON pages (template_id);

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
DELETE FROM pages WHERE wp_type NOT IN ('page', 'post', 'product', 'product_cat');

PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TEMP TABLE pages_kept AS SELECT * FROM pages;

DROP INDEX pages_template;
DROP INDEX pages_parent;
DROP INDEX pages_entity;
DROP INDEX pages_site_entity;
DROP INDEX pages_site_created_at;
DROP TABLE pages;

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
    updated_at TEXT NOT NULL, wp_link TEXT NOT NULL DEFAULT '', wp_slug TEXT NOT NULL DEFAULT '', wp_status TEXT NOT NULL DEFAULT '', wp_title TEXT NOT NULL DEFAULT '', wp_h1 TEXT NOT NULL DEFAULT '', keywords TEXT NOT NULL DEFAULT '[]', notes TEXT NOT NULL DEFAULT '[]', planned_path TEXT NOT NULL DEFAULT '',
    UNIQUE (site_id, path)
) STRICT;

INSERT INTO pages (id, site_id, path, slug, parent_page_id, wp_type, wp_id, title, h1, meta_title,
    meta_description, canonical, status, entity_id, template_id, content_hash, wp_modified_at, last_synced_at,
    drift, created_at, updated_at, wp_link, wp_slug, wp_status, wp_title, wp_h1, keywords, notes, planned_path)
SELECT id, site_id, path, slug, parent_page_id, wp_type, wp_id, title, h1, meta_title,
    meta_description, canonical, status, entity_id, template_id, content_hash, wp_modified_at, last_synced_at,
    drift, created_at, updated_at, wp_link, wp_slug, wp_status, wp_title, wp_h1, keywords, notes, planned_path
FROM pages_kept;

DROP TABLE pages_kept;

CREATE INDEX pages_site_created_at ON pages (site_id, created_at, id);
CREATE INDEX pages_site_entity ON pages (site_id, entity_id);
CREATE INDEX pages_entity ON pages (entity_id);
CREATE INDEX pages_parent ON pages (parent_page_id);
CREATE INDEX pages_template ON pages (template_id);

COMMIT;

PRAGMA foreign_keys = ON;
