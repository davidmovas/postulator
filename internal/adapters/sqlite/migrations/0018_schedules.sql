-- +goose Up
CREATE TABLE schedules (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cron TEXT NOT NULL DEFAULT '',
    interval_seconds INTEGER NOT NULL DEFAULT 0,
    target_entity_id TEXT REFERENCES entities (id) ON DELETE CASCADE,
    target_status TEXT NOT NULL DEFAULT '',
    target_limit INTEGER NOT NULL DEFAULT 0,
    template_id TEXT NOT NULL DEFAULT '',
    recipe TEXT NOT NULL DEFAULT '[]',
    publish_mode TEXT NOT NULL CHECK (publish_mode IN ('draft', 'publish')),
    budget_max_usd REAL NOT NULL DEFAULT 0,
    budget_max_tokens INTEGER NOT NULL DEFAULT 0,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    next_run_at TEXT,
    last_run_id TEXT REFERENCES runs (id) ON DELETE SET NULL,
    created_by TEXT NOT NULL CHECK (created_by IN ('user', 'agent', 'schedule')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (site_id, name)
) STRICT;

CREATE INDEX schedules_due ON schedules (enabled, next_run_at);
CREATE INDEX schedules_site ON schedules (site_id, created_at, id);

-- +goose Down
DROP INDEX schedules_site;
DROP INDEX schedules_due;
DROP TABLE schedules;
