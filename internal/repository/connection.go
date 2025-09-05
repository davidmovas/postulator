package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var db *sql.DB

// GetDB returns the database connection instance
func GetDB() *sql.DB {
	return db
}

// InitDatabase initializes the SQLite database connection
func InitDatabase() error {
	dbPath := getDatabasePath()
	var err error

	// Ensure the directory exists
	if err = os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize database schema
	if err = initSchema(); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
}

// CloseDatabase closes the database connection
func CloseDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// getDatabasePath returns the path where the database should be stored
func getDatabasePath() string {
	return filepath.Join(xdg.DataHome, "Postulator", "postulator.db")
}

// initSchema initializes the database schema
func initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    api_key TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    last_check DATETIME,
    status TEXT DEFAULT 'pending',
    strategy TEXT DEFAULT 'random',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT,
    keywords TEXT,
    prompt TEXT,
    category TEXT,
    tags TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS site_topics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    topic_id INTEGER NOT NULL,
    priority INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
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
    cron_expr TEXT NOT NULL,
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
    title TEXT NOT NULL,
    content TEXT,
    excerpt TEXT,
    keywords TEXT,
    tags TEXT,
    category TEXT,
    status TEXT DEFAULT 'generated',
    wordpress_id INTEGER,
    gpt_model TEXT,
    tokens INTEGER DEFAULT 0,
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
    status TEXT DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    error_msg TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('system', 'user')),
    content TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
CREATE INDEX IF NOT EXISTS idx_topics_is_active ON topics(is_active);
CREATE INDEX IF NOT EXISTS idx_site_topics_site_id ON site_topics(site_id);
CREATE INDEX IF NOT EXISTS idx_site_topics_topic_id ON site_topics(topic_id);
CREATE INDEX IF NOT EXISTS idx_schedules_site_id ON schedules(site_id);
CREATE INDEX IF NOT EXISTS idx_schedules_is_active ON schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_articles_site_id ON articles(site_id);
CREATE INDEX IF NOT EXISTS idx_articles_topic_id ON articles(topic_id);
CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status);
CREATE INDEX IF NOT EXISTS idx_posting_jobs_status ON posting_jobs(status);
CREATE INDEX IF NOT EXISTS idx_posting_jobs_site_id ON posting_jobs(site_id);
CREATE INDEX IF NOT EXISTS idx_prompts_type ON prompts(type);
CREATE INDEX IF NOT EXISTS idx_prompts_is_default ON prompts(is_default);
CREATE INDEX IF NOT EXISTS idx_prompts_is_active ON prompts(is_active);

-- Insert default prompts if they don't exist
INSERT OR IGNORE INTO prompts (name, type, content, is_default, description, is_active) VALUES
(
    'Default System Prompt',
    'system',
    'You are a professional content writer who creates high-quality, SEO-optimized articles for WordPress websites. You must respond with valid JSON matching the provided schema. Focus on creating engaging, informative content that provides value to readers while following SEO best practices.',
    TRUE,
    'Default system prompt for article generation',
    TRUE
),
(
    'Default User Prompt',
    'user',
    'Please write a comprehensive article about: {{title}}

Topic details:
{{prompt}}

Requirements:
- Write a complete HTML article with minimum 800 words
- Create an SEO-optimized title based on: {{title}}
- Include a brief excerpt (150-200 characters) 
- Provide relevant SEO keywords
- Suggest appropriate WordPress tags
- Assign a suitable category
- Use proper HTML formatting with headings, paragraphs, and lists where appropriate
- Make the content engaging and informative
- Ensure the article provides genuine value to readers

Please respond with valid JSON matching the provided schema.',
    TRUE,
    'Default user prompt for article generation with placeholder support',
    TRUE
);
`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}
