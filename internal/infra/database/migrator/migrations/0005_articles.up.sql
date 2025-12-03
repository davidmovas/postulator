-- +goose Up
-- =========================================================================
-- ARTICLES
-- =========================================================================

CREATE TABLE articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    job_id INTEGER,
    topic_id INTEGER,

    title TEXT NOT NULL,
    original_title TEXT NOT NULL,
    slug TEXT,
    content TEXT NOT NULL,
    excerpt TEXT,
    meta_description TEXT,

    wp_post_id INTEGER NOT NULL DEFAULT 0,
    wp_post_url TEXT NOT NULL DEFAULT '',
    wp_category_ids TEXT NOT NULL DEFAULT '[]',
    wp_tag_ids TEXT NOT NULL DEFAULT '[]',
    featured_media_id INTEGER,
    featured_media_url TEXT,
    author INTEGER,

    status TEXT NOT NULL DEFAULT 'draft',
    source TEXT NOT NULL DEFAULT 'generated',
    is_edited BOOLEAN NOT NULL DEFAULT 0,
    word_count INTEGER,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at DATETIME,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE SET NULL
);

CREATE INDEX idx_articles_site ON articles(site_id);
CREATE INDEX idx_articles_job ON articles(job_id);
CREATE INDEX idx_articles_topic ON articles(topic_id);
CREATE INDEX idx_articles_status ON articles(status);
CREATE INDEX idx_articles_source ON articles(source);
CREATE INDEX idx_articles_published ON articles(published_at);
CREATE INDEX idx_articles_wp_post ON articles(site_id, wp_post_id);
CREATE INDEX idx_articles_slug ON articles(slug);
CREATE INDEX idx_articles_author ON articles(author);

-- Partial unique index: only enforce uniqueness for published articles (wp_post_id > 0)
CREATE UNIQUE INDEX idx_articles_site_wp_post_unique ON articles(site_id, wp_post_id) WHERE wp_post_id > 0;

-- +goose Down
DROP INDEX IF EXISTS idx_articles_site_wp_post_unique;
DROP INDEX IF EXISTS idx_articles_author;
DROP INDEX IF EXISTS idx_articles_slug;
DROP INDEX IF EXISTS idx_articles_wp_post;
DROP INDEX IF EXISTS idx_articles_published;
DROP INDEX IF EXISTS idx_articles_source;
DROP INDEX IF EXISTS idx_articles_status;
DROP INDEX IF EXISTS idx_articles_topic;
DROP INDEX IF EXISTS idx_articles_job;
DROP INDEX IF EXISTS idx_articles_site;
DROP TABLE IF EXISTS articles;
