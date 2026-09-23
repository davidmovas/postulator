package tor

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestTheCommandIsDetachedAndRunsBesideTheExecutable(t *testing.T) {
	t.Parallel()

	exe := filepath.Join(t.TempDir(), "Tor Browser", "Browser", "firefox.exe")
	const url = "https://example.test/grinders/hand/"

	built := command(exe, url)
	if built.Path != exe {
		t.Errorf("Path = %q, want %q", built.Path, exe)
	}
	if len(built.Args) != 2 || built.Args[1] != url {
		t.Fatalf("Args = %v, want the executable and the address", built.Args)
	}
	if built.Dir != filepath.Dir(exe) {
		t.Errorf("Dir = %q, want %q", built.Dir, filepath.Dir(exe))
	}
	if built.SysProcAttr == nil {
		t.Fatal("the command carries no process attributes")
	}

	const want = windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP
	if built.SysProcAttr.CreationFlags != want {
		t.Errorf("CreationFlags = %#x, want %#x", built.SysProcAttr.CreationFlags, want)
	}
}
