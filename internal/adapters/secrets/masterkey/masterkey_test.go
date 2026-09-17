package masterkey_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/secrets/dpapi"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestLoadCreatesAndReuses(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "Postulator")

	first, err := masterkey.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(first) != masterkey.Length {
		t.Fatalf("length = %d, want %d", len(first), masterkey.Length)
	}
	if bytes.Equal(first, make([]byte, masterkey.Length)) {
		t.Fatal("the key must not be all zeroes")
	}

	stored, err := os.ReadFile(filepath.Join(dir, masterkey.FileName))
	if err != nil {
		t.Fatalf("read the key file: %v", err)
	}
	if bytes.Contains(stored, first) {
		t.Fatal("the key file must not contain the raw key")
	}

	second, err := masterkey.Load(dir)
	if err != nil {
		t.Fatalf("Load again: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("a second Load must return the same key")
	}
}

func TestLoadGeneratesDistinctKeys(t *testing.T) {
	t.Parallel()

	first, err := masterkey.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	second, err := masterkey.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Error("two directories must get different keys")
	}
}

func TestLoadRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		dir  func(t *testing.T) string
		want errors.Code
	}{
		{
			name: "empty directory",
			dir:  func(*testing.T) string { return "" },
			want: errors.Invalid,
		},
		{
			name: "corrupt key file",
			dir: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, masterkey.FileName), []byte("not protected"), 0o600); err != nil {
					t.Fatalf("write a corrupt key file: %v", err)
				}
				return dir
			},
			want: errors.Locked,
		},
		{
			name: "key file is a directory",
			dir: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, masterkey.FileName), 0o700); err != nil {
					t.Fatalf("create a directory in place of the key file: %v", err)
				}
				return dir
			},
			want: errors.Internal,
		},
		{
			name: "key of the wrong length",
			dir: func(t *testing.T) string {
				t.Helper()

				protected, err := dpapi.Protect([]byte("short"))
				if err != nil {
					t.Fatalf("Protect: %v", err)
				}

				dir := t.TempDir()
				if err = os.WriteFile(filepath.Join(dir, masterkey.FileName), protected, 0o600); err != nil {
					t.Fatalf("write the key file: %v", err)
				}
				return dir
			},
			want: errors.Internal,
		},
		{
			name: "temporary file path is a directory",
			dir: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, masterkey.FileName+".tmp"), 0o700); err != nil {
					t.Fatalf("create a directory in place of the temporary file: %v", err)
				}
				return dir
			},
			want: errors.Internal,
		},
		{
			name: "directory cannot be created",
			dir: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				blocker := filepath.Join(dir, "blocker")
				if err := os.WriteFile(blocker, []byte("file"), 0o600); err != nil {
					t.Fatalf("write the blocking file: %v", err)
				}
				return filepath.Join(blocker, "Postulator")
			},
			want: errors.Internal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := masterkey.Load(tc.dir(t)); errors.CodeOf(err) != tc.want {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), tc.want)
			}
		})
	}
}

func TestLoadIgnoresAStaleTempFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	first, err := masterkey.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	stale := filepath.Join(dir, masterkey.FileName+".tmp")
	if err = os.WriteFile(stale, []byte("left over from a crash"), 0o600); err != nil {
		t.Fatalf("write the stale temporary file: %v", err)
	}

	second, err := masterkey.Load(dir)
	if err != nil {
		t.Fatalf("Load with a stale temporary file present: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("a stale temporary file must not change the stored key")
	}
}

func TestCreateOverwritesAStaleTempFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stale := filepath.Join(dir, masterkey.FileName+".tmp")
	if err := os.WriteFile(stale, []byte("left over from a crash"), 0o600); err != nil {
		t.Fatalf("write the stale temporary file: %v", err)
	}

	key, err := masterkey.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(key) != masterkey.Length {
		t.Fatalf("length = %d, want %d", len(key), masterkey.Length)
	}

	if _, err = os.Stat(stale); !os.IsNotExist(err) {
		t.Error("the temporary file must not survive a successful write")
	}
}

func TestLoadReportsAKeyItCannotUnprotect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, masterkey.FileName), []byte("not protected"), 0o600); err != nil {
		t.Fatalf("write a corrupt key file: %v", err)
	}

	_, err := masterkey.Load(dir)
	if !errors.IsCode(err, errors.Locked) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Locked)
	}
	if !strings.Contains(err.Error(), `remove %APPDATA%\Postulator\master.key to reset`) {
		t.Errorf("message = %q, want the reset instruction", err.Error())
	}

	if _, statErr := os.Stat(filepath.Join(dir, masterkey.FileName)); statErr != nil {
		t.Error("the key file must never be deleted automatically")
	}
}
