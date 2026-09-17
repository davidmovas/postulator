package sqlite

import (
	"context"
	"database/sql"
	stderrors "errors"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const insertSetting = `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, '2026-09-17T00:00:00Z')`

func countSettings(t *testing.T, store *Store) int {
	t.Helper()

	var total int
	if err := store.reader.QueryRowContext(t.Context(), `SELECT count(*) FROM settings`).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	return total
}

func TestDoCommits(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	err := store.Do(t.Context(), func(c context.Context) error {
		_, execErr := store.writeFrom(c).ExecContext(c, insertSetting, "runs.workers", "2")
		return execErr
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got := countSettings(t, store); got != 1 {
		t.Errorf("rows = %d, want 1", got)
	}
}

func TestDoRollsBack(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	sentinel := errors.New(errors.Invalid, "abandon")

	err := store.Do(t.Context(), func(c context.Context) error {
		if _, execErr := store.writeFrom(c).ExecContext(c, insertSetting, "runs.workers", "2"); execErr != nil {
			return execErr
		}
		return sentinel
	})
	if !stderrors.Is(err, sentinel) {
		t.Fatalf("Do = %v, want the callback error", err)
	}
	if got := countSettings(t, store); got != 0 {
		t.Errorf("rows = %d, want 0", got)
	}
}

func TestDoNests(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	var outer, inner *sql.Tx

	err := store.Do(t.Context(), func(c context.Context) error {
		outer, _ = txFrom(c)
		return store.Do(c, func(nested context.Context) error {
			inner, _ = txFrom(nested)
			_, execErr := store.writeFrom(nested).ExecContext(nested, insertSetting, "runs.workers", "2")
			return execErr
		})
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if outer == nil || outer != inner {
		t.Error("the nested call must reuse the outer transaction")
	}
	if got := countSettings(t, store); got != 1 {
		t.Errorf("rows = %d, want 1", got)
	}
}

func TestNestedFailureRollsBackEverything(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	sentinel := errors.New(errors.Invalid, "abandon")

	err := store.Do(t.Context(), func(c context.Context) error {
		if _, execErr := store.writeFrom(c).ExecContext(c, insertSetting, "a", "1"); execErr != nil {
			return execErr
		}
		return store.Do(c, func(nested context.Context) error {
			if _, execErr := store.writeFrom(nested).ExecContext(nested, insertSetting, "b", "2"); execErr != nil {
				return execErr
			}
			return sentinel
		})
	})
	if !stderrors.Is(err, sentinel) {
		t.Fatalf("Do = %v, want the callback error", err)
	}
	if got := countSettings(t, store); got != 0 {
		t.Errorf("rows = %d, want 0", got)
	}
}

func TestExecutorRouting(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)

	if store.execFrom(t.Context()) != executor(store.reader) {
		t.Error("execFrom outside a transaction must return the reader")
	}
	if store.writeFrom(t.Context()) != executor(store.writer) {
		t.Error("writeFrom outside a transaction must return the writer")
	}

	err := store.Do(t.Context(), func(c context.Context) error {
		tx, ok := txFrom(c)
		if !ok {
			return errors.New(errors.Internal, "no transaction in the context")
		}
		if store.execFrom(c) != executor(tx) || store.writeFrom(c) != executor(tx) {
			return errors.New(errors.Internal, "both accessors must return the transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestDoRejectsAClosedStore(t *testing.T) {
	t.Parallel()

	store, err := Open(filepath.Join(t.TempDir(), "postulator.db"), nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err = store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	err = store.Do(t.Context(), func(context.Context) error { return nil })
	if !errors.IsCode(err, errors.Internal) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Internal)
	}
}
