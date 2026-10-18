-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

UPDATE import_mappings
SET columns = json_remove(
        json_set(columns, '$.page_type', coalesce(json_extract(columns, '$.entity_kind'), json_extract(columns, '$.page_kind'))),
        '$.entity_kind', '$.page_kind')
WHERE json_valid(columns)
    AND (json_type(columns, '$.entity_kind') IS NOT NULL OR json_type(columns, '$.page_kind') IS NOT NULL);

UPDATE import_mappings
SET options = json_set(options, '$.rowType', 'type')
WHERE json_valid(options) AND json_extract(options, '$.rowType') = 'kind';

ALTER TABLE entities
    DROP COLUMN kind;

DROP INDEX templates_site_kind;

ALTER TABLE templates
    DROP COLUMN page_kind;

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

ALTER TABLE templates
    ADD COLUMN page_kind TEXT NOT NULL DEFAULT 'guide';

UPDATE templates
SET page_kind = CASE name
    WHEN 'Hub' THEN 'hub'
    WHEN 'Category' THEN 'category'
    WHEN 'Comparison' THEN 'comparison'
    WHEN 'Product' THEN 'product'
    ELSE 'guide'
END;

CREATE INDEX templates_site_kind ON templates (site_id, page_kind);

CREATE TABLE entities_saved AS SELECT * FROM entities;

DROP TABLE entities;

CREATE TABLE "entities" (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    kind TEXT NOT NULL CHECK (kind IN ('hub', 'product', 'topic', 'category', 'custom')),
    intent TEXT NOT NULL DEFAULT '',
    keywords TEXT NOT NULL DEFAULT '[]',
    scope_entity_id TEXT REFERENCES entities (id) ON DELETE SET NULL,
    canonical_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    score REAL NOT NULL DEFAULT 0 CHECK (score >= 0),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (scope_entity_id IS NULL OR scope_entity_id <> id)
) STRICT;

ALTER TABLE entities
    ADD COLUMN definition TEXT NOT NULL DEFAULT '';

ALTER TABLE entities
    ADD COLUMN facts TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(facts) AND json_type(facts) = 'array');

ALTER TABLE entities
    ADD COLUMN notes TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(notes) AND json_type(notes) = 'array');

INSERT INTO entities (id, site_id, name, kind, intent, keywords, scope_entity_id, canonical_page_id, score, source,
    created_at, updated_at, definition, facts, notes)
SELECT saved.id, saved.site_id, saved.name,
    coalesce((
        SELECT CASE page_types.preset
            WHEN 'hub' THEN 'hub'
            WHEN 'product' THEN 'product'
            WHEN 'store_category' THEN 'category'
            WHEN 'catalog' THEN 'category'
            ELSE 'topic'
        END
        FROM pages
        JOIN page_types ON page_types.id = pages.page_type_id
        WHERE pages.id = saved.canonical_page_id
    ), 'topic'),
    saved.intent, saved.keywords, saved.scope_entity_id, saved.canonical_page_id, saved.score, saved.source,
    saved.created_at, saved.updated_at, saved.definition, saved.facts, saved.notes
FROM entities_saved AS saved;

DROP TABLE entities_saved;

CREATE INDEX entities_site_created_at ON entities (site_id, created_at, id);
CREATE INDEX entities_canonical_page ON entities (canonical_page_id);
CREATE INDEX entities_scope ON entities (scope_entity_id);
CREATE UNIQUE INDEX entities_site_scope_name ON entities (site_id, coalesce(scope_entity_id, ''), name);

UPDATE import_mappings
SET columns = json_remove(json_set(columns, '$.entity_kind', json_extract(columns, '$.page_type')), '$.page_type')
WHERE json_valid(columns) AND json_type(columns, '$.page_type') IS NOT NULL;

UPDATE import_mappings
SET options = json_set(options, '$.rowType', 'kind')
WHERE json_valid(options) AND json_extract(options, '$.rowType') = 'type';

COMMIT;

PRAGMA foreign_keys = ON;
