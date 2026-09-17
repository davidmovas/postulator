package sqlite

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/dbx"
)

type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type txKey struct{}

func withTx(parent context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(parent, txKey{}, tx)
}

func txFrom(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	return tx, ok
}

func (s *Store) execFrom(ctx context.Context) executor {
	if tx, ok := txFrom(ctx); ok {
		return tx
	}
	return s.reader
}

func (s *Store) writeFrom(ctx context.Context) executor {
	if tx, ok := txFrom(ctx); ok {
		return tx
	}
	return s.writer
}

func (s *Store) Do(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := txFrom(ctx); ok {
		return fn(ctx)
	}

	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return dbx.Convert(err, "begin the transaction")
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if rollback := tx.Rollback(); rollback != nil && !stderrors.Is(rollback, sql.ErrTxDone) {
			panic(fmt.Errorf("%v (rolling back: %w)", recovered, rollback))
		}
		panic(recovered)
	}()

	if err = fn(withTx(ctx, tx)); err != nil {
		rollback := tx.Rollback()
		if rollback != nil && !stderrors.Is(rollback, sql.ErrTxDone) {
			return stderrors.Join(err, dbx.Convert(rollback, "roll back the transaction"))
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		if cancelled := ctx.Err(); cancelled != nil {
			return dbx.Convert(cancelled, "commit the transaction")
		}
		return dbx.Convert(err, "commit the transaction")
	}
	return nil
}
