-- +goose Up
-- =========================================================================
-- SITES
-- =========================================================================

CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    wp_username TEXT NOT NULL,
    wp_password TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    last_health_check DATETIME,
    auto_health_check BOOLEAN NOT NULL DEFAULT 0,
    health_status TEXT DEFAULT 'unknown',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (status IN ('active', 'inactive', 'error')),
    CHECK (health_status IN ('healthy', 'unhealthy', 'unknown', 'error'))
);

CREATE INDEX idx_sites_status ON sites(status);
CREATE INDEX idx_sites_health ON sites(health_status);

-- =========================================================================
-- HEALTH CHECK HISTORY
-- =========================================================================

CREATE TABLE health_check_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    checked_at DATETIME NOT NULL,
    status TEXT NOT NULL,
    response_time_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    CHECK (status IN ('healthy', 'unhealthy', 'error'))
);

CREATE INDEX idx_health_check_site_date ON health_check_history(site_id, checked_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_health_check_site_date;
DROP TABLE IF EXISTS health_check_history;

DROP INDEX IF EXISTS idx_sites_health;
DROP INDEX IF EXISTS idx_sites_status;
DROP TABLE IF EXISTS sites;
