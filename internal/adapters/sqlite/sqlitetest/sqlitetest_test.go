package sqlitetest_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	if filepath.Base(store.Path()) != "postulator.db" {
		t.Errorf("path = %q, want a file named postulator.db", store.Path())
	}
	if err := store.Do(t.Context(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestOpenEncrypted(t *testing.T) {
	t.Parallel()

	key := sqlitetest.Key()
	if len(key) != 32 {
		t.Fatalf("Key length = %d, want 32", len(key))
	}

	store := sqlitetest.OpenEncrypted(t, key)
	if err := store.Do(t.Context(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("Do: %v", err)
	}
}
