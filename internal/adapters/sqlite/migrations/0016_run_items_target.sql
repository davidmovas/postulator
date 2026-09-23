-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE run_items_rebuilt (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    site_id TEXT NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    target_id TEXT NOT NULL,
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

INSERT INTO run_items_rebuilt (id, run_id, site_id, target_id, status, current_step, attempts, advance_seq,
    checkpoint, lease_until, wake_at, pause_reason, error, created_at, updated_at, finished_at)
SELECT i.id, i.run_id, r.site_id, i.page_id, i.status, i.current_step, i.attempts, i.advance_seq,
    i.checkpoint, i.lease_until, i.wake_at, i.pause_reason, i.error, i.created_at, i.updated_at, i.finished_at
FROM run_items i JOIN runs r ON r.id = i.run_id;

DROP INDEX run_items_page;
DROP INDEX run_items_status_lease;
DROP INDEX run_items_status_wake;
DROP INDEX run_items_run_created_at;
DROP TABLE run_items;

ALTER TABLE run_items_rebuilt RENAME TO run_items;

CREATE INDEX run_items_run_created_at ON run_items (run_id, created_at, id);
CREATE INDEX run_items_status_wake ON run_items (status, wake_at);
CREATE INDEX run_items_status_lease ON run_items (status, lease_until);
CREATE INDEX run_items_target ON run_items (target_id, created_at, id);
CREATE INDEX run_items_site ON run_items (site_id);

COMMIT;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE run_items_rebuilt (
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

INSERT INTO run_items_rebuilt (id, run_id, page_id, status, current_step, attempts, advance_seq,
    checkpoint, lease_until, wake_at, pause_reason, error, created_at, updated_at, finished_at)
SELECT i.id, i.run_id, i.target_id, i.status, i.current_step, i.attempts, i.advance_seq,
    i.checkpoint, i.lease_until, i.wake_at, i.pause_reason, i.error, i.created_at, i.updated_at, i.finished_at
FROM run_items i JOIN pages p ON p.id = i.target_id;

DROP INDEX run_items_site;
DROP INDEX run_items_target;
DROP INDEX run_items_status_lease;
DROP INDEX run_items_status_wake;
DROP INDEX run_items_run_created_at;
DROP TABLE run_items;

ALTER TABLE run_items_rebuilt RENAME TO run_items;

CREATE INDEX run_items_run_created_at ON run_items (run_id, created_at, id);
CREATE INDEX run_items_status_wake ON run_items (status, wake_at);
CREATE INDEX run_items_status_lease ON run_items (status, lease_until);
CREATE INDEX run_items_page ON run_items (page_id);

COMMIT;

PRAGMA foreign_keys = ON;
