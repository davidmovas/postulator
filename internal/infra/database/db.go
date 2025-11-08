package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/davidmovas/postulator/internal/infra/database/migrator"
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

	if err = migrator.Apply(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return &DB{DB: db}, nil
}

func (d *DB) Close() {
	if d == nil || d.DB == nil {
		return
	}
	_ = d.DB.Close()
}
