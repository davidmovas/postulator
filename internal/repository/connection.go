package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
	dbPath, err := getDatabasePath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize database schema
	if err := initSchema(); err != nil {
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
func getDatabasePath() (string, error) {
	// Get user's application data directory
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	// Create application-specific directory
	appDir := filepath.Join(appDataDir, "Postulator")
	dbPath := filepath.Join(appDir, "postulator.db")

	return dbPath, nil
}

// initSchema initializes the database schema
func initSchema() error {
	// Create all tables for the application
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
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}
