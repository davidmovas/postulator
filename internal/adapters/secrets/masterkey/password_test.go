package masterkey_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestTheKeyTravelsThroughEveryPasswordState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := masterkey.Config{Dir: dir}

	key, err := masterkey.Load(cfg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	protected, err := masterkey.Protected(cfg)
	if err != nil {
		t.Fatalf("Protected: %v", err)
	}
	if protected {
		t.Fatal("a fresh directory must not be protected by a password")
	}

	if err = masterkey.SetPassword(cfg, key, "hunter2"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}

	protected, err = masterkey.Protected(cfg)
	if err != nil {
		t.Fatalf("Protected: %v", err)
	}
	if !protected {
		t.Fatal("the directory must be protected once a password is set")
	}
	if _, statErr := os.Stat(filepath.Join(dir, masterkey.FileName)); !os.IsNotExist(statErr) {
		t.Fatal("the plain key file must be gone once a password is set")
	}

	wrapped, err := os.ReadFile(filepath.Join(dir, masterkey.ProtectedFileName))
	if err != nil {
		t.Fatalf("read the wrapped key file: %v", err)
	}
	if bytes.Contains(wrapped, key) {
		t.Fatal("the wrapped key file must not contain the raw key")
	}

	if _, err = masterkey.Load(cfg); !errors.IsCode(err, errors.Locked) {
		t.Fatalf("Load = %v, want %s", err, errors.Locked)
	}
	if _, err = masterkey.Unlock(cfg, "hunter3"); !errors.IsCode(err, errors.Locked) {
		t.Fatalf("Unlock with the wrong password = %v, want %s", err, errors.Locked)
	}

	unlocked, err := masterkey.Unlock(cfg, "hunter2")
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if !bytes.Equal(unlocked, key) {
		t.Fatal("Unlock returned another key")
	}

	if err = masterkey.SetPassword(cfg, unlocked, "hunter4"); err != nil {
		t.Fatalf("SetPassword again: %v", err)
	}
	if _, err = masterkey.Unlock(cfg, "hunter2"); !errors.IsCode(err, errors.Locked) {
		t.Fatalf("Unlock with the replaced password = %v, want %s", err, errors.Locked)
	}

	rewrapped, err := masterkey.Unlock(cfg, "hunter4")
	if err != nil {
		t.Fatalf("Unlock with the new password: %v", err)
	}
	if !bytes.Equal(rewrapped, key) {
		t.Fatal("rewrapping changed the key")
	}

	if err = masterkey.ClearPassword(cfg, rewrapped); err != nil {
		t.Fatalf("ClearPassword: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, masterkey.ProtectedFileName)); !os.IsNotExist(statErr) {
		t.Fatal("the wrapped key file must be gone once the password is removed")
	}

	plain, err := masterkey.Load(cfg)
	if err != nil {
		t.Fatalf("Load after ClearPassword: %v", err)
	}
	if !bytes.Equal(plain, key) {
		t.Fatal("removing the password changed the key")
	}
}

func TestPasswordTransitionsRefuse(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{9}, masterkey.Length)

	cases := []struct {
		name string
		call func(t *testing.T) error
		want errors.Code
	}{
		{
			name: "set without a directory",
			call: func(*testing.T) error { return masterkey.SetPassword(masterkey.Config{}, key, "hunter2") },
			want: errors.Invalid,
		},
		{
			name: "set an empty password",
			call: func(t *testing.T) error {
				t.Helper()
				return masterkey.SetPassword(masterkey.Config{Dir: t.TempDir()}, key, "")
			},
			want: errors.Invalid,
		},
		{
			name: "set with a key of the wrong length",
			call: func(t *testing.T) error {
				t.Helper()
				return masterkey.SetPassword(masterkey.Config{Dir: t.TempDir()}, []byte("short"), "hunter2")
			},
			want: errors.Invalid,
		},
		{
			name: "clear with a key of the wrong length",
			call: func(t *testing.T) error {
				t.Helper()
				return masterkey.ClearPassword(masterkey.Config{Dir: t.TempDir()}, []byte("short"))
			},
			want: errors.Invalid,
		},
		{
			name: "unlock without a wrapped key",
			call: func(t *testing.T) error {
				t.Helper()
				_, err := masterkey.Unlock(masterkey.Config{Dir: t.TempDir()}, "hunter2")
				return err
			},
			want: errors.NotFound,
		},
		{
			name: "unlock a corrupt wrapped key",
			call: func(t *testing.T) error {
				t.Helper()

				dir := t.TempDir()
				path := filepath.Join(dir, masterkey.ProtectedFileName)
				if err := os.WriteFile(path, []byte("not an envelope"), 0o600); err != nil {
					t.Fatalf("write the wrapped key file: %v", err)
				}
				_, err := masterkey.Unlock(masterkey.Config{Dir: dir}, "hunter2")
				return err
			},
			want: errors.Invalid,
		},
		{
			name: "protected without a directory",
			call: func(*testing.T) error {
				_, err := masterkey.Protected(masterkey.Config{})
				return err
			},
			want: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.call(t); errors.CodeOf(err) != tc.want {
				t.Fatalf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestUnlockRefusesAWrappedKeyOfTheWrongLength(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := masterkey.SetPassword(masterkey.Config{Dir: dir}, bytes.Repeat([]byte{3}, masterkey.Length), "hunter2"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}

	unlocked, err := masterkey.Unlock(masterkey.Config{Dir: dir}, "hunter2")
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if len(unlocked) != masterkey.Length {
		t.Fatalf("length = %d, want %d", len(unlocked), masterkey.Length)
	}
}

func TestPasswordTransitionsReportFilesystemFailures(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{5}, masterkey.Length)

	cases := []struct {
		name    string
		blocked string
		nested  bool
		call    func(dir string) error
	}{
		{
			name:    "the wrapped key path is taken",
			blocked: masterkey.ProtectedFileName,
			call:    func(dir string) error { return masterkey.SetPassword(masterkey.Config{Dir: dir}, key, "hunter2") },
		},
		{
			name:    "the plain key path is taken",
			blocked: masterkey.FileName,
			call:    func(dir string) error { return masterkey.ClearPassword(masterkey.Config{Dir: dir}, key) },
		},
		{
			name:    "the wrapped key cannot be read",
			blocked: masterkey.ProtectedFileName,
			call: func(dir string) error {
				_, err := masterkey.Unlock(masterkey.Config{Dir: dir}, "hunter2")
				return err
			},
		},
		{
			name:    "the superseded key cannot be removed",
			blocked: masterkey.FileName,
			nested:  true,
			call:    func(dir string) error { return masterkey.SetPassword(masterkey.Config{Dir: dir}, key, "hunter2") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			blocker := filepath.Join(dir, tc.blocked)
			if err := os.Mkdir(blocker, 0o700); err != nil {
				t.Fatalf("create the blocking directory: %v", err)
			}
			if tc.nested {
				if err := os.WriteFile(filepath.Join(blocker, "occupied"), []byte("x"), 0o600); err != nil {
					t.Fatalf("occupy the blocking directory: %v", err)
				}
			}

			if err := tc.call(dir); !errors.IsCode(err, errors.Internal) {
				t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Internal)
			}
		})
	}
}
