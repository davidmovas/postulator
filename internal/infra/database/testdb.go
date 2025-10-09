package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func SetupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tempDir := filepath.Join(os.TempDir(), "postulator_test_db")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, fmt.Sprintf("test_%s.db", t.Name()))

	_ = os.Remove(dbPath)

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		_ = os.Remove(dbPath)
	}

	return db, cleanup
}
