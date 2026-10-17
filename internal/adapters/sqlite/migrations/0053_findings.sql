-- +goose Up
CREATE TABLE findings (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    page_id TEXT REFERENCES pages (id) ON DELETE CASCADE,
    entity_id TEXT REFERENCES entities (id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'error')),
    subject_key TEXT NOT NULL,
    subject TEXT NOT NULL DEFAULT '{}',
    options TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL CHECK (status IN ('open', 'resolved', 'dismissed')),
    resolution TEXT NOT NULL DEFAULT '',
    resolved_by TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (site_id, code, subject_key)
) STRICT;

CREATE INDEX findings_site_status ON findings (site_id, status, created_at, id);
CREATE INDEX findings_page ON findings (page_id);

-- +goose Down
DROP INDEX findings_page;
DROP INDEX findings_site_status;
DROP TABLE findings;
