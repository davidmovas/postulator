package database

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type DB struct {
	*sql.DB
}

func NewDB(filename string) (*DB, error) {
	dir := filepath.Dir(filename)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		var f *os.File
		f, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("create db file: %w", err)
		}
		_ = f.Close()
	}

	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	has, err := hasSchema(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if !has {
		if err = InitSchema(db); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return &DB{DB: db}, nil
}

func (d *DB) Close() {
	if d == nil || d.DB == nil {
		return
	}
	_ = d.DB.Close()
}

func hasSchema(db *sql.DB) (bool, error) {
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='sites' LIMIT 1").Scan(&name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if name == "" {
		return false, nil
	}
	return false, fmt.Errorf("check schema: %w", err)
}
