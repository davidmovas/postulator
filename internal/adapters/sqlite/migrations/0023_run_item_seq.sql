-- +goose Up
ALTER TABLE run_items
    ADD COLUMN seq INTEGER NOT NULL DEFAULT 0;

CREATE INDEX run_items_run_seq ON run_items (run_id, seq, id);

-- +goose Down
DROP INDEX run_items_run_seq;

ALTER TABLE run_items
    DROP COLUMN seq;
