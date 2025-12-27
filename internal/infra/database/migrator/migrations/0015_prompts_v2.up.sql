-- +goose Up
-- =========================================================================
-- PROMPTS V2: Instructions + Context Config
-- =========================================================================

-- Add new columns for v2 format
ALTER TABLE prompts ADD COLUMN instructions TEXT NOT NULL DEFAULT '';
ALTER TABLE prompts ADD COLUMN context_config TEXT;
ALTER TABLE prompts ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- Create index for version
CREATE INDEX idx_prompts_version ON prompts(version);

-- +goose Down
DROP INDEX IF EXISTS idx_prompts_version;

-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- First, create a backup table with the original schema
CREATE TABLE prompts_backup (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'post_gen'
        CHECK (category IN ('post_gen', 'page_gen', 'link_suggest', 'link_apply', 'sitemap_gen')),
    is_builtin BOOLEAN NOT NULL DEFAULT 0,
    system_prompt TEXT NOT NULL,
    user_prompt TEXT NOT NULL,
    placeholders TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Copy data back (without new columns)
INSERT INTO prompts_backup (id, name, category, is_builtin, system_prompt, user_prompt, placeholders, created_at, updated_at)
SELECT id, name, category, is_builtin, system_prompt, user_prompt, placeholders, created_at, updated_at
FROM prompts;

-- Drop original table and recreate index
DROP TABLE prompts;
ALTER TABLE prompts_backup RENAME TO prompts;
CREATE INDEX idx_prompts_category ON prompts(category);
