-- +goose Up
-- =========================================================================
-- TOPICS
-- =========================================================================

CREATE TABLE topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX idx_topics_title ON topics(title);

-- =========================================================================
-- SITE TOPICS (Many-to-Many)
-- =========================================================================

CREATE TABLE site_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(site_id, topic_id)
);

CREATE INDEX idx_site_topics_site ON site_topics(site_id);
CREATE INDEX idx_site_topics_topic ON site_topics(topic_id);

-- =========================================================================
-- USED TOPICS (Track which topics were used per site)
-- =========================================================================

CREATE TABLE used_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(site_id, topic_id)
);

CREATE INDEX idx_used_topics_site ON used_topics(site_id);
CREATE INDEX idx_used_topics_topic ON used_topics(topic_id);

-- +goose Down
DROP INDEX IF EXISTS idx_used_topics_topic;
DROP INDEX IF EXISTS idx_used_topics_site;
DROP TABLE IF EXISTS used_topics;

DROP INDEX IF EXISTS idx_site_topics_topic;
DROP INDEX IF EXISTS idx_site_topics_site;
DROP TABLE IF EXISTS site_topics;

DROP INDEX IF EXISTS idx_topics_title;
DROP TABLE IF EXISTS topics;
