package tor

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/sys/windows"
)

func TestTheCommandIsDetachedAndRunsBesideTheExecutable(t *testing.T) {
	t.Parallel()

	exe := filepath.Join(t.TempDir(), "Tor Browser", "Browser", "firefox.exe")
	const url = "https://example.test/grinders/hand/"

	built := command(exe, allowRemote, url)
	if built.Path != exe {
		t.Errorf("Path = %q, want %q", built.Path, exe)
	}
	if !slices.Equal(built.Args, []string{exe, allowRemote, url}) {
		t.Fatalf("Args = %v, want the executable, the flag and the address", built.Args)
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

func TestARemoteWindowIsMatchedToItsProfile(t *testing.T) {
	t.Parallel()

	profile := `C:\Users\someone\Desktop\Tor Browser\Browser\TorBrowser\Data\Browser\profile.default`

	cases := []struct {
		name  string
		class string
		want  bool
	}{
		{name: "the window Tor Browser opens for links", class: "Mozilla_tor browser_" + profile + "_RemoteWindow", want: true},
		{name: "the same profile in another case", class: "Mozilla_tor browser_c:\\users\\someone\\desktop\\tor browser\\browser\\torbrowser\\data\\browser\\profile.default_RemoteWindow", want: true},
		{name: "another installation", class: `Mozilla_tor browser_D:\Tor Browser\Browser\TorBrowser\Data\Browser\profile.default_RemoteWindow`},
		{name: "a window that takes no links", class: "MozillaWindowClass"},
		{name: "a Firefox of its own", class: "Mozilla_firefox_default_RemoteWindow"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := takesLinksFor(tc.class, profile); got != tc.want {
				t.Fatalf("takesLinksFor(%q) = %v, want %v", tc.class, got, tc.want)
			}
		})
	}
}

func TestTheSystemDesktopReadsWhatIsRunning(t *testing.T) {
	t.Parallel()

	running, err := system{}.Running(os.Args[0])
	if err != nil || !running {
		t.Fatalf("Running(the test itself) = %v, %v; want true", running, err)
	}

	elsewhere := filepath.Join(t.TempDir(), "firefox.exe")
	if running, err = (system{}).Running(elsewhere); err != nil || running {
		t.Fatalf("Running(%s) = %v, %v; want false", elsewhere, running, err)
	}

	accepting, err := system{}.Accepting(filepath.Join(t.TempDir(), "profile.default"))
	if err != nil || accepting {
		t.Fatalf("Accepting(a profile nobody opened) = %v, %v; want false", accepting, err)
	}
}
