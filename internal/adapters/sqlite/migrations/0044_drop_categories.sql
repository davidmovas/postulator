-- +goose Up
DROP INDEX pages_category;

ALTER TABLE pages
    DROP COLUMN category_id;

DROP INDEX category_terms_site_taxonomy_term;
DROP TABLE category_terms;

DROP INDEX categories_site_parent;
DROP INDEX categories_site_parent_key;
DROP TABLE categories;

-- +goose Down
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

CREATE TABLE category_terms (
    category_id TEXT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    taxonomy TEXT NOT NULL CHECK (taxonomy IN ('category', 'product_cat')),
    term_id INTEGER NOT NULL CHECK (term_id > 0),
    parent_term_id INTEGER NOT NULL DEFAULT 0 CHECK (parent_term_id >= 0),
    name TEXT NOT NULL,
    run_id TEXT NOT NULL DEFAULT '',
    seen_at TEXT NOT NULL,
    PRIMARY KEY (category_id, taxonomy)
) STRICT;

CREATE INDEX category_terms_site_taxonomy_term ON category_terms (site_id, taxonomy, term_id);

ALTER TABLE pages
    ADD COLUMN category_id TEXT NOT NULL DEFAULT '';

CREATE INDEX pages_category ON pages (category_id);
