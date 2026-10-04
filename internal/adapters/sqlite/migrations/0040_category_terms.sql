-- +goose Up
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

-- +goose Down
DROP INDEX category_terms_site_taxonomy_term;
DROP TABLE category_terms;
