package sqlite

import (
	"context"
	stderrors "errors"
	"net/url"
	"path/filepath"

	sqlitedriver "github.com/ncruces/go-sqlite3/driver"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Store) Snapshot(ctx context.Context, path string) error {
	if path == "" {
		return errors.New(errors.Invalid, "the snapshot path must not be empty")
	}

	conn, err := s.writer.Conn(ctx)
	if err != nil {
		return dbx.Convert(err, "reserve the writer for the snapshot")
	}

	copied := conn.Raw(func(driverConn any) error {
		raw, ok := driverConn.(sqlitedriver.Conn)
		if !ok {
			return errors.New(errors.Internal, "the connection does not carry a sqlite handle")
		}
		return raw.Raw().Backup("main", snapshotURI(path))
	})
	return stderrors.Join(
		dbx.Convert(copied, "copy the database into the snapshot"),
		dbx.Convert(conn.Close(), "release the writer after the snapshot"),
	)
}

func snapshotURI(path string) string {
	location := url.URL{Path: filepath.ToSlash(path)}
	return "file:" + location.EscapedPath()
}
