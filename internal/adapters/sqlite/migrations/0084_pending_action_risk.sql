-- +goose Up
ALTER TABLE pending_actions ADD COLUMN risk TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE pending_actions DROP COLUMN risk;
