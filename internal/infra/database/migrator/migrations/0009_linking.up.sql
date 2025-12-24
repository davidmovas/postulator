-- +goose Up
-- =========================================================================
-- LINKING TASKS
-- =========================================================================

CREATE TABLE linking_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    site_ids TEXT NOT NULL,
    article_ids TEXT,

    max_links_per_article INTEGER DEFAULT 3,
    min_link_distance INTEGER DEFAULT 100,

    prompt_id INTEGER,
    ai_provider_id INTEGER NOT NULL,

    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    applied_at DATETIME,

    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE SET NULL,
    FOREIGN KEY (ai_provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT
);

CREATE INDEX idx_linking_tasks_status ON linking_tasks(status);

-- =========================================================================
-- LINKING PROPOSALS
-- =========================================================================

CREATE TABLE linking_proposals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    source_article_id INTEGER NOT NULL,
    target_article_id INTEGER NOT NULL,
    anchor_text TEXT NOT NULL,
    position INTEGER NOT NULL,
    confidence REAL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES linking_tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (source_article_id) REFERENCES articles(id) ON DELETE CASCADE,
    FOREIGN KEY (target_article_id) REFERENCES articles(id) ON DELETE CASCADE
);

CREATE INDEX idx_linking_proposals_task ON linking_proposals(task_id);
CREATE INDEX idx_linking_proposals_source ON linking_proposals(source_article_id);
CREATE INDEX idx_linking_proposals_target ON linking_proposals(target_article_id);

-- =========================================================================
-- ARTICLE LINKS
-- =========================================================================

CREATE TABLE article_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    link_type TEXT NOT NULL,
    target_article_id INTEGER,
    url TEXT NOT NULL,
    anchor_text TEXT NOT NULL,
    position INTEGER,
    task_id INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    FOREIGN KEY (target_article_id) REFERENCES articles(id) ON DELETE SET NULL,
    FOREIGN KEY (task_id) REFERENCES linking_tasks(id) ON DELETE SET NULL,

    CHECK (link_type IN ('internal', 'external'))
);

CREATE INDEX idx_article_links_article ON article_links(article_id);
CREATE INDEX idx_article_links_target ON article_links(target_article_id);
CREATE INDEX idx_article_links_type ON article_links(link_type);
CREATE INDEX idx_article_links_task ON article_links(task_id);

-- +goose Down
DROP INDEX IF EXISTS idx_article_links_task;
DROP INDEX IF EXISTS idx_article_links_type;
DROP INDEX IF EXISTS idx_article_links_target;
DROP INDEX IF EXISTS idx_article_links_article;
DROP TABLE IF EXISTS article_links;

DROP INDEX IF EXISTS idx_linking_proposals_target;
DROP INDEX IF EXISTS idx_linking_proposals_source;
DROP INDEX IF EXISTS idx_linking_proposals_task;
DROP TABLE IF EXISTS linking_proposals;

DROP INDEX IF EXISTS idx_linking_tasks_status;
DROP TABLE IF EXISTS linking_tasks;
