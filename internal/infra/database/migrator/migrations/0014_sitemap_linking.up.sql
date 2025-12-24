-- +goose Up
-- =========================================================================
-- SITEMAP LINK PLANS
-- =========================================================================

CREATE TABLE link_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sitemap_id INTEGER NOT NULL,
    site_id INTEGER NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    provider_id INTEGER,
    prompt_id INTEGER,
    error TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (sitemap_id) REFERENCES sitemaps(id) ON DELETE CASCADE,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (provider_id) REFERENCES ai_providers(id) ON DELETE SET NULL,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE SET NULL
);

CREATE INDEX idx_link_plans_sitemap ON link_plans(sitemap_id);
CREATE INDEX idx_link_plans_site ON link_plans(site_id);
CREATE INDEX idx_link_plans_status ON link_plans(status);

-- =========================================================================
-- PLANNED LINKS (for sitemap nodes)
-- =========================================================================

CREATE TABLE planned_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    source_node_id INTEGER NOT NULL,
    target_node_id INTEGER NOT NULL,
    anchor_text TEXT,
    anchor_context TEXT,
    status TEXT NOT NULL DEFAULT 'planned',
    source TEXT NOT NULL DEFAULT 'manual',
    position INTEGER,
    confidence REAL,
    error TEXT,
    applied_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (plan_id) REFERENCES link_plans(id) ON DELETE CASCADE,
    FOREIGN KEY (source_node_id) REFERENCES sitemap_nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (target_node_id) REFERENCES sitemap_nodes(id) ON DELETE CASCADE,

    UNIQUE(plan_id, source_node_id, target_node_id)
);

CREATE INDEX idx_planned_links_plan ON planned_links(plan_id);
CREATE INDEX idx_planned_links_source ON planned_links(source_node_id);
CREATE INDEX idx_planned_links_target ON planned_links(target_node_id);
CREATE INDEX idx_planned_links_status ON planned_links(status);

-- +goose Down
DROP INDEX IF EXISTS idx_planned_links_status;
DROP INDEX IF EXISTS idx_planned_links_target;
DROP INDEX IF EXISTS idx_planned_links_source;
DROP INDEX IF EXISTS idx_planned_links_plan;
DROP TABLE IF EXISTS planned_links;

DROP INDEX IF EXISTS idx_link_plans_status;
DROP INDEX IF EXISTS idx_link_plans_site;
DROP INDEX IF EXISTS idx_link_plans_sitemap;
DROP TABLE IF EXISTS link_plans;
