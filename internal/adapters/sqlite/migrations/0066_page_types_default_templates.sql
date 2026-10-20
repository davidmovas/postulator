-- +goose Up
UPDATE page_types
SET template_id = (
    SELECT shipped.id
    FROM templates shipped
    WHERE shipped.scope = 'global'
      AND shipped.name = CASE page_types.preset WHEN 'hub' THEN 'Hub' WHEN 'entity' THEN 'Guide' WHEN 'product' THEN 'Product' END
    ORDER BY shipped.created_at, shipped.id
    LIMIT 1
)
WHERE template_id IS NULL
  AND preset IN ('hub', 'entity', 'product')
  AND site_id IN (SELECT id FROM sites WHERE default_template_id IS NULL);

-- +goose Down
UPDATE page_types
SET template_id = NULL
WHERE preset IN ('hub', 'entity', 'product')
  AND site_id IN (SELECT id FROM sites WHERE default_template_id IS NULL)
  AND template_id = (
    SELECT shipped.id
    FROM templates shipped
    WHERE shipped.scope = 'global'
      AND shipped.name = CASE page_types.preset WHEN 'hub' THEN 'Hub' WHEN 'entity' THEN 'Guide' WHEN 'product' THEN 'Product' END
    ORDER BY shipped.created_at, shipped.id
    LIMIT 1
);
