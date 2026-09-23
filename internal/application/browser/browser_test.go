package browser_test

import (
	stderrors "errors"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/browser"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func asKernel(err error, target **errors.Error) bool {
	return stderrors.As(err, target)
}

func contains(text, needle string) bool {
	return strings.Contains(text, needle)
}

type fakeBrowser struct {
	path       string
	source     string
	found      bool
	failure    error
	configured string
	opened     string
}

func (f *fakeBrowser) Locate(configured string) (path, source string, ok bool) {
	f.configured = configured
	return f.path, f.source, f.found
}

func (f *fakeBrowser) Open(path, address string) error {
	f.opened = path + " " + address
	return f.failure
}

func TestOpenRefusesAnAddressThatIsNotWeb(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
	}{
		{name: "empty", url: "   "},
		{name: "a file", url: "file:///C:/Windows/System32/calc.exe"},
		{name: "a script", url: "javascript:alert(1)"},
		{name: "a custom scheme", url: "postulator://open"},
		{name: "no scheme", url: "example.test/espresso-machines/"},
		{name: "no host", url: "https:///espresso-machines/"},
		{name: "unreadable", url: "https://exa mple.test/\x7f"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			found := &fakeBrowser{path: "C:/Tor Browser/Browser/firefox.exe", source: "detected", found: true}
			service := browser.New(browser.Deps{Browser: found})

			_, err := service.Open(t.Context(), browser.OpenRequest{URL: tc.url})
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Open = %v, want INVALID", err)
			}
			if found.opened != "" {
				t.Fatalf("a refused address still reached the browser: %q", found.opened)
			}
		})
	}
}

func TestOpenSaysWhenTorIsMissing(t *testing.T) {
	t.Parallel()

	service := browser.New(browser.Deps{Browser: &fakeBrowser{}})

	_, err := service.Open(t.Context(), browser.OpenRequest{URL: "https://example.test/grinders/"})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Open = %v, want INVALID", err)
	}

	var kernel *errors.Error
	if !asKernel(err, &kernel) {
		t.Fatalf("Open answered %v, want a kernel error", err)
	}
	if kernel.Details["code"] != browser.MissingCode {
		t.Fatalf("details = %v, want code %q", kernel.Details, browser.MissingCode)
	}
}

func TestOpenReportsALaunchThatFailed(t *testing.T) {
	t.Parallel()

	service := browser.New(browser.Deps{Browser: &fakeBrowser{
		path: "C:/Tor Browser/Browser/firefox.exe", source: "detected", found: true,
		failure: errors.New(errors.External, "start Tor Browser"),
	}})

	_, err := service.Open(t.Context(), browser.OpenRequest{URL: "https://example.test/grinders/"})
	if !errors.IsCode(err, errors.External) {
		t.Fatalf("Open = %v, want EXTERNAL", err)
	}
}

func TestOpenNeverNamesTheLinkItWasGiven(t *testing.T) {
	t.Parallel()

	const secret = "https://example.test/?postulator_preview=abc123secrettoken"

	cases := []struct {
		name    string
		browser *fakeBrowser
	}{
		{name: "no browser", browser: &fakeBrowser{}},
		{
			name: "a launch that failed",
			browser: &fakeBrowser{
				path: "firefox.exe", source: "detected", found: true,
				failure: errors.New(errors.External, "start Tor Browser"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := browser.New(browser.Deps{Browser: tc.browser})
			_, err := service.Open(t.Context(), browser.OpenRequest{URL: secret})
			if err == nil {
				t.Fatal("Open answered no error")
			}
			if contains(err.Error(), "abc123secrettoken") {
				t.Fatalf("the failure names the signed link: %q", err.Error())
			}
		})
	}
}

func TestOpenHandsTheConfiguredPathToTheLocator(t *testing.T) {
	t.Parallel()

	found := &fakeBrowser{path: "D:/Tor/firefox.exe", source: "setting", found: true}
	service := browser.New(browser.Deps{
		Browser: found,
		TorPath: func() string { return "D:/Tor/firefox.exe" },
	})

	if _, err := service.Open(t.Context(), browser.OpenRequest{
		URL: "https://example.test/espresso-machines/under-500/",
	}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if found.configured != "D:/Tor/firefox.exe" {
		t.Errorf("the locator was given %q, want the configured path", found.configured)
	}
	if found.opened != "D:/Tor/firefox.exe https://example.test/espresso-machines/under-500/" {
		t.Errorf("the browser opened %q", found.opened)
	}
}

func TestLocateReportsWhatItFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		browser *fakeBrowser
		want    browser.LocateResponse
	}{
		{
			name:    "nothing installed",
			browser: &fakeBrowser{},
			want:    browser.LocateResponse{},
		},
		{
			name:    "detected",
			browser: &fakeBrowser{path: "C:/Tor/firefox.exe", source: "detected", found: true},
			want:    browser.LocateResponse{Path: "C:/Tor/firefox.exe", Source: "detected", Installed: true},
		},
		{
			name:    "configured",
			browser: &fakeBrowser{path: "D:/Tor/firefox.exe", source: "setting", found: true},
			want:    browser.LocateResponse{Path: "D:/Tor/firefox.exe", Source: "setting", Installed: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := browser.New(browser.Deps{Browser: tc.browser}).Locate(t.Context(), browser.LocateRequest{})
			if err != nil {
				t.Fatalf("Locate: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Locate = %+v, want %+v", got, tc.want)
			}
		})
	}
}
