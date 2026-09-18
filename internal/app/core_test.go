package app_test

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestOpenWiresTheUseCasesAndSeedsTheStarterTemplates(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	for round := range 2 {
		core, err := app.Open(t.Context(), cfg, zaptest.NewLogger(t))
		if err != nil {
			t.Fatalf("Open round %d: %v", round, err)
		}
		if core.Events == nil || core.Sites == nil || core.Graph == nil || core.Pages == nil || core.Templates == nil {
			t.Fatal("the core must carry the relay and the four services")
		}
		if core.LLM == nil || core.Catalog == nil || core.Profiles == nil || core.Ledger == nil || core.Models == nil {
			t.Fatal("the core must carry the llm stack")
		}
		if core.Steps == nil || core.Engine == nil || core.Runs == nil {
			t.Fatal("the core must carry the run engine")
		}
		if names := core.Steps.Names(); len(names) != 10 || names[0] != "resolve_context" {
			t.Fatalf("round %d: the step registry holds %v", round, names)
		}
		if _, err := core.Runs.List(t.Context(), runs.ListRequest{}); err != nil {
			t.Fatalf("round %d: ListRuns: %v", round, err)
		}

		listed, err := core.Models.ListModels(t.Context(), models.ListModelsRequest{})
		if err != nil || len(listed.Models) == 0 {
			t.Errorf("round %d: catalog models = %d, %v; want the embedded catalog", round, len(listed.Models), err)
		}

		seeded, err := core.Templates.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "global"})
		if err != nil {
			t.Fatalf("ListTemplates: %v", err)
		}
		if len(seeded.Items) != 5 {
			t.Errorf("round %d: seeded templates = %d, want 5", round, len(seeded.Items))
		}
		policies, err := core.Templates.ListPolicies(t.Context(), templates.ListPoliciesRequest{Scope: "global"})
		if err != nil || len(policies.Items) != 1 {
			t.Errorf("round %d: seeded policies = %d, %v; want 1", round, len(policies.Items), err)
		}

		if err = core.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

func TestOpenAndClose(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"),
		KeyDir:       home,
	}, zaptest.NewLogger(t))
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

	first, err := app.Open(t.Context(), cfg, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err = first.Secrets.Put(t.Context(), "wp.site.token", "abcd EFGH"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err = first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second, err := app.Open(t.Context(), cfg, zaptest.NewLogger(t))
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

			if _, err := app.Open(t.Context(), tc.cfg, zaptest.NewLogger(t)); !errors.IsCode(err, errors.Invalid) {
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

func TestOpenNamesBothFilesWhenTheKeyCannotBeUnprotected(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	keyPath := filepath.Join(home, "master.key")
	if err := os.WriteFile(keyPath, []byte("not protected"), 0o600); err != nil {
		t.Fatalf("write a corrupt key file: %v", err)
	}

	_, err := app.Open(t.Context(), cfg, zaptest.NewLogger(t))
	if !errors.IsCode(err, errors.Locked) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Locked)
	}
	for _, name := range []string{keyPath, cfg.DatabasePath} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("message = %q, want it to name %q", err.Error(), name)
		}
	}
}

func TestOpenNamesBothFilesWhenTheDatabaseIsUnreadable(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	junk := make([]byte, 64)
	if _, err := rand.Read(junk); err != nil {
		t.Fatalf("generate junk: %v", err)
	}
	if err := os.WriteFile(cfg.DatabasePath, junk, 0o600); err != nil {
		t.Fatalf("write junk: %v", err)
	}

	_, err := app.Open(t.Context(), cfg, zaptest.NewLogger(t))
	if !errors.IsCode(err, errors.Locked) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Locked)
	}
	for _, name := range []string{filepath.Join(home, "master.key"), cfg.DatabasePath} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("message = %q, want it to name %q", err.Error(), name)
		}
	}
}
