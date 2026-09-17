package sqlite

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
)

func TestMigrationsAreEmbedded(t *testing.T) {
	t.Parallel()

	fsys, err := migrations()
	if err != nil {
		t.Fatalf("migrations: %v", err)
	}

	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	want := []string{"0001_app_meta.sql", "0002_settings.sql", "0003_secrets.sql"}
	if !slices.Equal(names, want) {
		t.Fatalf("embedded migrations = %v, want %v", names, want)
	}

	for _, name := range names {
		body, readErr := fs.ReadFile(fsys, name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		for _, marker := range []string{"-- +goose Up", "-- +goose Down", "STRICT"} {
			if !strings.Contains(string(body), marker) {
				t.Errorf("%s is missing %q", name, marker)
			}
		}
	}
}

func TestMigrationsRoundTrip(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)

	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}

	version, err := provider.GetDBVersion(t.Context())
	if err != nil {
		t.Fatalf("version after up: %v", err)
	}
	if version != 3 {
		t.Fatalf("version after up = %d, want 3", version)
	}

	if _, err = provider.DownTo(t.Context(), 0); err != nil {
		t.Fatalf("down: %v", err)
	}

	for _, table := range []string{"app_meta", "settings", "secrets"} {
		var name string
		scanErr := store.writer.QueryRowContext(t.Context(),
			`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if scanErr == nil {
			t.Errorf("%s survived the down migration", table)
		}
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}

	version, err = provider.GetDBVersion(t.Context())
	if err != nil {
		t.Fatalf("version after the second up: %v", err)
	}
	if version != 3 {
		t.Errorf("version after the second up = %d, want 3", version)
	}
}
