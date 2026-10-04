-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE entities_rebuilt (
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

INSERT INTO entities_rebuilt (id, site_id, name, kind, intent, keywords, scope_entity_id, canonical_page_id,
    score, source, created_at, updated_at)
SELECT id, site_id, name, kind, intent, keywords,
    (
        SELECT parent.to_entity_id
        FROM edges AS parent
        WHERE parent.from_entity_id = entities.id
            AND parent.kind = 'parent'
            AND parent.status = 'approved'
        ORDER BY parent.created_at, parent.id
        LIMIT 1
    ),
    canonical_page_id, score, source, created_at, updated_at
FROM entities;

DROP INDEX entities_canonical_page;
DROP INDEX entities_site_created_at;
DROP TABLE entities;

ALTER TABLE entities_rebuilt RENAME TO entities;

CREATE INDEX entities_site_created_at ON entities (site_id, created_at, id);
CREATE INDEX entities_canonical_page ON entities (canonical_page_id);
CREATE INDEX entities_scope ON entities (scope_entity_id);
CREATE UNIQUE INDEX entities_site_scope_name ON entities (site_id, coalesce(scope_entity_id, ''), name);

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE entities_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    kind TEXT NOT NULL CHECK (kind IN ('hub', 'product', 'topic', 'category', 'custom')),
    intent TEXT NOT NULL DEFAULT '',
    canonical_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    score REAL NOT NULL DEFAULT 0 CHECK (score >= 0),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    keywords TEXT NOT NULL DEFAULT '[]',
    UNIQUE (site_id, name)
) STRICT;

INSERT INTO entities_rebuilt (id, site_id, name, kind, intent, canonical_page_id, score, source,
    created_at, updated_at, keywords)
SELECT id, site_id,
    CASE
        WHEN EXISTS (
            SELECT 1
            FROM entities AS earlier
            WHERE earlier.site_id = entities.site_id
                AND earlier.name = entities.name
                AND earlier.rowid < entities.rowid
        ) THEN name || ' (' || substr(id, 1, 8) || ')'
        ELSE name
    END,
    kind, intent, canonical_page_id, score, source, created_at, updated_at, keywords
FROM entities;

DROP INDEX entities_site_scope_name;
DROP INDEX entities_scope;
DROP INDEX entities_canonical_page;
DROP INDEX entities_site_created_at;
DROP TABLE entities;

ALTER TABLE entities_rebuilt RENAME TO entities;

CREATE INDEX entities_site_created_at ON entities (site_id, created_at, id);
CREATE INDEX entities_canonical_page ON entities (canonical_page_id);

COMMIT;

PRAGMA foreign_keys = ON;
