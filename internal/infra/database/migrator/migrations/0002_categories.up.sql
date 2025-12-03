-- +goose Up
-- =========================================================================
-- CATEGORIES
-- =========================================================================

CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    wp_category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT,
    description TEXT,
    count INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    UNIQUE(site_id, wp_category_id)
);

CREATE INDEX idx_categories_site ON categories(site_id);
CREATE INDEX idx_categories_wp_id ON categories(site_id, wp_category_id);

-- +goose Down
DROP INDEX IF EXISTS idx_categories_wp_id;
DROP INDEX IF EXISTS idx_categories_site;
DROP TABLE IF EXISTS categories;
