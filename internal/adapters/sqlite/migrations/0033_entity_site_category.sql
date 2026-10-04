-- +goose Up
ALTER TABLE entities
    ADD COLUMN site_category INTEGER NOT NULL DEFAULT 0 CHECK (site_category IN (0, 1));

-- +goose Down
ALTER TABLE entities
    DROP COLUMN site_category;
