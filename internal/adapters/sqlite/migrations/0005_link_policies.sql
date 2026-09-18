-- +goose Up
CREATE TABLE link_policies (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('global', 'site')),
    site_id TEXT REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    rules TEXT NOT NULL,
    forbid_external INTEGER NOT NULL CHECK (forbid_external IN (0, 1)),
    forbid_self INTEGER NOT NULL CHECK (forbid_self IN (0, 1)),
    anchor_strategy TEXT NOT NULL CHECK (anchor_strategy IN ('prefer_user', 'rotate')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((scope = 'global' AND site_id IS NULL) OR (scope = 'site' AND site_id IS NOT NULL))
) STRICT;

CREATE UNIQUE INDEX link_policies_scope_name ON link_policies (scope, coalesce(site_id, ''), name);
CREATE INDEX link_policies_created_at ON link_policies (created_at, id);
CREATE INDEX link_policies_name ON link_policies (name, id);
CREATE INDEX link_policies_site ON link_policies (site_id);

-- +goose Down
DROP INDEX link_policies_site;
DROP INDEX link_policies_name;
DROP INDEX link_policies_created_at;
DROP INDEX link_policies_scope_name;
DROP TABLE link_policies;
