-- +goose Up
-- =========================================================================
-- SITEMAPS (Site Structure Trees)
-- =========================================================================

CREATE TABLE sitemaps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    source TEXT NOT NULL DEFAULT 'manual',
    status TEXT NOT NULL DEFAULT 'draft',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,

    CHECK (source IN ('manual', 'imported', 'generated', 'scanned')),
    CHECK (status IN ('draft', 'active', 'archived'))
);

CREATE INDEX idx_sitemaps_site ON sitemaps(site_id);
CREATE INDEX idx_sitemaps_status ON sitemaps(status);

-- =========================================================================
-- SITEMAP NODES (Tree Nodes / Pages)
-- =========================================================================

CREATE TABLE sitemap_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sitemap_id INTEGER NOT NULL,
    parent_id INTEGER,

    -- Node identification
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    is_root BOOLEAN NOT NULL DEFAULT FALSE,

    -- Tree structure
    depth INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    path TEXT NOT NULL DEFAULT '/',

    -- Content type and linking
    content_type TEXT NOT NULL DEFAULT 'none',
    article_id INTEGER,
    wp_page_id INTEGER,
    wp_url TEXT,

    -- Source tracking
    source TEXT NOT NULL DEFAULT 'manual',

    -- Sync status
    is_synced BOOLEAN NOT NULL DEFAULT FALSE,
    last_synced_at DATETIME,
    wp_title TEXT,
    wp_slug TEXT,

    -- Content status
    content_status TEXT NOT NULL DEFAULT 'none',

    -- React Flow positions
    position_x REAL,
    position_y REAL,

    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (sitemap_id) REFERENCES sitemaps(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES sitemap_nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE SET NULL,

    CHECK (source IN ('manual', 'imported', 'generated', 'scanned')),
    CHECK (content_type IN ('page', 'post', 'none')),
    CHECK (content_status IN ('none', 'ai_draft', 'pending', 'draft', 'published'))
);

CREATE INDEX idx_sitemap_nodes_sitemap ON sitemap_nodes(sitemap_id);
CREATE INDEX idx_sitemap_nodes_parent ON sitemap_nodes(parent_id);
CREATE INDEX idx_sitemap_nodes_article ON sitemap_nodes(article_id);
CREATE INDEX idx_sitemap_nodes_depth ON sitemap_nodes(sitemap_id, depth);
CREATE INDEX idx_sitemap_nodes_path ON sitemap_nodes(path);
CREATE INDEX idx_sitemap_nodes_slug ON sitemap_nodes(sitemap_id, slug);
CREATE INDEX idx_sitemap_nodes_content_status ON sitemap_nodes(content_status);

-- Unique slug within same parent (siblings can't have same slug)
CREATE UNIQUE INDEX idx_sitemap_nodes_unique_slug
    ON sitemap_nodes(sitemap_id, COALESCE(parent_id, 0), slug);

-- =========================================================================
-- SITEMAP NODE KEYWORDS
-- =========================================================================

CREATE TABLE sitemap_node_keywords (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL,
    keyword TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (node_id) REFERENCES sitemap_nodes(id) ON DELETE CASCADE
);

CREATE INDEX idx_sitemap_node_keywords_node ON sitemap_node_keywords(node_id);
CREATE INDEX idx_sitemap_node_keywords_keyword ON sitemap_node_keywords(keyword);

-- Prevent duplicate keywords per node
CREATE UNIQUE INDEX idx_sitemap_node_keywords_unique ON sitemap_node_keywords(node_id, keyword);

-- +goose Down
DROP INDEX IF EXISTS idx_sitemap_node_keywords_unique;
DROP INDEX IF EXISTS idx_sitemap_node_keywords_keyword;
DROP INDEX IF EXISTS idx_sitemap_node_keywords_node;
DROP TABLE IF EXISTS sitemap_node_keywords;

DROP INDEX IF EXISTS idx_sitemap_nodes_unique_slug;
DROP INDEX IF EXISTS idx_sitemap_nodes_content_status;
DROP INDEX IF EXISTS idx_sitemap_nodes_slug;
DROP INDEX IF EXISTS idx_sitemap_nodes_path;
DROP INDEX IF EXISTS idx_sitemap_nodes_depth;
DROP INDEX IF EXISTS idx_sitemap_nodes_article;
DROP INDEX IF EXISTS idx_sitemap_nodes_parent;
DROP INDEX IF EXISTS idx_sitemap_nodes_sitemap;
DROP TABLE IF EXISTS sitemap_nodes;

DROP INDEX IF EXISTS idx_sitemaps_status;
DROP INDEX IF EXISTS idx_sitemaps_site;
DROP TABLE IF EXISTS sitemaps;
