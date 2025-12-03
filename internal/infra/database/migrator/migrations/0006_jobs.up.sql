-- +goose Up
-- =========================================================================
-- JOBS
-- =========================================================================

CREATE TABLE jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    site_id INTEGER NOT NULL,
    prompt_id INTEGER NOT NULL,
    ai_provider_id INTEGER NOT NULL,
    placeholders_values TEXT,

    topic_strategy TEXT NOT NULL DEFAULT 'unique',
    category_strategy TEXT NOT NULL DEFAULT 'fixed',
    requires_validation BOOLEAN NOT NULL DEFAULT 0,

    schedule_type TEXT NOT NULL,
    schedule_config TEXT,

    jitter_enabled BOOLEAN NOT NULL DEFAULT 0,
    jitter_minutes INTEGER DEFAULT 30,

    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE RESTRICT,
    FOREIGN KEY (ai_provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT,

    CHECK (status IN ('active', 'paused', 'completed'))
);

CREATE INDEX idx_jobs_site ON jobs(site_id);
CREATE INDEX idx_jobs_status ON jobs(status);

-- =========================================================================
-- JOB STATE
-- =========================================================================

CREATE TABLE job_state (
    job_id INTEGER PRIMARY KEY,
    last_run_at DATETIME,
    next_run_at DATETIME,
    next_run_base DATETIME,
    total_executions INTEGER DEFAULT 0,
    failed_executions INTEGER DEFAULT 0,
    last_category_index INTEGER DEFAULT 0,

    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_job_state_next_run ON job_state(next_run_at) WHERE next_run_at IS NOT NULL;

-- =========================================================================
-- JOB CATEGORIES
-- =========================================================================

CREATE TABLE job_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    order_index INTEGER NOT NULL DEFAULT 0,

    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
    UNIQUE(job_id, category_id)
);

CREATE INDEX idx_job_categories_job ON job_categories(job_id);
CREATE INDEX idx_job_categories_order ON job_categories(job_id, order_index);

-- =========================================================================
-- JOB TOPICS
-- =========================================================================

CREATE TABLE job_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    order_index INTEGER NOT NULL DEFAULT 0,

    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(job_id, topic_id)
);

CREATE INDEX idx_job_topics_job ON job_topics(job_id);
CREATE INDEX idx_job_topics_order ON job_topics(job_id, order_index);

-- +goose Down
DROP INDEX IF EXISTS idx_job_topics_order;
DROP INDEX IF EXISTS idx_job_topics_job;
DROP TABLE IF EXISTS job_topics;

DROP INDEX IF EXISTS idx_job_categories_order;
DROP INDEX IF EXISTS idx_job_categories_job;
DROP TABLE IF EXISTS job_categories;

DROP INDEX IF EXISTS idx_job_state_next_run;
DROP TABLE IF EXISTS job_state;

DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_site;
DROP TABLE IF EXISTS jobs;
