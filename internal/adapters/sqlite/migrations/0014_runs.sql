-- +goose Up
CREATE TABLE runs (
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

CREATE INDEX runs_site_created_at ON runs (site_id, created_at, id);
CREATE INDEX runs_status ON runs (status, created_at, id);
CREATE INDEX runs_parent ON runs (parent_run_id);

CREATE TABLE run_items (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    page_id TEXT NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'waiting', 'paused', 'completed', 'failed', 'cancelled')),
    current_step TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    advance_seq INTEGER NOT NULL DEFAULT 0,
    checkpoint TEXT NOT NULL DEFAULT '{}',
    lease_until TEXT,
    wake_at TEXT,
    pause_reason TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    finished_at TEXT
) STRICT;

CREATE INDEX run_items_run_created_at ON run_items (run_id, created_at, id);
CREATE INDEX run_items_status_wake ON run_items (status, wake_at);
CREATE INDEX run_items_status_lease ON run_items (status, lease_until);
CREATE INDEX run_items_page ON run_items (page_id);

CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    item_id TEXT NOT NULL REFERENCES run_items (id) ON DELETE CASCADE,
    step TEXT NOT NULL,
    kind TEXT NOT NULL,
    blob BLOB NOT NULL,
    size INTEGER NOT NULL,
    hash TEXT NOT NULL,
    purged INTEGER NOT NULL DEFAULT 0 CHECK (purged IN (0, 1)),
    expires_at TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (item_id, step, kind)
) STRICT;

CREATE INDEX artifacts_item_kind ON artifacts (item_id, kind);
CREATE INDEX artifacts_run_kind ON artifacts (run_id, kind);
CREATE INDEX artifacts_purge ON artifacts (purged, kind, created_at);

CREATE TABLE step_execs (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    item_id TEXT NOT NULL REFERENCES run_items (id) ON DELETE CASCADE,
    step TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('started', 'done', 'failed')),
    input_hash TEXT NOT NULL,
    artifact_id TEXT REFERENCES artifacts (id) ON DELETE SET NULL,
    tokens INTEGER NOT NULL DEFAULT 0,
    usd REAL NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL,
    finished_at TEXT,
    error TEXT NOT NULL DEFAULT '',
    UNIQUE (item_id, step, attempt)
) STRICT;

CREATE INDEX step_execs_item_step ON step_execs (item_id, step, status);
CREATE INDEX step_execs_run ON step_execs (run_id, started_at, id);

CREATE TABLE run_events (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    seq INTEGER NOT NULL,
    type TEXT NOT NULL,
    at TEXT NOT NULL,
    payload TEXT NOT NULL,
    UNIQUE (run_id, seq)
) STRICT;

CREATE INDEX run_events_run_seq ON run_events (run_id, seq);

-- +goose Down
DROP INDEX run_events_run_seq;
DROP TABLE run_events;
DROP INDEX step_execs_run;
DROP INDEX step_execs_item_step;
DROP TABLE step_execs;
DROP INDEX artifacts_purge;
DROP INDEX artifacts_run_kind;
DROP INDEX artifacts_item_kind;
DROP TABLE artifacts;
DROP INDEX run_items_page;
DROP INDEX run_items_status_lease;
DROP INDEX run_items_status_wake;
DROP INDEX run_items_run_created_at;
DROP TABLE run_items;
DROP INDEX runs_parent;
DROP INDEX runs_status;
DROP INDEX runs_site_created_at;
DROP TABLE runs;
