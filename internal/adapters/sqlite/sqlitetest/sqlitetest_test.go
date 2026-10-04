package sqlitetest_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	if err := store.Do(t.Context(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestKeyFitsTheCipher(t *testing.T) {
	t.Parallel()

	if key := sqlitetest.Key(); len(key) != 32 {
		t.Fatalf("Key length = %d, want 32", len(key))
	}
}
