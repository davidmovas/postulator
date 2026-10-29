-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE entities_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    intent TEXT NOT NULL DEFAULT '',
    keywords TEXT NOT NULL DEFAULT '[]',
    scope_entity_id TEXT REFERENCES entities (id) ON DELETE SET NULL,
    canonical_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    score REAL NOT NULL DEFAULT 0 CHECK (score >= 0),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai', 'site')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    definition TEXT NOT NULL DEFAULT '',
    facts TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(facts) AND json_type(facts) = 'array'),
    notes TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(notes) AND json_type(notes) = 'array'),
    CHECK (scope_entity_id IS NULL OR scope_entity_id <> id)
) STRICT;

INSERT INTO entities_rebuilt (id, site_id, name, intent, keywords, scope_entity_id, canonical_page_id, score, source,
    created_at, updated_at, definition, facts, notes)
SELECT id, site_id, name, intent, keywords, scope_entity_id, canonical_page_id, score, source,
    created_at, updated_at, definition, facts, notes
FROM entities;

DROP INDEX entities_site_scope_name;
DROP INDEX entities_scope;
DROP INDEX entities_canonical_page;
DROP INDEX entities_site_created_at;
DROP TABLE entities;

ALTER TABLE entities_rebuilt RENAME TO entities;

CREATE INDEX entities_site_created_at ON entities (site_id, created_at, id);
CREATE INDEX entities_canonical_page ON entities (canonical_page_id);
CREATE INDEX entities_scope ON entities (scope_entity_id);
CREATE UNIQUE INDEX entities_site_scope_name ON entities (site_id, coalesce(scope_entity_id, ''), name);

CREATE TABLE edges_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    to_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('parent', 'related')),
    weight REAL NOT NULL CHECK (weight >= 0 AND weight <= 1),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai', 'site')),
    status TEXT NOT NULL CHECK (status IN ('approved', 'proposed', 'rejected')),
    created_at TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    CHECK (from_entity_id <> to_entity_id),
    CHECK (kind <> 'related' OR from_entity_id < to_entity_id),
    CHECK (kind <> 'parent' OR weight = 1),
    UNIQUE (site_id, from_entity_id, to_entity_id, kind)
) STRICT;

INSERT INTO edges_rebuilt (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at, reason)
SELECT id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at, reason
FROM edges;

DROP INDEX edges_to_entity;
DROP INDEX edges_from_entity;
DROP INDEX edges_site_created_at;
DROP TABLE edges;

ALTER TABLE edges_rebuilt RENAME TO edges;

CREATE INDEX edges_site_created_at ON edges (site_id, created_at, id);
CREATE INDEX edges_from_entity ON edges (from_entity_id);
CREATE INDEX edges_to_entity ON edges (to_entity_id);

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

UPDATE entities SET source = 'user' WHERE source = 'site';
UPDATE edges SET source = 'user' WHERE source = 'site';

CREATE TABLE entities_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    intent TEXT NOT NULL DEFAULT '',
    keywords TEXT NOT NULL DEFAULT '[]',
    scope_entity_id TEXT REFERENCES entities (id) ON DELETE SET NULL,
    canonical_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    score REAL NOT NULL DEFAULT 0 CHECK (score >= 0),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    definition TEXT NOT NULL DEFAULT '',
    facts TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(facts) AND json_type(facts) = 'array'),
    notes TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(notes) AND json_type(notes) = 'array'),
    CHECK (scope_entity_id IS NULL OR scope_entity_id <> id)
) STRICT;

INSERT INTO entities_rebuilt (id, site_id, name, intent, keywords, scope_entity_id, canonical_page_id, score, source,
    created_at, updated_at, definition, facts, notes)
SELECT id, site_id, name, intent, keywords, scope_entity_id, canonical_page_id, score, source,
    created_at, updated_at, definition, facts, notes
FROM entities;

DROP INDEX entities_site_scope_name;
DROP INDEX entities_scope;
DROP INDEX entities_canonical_page;
DROP INDEX entities_site_created_at;
DROP TABLE entities;

ALTER TABLE entities_rebuilt RENAME TO entities;

CREATE INDEX entities_site_created_at ON entities (site_id, created_at, id);
CREATE INDEX entities_canonical_page ON entities (canonical_page_id);
CREATE INDEX entities_scope ON entities (scope_entity_id);
CREATE UNIQUE INDEX entities_site_scope_name ON entities (site_id, coalesce(scope_entity_id, ''), name);

CREATE TABLE edges_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    to_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('parent', 'related')),
    weight REAL NOT NULL CHECK (weight >= 0 AND weight <= 1),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    status TEXT NOT NULL CHECK (status IN ('approved', 'proposed', 'rejected')),
    created_at TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    CHECK (from_entity_id <> to_entity_id),
    CHECK (kind <> 'related' OR from_entity_id < to_entity_id),
    CHECK (kind <> 'parent' OR weight = 1),
    UNIQUE (site_id, from_entity_id, to_entity_id, kind)
) STRICT;

INSERT INTO edges_rebuilt (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at, reason)
SELECT id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at, reason
FROM edges;

DROP INDEX edges_to_entity;
DROP INDEX edges_from_entity;
DROP INDEX edges_site_created_at;
DROP TABLE edges;

ALTER TABLE edges_rebuilt RENAME TO edges;

CREATE INDEX edges_site_created_at ON edges (site_id, created_at, id);
CREATE INDEX edges_from_entity ON edges (from_entity_id);
CREATE INDEX edges_to_entity ON edges (to_entity_id);

COMMIT;

PRAGMA foreign_keys = ON;
