package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"Postulator/internal/schema"

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
	return schema.InitSchema(db)
}

// InitSchemaForDB initializes the database schema for the provided database connection
func InitSchemaForDB(database *sql.DB) error {
	return schema.InitSchema(database)
}
