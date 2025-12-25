-- +goose Up
-- =========================================================================
-- SITE STATISTICS
-- =========================================================================

CREATE TABLE site_statistics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    date DATETIME NOT NULL,
    articles_published INTEGER DEFAULT 0,
    articles_failed INTEGER DEFAULT 0,
    total_words INTEGER DEFAULT 0,
    internal_links_created INTEGER DEFAULT 0,
    external_links_created INTEGER DEFAULT 0,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    UNIQUE(site_id, date)
);

CREATE INDEX idx_site_statistics_site_date ON site_statistics(site_id, date);

-- =========================================================================
-- CATEGORY STATISTICS
-- =========================================================================

CREATE TABLE category_statistics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    date DATETIME NOT NULL,
    category_id INTEGER NOT NULL,
    articles_published INTEGER DEFAULT 0,
    total_words INTEGER DEFAULT 0,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
    UNIQUE(site_id, category_id, date)
);

CREATE INDEX idx_category_statistics_site_cat_date ON category_statistics(site_id, category_id, date);

-- +goose Down
DROP INDEX IF EXISTS idx_category_statistics_site_cat_date;
DROP TABLE IF EXISTS category_statistics;

DROP INDEX IF EXISTS idx_site_statistics_site_date;
DROP TABLE IF EXISTS site_statistics;
