-- +goose Up
CREATE TABLE page_type_rules (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('rule', 'fallback')),
    position INTEGER NOT NULL CHECK (position >= 0),
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    page_type_id TEXT NOT NULL REFERENCES page_types (id),
    conditions TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(conditions)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX page_type_rules_site ON page_type_rules (site_id, position, id);

CREATE INDEX page_type_rules_type ON page_type_rules (page_type_id);

CREATE UNIQUE INDEX page_type_rules_fallback ON page_type_rules (site_id) WHERE kind = 'fallback';

CREATE TEMP TABLE rule_seeds (position INTEGER, kind TEXT, name TEXT, preset TEXT, role TEXT, type_name TEXT, conditions TEXT);

INSERT INTO rule_seeds VALUES
    (1, 'rule', 'Technical pages, such as the cart, the checkout and the account', 'technical', 'technical', 'Technical', '{"technical":true}'),
    (2, 'rule', 'Products', 'product', 'product', 'Product', '{"postTypes":["product"]}'),
    (3, 'rule', 'Store categories', 'store_category', 'archive', 'Store category', '{"postTypes":["product_cat"]}'),
    (4, 'rule', 'The shop', 'catalog', 'catalog', 'Catalog', '{"shop":true}'),
    (5, 'rule', 'The front page', 'landing', 'hub', 'Landing', '{"frontPage":true}'),
    (6, 'rule', 'A product listing at the top level', 'landing', 'hub', 'Landing', '{"level":{"max":1},"listing":{"present":true}}'),
    (7, 'rule', 'A product listing below the top level', 'category', 'hub', 'Category', '{"level":{"min":2},"listing":{"present":true}}'),
    (8, 'rule', 'Blog posts', 'guide', 'entity', 'Guide', '{"postTypes":["post"]}'),
    (9, 'rule', 'An article, a blog post or a how-to by its schema', 'guide', 'entity', 'Guide', '{"schema":["Article","BlogPosting","HowTo"]}'),
    (10, 'rule', 'A guide''s word in the address', 'guide', 'entity', 'Guide',
        '{"urls":[{"mode":"segment","value":"blog"},{"mode":"segment","value":"guide"},{"mode":"segment","value":"guides"},{"mode":"segment","value":"how-to"},{"mode":"segment","value":"learn"},{"mode":"segment","value":"research"}]}'),
    (11, 'rule', 'Under a guide', 'guide', 'entity', 'Guide', '{"ancestor":{"typeId":"@guide"}}'),
    (12, 'rule', 'An about, contact or FAQ page by its schema', 'info', 'entity', 'Info', '{"schema":["AboutPage","ContactPage","FAQPage"]}'),
    (13, 'rule', 'An about, contact, shipping or returns page by its address', 'info', 'entity', 'Info',
        '{"slugs":["about","about-us","our-story","team","careers","contact","contact-us","shipping","shipping-policy","delivery","delivery-information","faq","faqs","returns","returns-policy","refund-policy","refunds","refund_returns","payment","payments","payment-methods","warranty","help","support"]}'),
    (14, 'rule', 'A long article with sections', 'guide', 'entity', 'Guide', '{"listing":{"present":false},"words":{"min":800},"minH2":2}'),
    (15, 'fallback', 'Anything no rule matched', 'info', 'entity', 'Info', '{}');

CREATE TEMP TABLE rule_roles (preset TEXT, role TEXT);

INSERT INTO rule_roles VALUES
    ('landing', 'hub'), ('category', 'hub'), ('hub', 'hub'), ('guide', 'entity'), ('info', 'entity'), ('entity', 'entity'),
    ('product', 'product'), ('store_category', 'archive'), ('catalog', 'catalog'), ('technical', 'technical');

WITH targets AS (
    SELECT
        sites.id AS site_id,
        seeds.position,
        seeds.kind,
        seeds.name,
        seeds.conditions,
        coalesce(
            (SELECT typed.id FROM page_types typed WHERE typed.site_id = sites.id AND typed.preset = seeds.preset
             ORDER BY typed.created_at, typed.id LIMIT 1),
            (SELECT named.id FROM page_types named JOIN rule_roles ON rule_roles.preset = named.preset
             WHERE named.site_id = sites.id AND named.name = seeds.type_name AND rule_roles.role = seeds.role
             ORDER BY named.created_at, named.id LIMIT 1)
        ) AS type_id,
        coalesce(
            (SELECT typed.id FROM page_types typed WHERE typed.site_id = sites.id AND typed.preset = 'guide'
             ORDER BY typed.created_at, typed.id LIMIT 1),
            (SELECT named.id FROM page_types named JOIN rule_roles ON rule_roles.preset = named.preset
             WHERE named.site_id = sites.id AND named.name = 'Guide' AND rule_roles.role = 'entity'
             ORDER BY named.created_at, named.id LIMIT 1)
        ) AS guide_id
    FROM sites
    CROSS JOIN rule_seeds seeds
    WHERE EXISTS (SELECT 1 FROM page_types WHERE page_types.site_id = sites.id)
)
INSERT INTO page_type_rules (id, site_id, kind, position, name, enabled, page_type_id, conditions, created_at, updated_at)
SELECT
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))), 2) || '-'
        || substr('89ab', 1 + (abs(random()) % 4), 1) || substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6))),
    site_id,
    kind,
    row_number() OVER (PARTITION BY site_id ORDER BY position),
    name,
    1,
    type_id,
    CASE WHEN instr(conditions, '@guide') > 0 THEN replace(conditions, '@guide', guide_id) ELSE conditions END,
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM targets
WHERE type_id IS NOT NULL
  AND (instr(conditions, '@guide') = 0 OR guide_id IS NOT NULL);

DROP TABLE rule_seeds;

DROP TABLE rule_roles;

-- +goose Down
DROP TABLE page_type_rules;
