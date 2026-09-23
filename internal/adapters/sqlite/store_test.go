package sqlite

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func openStore(t *testing.T, key []byte) *Store {
	t.Helper()

	store, err := Open(Config{Path: filepath.Join(t.TempDir(), "postulator.db"), Key: key})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close: %v", closeErr)
		}
	})
	return store
}

func testKey() []byte {
	key := make([]byte, keyLength)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestOpenAppliesPragmas(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		key  []byte
	}{
		{name: "plain", key: nil},
		{name: "adiantum", key: testKey()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := openStore(t, tc.key)
			pragmas := []struct {
				name string
				want string
			}{
				{name: "journal_mode", want: "wal"},
				{name: "foreign_keys", want: "1"},
				{name: "busy_timeout", want: "5000"},
				{name: "synchronous", want: "1"},
			}

			for _, pragma := range pragmas {
				var got string
				if err := store.writer.QueryRowContext(t.Context(), "PRAGMA "+pragma.name).Scan(&got); err != nil {
					t.Fatalf("PRAGMA %s: %v", pragma.name, err)
				}
				if got != pragma.want {
					t.Errorf("PRAGMA %s = %q, want %q", pragma.name, got, pragma.want)
				}
			}
		})
	}
}

func TestOpenCreatesTheSchema(t *testing.T) {
	t.Parallel()

	store := openStore(t, testKey())
	for _, table := range []string{"app_meta", "settings", "secrets"} {
		var name string
		err := store.reader.QueryRowContext(t.Context(),
			`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}

	var version string
	err := store.reader.QueryRowContext(t.Context(),
		`SELECT value FROM app_meta WHERE key = 'schema_version'`).Scan(&version)
	if err != nil {
		t.Fatalf("schema_version: %v", err)
	}
	if version != "1" {
		t.Errorf("schema_version = %q, want %q", version, "1")
	}
}

func TestOpenRejectsBadArguments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		key  []byte
	}{
		{name: "empty path", path: "", key: nil},
		{name: "short key", path: filepath.Join(t.TempDir(), "a.db"), key: []byte("too short")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store, err := Open(Config{Path: tc.path, Key: tc.key})
			if store != nil {
				t.Fatal("no store may be returned")
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestOpenRejectsTheWrongKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "postulator.db")
	first, err := Open(Config{Path: path, Key: testKey()})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err = first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	other := testKey()
	other[0] ^= 0xff

	second, err := Open(Config{Path: path, Key: other})
	if err == nil {
		if closeErr := second.Close(); closeErr != nil {
			t.Errorf("close: %v", closeErr)
		}
		t.Fatal("a database opened with the wrong key must fail")
	}
	if !errors.IsCode(err, errors.Locked) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Locked)
	}
}

func TestReaderPoolRefusesWrites(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	_, err := store.reader.ExecContext(t.Context(),
		`INSERT INTO settings (key, value, updated_at) VALUES ('a', '1', '2026-09-17T00:00:00Z')`)
	if err == nil {
		t.Fatal("the reader pool must refuse a write")
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "postulator.db")
	store, err := Open(Config{Path: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close: %v", closeErr)
		}
	})

	if store.Path() != path {
		t.Errorf("Path = %q, want %q", store.Path(), path)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	if err := store.migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestOpenReportsAFileThatIsNotADatabase(t *testing.T) {
	t.Parallel()

	junk := make([]byte, 64)
	if _, err := rand.Read(junk); err != nil {
		t.Fatalf("generate junk: %v", err)
	}

	path := filepath.Join(t.TempDir(), "postulator.db")
	if err := os.WriteFile(path, junk, 0o600); err != nil {
		t.Fatalf("write junk: %v", err)
	}

	const recovery = `remove C:\Postulator\master.key and C:\Postulator\postulator.db to reset the application state`

	store, err := Open(Config{Path: path, Recovery: recovery})
	if store != nil {
		t.Fatal("no store may be returned")
	}
	if !errors.IsCode(err, errors.Locked) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Locked)
	}
	if !strings.Contains(err.Error(), recovery) {
		t.Errorf("message = %q, want the recovery instruction", err.Error())
	}
}

func TestOpenWithoutARecoveryInstruction(t *testing.T) {
	t.Parallel()

	junk := make([]byte, 64)
	if _, err := rand.Read(junk); err != nil {
		t.Fatalf("generate junk: %v", err)
	}

	path := filepath.Join(t.TempDir(), "postulator.db")
	if err := os.WriteFile(path, junk, 0o600); err != nil {
		t.Fatalf("write junk: %v", err)
	}

	_, err := Open(Config{Path: path})
	if !errors.IsCode(err, errors.Locked) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Locked)
	}
}
