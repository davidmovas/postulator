-- Сайты
CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    wp_username TEXT NOT NULL,
    wp_password TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active', -- active, inactive, error
    last_health_check DATETIME,
    health_status TEXT DEFAULT 'unknown', -- healthy, unhealthy, unknown
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Категории WordPress для каждого сайта
CREATE TABLE site_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    wp_category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT,
    count INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    UNIQUE(site_id, wp_category_id)
);

-- Топики (тайтлы для генерации статей)
CREATE TABLE topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Связь топиков с сайтами и категориями (многие ко многим)
CREATE TABLE site_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL, -- обязательная категория для публикации
    strategy TEXT NOT NULL DEFAULT 'unique', -- unique, reuse_with_variation
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES site_categories(id) ON DELETE CASCADE,
    UNIQUE(site_id, topic_id)
);

-- Использованные топики (для отслеживания UNIQUE стратегии)
CREATE TABLE used_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(site_id, topic_id)
);

-- Промпты
CREATE TABLE prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    system_prompt TEXT NOT NULL,
    user_prompt TEXT NOT NULL,
    placeholders TEXT, -- JSON массив доступных плейсхолдеров: ["title", "category", "site_name"]
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- AI провайдеры и их конфигурация
CREATE TABLE ai_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE, -- openai, anthropic, etc
    api_key TEXT NOT NULL,
    model TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Джобы (задачи на генерацию и публикацию)
CREATE TABLE jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    site_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL, -- в какую категорию публиковать
    prompt_id INTEGER NOT NULL,
    ai_provider_id INTEGER NOT NULL,
    ai_model TEXT NOT NULL, -- модель AI для использования (gpt-4, gpt-3.5-turbo, claude-3-opus, etc)
    requires_validation BOOLEAN NOT NULL DEFAULT 0,
    schedule_type TEXT NOT NULL, -- manual, once, daily, weekly, monthly
    schedule_time TIME, -- время для daily (например 09:00:00)
    schedule_day INTEGER, -- день недели для weekly (1-7) или день месяца для monthly (1-31)
    jitter_enabled BOOLEAN NOT NULL DEFAULT 0,
    jitter_minutes INTEGER DEFAULT 30, -- +- минуты
    status TEXT NOT NULL DEFAULT 'active', -- active, paused, completed, error
    last_run_at DATETIME,
    next_run_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES site_categories(id) ON DELETE CASCADE,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE RESTRICT,
    FOREIGN KEY (ai_provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT
);

-- Топики привязанные к джобе (многие ко многим)
CREATE TABLE job_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE,
    UNIQUE(job_id, topic_id)
);

-- Опубликованные статьи
CREATE TABLE articles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    job_id INTEGER, -- может быть NULL если создано вручную
    topic_id INTEGER NOT NULL,
    title TEXT NOT NULL, -- финальный тайтл (может быть сгенерирован AI если стратегия reuse_with_variation)
    original_title TEXT NOT NULL, -- оригинальный тайтл из топика
    content TEXT NOT NULL,
    excerpt TEXT,
    wp_post_id INTEGER NOT NULL,
    wp_post_url TEXT NOT NULL,
    wp_category_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'published', -- draft, published, failed
    word_count INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE SET NULL,
    FOREIGN KEY (topic_id) REFERENCES topics(id) ON DELETE CASCADE
);

-- Ссылки в статьях (для отслеживания внутренней перелиновки и внешних ссылок)
CREATE TABLE article_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id INTEGER NOT NULL,
    link_type TEXT NOT NULL, -- internal, external
    target_article_id INTEGER, -- ID статьи на которую ссылаемся (если internal)
    url TEXT NOT NULL, -- полный URL
    anchor_text TEXT NOT NULL, -- текст гиперссылки
    position INTEGER, -- позиция в тексте (символ)
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
    FOREIGN KEY (target_article_id) REFERENCES articles(id) ON DELETE SET NULL
);

-- Выполнения джоб
CREATE TABLE job_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    generated_title TEXT,
    generated_content TEXT,
    status TEXT NOT NULL, -- pending, generating, pending_validation, validated, publishing, published, failed
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

-- Статистика по сайтам
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

-- Индексы для производительности
CREATE INDEX idx_sites_status ON sites(status);
CREATE INDEX idx_site_topics_site_id ON site_topics(site_id);
CREATE INDEX idx_site_topics_topic_id ON site_topics(topic_id);
CREATE INDEX idx_used_topics_site_id ON used_topics(site_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_next_run ON jobs(next_run_at) WHERE status = 'active';
CREATE INDEX idx_job_topics_job_id ON job_topics(job_id);
CREATE INDEX idx_articles_site_id ON articles(site_id);
CREATE INDEX idx_articles_job_id ON articles(job_id);
CREATE INDEX idx_articles_published_at ON articles(published_at);
CREATE INDEX idx_article_links_article_id ON article_links(article_id);
CREATE INDEX idx_article_links_target_article_id ON article_links(target_article_id);
CREATE INDEX idx_article_links_type ON article_links(link_type);
CREATE INDEX idx_job_executions_job_id ON job_executions(job_id);
CREATE INDEX idx_job_executions_status ON job_executions(status);
CREATE INDEX idx_site_statistics_site_date ON site_statistics(site_id, date);