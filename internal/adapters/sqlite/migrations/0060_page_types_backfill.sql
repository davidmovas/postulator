-- +goose Up
INSERT INTO page_types (id, site_id, name, preset, template_id, created_at, updated_at)
SELECT
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))), 2) || '-'
        || substr('89ab', 1 + (abs(random()) % 4), 1) || substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6))),
    sites.id, presets.name, presets.preset, NULL,
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+' || presets.position || ' seconds'),
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM sites
CROSS JOIN (
    SELECT 'hub' AS preset, 'Hub' AS name, 0 AS position
    UNION ALL SELECT 'entity', 'Entity', 1
    UNION ALL SELECT 'product', 'Product', 2
    UNION ALL SELECT 'store_category', 'Store category', 3
    UNION ALL SELECT 'catalog', 'Catalog', 4
    UNION ALL SELECT 'technical', 'Technical', 5
) AS presets
WHERE NOT EXISTS (SELECT 1 FROM page_types WHERE page_types.site_id = sites.id);

INSERT INTO field_scopes (page_type_id, field_key, enabled, ai_writes, instruction, min_len, max_len, rewrites, link_policy, label, provider, updated_at)
SELECT page_types.id, defaults.field_key, 1, 1, '', 0, 0, 0, defaults.link_policy, '', '', strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM page_types
JOIN (
    SELECT 'hub' AS preset, 'body' AS field_key,
        '{"links":true,"rules":{"upDepth":1,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":20,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":true},"forbidExternal":true,"forbidSelf":true,"anchorStrategy":"prefer_user"}' AS link_policy
    UNION ALL SELECT 'hub', 'seo_title',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'hub', 'seo_description',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'entity', 'body',
        '{"links":true,"rules":{"upDepth":2,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":12,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":false},"forbidExternal":true,"forbidSelf":true,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'entity', 'seo_title',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'entity', 'seo_description',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'product', 'product_description',
        '{"links":true,"rules":{"upDepth":1,"downLinks":false,"siblingMinWeight":1,"maxLinks":3,"maxPerTarget":1,"parentLinkWithinParagraphs":1,"childrenSection":false},"forbidExternal":true,"forbidSelf":true,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'product', 'product_short_description',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'product', 'seo_title',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
    UNION ALL SELECT 'product', 'seo_description',
        '{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}'
) AS defaults ON defaults.preset = page_types.preset
WHERE NOT EXISTS (SELECT 1 FROM field_scopes WHERE field_scopes.page_type_id = page_types.id);

WITH typed AS (
    SELECT
        pages.id AS page_id,
        pages.site_id,
        CASE
            WHEN pages.wp_type = 'product' THEN 'product'
            WHEN pages.wp_type = 'product_cat' THEN 'store_category'
            WHEN pages.technical_reason = 'shop' THEN 'catalog'
            WHEN pages.technical_reason <> '' THEN 'technical'
            WHEN templates.page_kind IN ('hub', 'category') THEN 'hub'
            WHEN templates.page_kind = 'product' THEN 'product'
            WHEN templates.page_kind IN ('guide', 'comparison') THEN 'entity'
            WHEN entities.kind IN ('hub', 'category') THEN 'hub'
            WHEN entities.kind = 'product' THEN 'product'
            ELSE 'entity'
        END AS preset,
        CASE
            WHEN pages.wp_type = 'product' THEN 'product'
            WHEN pages.wp_type = 'product_cat' THEN 'store_category'
            WHEN pages.technical_reason = 'shop' THEN 'shop'
            WHEN pages.technical_reason <> '' THEN 'technical'
            WHEN templates.page_kind IN ('hub', 'category', 'product', 'guide', 'comparison') THEN 'kind'
            WHEN entities.kind IS NOT NULL THEN 'kind'
            ELSE 'default'
        END AS reason
    FROM pages
    LEFT JOIN templates ON templates.id = pages.template_id
    LEFT JOIN entities ON entities.id = pages.entity_id
)
UPDATE pages
SET page_type_id = page_types.id, page_type_source = 'auto', page_type_reason = typed.reason
FROM typed
JOIN page_types ON page_types.site_id = typed.site_id AND page_types.preset = typed.preset
WHERE pages.id = typed.page_id AND pages.page_type_id IS NULL;

-- +goose Down
UPDATE pages
SET page_type_id = NULL, page_type_source = '', page_type_reason = '';

DELETE FROM field_scopes;

DELETE FROM page_types;
