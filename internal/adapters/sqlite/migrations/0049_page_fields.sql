-- +goose Up
CREATE TABLE page_fields (
    page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    field_key TEXT NOT NULL,
    current_value TEXT NOT NULL DEFAULT '',
    current_hash TEXT NOT NULL DEFAULT '',
    base_value TEXT NOT NULL DEFAULT '',
    base_hash TEXT NOT NULL DEFAULT '',
    observed_at TEXT NOT NULL,
    written_at TEXT,
    written_run_id TEXT,
    PRIMARY KEY (page_id, field_key)
) STRICT;

-- +goose Down
DROP TABLE page_fields;
