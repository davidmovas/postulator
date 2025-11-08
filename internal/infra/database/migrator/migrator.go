package migrator

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*
var migrationsFS embed.FS

func Apply(ctx context.Context, db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	goose.SetBaseFS(migrationsFS)

	hasGoose, err := tableExists(ctx, db, "goose_db_version")
	if err != nil {
		return err
	}
	if !hasGoose {
		hasSites, err2 := tableExists(ctx, db, "sites")
		if err2 != nil {
			return err2
		}
		if hasSites {
			if err = createGooseVersionTable(ctx, db); err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO goose_db_version (version_id, is_applied) VALUES (1, TRUE)"); err != nil {
				return fmt.Errorf("baseline goose version insert: %w", err)
			}
		}
	}

	if err = goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var n string
	err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=? LIMIT 1", name).Scan(&n)
	switch {
	case err == nil && n != "":
		return true, nil
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("check table %s: %w", name, err)
	default:
		return n != "", nil
	}
}

func createGooseVersionTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version_id BIGINT NOT NULL,
			is_applied BOOLEAN NOT NULL,
			tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create goose version table: %w", err)
	}
	return nil
}
