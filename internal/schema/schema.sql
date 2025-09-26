CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(28) NOT NULL,
    url VARCHAR(255) NOT NULL,
    username VARCHAR(64) NOT NULL,
    password VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    last_check DATETIME,
    status VARCHAR(64) DEFAULT 'pending',
    strategy VARCHAR(64) DEFAULT 'unique',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(url)
);

CREATE TABLE IF NOT EXISTS topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    keywords VARCHAR(128),
    category VARCHAR(64),
    tags VARCHAR(128),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS site_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    priority INTEGER DEFAULT 1,
    last_used_at DATETIME,
    usage_count INTEGER DEFAULT 0,
    round_robin_pos INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(site_id, topic_id)
);

CREATE TABLE IF NOT EXISTS schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    cron_expr VARCHAR(28) NOT NULL,
    posts_per_day INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    last_run DATETIME,
    next_run DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    excerpt TEXT,
    keywords VARCHAR(128),
    tags VARCHAR(64),
    category VARCHAR(64),
    status VARCHAR(24) DEFAULT 'generated',
    wordpress_id INTEGER,
    gpt_model VARCHAR(24),
    tokens INTEGER DEFAULT 0,
    slug VARCHAR(255),
    outline TEXT,
    error_msg TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS posting_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL DEFAULT 'scheduled',
    site_id INTEGER NOT NULL,
    article_id INTEGER,
    status VARCHAR(24) DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    error_msg VARCHAR(255),
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(64) NOT NULL,
    system_prompt TEXT NOT NULL,
    user_prompt TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS site_prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    prompt_id INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE CASCADE,
    UNIQUE(site_id, prompt_id)
);

CREATE TABLE IF NOT EXISTS topic_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    article_id INTEGER NOT NULL,
    strategy TEXT NOT NULL,
    used_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_sites_status ON sites(status);
CREATE INDEX IF NOT EXISTS idx_sites_is_active ON sites(is_active);
CREATE INDEX IF NOT EXISTS idx_site_topics_site_id ON site_topics(site_id);
CREATE INDEX IF NOT EXISTS idx_site_topics_topic_id ON site_topics(topic_id);
CREATE INDEX IF NOT EXISTS idx_site_topics_usage_count ON site_topics(site_id, usage_count);
CREATE INDEX IF NOT EXISTS idx_site_topics_last_used ON site_topics(site_id, last_used_at);
CREATE INDEX IF NOT EXISTS idx_site_topics_round_robin ON site_topics(site_id, round_robin_pos);
CREATE INDEX IF NOT EXISTS idx_schedules_site_id ON schedules(site_id);
CREATE INDEX IF NOT EXISTS idx_schedules_is_active ON schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_articles_site_id ON articles(site_id);
CREATE INDEX IF NOT EXISTS idx_articles_topic_id ON articles(topic_id);
CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_articles_site_slug ON articles(site_id, slug);
CREATE INDEX IF NOT EXISTS idx_posting_jobs_status ON posting_jobs(status);
CREATE INDEX IF NOT EXISTS idx_posting_jobs_site_id ON posting_jobs(site_id);
CREATE INDEX IF NOT EXISTS idx_prompts_is_default ON prompts(is_default);
CREATE INDEX IF NOT EXISTS idx_site_prompts_site_id ON site_prompts(site_id);
CREATE INDEX IF NOT EXISTS idx_site_prompts_prompt_id ON site_prompts(prompt_id);

-- Insert default prompt if it doesn't exist
INSERT OR IGNORE INTO prompts (name, system_prompt, user_prompt, is_default) VALUES
(
    'Default Prompt',
    'Пиши структурированный текст с подзаголовками, без лишней воды. Минимум {min_words} слов. Язык сайта.',
    'Создай статью на тему: {topic_title}

Параметры сайта:
- Название сайта: {site_name}
- URL: {site_url}
- Категория: {category}
- Ключевые слова: {keywords}
- Теги: {tags}
- Тон: {tone}
- Стиль: {style}
- Минимум слов: {min_words}

Верни результат в формате JSON:
{
  "title": "заголовок статьи",
  "outline": ["раздел 1", "раздел 2", "раздел 3"],
  "content_html": "полный HTML контент статьи с подзаголовками",
  "excerpt": "краткое описание",
  "keywords": ["ключевое1", "ключевое2"],
  "tags": ["тег1", "тег2"]
}',
    TRUE
);