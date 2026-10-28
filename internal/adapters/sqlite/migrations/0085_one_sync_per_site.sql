-- +goose Up
UPDATE run_items
SET status = 'cancelled',
    lease_until = NULL,
    wake_at = NULL,
    error = 'cancelled on upgrade: another sync of this site was already queued',
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
    finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE status IN ('pending', 'running', 'waiting', 'paused')
  AND run_id IN (
    SELECT later.id
    FROM runs later
    WHERE later.kind = 'sync'
      AND later.status IN ('pending', 'running', 'waiting', 'paused')
      AND EXISTS (
        SELECT 1
        FROM runs earlier
        WHERE earlier.site_id = later.site_id
          AND earlier.kind = 'sync'
          AND earlier.status IN ('pending', 'running', 'waiting', 'paused')
          AND (earlier.created_at < later.created_at OR (earlier.created_at = later.created_at AND earlier.id < later.id))
      )
  );

UPDATE runs
SET status = 'cancelled',
    error = 'cancelled on upgrade: another sync of this site was already queued',
    finished_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE kind = 'sync'
  AND status IN ('pending', 'running', 'waiting', 'paused')
  AND EXISTS (
    SELECT 1
    FROM runs earlier
    WHERE earlier.site_id = runs.site_id
      AND earlier.kind = 'sync'
      AND earlier.status IN ('pending', 'running', 'waiting', 'paused')
      AND (earlier.created_at < runs.created_at OR (earlier.created_at = runs.created_at AND earlier.id < runs.id))
  );

CREATE UNIQUE INDEX runs_one_active_sync ON runs (site_id)
WHERE kind = 'sync' AND status IN ('pending', 'running', 'waiting', 'paused');

-- +goose Down
DROP INDEX runs_one_active_sync;
