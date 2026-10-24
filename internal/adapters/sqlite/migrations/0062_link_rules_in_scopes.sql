-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

WITH effective AS (
    SELECT page_types.id AS page_type_id, policy.forbid_external, policy.forbid_self, policy.anchor_strategy
    FROM page_types
    JOIN sites ON sites.id = page_types.site_id
    JOIN link_policies AS policy ON policy.id = coalesce(
        (SELECT chosen.id FROM link_policies AS chosen WHERE chosen.id = sites.default_link_policy_id),
        (
            SELECT fallback.id
            FROM link_policies AS fallback
            WHERE fallback.scope = 'global' AND fallback.name = 'Default'
            ORDER BY fallback.created_at, fallback.id
            LIMIT 1
        )
    )
)
UPDATE field_scopes
SET link_policy = json_set(link_policy,
        '$.forbidExternal', json(iif(effective.forbid_external = 1, 'true', 'false')),
        '$.forbidSelf', json(iif(effective.forbid_self = 1, 'true', 'false')),
        '$.anchorStrategy', effective.anchor_strategy)
FROM effective
WHERE effective.page_type_id = field_scopes.page_type_id
    AND json_extract(field_scopes.link_policy, '$.links') = 1;

UPDATE templates
SET spec = json_remove(spec, '$.linkRules')
WHERE json_type(spec, '$.linkRules') IS NOT NULL;

UPDATE template_overrides
SET patch = json_remove(patch, '$.linkRules')
WHERE json_valid(patch) AND json_type(patch, '$.linkRules') IS NOT NULL;

CREATE TABLE sites_rebuilt (
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
    model_profiles TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    commerce TEXT NOT NULL DEFAULT '' CHECK (commerce IN ('', 'absent', 'forbidden', 'ready')),
    settings TEXT NOT NULL DEFAULT '{}'
) STRICT;

INSERT INTO sites_rebuilt (id, name, base_url, username, secret_ref, status, allow_insecure, plugin_installed,
    plugin_version, plugin_capabilities, plugin_seo, default_template_id, model_profiles, created_at, updated_at,
    commerce, settings)
SELECT id, name, base_url, username, secret_ref, status, allow_insecure, plugin_installed,
    plugin_version, plugin_capabilities, plugin_seo, default_template_id, model_profiles, created_at, updated_at,
    commerce, settings
FROM sites;

DROP INDEX sites_name;
DROP INDEX sites_created_at;
DROP TABLE sites;

ALTER TABLE sites_rebuilt RENAME TO sites;

CREATE INDEX sites_created_at ON sites (created_at, id);
CREATE INDEX sites_name ON sites (name, id);

DROP INDEX link_policies_site;
DROP INDEX link_policies_name;
DROP INDEX link_policies_created_at;
DROP INDEX link_policies_scope_name;
DROP TABLE link_policies;

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

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

INSERT INTO link_policies (id, scope, site_id, name, rules, forbid_external, forbid_self, anchor_strategy, created_at, updated_at)
VALUES (
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))), 2) || '-'
        || substr('89ab', 1 + (abs(random()) % 4), 1) || substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6))),
    'global', NULL, 'Default',
    '{"upDepth":2,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":12,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":true}',
    1, 1, 'prefer_user',
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
);

CREATE TABLE sites_saved AS SELECT * FROM sites;

DROP INDEX sites_name;
DROP INDEX sites_created_at;
DROP TABLE sites;

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

ALTER TABLE sites
    ADD COLUMN commerce TEXT NOT NULL DEFAULT '' CHECK (commerce IN ('', 'absent', 'forbidden', 'ready'));

ALTER TABLE sites
    ADD COLUMN settings TEXT NOT NULL DEFAULT '{}';

INSERT INTO sites (id, name, base_url, username, secret_ref, status, allow_insecure, plugin_installed,
    plugin_version, plugin_capabilities, plugin_seo, default_template_id, model_profiles, created_at, updated_at,
    commerce, settings)
SELECT id, name, base_url, username, secret_ref, status, allow_insecure, plugin_installed,
    plugin_version, plugin_capabilities, plugin_seo, default_template_id, model_profiles, created_at, updated_at,
    commerce, settings
FROM sites_saved;

DROP TABLE sites_saved;

CREATE INDEX sites_created_at ON sites (created_at, id);
CREATE INDEX sites_name ON sites (name, id);

UPDATE templates
SET spec = json_set(spec, '$.linkRules', json(CASE name
    WHEN 'Hub' THEN '{"upDepth":1,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":20,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":true}'
    WHEN 'Category' THEN '{"upDepth":1,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":15,"maxPerTarget":1,"parentLinkWithinParagraphs":1,"childrenSection":true}'
    WHEN 'Comparison' THEN '{"upDepth":2,"downLinks":false,"siblingMinWeight":0.7,"maxLinks":10,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":false}'
    WHEN 'Product' THEN '{"upDepth":2,"downLinks":false,"siblingMinWeight":0.6,"maxLinks":8,"maxPerTarget":1,"parentLinkWithinParagraphs":1,"childrenSection":false}'
    ELSE '{"upDepth":2,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":12,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":false}'
END))
WHERE json_type(spec, '$.linkRules') IS NULL;

COMMIT;

PRAGMA foreign_keys = ON;
