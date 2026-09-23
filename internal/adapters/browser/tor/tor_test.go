package tor_test

import (
	stderrors "errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/browser/tor"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const childVariable = "POSTULATOR_TOR_TEST_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(childVariable) != "" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func install(t *testing.T, root string, withTorDirectory bool) string {
	t.Helper()

	browser := filepath.Join(root, "Tor Browser", "Browser")
	if err := os.MkdirAll(browser, 0o750); err != nil {
		t.Fatalf("lay out %s: %v", browser, err)
	}
	if withTorDirectory {
		if err := os.MkdirAll(filepath.Join(browser, "TorBrowser"), 0o750); err != nil {
			t.Fatalf("lay out the TorBrowser directory: %v", err)
		}
	}

	exe := filepath.Join(browser, "firefox.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o600); err != nil {
		t.Fatalf("write %s: %v", exe, err)
	}
	return exe
}

func lookupOf(values map[string]string) tor.Lookup {
	return func(name string) string { return values[name] }
}

func TestLocateWalksTheRootsInOrder(t *testing.T) {
	t.Parallel()

	profile := t.TempDir()
	local := t.TempDir()
	files := t.TempDir()

	cases := []struct {
		name    string
		install string
		lookup  map[string]string
		want    func() string
	}{
		{
			name:    "the desktop",
			install: filepath.Join(profile, "Desktop"),
			lookup:  map[string]string{"USERPROFILE": profile},
			want:    func() string { return filepath.Join(profile, "Desktop") },
		},
		{
			name:    "the OneDrive desktop",
			install: filepath.Join(profile, "OneDrive", "Desktop"),
			lookup:  map[string]string{"USERPROFILE": profile},
			want:    func() string { return filepath.Join(profile, "OneDrive", "Desktop") },
		},
		{
			name:    "the local application data",
			install: local,
			lookup:  map[string]string{"LOCALAPPDATA": local},
			want:    func() string { return local },
		},
		{
			name:    "the program files",
			install: files,
			lookup:  map[string]string{"PROGRAMFILES": files},
			want:    func() string { return files },
		},
		{
			name:    "the downloads folder",
			install: filepath.Join(profile, "Downloads"),
			lookup:  map[string]string{"USERPROFILE": profile},
			want:    func() string { return filepath.Join(profile, "Downloads") },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installed := install(t, tc.install, true)
			t.Cleanup(func() {
				if err := os.RemoveAll(filepath.Join(tc.install, "Tor Browser")); err != nil {
					t.Errorf("clean up: %v", err)
				}
			})

			path, source, ok := tor.New(lookupOf(tc.lookup)).Locate("")
			if !ok {
				t.Fatalf("Locate found nothing under %s", tc.want())
			}
			if path != installed {
				t.Errorf("path = %q, want %q", path, installed)
			}
			if source != tor.SourceDetected {
				t.Errorf("source = %q, want %q", source, tor.SourceDetected)
			}
		})
	}
}

func TestLocateRefusesAFirefoxThatIsNotTor(t *testing.T) {
	t.Parallel()

	profile := t.TempDir()
	install(t, filepath.Join(profile, "Desktop"), false)

	if path, source, ok := tor.New(lookupOf(map[string]string{"USERPROFILE": profile})).Locate(""); ok {
		t.Fatalf("Locate accepted %q from %q; a Tor Browser carries a TorBrowser directory", path, source)
	}
}

func TestLocatePrefersTheConfiguredPath(t *testing.T) {
	t.Parallel()

	profile := t.TempDir()
	elsewhere := t.TempDir()
	detected := install(t, filepath.Join(profile, "Desktop"), true)
	configured := install(t, elsewhere, true)

	browser := tor.New(lookupOf(map[string]string{"USERPROFILE": profile}))

	path, source, ok := browser.Locate(configured)
	if !ok || path != configured || source != tor.SourceSetting {
		t.Fatalf("Locate = %q, %q, %v; want the configured path", path, source, ok)
	}

	path, source, ok = browser.Locate("   ")
	if !ok || path != detected || source != tor.SourceDetected {
		t.Fatalf("Locate = %q, %q, %v; want the detected path", path, source, ok)
	}
}

func TestLocateFallsBackWhenTheConfiguredPathIsNotTor(t *testing.T) {
	t.Parallel()

	profile := t.TempDir()
	elsewhere := t.TempDir()
	detected := install(t, filepath.Join(profile, "Desktop"), true)
	plainFirefox := install(t, elsewhere, false)

	path, source, ok := tor.New(lookupOf(map[string]string{"USERPROFILE": profile})).Locate(plainFirefox)
	if !ok || path != detected || source != tor.SourceDetected {
		t.Fatalf("Locate = %q, %q, %v; want the detected Tor Browser", path, source, ok)
	}
}

func TestLocateFindsNothingWithoutRoots(t *testing.T) {
	t.Parallel()

	if _, _, ok := tor.New(lookupOf(nil)).Locate(""); ok {
		t.Fatal("Locate found a browser with no roots to look in")
	}
}

type started struct {
	path string
	args []string
}

type desktop struct {
	accepting  bool
	running    bool
	probeErr   error
	startErr   error
	asked      string
	askedImage string
	starts     []started
}

func (d *desktop) Accepting(profile string) (bool, error) {
	d.asked = profile
	return d.accepting, d.probeErr
}

func (d *desktop) Running(exe string) (bool, error) {
	d.askedImage = exe
	return d.running, d.probeErr
}

func (d *desktop) Start(path string, args ...string) error {
	d.starts = append(d.starts, started{path: path, args: args})
	return d.startErr
}

func TestOpenAddsATabOrStartsTorBrowserSoLaterLinksBecomeTabs(t *testing.T) {
	t.Parallel()

	exe := filepath.Join(t.TempDir(), "Tor Browser", "Browser", "firefox.exe")
	profile := filepath.Join(filepath.Dir(exe), "TorBrowser", "Data", "Browser", "profile.default")
	const address = "https://example.test/espresso-machines/"

	cases := []struct {
		name      string
		desktop   desktop
		wantArgs  []string
		wantCode  errors.Code
		wantLeave bool
	}{
		{
			name:     "an open Tor Browser that takes links gets a new tab",
			desktop:  desktop{accepting: true, running: true},
			wantArgs: []string{"--allow-remote", "-new-tab", address},
		},
		{
			name:     "a closed Tor Browser is started so it takes the links after this one",
			desktop:  desktop{},
			wantArgs: []string{"--allow-remote", address},
		},
		{
			name:      "an open Tor Browser that takes no links is left alone",
			desktop:   desktop{running: true},
			wantCode:  errors.Conflict,
			wantLeave: true,
		},
		{
			name:      "a desktop that cannot be read starts nothing",
			desktop:   desktop{probeErr: errors.New(errors.External, "the window list could not be read")},
			wantCode:  errors.External,
			wantLeave: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seen := tc.desktop
			err := tor.New(lookupOf(nil), tor.WithDesktop(&seen)).Open(exe, address)

			if tc.wantCode != "" {
				if !errors.IsCode(err, tc.wantCode) {
					t.Fatalf("Open = %v, want %s", err, tc.wantCode)
				}
			} else if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if tc.wantLeave {
				if len(seen.starts) != 0 {
					t.Fatalf("Open started %+v; a second Tor Browser asks the person to close the first", seen.starts)
				}
				return
			}

			if seen.asked != profile {
				t.Errorf("Open looked for the links of %q, want the profile %q", seen.asked, profile)
			}
			if len(seen.starts) != 1 || seen.starts[0].path != exe || !slices.Equal(seen.starts[0].args, tc.wantArgs) {
				t.Fatalf("Open started %+v, want %s %v", seen.starts, exe, tc.wantArgs)
			}
		})
	}
}

func TestOpenNamesWhatToDoWhenTorBrowserTakesNoLinks(t *testing.T) {
	t.Parallel()

	exe := filepath.Join(t.TempDir(), "Tor Browser", "Browser", "firefox.exe")
	err := tor.New(lookupOf(nil), tor.WithDesktop(&desktop{running: true})).Open(exe, "https://example.test/")

	var refusal *errors.Error
	if !stderrors.As(err, &refusal) || refusal.Details["code"] != tor.ClosedToLinksCode {
		t.Fatalf("Open = %v, want the %s code", err, tor.ClosedToLinksCode)
	}
	if !strings.Contains(err.Error(), "close Tor Browser") {
		t.Fatalf("the refusal %q does not say what to do", err.Error())
	}
}

func TestOpenStartsTheProcessAndReturns(t *testing.T) {
	t.Setenv(childVariable, "1")

	stand := filepath.Join(t.TempDir(), "Tor Browser", "Browser", "firefox.exe")
	if err := os.MkdirAll(filepath.Dir(stand), 0o750); err != nil {
		t.Fatalf("lay out the stand-in: %v", err)
	}
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatalf("read the test binary: %v", err)
	}
	if err = os.WriteFile(stand, binary, 0o700); err != nil {
		t.Fatalf("write the stand-in: %v", err)
	}

	if err = tor.New(lookupOf(nil)).Open(stand, "https://example.test/espresso-machines/"); err != nil {
		t.Fatalf("Open: %v", err)
	}
}

func TestOpenReportsAnExecutableItCannotStart(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "Tor Browser", "Browser", "firefox.exe")
	err := tor.New(lookupOf(nil)).Open(missing, "https://example.test/")
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Open = %v, want an EXTERNAL failure", err)
	}
}
