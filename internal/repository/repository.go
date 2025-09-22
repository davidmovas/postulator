package repository

import (
	"Postulator/internal/schema"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	"github.com/adrg/xdg"
	br "github.com/lann/builder"
)

var (
	_ SiteRepository       = (*Repository)(nil)
	_ TopicRepository      = (*Repository)(nil)
	_ SiteTopicRepository  = (*Repository)(nil)
	_ TopicUsageRepository = (*Repository)(nil)
	_ PromptRepository     = (*Repository)(nil)
	_ SitePromptRepository = (*Repository)(nil)
)

var builder = squirrel.StatementBuilderType(br.EmptyBuilder).PlaceholderFormat(squirrel.Dollar)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

func InitDatabase() (*sql.DB, error) {
	dbPath := getDatabasePath()
	var err error

	// Ensure the directory exists
	if err = os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize database schema
	if err = initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return db, nil
}

// getDatabasePath returns the path where the database should be stored
func getDatabasePath() string {
	return filepath.Join(xdg.DataHome, "Postulator", "postulator.db")
}

func initSchema(db *sql.DB) error {
	return schema.InitSchema(db)
}
