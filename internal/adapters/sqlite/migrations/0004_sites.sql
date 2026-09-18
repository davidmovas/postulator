-- +goose Up
CREATE TABLE sites (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL COLLATE NOCASE,
    base_url TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    secret_ref TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'paused', 'error')),
    allow_insecure INTEGER NOT NULL DEFAULT 0 CHECK (allow_insecure IN (0, 1)),
    plugin_installed INTEGER NOT NULL DEFAULT 0 CHECK (plugin_installed IN (0, 1)),
    plugin_version TEXT NOT NULL DEFAULT '',
    plugin_capabilities TEXT NOT NULL DEFAULT '[]',
    plugin_seo TEXT NOT NULL DEFAULT '',
    default_template_id TEXT REFERENCES templates (id) ON DELETE SET NULL,
    default_link_policy_id TEXT REFERENCES link_policies (id) ON DELETE SET NULL,
    model_profiles TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX sites_created_at ON sites (created_at, id);
CREATE INDEX sites_name ON sites (name, id);

-- +goose Down
DROP INDEX sites_name;
DROP INDEX sites_created_at;
DROP TABLE sites;
