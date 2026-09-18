package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func openCore(t *testing.T, cfg app.Config) *app.Core {
	t.Helper()

	core, err := app.Open(t.Context(), cfg, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return core
}

func siteNames(t *testing.T, core *app.Core) int {
	t.Helper()

	listed, err := core.Sites.List(t.Context(), sites.ListRequest{})
	if err != nil {
		t.Fatalf("List the sites: %v", err)
	}
	return len(listed.Items)
}

func TestTheMasterPasswordLocksAndUnlocksTheWholeCore(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	core := openCore(t, cfg)
	t.Cleanup(func() {
		if err := core.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if core.Locked() {
		t.Fatal("a core without a master password opens locked")
	}
	if _, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Shop", BaseURL: "https://shop.example", Username: "editor", Password: "abcd EFGH ijkl MNOP qrst UVWX",
	}); err != nil {
		t.Fatalf("Create a site: %v", err)
	}

	if err := core.Lock(); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Lock without a master password = %v, want %s", err, errors.Invalid)
	}

	if err := core.SetMasterPassword(t.Context(), "", "hunter2"); err != nil {
		t.Fatalf("SetMasterPassword: %v", err)
	}
	protected, err := core.Protected()
	if err != nil || !protected {
		t.Fatalf("Protected = %v, %v, want true", protected, err)
	}
	if _, statErr := os.Stat(filepath.Join(home, masterkey.FileName)); !os.IsNotExist(statErr) {
		t.Fatal("the plain key file survives a master password")
	}

	if err = core.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if !core.Locked() || core.Store != nil || core.Sites != nil {
		t.Fatal("a locked core still carries its composition")
	}

	if err = core.Unlock(t.Context(), "hunter3"); !errors.IsCode(err, errors.Locked) {
		t.Fatalf("Unlock with the wrong password = %v, want %s", err, errors.Locked)
	}
	if !core.Locked() {
		t.Fatal("a refused password unlocked the core")
	}

	if err = core.Unlock(t.Context(), "hunter2"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if core.Locked() || core.Sites == nil {
		t.Fatal("the core is still locked after the right password")
	}
	if got := siteNames(t, core); got != 1 {
		t.Fatalf("the unlocked core lists %d sites, want the one it wrote before locking", got)
	}

	if err = core.SetMasterPassword(t.Context(), "hunter2", ""); err != nil {
		t.Fatalf("SetMasterPassword to remove it: %v", err)
	}
	protected, err = core.Protected()
	if err != nil || protected {
		t.Fatalf("Protected after removing the password = %v, %v, want false", protected, err)
	}
	if got := siteNames(t, core); got != 1 {
		t.Fatalf("removing the password lost the data: %d sites", got)
	}
}

func TestACoreWithAMasterPasswordStartsLocked(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}

	first := openCore(t, cfg)
	if err := first.SetMasterPassword(t.Context(), "", "hunter2"); err != nil {
		t.Fatalf("SetMasterPassword: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second := openCore(t, cfg)
	t.Cleanup(func() {
		if err := second.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if !second.Locked() {
		t.Fatal("a protected core opens unlocked")
	}
	if second.Store != nil || second.Engine != nil || second.Events == nil {
		t.Fatal("a locked core must carry the relay and nothing else")
	}
	if _, err := second.ExportBackup(t.Context(), filepath.Join(home, "backup.pstx"), "hunter2"); !errors.IsCode(err, errors.Locked) {
		t.Fatalf("ExportBackup while locked = %v, want %s", err, errors.Locked)
	}
	if err := second.ImportBackup(t.Context(), filepath.Join(home, "backup.pstx"), "hunter2"); !errors.IsCode(err, errors.Locked) {
		t.Fatalf("ImportBackup while locked = %v, want %s", err, errors.Locked)
	}

	if err := second.Unlock(t.Context(), "hunter2"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if second.Store == nil || second.Engine == nil || second.Runs == nil {
		t.Fatal("unlocking did not compose the core")
	}

	if err := second.Unlock(t.Context(), "hunter2"); err != nil {
		t.Fatalf("Unlock twice: %v", err)
	}
	if err := second.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if err := second.Lock(); err != nil {
		t.Fatalf("Lock twice: %v", err)
	}
	if err := second.Unlock(t.Context(), "hunter2"); err != nil {
		t.Fatalf("Unlock after a second lock: %v", err)
	}
}

func TestTheBackupRoundTripsThroughTheCore(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	cfg := app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home}
	backup := filepath.Join(home, "postulator.pstx")

	core := openCore(t, cfg)
	t.Cleanup(func() {
		if err := core.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if _, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Shop", BaseURL: "https://shop.example", Username: "editor", Password: "abcd EFGH ijkl MNOP qrst UVWX",
	}); err != nil {
		t.Fatalf("Create a site: %v", err)
	}

	written, err := core.ExportBackup(t.Context(), backup, "hunter2")
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}
	if written == 0 {
		t.Fatal("the backup file is empty")
	}

	if _, err = core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Blog", BaseURL: "https://blog.example", Username: "editor", Password: "abcd EFGH ijkl MNOP qrst UVWX",
	}); err != nil {
		t.Fatalf("Create a second site: %v", err)
	}
	if got := siteNames(t, core); got != 2 {
		t.Fatalf("the store holds %d sites, want 2", got)
	}

	if err = core.ImportBackup(t.Context(), backup, "hunter3"); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ImportBackup with the wrong password = %v, want %s", err, errors.Invalid)
	}
	if err = core.ImportBackup(t.Context(), backup, "hunter2"); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	if got := siteNames(t, core); got != 1 {
		t.Fatalf("the restored store holds %d sites, want the one the backup carried", got)
	}
	if core.Locked() || core.Engine == nil {
		t.Fatal("the core did not come back up after the restore")
	}
}
