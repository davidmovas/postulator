package masterkey

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestWriteAtomicallyLeavesNoTemporaryFileBehind(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, FileName)

	if err := writeAtomically(path, []byte("protected")); err != nil {
		t.Fatalf("writeAtomically: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(contents) != "protected" {
		t.Errorf("contents = %q, want %q", contents, "protected")
	}
	if _, err = os.Stat(path + tempSuffix); !os.IsNotExist(err) {
		t.Error("the temporary file must not survive")
	}
}

func TestWriteAtomicallyCleansUpWhenTheRenameFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("create a directory in place of the key file: %v", err)
	}

	err := writeAtomically(path, []byte("protected"))
	if !errors.IsCode(err, errors.Internal) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Internal)
	}
	if _, statErr := os.Stat(path + tempSuffix); !os.IsNotExist(statErr) {
		t.Error("a failed rename must remove the temporary file")
	}
}

func TestRemoveIgnoresAnAbsentFile(t *testing.T) {
	t.Parallel()

	if err := remove(filepath.Join(t.TempDir(), "absent")); err != nil {
		t.Errorf("remove on an absent file must succeed, got %v", err)
	}
}
