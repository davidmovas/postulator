-- +goose Up
CREATE TABLE field_scopes (
    page_type_id TEXT NOT NULL REFERENCES page_types (id) ON DELETE CASCADE,
    field_key TEXT NOT NULL CHECK (length(field_key) > 0),
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    ai_writes INTEGER NOT NULL DEFAULT 1 CHECK (ai_writes IN (0, 1)),
    instruction TEXT NOT NULL DEFAULT '',
    min_len INTEGER NOT NULL DEFAULT 0 CHECK (min_len >= 0),
    max_len INTEGER NOT NULL DEFAULT 0 CHECK (max_len >= 0),
    rewrites INTEGER NOT NULL DEFAULT 0 CHECK (rewrites IN (0, 1)),
    link_policy TEXT NOT NULL CHECK (json_valid(link_policy) AND coalesce(json_type(link_policy, '$.links'), '') IN ('true', 'false')),
    label TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '' CHECK (provider IN ('', 'plugin', 'store')),
    updated_at TEXT NOT NULL,
    PRIMARY KEY (page_type_id, field_key)
) STRICT;

-- +goose Down
DROP TABLE field_scopes;
