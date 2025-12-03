-- +goose Up
-- =========================================================================
-- JOB EXECUTIONS
-- =========================================================================

CREATE TABLE job_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    article_id INTEGER,

    prompt_id INTEGER NOT NULL,
    ai_provider_id INTEGER NOT NULL,
    ai_model TEXT NOT NULL,
    category_ids TEXT NOT NULL,

    status TEXT NOT NULL,
    error_message TEXT,

    generation_time_ms INTEGER,
    tokens_used INTEGER,

    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    generated_at DATETIME,
    validated_at DATETIME,
    published_at DATETIME,
    completed_at DATETIME,

    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE SET NULL,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE RESTRICT,
    FOREIGN KEY (ai_provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT
);

CREATE INDEX idx_job_executions_job ON job_executions(job_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);
CREATE INDEX idx_job_executions_article ON job_executions(article_id);

-- +goose Down
DROP INDEX IF EXISTS idx_job_executions_article;
DROP INDEX IF EXISTS idx_job_executions_status;
DROP INDEX IF EXISTS idx_job_executions_job;
DROP TABLE IF EXISTS job_executions;
