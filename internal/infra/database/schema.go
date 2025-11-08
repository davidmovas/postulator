package database

import (
	"database/sql"
)

// Deprecated: migrations are handled by pressly/goose in internal/infra/database/migrator.
// This function is kept for backward compatibility and does nothing.
func InitSchema(_ *sql.DB) error {
	return nil
}
