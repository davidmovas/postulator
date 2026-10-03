-- +goose Up
ALTER TABLE pages
    ADD COLUMN planned_path TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE pages
    DROP COLUMN planned_path;
