-- +goose Up
-- Add SEO and WordPress fields to articles table
ALTER TABLE articles ADD COLUMN slug TEXT;
ALTER TABLE articles ADD COLUMN wp_tag_ids TEXT NOT NULL DEFAULT '[]';
ALTER TABLE articles ADD COLUMN featured_media_id INTEGER;
ALTER TABLE articles ADD COLUMN featured_media_url TEXT;
ALTER TABLE articles ADD COLUMN meta_description TEXT;
ALTER TABLE articles ADD COLUMN author INTEGER;

-- Create indexes for new fields
CREATE INDEX IF NOT EXISTS idx_articles_slug ON articles(slug);
CREATE INDEX IF NOT EXISTS idx_articles_author ON articles(author);

-- +goose Down
-- Remove indexes
DROP INDEX IF EXISTS idx_articles_slug;
DROP INDEX IF EXISTS idx_articles_author;

-- Note: SQLite has limited ALTER TABLE support, dropping columns requires table recreation
-- For safety, columns are not dropped in down migration
