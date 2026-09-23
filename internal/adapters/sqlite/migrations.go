package sqlite

import (
	"embed"
	"io/fs"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func migrations() (fs.FS, error) {
	fsys, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "read embedded migrations")
	}
	return fsys, nil
}
