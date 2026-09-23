-- +goose Up
CREATE TABLE entities (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    kind TEXT NOT NULL CHECK (kind IN ('hub', 'product', 'topic', 'category', 'custom')),
    intent TEXT NOT NULL DEFAULT '',
    primary_keyword TEXT NOT NULL DEFAULT '',
    secondary_keywords TEXT NOT NULL DEFAULT '[]',
    canonical_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    score REAL NOT NULL DEFAULT 0 CHECK (score >= 0),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (site_id, name)
) STRICT;

CREATE INDEX entities_site_created_at ON entities (site_id, created_at, id);
CREATE INDEX entities_canonical_page ON entities (canonical_page_id);

CREATE TABLE entity_anchors (
    entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    text TEXT NOT NULL COLLATE NOCASE,
    source TEXT NOT NULL CHECK (source IN ('user', 'ai')),
    weight REAL NOT NULL CHECK (weight >= 0 AND weight <= 1),
    PRIMARY KEY (entity_id, position),
    UNIQUE (entity_id, text)
) STRICT;

-- +goose Down
DROP TABLE entity_anchors;
DROP INDEX entities_canonical_page;
DROP INDEX entities_site_created_at;
DROP TABLE entities;
