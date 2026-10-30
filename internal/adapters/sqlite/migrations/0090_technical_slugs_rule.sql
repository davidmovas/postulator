-- +goose Up
CREATE TEMP TABLE technical_slugs_sites AS
SELECT
    sites.id AS site_id,
    (SELECT typed.id FROM page_types typed WHERE typed.site_id = sites.id AND typed.preset = 'technical'
     ORDER BY typed.created_at, typed.id LIMIT 1) AS type_id
FROM sites
WHERE EXISTS (SELECT 1 FROM page_type_rules WHERE page_type_rules.site_id = sites.id);

DELETE FROM technical_slugs_sites WHERE type_id IS NULL;

UPDATE page_type_rules
SET position = position + 1
WHERE position >= 2
  AND site_id IN (SELECT site_id FROM technical_slugs_sites);

INSERT INTO page_type_rules (id, site_id, kind, position, name, enabled, page_type_id, conditions, created_at, updated_at)
SELECT
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))), 2) || '-'
        || substr('89ab', 1 + (abs(random()) % 4), 1) || substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6))),
    site_id,
    'rule',
    2,
    'The cart, the checkout and the account by their address',
    1,
    type_id,
    '{"postTypes":["page"],"slugs":["cart","basket","checkout","my-account","account","wishlist","compare","order-tracking","track-order","login","register","lost-password"]}',
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM technical_slugs_sites;

DROP TABLE technical_slugs_sites;

-- +goose Down
CREATE TEMP TABLE technical_slugs_rules AS
SELECT id, site_id, position
FROM page_type_rules
WHERE kind = 'rule' AND name = 'The cart, the checkout and the account by their address';

DELETE FROM page_type_rules WHERE id IN (SELECT id FROM technical_slugs_rules);

UPDATE page_type_rules
SET position = position - 1
WHERE EXISTS (
    SELECT 1 FROM technical_slugs_rules seeded
    WHERE seeded.site_id = page_type_rules.site_id AND page_type_rules.position > seeded.position
);

DROP TABLE technical_slugs_rules;
