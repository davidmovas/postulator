-- +goose Up
DROP TABLE page_renders;

CREATE TABLE page_renders (
    page_id TEXT PRIMARY KEY REFERENCES pages (id) ON DELETE CASCADE,
    render_hash TEXT NOT NULL,
    content_root TEXT NOT NULL CHECK (content_root IN ('found', 'empty_raw', 'dynamic_only', 'archive', 'not_found')),
    regions TEXT NOT NULL DEFAULT '[]',
    jsonld TEXT NOT NULL DEFAULT '{}',
    trail TEXT NOT NULL DEFAULT '[]',
    headings TEXT NOT NULL DEFAULT '[]',
    word_count_content INTEGER NOT NULL DEFAULT 0,
    word_count_generated INTEGER NOT NULL DEFAULT 0,
    builder TEXT NOT NULL DEFAULT '',
    url_form TEXT NOT NULL DEFAULT '',
    robots TEXT NOT NULL DEFAULT '',
    final_path TEXT NOT NULL DEFAULT '',
    extracted_at TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE page_renders;

CREATE TABLE page_renders (
    page_id TEXT PRIMARY KEY REFERENCES pages (id) ON DELETE CASCADE,
    render_hash TEXT NOT NULL,
    content_root TEXT NOT NULL CHECK (content_root IN ('found', 'empty_raw', 'dynamic_only', 'not_found')),
    regions TEXT NOT NULL DEFAULT '[]',
    jsonld TEXT NOT NULL DEFAULT '{}',
    trail TEXT NOT NULL DEFAULT '[]',
    headings TEXT NOT NULL DEFAULT '[]',
    word_count_content INTEGER NOT NULL DEFAULT 0,
    word_count_generated INTEGER NOT NULL DEFAULT 0,
    builder TEXT NOT NULL DEFAULT '',
    url_form TEXT NOT NULL DEFAULT '',
    robots TEXT NOT NULL DEFAULT '',
    extracted_at TEXT NOT NULL
) STRICT;
