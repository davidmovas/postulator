-- +goose Up
CREATE TABLE page_identity (
    page_id TEXT PRIMARY KEY REFERENCES pages (id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    status TEXT NOT NULL,
    wp_type TEXT NOT NULL,
    wp_parent_id INTEGER,
    modified_gmt TEXT,
    observed_at TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE page_identity;
