-- +goose Up
ALTER TABLE page_links
    ADD COLUMN block_hash TEXT;

-- +goose Down
ALTER TABLE page_links
    DROP COLUMN block_hash;
