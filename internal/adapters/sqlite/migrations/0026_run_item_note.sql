-- +goose Up
ALTER TABLE run_items
    ADD COLUMN note TEXT NOT NULL DEFAULT '';

CREATE INDEX run_items_held ON run_items (status, pause_reason);

-- +goose Down
DROP INDEX run_items_held;

ALTER TABLE run_items
    DROP COLUMN note;
