package sqlitetest

import (
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
)

const (
	fileName  = "postulator.db"
	keyLength = 32
)

func Key() []byte {
	key := make([]byte, keyLength)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func Open(t testing.TB) *sqlite.Store {
	t.Helper()
	return open(t, nil)
}

func OpenEncrypted(t testing.TB, key []byte) *sqlite.Store {
	t.Helper()
	return open(t, key)
}

func open(t testing.TB, key []byte) *sqlite.Store {
	t.Helper()

	store, err := sqlite.Open(filepath.Join(t.TempDir(), fileName), key)
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close the store: %v", closeErr)
		}
	})
	return store
}
