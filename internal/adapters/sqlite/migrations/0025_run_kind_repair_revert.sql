-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE runs_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('generate', 'relink', 'audit', 'sync', 'import', 'repair', 'revert', 'custom')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'waiting', 'paused', 'completed', 'failed', 'cancelled')),
    targets TEXT NOT NULL,
    recipe TEXT NOT NULL,
    template_id TEXT NOT NULL DEFAULT '',
    template_version INTEGER NOT NULL DEFAULT 0,
    publish_mode TEXT NOT NULL CHECK (publish_mode IN ('draft', 'publish')),
    budget_max_usd REAL NOT NULL DEFAULT 0,
    budget_max_tokens INTEGER NOT NULL DEFAULT 0,
    stats_items INTEGER NOT NULL DEFAULT 0,
    stats_done INTEGER NOT NULL DEFAULT 0,
    stats_failed INTEGER NOT NULL DEFAULT 0,
    stats_tokens INTEGER NOT NULL DEFAULT 0,
    stats_usd REAL NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL CHECK (created_by IN ('user', 'agent', 'schedule')),
    parent_run_id TEXT REFERENCES runs (id) ON DELETE SET NULL,
    pause_reason TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    deadline_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT
) STRICT;

INSERT INTO runs_rebuilt (id, site_id, kind, status, targets, recipe, template_id, template_version,
    publish_mode, budget_max_usd, budget_max_tokens, stats_items, stats_done, stats_failed, stats_tokens,
    stats_usd, created_by, parent_run_id, pause_reason, error, deadline_at, created_at, started_at, finished_at)
SELECT id, site_id, kind, status, targets, recipe, template_id, template_version,
    publish_mode, budget_max_usd, budget_max_tokens, stats_items, stats_done, stats_failed, stats_tokens,
    stats_usd, created_by, parent_run_id, pause_reason, error, deadline_at, created_at, started_at, finished_at
FROM runs;

DROP INDEX runs_parent;
DROP INDEX runs_status;
DROP INDEX runs_site_created_at;
DROP TABLE runs;

ALTER TABLE runs_rebuilt RENAME TO runs;

CREATE INDEX runs_site_created_at ON runs (site_id, created_at, id);
CREATE INDEX runs_status ON runs (status, created_at, id);
CREATE INDEX runs_parent ON runs (parent_run_id);

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
DELETE FROM runs WHERE kind IN ('repair', 'revert');

PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE runs_rebuilt (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('generate', 'relink', 'audit', 'sync', 'import', 'custom')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'waiting', 'paused', 'completed', 'failed', 'cancelled')),
    targets TEXT NOT NULL,
    recipe TEXT NOT NULL,
    template_id TEXT NOT NULL DEFAULT '',
    template_version INTEGER NOT NULL DEFAULT 0,
    publish_mode TEXT NOT NULL CHECK (publish_mode IN ('draft', 'publish')),
    budget_max_usd REAL NOT NULL DEFAULT 0,
    budget_max_tokens INTEGER NOT NULL DEFAULT 0,
    stats_items INTEGER NOT NULL DEFAULT 0,
    stats_done INTEGER NOT NULL DEFAULT 0,
    stats_failed INTEGER NOT NULL DEFAULT 0,
    stats_tokens INTEGER NOT NULL DEFAULT 0,
    stats_usd REAL NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL CHECK (created_by IN ('user', 'agent', 'schedule')),
    parent_run_id TEXT REFERENCES runs (id) ON DELETE SET NULL,
    pause_reason TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    deadline_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT
) STRICT;

INSERT INTO runs_rebuilt (id, site_id, kind, status, targets, recipe, template_id, template_version,
    publish_mode, budget_max_usd, budget_max_tokens, stats_items, stats_done, stats_failed, stats_tokens,
    stats_usd, created_by, parent_run_id, pause_reason, error, deadline_at, created_at, started_at, finished_at)
SELECT id, site_id, kind, status, targets, recipe, template_id, template_version,
    publish_mode, budget_max_usd, budget_max_tokens, stats_items, stats_done, stats_failed, stats_tokens,
    stats_usd, created_by, parent_run_id, pause_reason, error, deadline_at, created_at, started_at, finished_at
FROM runs;

DROP INDEX runs_parent;
DROP INDEX runs_status;
DROP INDEX runs_site_created_at;
DROP TABLE runs;

ALTER TABLE runs_rebuilt RENAME TO runs;

CREATE INDEX runs_site_created_at ON runs (site_id, created_at, id);
CREATE INDEX runs_status ON runs (status, created_at, id);
CREATE INDEX runs_parent ON runs (parent_run_id);

COMMIT;

PRAGMA foreign_keys = ON;
