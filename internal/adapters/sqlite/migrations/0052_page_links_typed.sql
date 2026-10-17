-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE page_links_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    to_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    to_url TEXT NOT NULL,
    anchor_text TEXT NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('generated', 'observed')),
    type TEXT NOT NULL CHECK (type IN ('breadcrumb', 'listing', 'navigation', 'contextual', 'unknown')),
    region TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL CHECK (source IN ('raw', 'render')),
    position INTEGER,
    ours INTEGER NOT NULL DEFAULT 0 CHECK (ours IN (0, 1)),
    observed_at TEXT NOT NULL
) STRICT;

INSERT INTO page_links_rebuilt (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, type, region,
    source, position, ours, observed_at)
SELECT id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, 'contextual', '',
    'raw', NULL, CASE WHEN origin = 'generated' THEN 1 ELSE 0 END, observed_at
FROM page_links;

DROP INDEX page_links_site;
DROP INDEX page_links_to;
DROP INDEX page_links_from;
DROP TABLE page_links;

ALTER TABLE page_links_rebuilt RENAME TO page_links;

CREATE INDEX page_links_from ON page_links (from_page_id, observed_at, id);
CREATE INDEX page_links_to ON page_links (to_page_id);
CREATE INDEX page_links_site ON page_links (site_id);
CREATE INDEX page_links_from_source ON page_links (from_page_id, source);

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TEMP TABLE page_links_kept AS
SELECT id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at
FROM page_links;

DROP INDEX page_links_from_source;
DROP INDEX page_links_site;
DROP INDEX page_links_to;
DROP INDEX page_links_from;
DROP TABLE page_links;

CREATE TABLE page_links (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    from_page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    to_page_id TEXT REFERENCES pages (id) ON DELETE SET NULL,
    to_url TEXT NOT NULL,
    anchor_text TEXT NOT NULL,
    origin TEXT NOT NULL CHECK (origin IN ('generated', 'observed')),
    observed_at TEXT NOT NULL
) STRICT;

INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at)
SELECT id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at
FROM page_links_kept;

DROP TABLE page_links_kept;

CREATE INDEX page_links_from ON page_links (from_page_id, observed_at, id);
CREATE INDEX page_links_to ON page_links (to_page_id);
CREATE INDEX page_links_site ON page_links (site_id);

COMMIT;

PRAGMA foreign_keys = ON;
