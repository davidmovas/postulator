-- ============================================================================
-- SITES
-- ============================================================================

CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    wp_username TEXT NOT NULL,
    wp_password TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    last_health_check DATETIME,
    health_status TEXT DEFAULT 'unknown',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (status IN ('active', 'inactive', 'error')),
    CHECK (health_status IN ('healthy', 'unhealthy', 'unknown'))
);

CREATE INDEX idx_sites_status ON sites(status);
CREATE INDEX idx_sites_health ON sites(health_status);

-- ============================================================================
-- CATEGORIES
-- ============================================================================

CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    wp_category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT,
    description TEXT,
    count INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    UNIQUE(site_id, wp_category_id)
);

CREATE INDEX idx_categories_site ON categories(site_id);
CREATE INDEX idx_categories_wp_id ON categories(site_id, wp_category_id);

-- ============================================================================
-- TOPICS
-- ============================================================================

CREATE TABLE topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATEtIME
);

CREATE INDEX idx_topics_title ON topics(title);

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

-- ============================================================================
-- AI PROVIDERS & PROMPTS
-- ============================================================================

CREATE TABLE ai_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    api_key TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (provider IN ('openai', 'anthropic', 'google'))
);

CREATE INDEX idx_ai_providers_active ON ai_providers(is_active);

CREATE TABLE prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    system_prompt TEXT NOT NULL,
    user_prompt TEXT NOT NULL,
    placeholders TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- JOBS
-- ============================================================================

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

CREATE TABLE job_state (
    job_id INTEGER PRIMARY KEY,
    last_run_at DATETIME,
    next_run_at DATETIME,
    total_executions INTEGER DEFAULT 0,
    failed_executions INTEGER DEFAULT 0,
    last_category_index INTEGER DEFAULT 0,

    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_job_state_next_run ON job_state(next_run_at) WHERE next_run_at IS NOT NULL;

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

-- ============================================================================
-- JOB EXECUTIONS
-- ============================================================================

CREATE TABLE job_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    article_id INTEGER,

    prompt_id INTEGER NOT NULL,
    ai_provider_id INTEGER NOT NULL,
    ai_model TEXT NOT NULL,
    category_id INTEGER NOT NULL,

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
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE SET NULL,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE RESTRICT,
    FOREIGN KEY (ai_provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
);

CREATE INDEX idx_job_executions_job ON job_executions(job_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);
CREATE INDEX idx_job_executions_article ON job_executions(article_id);

-- ============================================================================
-- ARTICLES
-- ============================================================================

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
    wp_category_ids TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'published',
    source TEXT NOT NULL DEFAULT 'generated',
    is_edited BOOLEAN NOT NULL DEFAULT 0,
    word_count INTEGER,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at DATETIME,

    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE SET NULL,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);

CREATE INDEX idx_articles_site ON articles(site_id);
CREATE INDEX idx_articles_job ON articles(job_id);
CREATE INDEX idx_articles_topic ON articles(topic_id);
CREATE INDEX idx_articles_status ON articles(status);
CREATE INDEX idx_articles_source ON articles(source);
CREATE INDEX idx_articles_published ON articles(published_at);
CREATE INDEX idx_articles_wp_post ON articles(site_id, wp_post_id);

-- ============================================================================
-- ARTICLE LINKS
-- ============================================================================

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
    FOREIGN KEY (task_id) REFERENCES interlinking_tasks(id) ON DELETE SET NULL,

    CHECK (link_type IN ('internal', 'external'))
);

CREATE INDEX idx_article_links_article ON article_links(article_id);
CREATE INDEX idx_article_links_target ON article_links(target_article_id);
CREATE INDEX idx_article_links_type ON article_links(link_type);
CREATE INDEX idx_article_links_task ON article_links(task_id);

-- ============================================================================
-- INTERLINKING
-- ============================================================================

CREATE TABLE interlinking_tasks (
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

CREATE INDEX idx_interlinking_tasks_status ON interlinking_tasks(status);

CREATE TABLE interlinking_proposals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    source_article_id INTEGER NOT NULL,
    target_article_id INTEGER NOT NULL,
    anchor_text TEXT NOT NULL,
    position INTEGER NOT NULL,
    confidence REAL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (task_id) REFERENCES interlinking_tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (source_article_id) REFERENCES articles(id) ON DELETE CASCADE,
    FOREIGN KEY (target_article_id) REFERENCES articles(id) ON DELETE CASCADE
);

CREATE INDEX idx_interlinking_proposals_task ON interlinking_proposals(task_id);
CREATE INDEX idx_interlinking_proposals_source ON interlinking_proposals(source_article_id);
CREATE INDEX idx_interlinking_proposals_target ON interlinking_proposals(target_article_id);

-- ============================================================================
-- STATISTICS
-- ============================================================================

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