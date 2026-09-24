-- +goose Up
ALTER TABLE run_items
    ADD COLUMN blocked_by TEXT;

CREATE INDEX run_items_blocked_by ON run_items (blocked_by);

-- +goose Down
DROP INDEX run_items_blocked_by;

ALTER TABLE run_items
    DROP COLUMN blocked_by;
