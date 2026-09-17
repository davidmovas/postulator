package app_test

import (
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestOpenAndClose(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"),
		KeyDir:       home,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if core.Store == nil || core.Secrets == nil || core.Settings == nil {
		t.Fatal("the core must carry the store, the secrets and the settings")
	}
	if len(core.UnknownSettings) != 0 {
		t.Errorf("UnknownSettings = %v, want none", core.UnknownSettings)
	}

	if err = core.Secrets.Put(t.Context(), "wp.site.token", "abcd EFGH"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	value, err := core.Secrets.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if value != "abcd EFGH" {
		t.Errorf("Get = %q, want %q", value, "abcd EFGH")
	}

	if err = core.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOpenReopensTheSameDatabase(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	first, err := app.Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err = first.Secrets.Put(t.Context(), "wp.site.token", "abcd EFGH"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err = first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second, err := app.Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("Open again: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := second.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	value, err := second.Secrets.Get(t.Context(), "wp.site.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if value != "abcd EFGH" {
		t.Errorf("Get = %q, want %q", value, "abcd EFGH")
	}
}

func TestOpenRejectsAnIncompleteConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  app.Config
	}{
		{name: "no database path", cfg: app.Config{KeyDir: t.TempDir()}},
		{name: "no key directory", cfg: app.Config{DatabasePath: filepath.Join(t.TempDir(), "postulator.db")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := app.Open(t.Context(), tc.cfg); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg, err := app.DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig: %v", err)
	}
	if filepath.Base(cfg.DatabasePath) != "postulator.db" {
		t.Errorf("DatabasePath = %q, want a file named postulator.db", cfg.DatabasePath)
	}
	if filepath.Base(cfg.KeyDir) != "Postulator" {
		t.Errorf("KeyDir = %q, want a directory named Postulator", cfg.KeyDir)
	}
	if filepath.Dir(cfg.DatabasePath) != cfg.KeyDir {
		t.Errorf("the database and the key must share a directory, got %q and %q", cfg.DatabasePath, cfg.KeyDir)
	}
}
