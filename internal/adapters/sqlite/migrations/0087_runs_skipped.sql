-- +goose Up
ALTER TABLE runs ADD COLUMN skipped TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE runs DROP COLUMN skipped;
