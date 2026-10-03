-- +goose Up
ALTER TABLE sites
    ADD COLUMN commerce TEXT NOT NULL DEFAULT '' CHECK (commerce IN ('', 'absent', 'forbidden', 'ready'));

-- +goose Down
ALTER TABLE sites
    DROP COLUMN commerce;
