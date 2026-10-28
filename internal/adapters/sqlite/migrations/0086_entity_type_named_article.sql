-- +goose Up
UPDATE page_types
SET name = CASE
        WHEN EXISTS (SELECT 1 FROM page_types taken WHERE taken.site_id = page_types.site_id AND taken.name = 'Article')
            THEN 'Article (2.5)'
        ELSE 'Article'
    END,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE name = 'Entity' AND preset = 'entity';

-- +goose Down
UPDATE page_types
SET name = 'Entity',
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE preset = 'entity'
  AND name IN ('Article', 'Article (2.5)')
  AND NOT EXISTS (SELECT 1 FROM page_types taken WHERE taken.site_id = page_types.site_id AND taken.name = 'Entity');
