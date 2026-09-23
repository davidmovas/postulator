-- +goose Up
CREATE TABLE app_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TEXT NOT NULL
) STRICT;

INSERT INTO app_meta (key, value, created_at)
VALUES ('schema_version', '1', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));

-- +goose Down
DROP TABLE app_meta;
