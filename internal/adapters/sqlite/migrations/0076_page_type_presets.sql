-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

DROP TRIGGER page_map_type_moved;

DROP INDEX page_types_site_preset;

ALTER TABLE page_types
    ADD COLUMN preset_widened TEXT NOT NULL DEFAULT 'hub' CHECK (preset_widened IN (
        'landing', 'category', 'guide', 'info', 'product', 'store_category', 'catalog', 'technical', 'hub', 'entity'
    ));

UPDATE page_types SET preset_widened = preset;

ALTER TABLE page_types
    DROP COLUMN preset;

ALTER TABLE page_types
    RENAME COLUMN preset_widened TO preset;

CREATE INDEX page_types_site_preset ON page_types (site_id, preset, created_at);

-- +goose StatementBegin
CREATE TRIGGER page_map_type_moved AFTER UPDATE OF preset, site_id ON page_types
WHEN OLD.preset IS NOT NEW.preset OR OLD.site_id IS NOT NEW.site_id
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id IN (OLD.site_id, NEW.site_id);
END;
-- +goose StatementEnd

CREATE TEMP TABLE added_presets (preset TEXT, name TEXT, role TEXT, template TEXT, position INTEGER);

INSERT INTO added_presets VALUES
    ('landing', 'Landing', 'hub', 'Hub', 0),
    ('category', 'Category', 'hub', 'Hub', 1),
    ('guide', 'Guide', 'entity', 'Guide', 2),
    ('info', 'Info', 'entity', 'Guide', 3);

CREATE TEMP TABLE preset_roles (preset TEXT, role TEXT);

INSERT INTO preset_roles VALUES
    ('landing', 'hub'), ('category', 'hub'), ('hub', 'hub'), ('guide', 'entity'), ('info', 'entity'), ('entity', 'entity'),
    ('product', 'product'), ('store_category', 'archive'), ('catalog', 'catalog'), ('technical', 'technical');

INSERT INTO page_types (id, site_id, name, preset, template_id, created_at, updated_at)
SELECT
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))), 2) || '-'
        || substr('89ab', 1 + (abs(random()) % 4), 1) || substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6))),
    sites.id,
    CASE
        WHEN NOT EXISTS (SELECT 1 FROM page_types taken WHERE taken.site_id = sites.id AND taken.name = added.name) THEN added.name
        WHEN NOT EXISTS (SELECT 1 FROM page_types taken WHERE taken.site_id = sites.id AND taken.name = added.name || ' pages')
            THEN added.name || ' pages'
        WHEN NOT EXISTS (SELECT 1 FROM page_types taken WHERE taken.site_id = sites.id AND taken.name = added.name || ' pages 2')
            THEN added.name || ' pages 2'
        ELSE added.name || ' pages ' || lower(hex(randomblob(3)))
    END,
    added.preset,
    CASE WHEN sites.default_template_id IS NULL THEN (
        SELECT shipped.id
        FROM templates shipped
        WHERE shipped.scope = 'global' AND shipped.name = added.template
        ORDER BY shipped.created_at, shipped.id
        LIMIT 1
    ) END,
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+' || added.position || ' seconds'),
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM sites
CROSS JOIN added_presets added
WHERE EXISTS (SELECT 1 FROM page_types typed WHERE typed.site_id = sites.id)
  AND NOT EXISTS (SELECT 1 FROM page_types typed WHERE typed.site_id = sites.id AND typed.preset = added.preset)
  AND NOT EXISTS (
      SELECT 1
      FROM page_types named
      JOIN preset_roles ON preset_roles.preset = named.preset
      WHERE named.site_id = sites.id AND named.name = added.name AND preset_roles.role = added.role
  );

INSERT INTO field_scopes (page_type_id, field_key, enabled, ai_writes, instruction, min_len, max_len, rewrites, link_policy, label, provider, updated_at)
SELECT page_types.id, defaults.field_key, 1, 1, '', 0, 0, 0, defaults.link_policy, '', '', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM page_types
JOIN (
    SELECT 'hub' AS role, 'body' AS field_key,
        '{"links":true,"rules":{"upDepth":1,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":20,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":true},"forbidExternal":true,"forbidSelf":true,"anchorStrategy":"prefer_user"}' AS link_policy
    UNION ALL SELECT 'hub', 'seo_title',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'hub', 'seo_description',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'guide', 'body',
        '{"links":true,"rules":{"upDepth":2,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":12,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":false},"forbidExternal":true,"forbidSelf":true,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'guide', 'seo_title',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'guide', 'seo_description',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'info', 'seo_title',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'info', 'seo_description',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
) AS defaults ON defaults.role = CASE page_types.preset WHEN 'landing' THEN 'hub' WHEN 'category' THEN 'hub' ELSE page_types.preset END
WHERE page_types.preset IN ('landing', 'category', 'guide', 'info')
  AND NOT EXISTS (SELECT 1 FROM field_scopes WHERE field_scopes.page_type_id = page_types.id);

DROP TABLE added_presets;

DROP TABLE preset_roles;

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

DROP TRIGGER page_map_type_moved;

DROP INDEX page_types_site_preset;

ALTER TABLE page_types
    ADD COLUMN preset_narrowed TEXT NOT NULL DEFAULT 'hub' CHECK (preset_narrowed IN (
        'hub', 'entity', 'product', 'store_category', 'catalog', 'technical'
    ));

UPDATE page_types
SET preset_narrowed = CASE preset
    WHEN 'landing' THEN 'hub'
    WHEN 'category' THEN 'hub'
    WHEN 'guide' THEN 'entity'
    WHEN 'info' THEN 'entity'
    ELSE preset
END;

ALTER TABLE page_types
    DROP COLUMN preset;

ALTER TABLE page_types
    RENAME COLUMN preset_narrowed TO preset;

CREATE INDEX page_types_site_preset ON page_types (site_id, preset, created_at);

-- +goose StatementBegin
CREATE TRIGGER page_map_type_moved AFTER UPDATE OF preset, site_id ON page_types
WHEN OLD.preset IS NOT NEW.preset OR OLD.site_id IS NOT NEW.site_id
BEGIN
    UPDATE page_map_sites SET generation = generation + 1 WHERE site_id IN (OLD.site_id, NEW.site_id);
END;
-- +goose StatementEnd

COMMIT;

PRAGMA foreign_keys = ON;
