-- +goose Up
CREATE TABLE page_terms (
    page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    taxonomy TEXT NOT NULL,
    term_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_term_id INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (page_id, taxonomy, term_id)
) STRICT;

CREATE INDEX page_terms_taxonomy_term ON page_terms (taxonomy, term_id);

-- +goose Down
DROP INDEX page_terms_taxonomy_term;
DROP TABLE page_terms;
