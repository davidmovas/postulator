package sqlite

import (
	"context"
	"database/sql"
	stderrors "errors"
	"os"
	"path/filepath"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/pressly/goose/v3"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	readerConns   = 4
	directoryMode = 0o700
)

type Store struct {
	writer *sql.DB
	reader *sql.DB
	path   string
}

func Open(path string, key []byte) (*Store, error) {
	if path == "" {
		return nil, errors.New(errors.Invalid, "database path must not be empty")
	}
	if key != nil && len(key) != keyLength {
		return nil, errors.New(errors.Invalid, "database key must be 32 bytes")
	}
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the database directory")
	}

	writer, err := driver.Open(dsn(path, key, false))
	if err != nil {
		return nil, dbx.Convert(err, "open the database for writing")
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	writer.SetConnMaxLifetime(0)

	store := &Store{writer: writer, path: path}
	if err = store.migrate(context.Background()); err != nil {
		return nil, stderrors.Join(err, closeDB(writer, "close the writer after a failed migration"))
	}

	reader, err := driver.Open(dsn(path, key, true))
	if err != nil {
		converted := dbx.Convert(err, "open the database for reading")
		return nil, stderrors.Join(converted, closeDB(writer, "close the writer after a failed reader open"))
	}
	reader.SetMaxOpenConns(readerConns)
	reader.SetMaxIdleConns(readerConns)
	reader.SetConnMaxLifetime(0)

	store.reader = reader
	return store, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Close() error {
	return stderrors.Join(
		closeDB(s.reader, "close the database reader"),
		closeDB(s.writer, "close the database writer"),
	)
}

func (s *Store) provider() (*goose.Provider, error) {
	fsys, err := migrations()
	if err != nil {
		return nil, err
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, s.writer, fsys)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "prepare the migrations")
	}
	return provider, nil
}

func (s *Store) migrate(ctx context.Context) error {
	provider, err := s.provider()
	if err != nil {
		return err
	}
	if _, err = provider.Up(ctx); err != nil {
		return errors.Wrap(err, errors.Internal, "apply the migrations")
	}
	return nil
}

func closeDB(db *sql.DB, message string) error {
	if db == nil {
		return nil
	}
	return dbx.Convert(db.Close(), message)
}
