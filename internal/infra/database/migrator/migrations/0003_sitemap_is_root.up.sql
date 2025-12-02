-- +goose Up
ALTER TABLE sitemap_nodes ADD COLUMN is_root BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly
-- This would require recreating the table
