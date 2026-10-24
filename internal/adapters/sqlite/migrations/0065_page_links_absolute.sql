-- +goose Up
ALTER TABLE page_links
    ADD COLUMN absolute INTEGER NOT NULL DEFAULT 0 CHECK (absolute IN (0, 1));

-- +goose Down
ALTER TABLE page_links
    DROP COLUMN absolute;
