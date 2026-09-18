package sqlite_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestSnapshotCopiesAnEncryptedDatabaseInTheClear(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate the key: %v", err)
	}

	store, err := sqlite.Open(sqlite.Config{Path: filepath.Join(dir, "postulator.db"), Key: key})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	settings := sqlite.NewSettingsRepo(store, clock.System{})
	if err = settings.Set(t.Context(), "runs.workers", json.RawMessage("4")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	snapshot := filepath.Join(dir, "snapshot.db")
	if err = store.Snapshot(t.Context(), snapshot); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	raw, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatalf("read the snapshot: %v", err)
	}
	if !bytes.HasPrefix(raw, []byte("SQLite format 3")) {
		t.Fatal("the snapshot is not a plain sqlite database")
	}

	copied, err := sqlite.Open(sqlite.Config{Path: snapshot})
	if err != nil {
		t.Fatalf("open the snapshot: %v", err)
	}
	defer func() {
		if closeErr := copied.Close(); closeErr != nil {
			t.Errorf("close the snapshot: %v", closeErr)
		}
	}()

	stored, found, err := sqlite.NewSettingsRepo(copied, clock.System{}).Get(t.Context(), "runs.workers")
	if err != nil {
		t.Fatalf("Get from the snapshot: %v", err)
	}
	if !found || string(stored) != "4" {
		t.Fatalf("the snapshot holds %s (found %v), want 4", stored, found)
	}
}

func TestSnapshotRefuses(t *testing.T) {
	t.Parallel()

	store, err := sqlite.Open(sqlite.Config{Path: filepath.Join(t.TempDir(), "postulator.db")})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	if err = store.Snapshot(t.Context(), ""); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Snapshot with no path = %v, want %s", err, errors.Invalid)
	}
	if err = store.Snapshot(t.Context(), t.TempDir()); err == nil {
		t.Fatal("Snapshot over a directory returned no error")
	}
}
