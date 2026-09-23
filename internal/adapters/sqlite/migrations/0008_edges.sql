-- +goose Up
CREATE TABLE edges (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    to_entity_id TEXT NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('parent', 'related')),
    weight REAL NOT NULL CHECK (weight >= 0 AND weight <= 1),
    source TEXT NOT NULL CHECK (source IN ('import', 'user', 'ai')),
    status TEXT NOT NULL CHECK (status IN ('approved', 'proposed', 'rejected')),
    created_at TEXT NOT NULL,
    CHECK (from_entity_id <> to_entity_id),
    CHECK (kind <> 'related' OR from_entity_id < to_entity_id),
    CHECK (kind <> 'parent' OR weight = 1),
    UNIQUE (site_id, from_entity_id, to_entity_id, kind)
) STRICT;

CREATE INDEX edges_site_created_at ON edges (site_id, created_at, id);
CREATE INDEX edges_from_entity ON edges (from_entity_id);
CREATE INDEX edges_to_entity ON edges (to_entity_id);

-- +goose Down
DROP INDEX edges_to_entity;
DROP INDEX edges_from_entity;
DROP INDEX edges_site_created_at;
DROP TABLE edges;
