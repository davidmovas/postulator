-- +goose Up
CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'site')),
    site_id TEXT REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    page_kind TEXT NOT NULL,
    version INTEGER NOT NULL CHECK (version >= 1),
    spec TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((scope = 'global' AND site_id IS NULL) OR (scope = 'site' AND site_id IS NOT NULL))
) STRICT;

CREATE UNIQUE INDEX templates_scope_name ON templates (scope, coalesce(site_id, ''), name);
CREATE INDEX templates_created_at ON templates (created_at, id);
CREATE INDEX templates_name ON templates (name, id);
CREATE INDEX templates_site_kind ON templates (site_id, page_kind);

-- +goose Down
DROP INDEX templates_site_kind;
DROP INDEX templates_name;
DROP INDEX templates_created_at;
DROP INDEX templates_scope_name;
DROP TABLE templates;
