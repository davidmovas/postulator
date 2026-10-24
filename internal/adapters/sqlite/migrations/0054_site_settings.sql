-- +goose Up
ALTER TABLE sites
    ADD COLUMN settings TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE sites
    DROP COLUMN settings;
