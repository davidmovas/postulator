package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/secrets/export"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

type garbage struct{}

func (garbage) Snapshot(_ context.Context, path string) error {
	return os.WriteFile(path, []byte("this is not a database"), 0o600)
}

func brokenBackup(t *testing.T, path, password string) {
	t.Helper()

	if err := export.New(garbage{}, clock.System{}).Write(t.Context(), path, password); err != nil {
		t.Fatalf("write the broken backup: %v", err)
	}
}

func twoSites(t *testing.T, core *app.Core) {
	t.Helper()

	for _, name := range []string{"Shop", "Blog"} {
		if _, err := core.Sites.Create(t.Context(), sites.CreateRequest{
			Name: name, BaseURL: "https://" + strings.ToLower(name) + ".example", Username: "editor",
			Password: "abcd EFGH ijkl MNOP qrst UVWX",
		}); err != nil {
			t.Fatalf("create the site %s: %v", name, err)
		}
	}
}

func TestAnImportThatCannotBeRestoredLeavesTheCoreOnWhatItHeld(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}
	backup := filepath.Join(home, "broken.pstx")
	brokenBackup(t, backup, "hunter2")

	core := openCore(t, cfg)
	t.Cleanup(func() {
		if err := core.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	twoSites(t, core)

	err := core.ImportBackup(t.Context(), backup, "hunter2")
	if err == nil {
		t.Fatal("ImportBackup accepted a backup carrying no database")
	}
	if core.Locked() {
		t.Fatalf("the core stayed locked after a failed import: %v", err)
	}
	if got := siteNames(t, core); got != 2 {
		t.Fatalf("the core holds %d sites, want the two it held before the import", got)
	}
	if core.Engine == nil || core.Sites == nil {
		t.Fatal("the core came back without its use cases")
	}
}

type failingOpen struct {
	failAt map[int]bool
	calls  atomic.Int64
}

func (f *failingOpen) open(cfg sqlite.Config) (*sqlite.Store, error) {
	if f.failAt[int(f.calls.Add(1))] {
		return nil, os.ErrPermission
	}
	return sqlite.Open(cfg)
}

func TestAnImportThatCannotRecomposePutsBackWhatItReplaced(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	faults := &failingOpen{failAt: map[int]bool{2: true}}
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home, Open: faults.open}
	backup := filepath.Join(home, "postulator.pstx")

	core := openCore(t, cfg)
	t.Cleanup(func() {
		if err := core.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	twoSites(t, core)

	if _, err := core.ExportBackup(t.Context(), backup, "hunter2"); err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}
	if _, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Third", BaseURL: "https://third.example", Username: "editor",
		Password: "abcd EFGH ijkl MNOP qrst UVWX",
	}); err != nil {
		t.Fatalf("create a third site: %v", err)
	}

	err := core.ImportBackup(t.Context(), backup, "hunter2")
	if err == nil {
		t.Fatal("ImportBackup reported success although the core could not be recomposed")
	}
	if core.Locked() {
		t.Fatalf("the core stayed locked although the rollback could recompose it: %v", err)
	}
	if got := siteNames(t, core); got != 3 {
		t.Fatalf("the core holds %d sites, want the three it held before the import", got)
	}
}

func TestAnImportThatCannotBeRolledBackNamesWhereTheDatabaseIsKept(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	faults := &failingOpen{failAt: map[int]bool{2: true, 3: true}}
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home, Open: faults.open}
	backup := filepath.Join(home, "postulator.pstx")

	core, err := app.Open(t.Context(), cfg, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	twoSites(t, core)

	if _, err = core.ExportBackup(t.Context(), backup, "hunter2"); err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	err = core.ImportBackup(t.Context(), backup, "hunter2")
	if err == nil {
		t.Fatal("ImportBackup reported success although nothing could be opened")
	}
	if !core.Locked() {
		t.Fatal("the core is not locked although neither the import nor the rollback could open the database")
	}

	kept := keptDatabase(t, err.Error())
	if _, statErr := os.Stat(kept); statErr != nil {
		t.Fatalf("the database named at %s is not there: %v", kept, statErr)
	}
	if removeErr := os.RemoveAll(filepath.Dir(kept)); removeErr != nil {
		t.Fatalf("remove the kept database: %v", removeErr)
	}
}

func keptDatabase(t *testing.T, message string) string {
	t.Helper()

	const marker = "kept at "
	at := strings.Index(message, marker)
	if at < 0 {
		t.Fatalf("the refusal does not say where the database was kept: %s", message)
	}
	rest := message[at+len(marker):]
	end := strings.Index(rest, export.DatabaseName)
	if end < 0 {
		t.Fatalf("the refusal does not name the database file: %s", message)
	}
	return rest[:end+len(export.DatabaseName)]
}
