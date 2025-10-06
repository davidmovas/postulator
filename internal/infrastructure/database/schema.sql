CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    wp_username TEXT NOT NULL,
    wp_password TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'inactive', 'error')),
    last_health_check DATETIME,
    health_status TEXT DEFAULT 'unknown' CHECK(health_status IN ('healthy', 'unhealthy', 'unknown')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sites_status ON sites(status);
CREATE INDEX idx_sites_health_status ON sites(health_status);

CREATE TABLE site_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    wp_category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    UNIQUE(site_id, wp_category_id)
);

CREATE INDEX idx_site_categories_site_id ON site_categories(site_id);

CREATE TABLE topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_topics_title ON topics(title);

CREATE TABLE site_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    strategy TEXT NOT NULL DEFAULT 'unique' CHECK(strategy IN ('unique', 'reuse_with_variation')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES site_categories(id) ON DELETE CASCADE,
    UNIQUE(site_id, topic_id)
);

CREATE INDEX idx_site_topics_site_id ON site_topics(site_id);
CREATE INDEX idx_site_topics_topic_id ON site_topics(topic_id);
CREATE INDEX idx_site_topics_category_id ON site_topics(category_id);
CREATE INDEX idx_site_topics_strategy ON site_topics(strategy);

CREATE TABLE used_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(site_id, topic_id)
);

CREATE INDEX idx_used_topics_site_id ON used_topics(site_id);
CREATE INDEX idx_used_topics_topic_id ON used_topics(topic_id);

CREATE TABLE prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    system_prompt TEXT NOT NULL,
    user_prompt TEXT NOT NULL,
    placeholders TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ai_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    api_key TEXT NOT NULL,
    model TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ai_providers_is_active ON ai_providers(is_active);

CREATE TABLE jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    site_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    prompt_id INTEGER NOT NULL,
    ai_provider_id INTEGER NOT NULL,
    requires_validation BOOLEAN NOT NULL DEFAULT 0,
    schedule_type TEXT NOT NULL CHECK(schedule_type IN ('manual', 'once', 'daily', 'weekly', 'monthly')),
    schedule_time TIME,
    schedule_day INTEGER,
    jitter_enabled BOOLEAN NOT NULL DEFAULT 0,
    jitter_minutes INTEGER DEFAULT 30,
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'paused', 'completed', 'error')),
    last_run_at DATETIME,
    next_run_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES site_categories(id) ON DELETE CASCADE,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE RESTRICT,
    FOREIGN KEY (ai_provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT
);

CREATE INDEX idx_jobs_site_id ON jobs(site_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_next_run_at ON jobs(next_run_at) WHERE status = 'active';
CREATE INDEX idx_jobs_schedule_type ON jobs(schedule_type);

CREATE TABLE job_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(job_id, topic_id)
);

CREATE INDEX idx_job_topics_job_id ON job_topics(job_id);
CREATE INDEX idx_job_topics_topic_id ON job_topics(topic_id);

CREATE TABLE articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    job_id INTEGER,
    topic_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    original_title TEXT NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    wp_post_id INTEGER NOT NULL,
    wp_post_url TEXT NOT NULL,
    wp_category_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'published' CHECK(status IN ('draft', 'published', 'failed')),
    word_count INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE SET NULL,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);

CREATE INDEX idx_articles_site_id ON articles(site_id);
CREATE INDEX idx_articles_job_id ON articles(job_id);
CREATE INDEX idx_articles_topic_id ON articles(topic_id);
CREATE INDEX idx_articles_status ON articles(status);
CREATE INDEX idx_articles_published_at ON articles(published_at);
CREATE INDEX idx_articles_wp_post_id ON articles(site_id, wp_post_id);

CREATE TABLE article_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    link_type TEXT NOT NULL CHECK(link_type IN ('internal', 'external')),
    target_article_id INTEGER,
    url TEXT NOT NULL,
    anchor_text TEXT NOT NULL,
    position INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    FOREIGN KEY (target_article_id) REFERENCES articles(id) ON DELETE SET NULL
);

CREATE INDEX idx_article_links_article_id ON article_links(article_id);
CREATE INDEX idx_article_links_target_article_id ON article_links(target_article_id);
CREATE INDEX idx_article_links_link_type ON article_links(link_type);

CREATE TABLE job_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    generated_title TEXT,
    generated_content TEXT,
    status TEXT NOT NULL CHECK(status IN ('pending', 'generating', 'pending_validation', 'validated', 'publishing', 'published', 'failed')),
    error_message TEXT,
    article_id INTEGER,
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    generated_at DATETIME,
    validated_at DATETIME,
    published_at DATETIME,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE SET NULL
);

CREATE INDEX idx_job_executions_job_id ON job_executions(job_id);
CREATE INDEX idx_job_executions_topic_id ON job_executions(topic_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);
CREATE INDEX idx_job_executions_article_id ON job_executions(article_id);

CREATE TABLE site_statistics (
                                 id INTEGER PRIMARY KEY AUTOINCREMENT,
                                 site_id INTEGER NOT NULL,
                                 date DATE NOT NULL,
                                 articles_published INTEGER DEFAULT 0,
                                 articles_failed INTEGER DEFAULT 0,
                                 total_words INTEGER DEFAULT 0,
                                 internal_links_created INTEGER DEFAULT 0,
                                 external_links_created INTEGER DEFAULT 0,
                                 FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
                                 UNIQUE(site_id, date)
);

CREATE INDEX idx_site_statistics_site_date ON site_statistics(site_id, date);

CREATE TRIGGER update_sites_updated_at
    AFTER UPDATE ON sites
BEGIN
UPDATE sites SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_prompts_updated_at
    AFTER UPDATE ON prompts
BEGIN
UPDATE prompts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_ai_providers_updated_at
    AFTER UPDATE ON ai_providers
BEGIN
UPDATE ai_providers SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_jobs_updated_at
    AFTER UPDATE ON jobs
BEGIN
UPDATE jobs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;