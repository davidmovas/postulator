-- +goose Up
-- Add columns for storing original WP data to track local modifications
ALTER TABLE sitemap_nodes ADD COLUMN wp_title TEXT;
ALTER TABLE sitemap_nodes ADD COLUMN wp_slug TEXT;

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly, but goose will handle this
