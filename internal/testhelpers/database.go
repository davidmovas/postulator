package testhelpers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"Postulator/internal/schema"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// TestDB represents a test database instance
type TestDB struct {
	DB   *sql.DB
	Path string
}

// SetupTestDB creates a temporary SQLite database for testing
func SetupTestDB(t *testing.T) *TestDB {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.db", time.Now().UnixNano()))

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Initialize schema using production schema
	if err = schema.InitSchema(db); err != nil {
		_ = db.Close()
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	return &TestDB{
		DB:   db,
		Path: dbPath,
	}
}

// Close closes the test database and cleans up the file
func (tdb *TestDB) Close() error {
	if tdb.DB != nil {
		if err := tdb.DB.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	// Remove the database file
	if tdb.Path != "" {
		if err := os.Remove(tdb.Path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove database file: %w", err)
		}
	}

	return nil
}

// ClearAllTables clears all data from test database tables
func (tdb *TestDB) ClearAllTables(t *testing.T) {
	tables := []string{
		"topic_usage",
		"posting_jobs",
		"articles",
		"schedules",
		"site_topics",
		"site_prompts",
		"topics",
		"sites",
		"prompts",
		"settings",
	}

	for _, table := range tables {
		_, err := tdb.DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Fatalf("Failed to clear table %s: %v", table, err)
		}
	}
}

// InsertTestData inserts common test data
func (tdb *TestDB) InsertTestData(t *testing.T) {
	// Insert test sites
	_, err := tdb.DB.Exec(`
		INSERT INTO sites (id, name, url, username, password, is_active, last_check, status, strategy, created_at, updated_at) 
		VALUES 
		(1, 'Test Site 1', 'https://test1.com', 'user1', 'pass1', 1, datetime('now'), 'connected', 'random', datetime('now'), datetime('now')),
		(2, 'Test Site 2', 'https://test2.com', 'user2', 'pass2', 1, datetime('now'), 'connected', 'unique', datetime('now'), datetime('now')),
		(3, 'Inactive Site', 'https://test3.com', 'user3', 'pass3', 0, datetime('now'), 'error', 'round_robin', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert test sites: %v", err)
	}

	// Insert test topics
	_, err = tdb.DB.Exec(`
		INSERT INTO topics (id, title, keywords, category, tags, is_active, created_at, updated_at)
		VALUES 
		(1, 'AI Technology', 'ai,ml,tech', 'Technology', 'ai,tech', 1, datetime('now'), datetime('now')),
		(2, 'Web Development', 'web,dev,js', 'Programming', 'web,dev', 1, datetime('now'), datetime('now')),
		(3, 'Inactive Topic', 'inactive', 'Test', 'test', 0, datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert test topics: %v", err)
	}

	// Insert test site_topics associations
	_, err = tdb.DB.Exec(`
		INSERT INTO site_topics (site_id, topic_id, priority, is_active, last_used_at, usage_count, round_robin_pos, created_at, updated_at)
		VALUES 
		(1, 1, 1, 1, datetime('now'), 1, 0, datetime('now'), datetime('now')),
		(1, 2, 2, 1, datetime('now'), 1, 0, datetime('now'), datetime('now')),
		(2, 1, 1, 1, datetime('now'), 2, 1, datetime('now'), datetime('now')),
		(2, 2, 1, 1, datetime('now'), 1, 0, datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert test site_topics: %v", err)
	}
}
