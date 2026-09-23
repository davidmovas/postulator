package tor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	SourceSetting  = "setting"
	SourceDetected = "detected"

	ClosedToLinksCode = "tor_closed_to_links"

	folderName     = "Tor Browser"
	executableName = "firefox.exe"
	markerName     = "TorBrowser"

	allowRemote = "--allow-remote"
	newTab      = "-new-tab"

	profileVariable = "USERPROFILE"
	localVariable   = "LOCALAPPDATA"
	programVariable = "PROGRAMFILES"
)

type Lookup func(name string) string

type desktop interface {
	Accepting(profile string) (bool, error)
	Running(exe string) (bool, error)
	Start(path string, args ...string) error
}

type Option func(*Browser)

func WithDesktop(d desktop) Option {
	return func(b *Browser) {
		if d != nil {
			b.desktop = d
		}
	}
}

type Browser struct {
	lookup  Lookup
	desktop desktop
}

func New(lookup Lookup, opts ...Option) *Browser {
	if lookup == nil {
		lookup = os.Getenv
	}
	browser := &Browser{lookup: lookup, desktop: system{}}
	for _, opt := range opts {
		opt(browser)
	}
	return browser
}

func (b *Browser) Locate(configured string) (path, source string, ok bool) {
	if trimmed := strings.TrimSpace(configured); trimmed != "" && isTorBrowser(trimmed) {
		return trimmed, SourceSetting, true
	}

	for _, root := range b.roots() {
		candidate := filepath.Join(root, folderName, "Browser", executableName)
		if isTorBrowser(candidate) {
			return candidate, SourceDetected, true
		}
	}
	return "", "", false
}

func (b *Browser) roots() []string {
	profile := strings.TrimSpace(b.lookup(profileVariable))
	roots := make([]string, 0, 5)

	if profile != "" {
		roots = append(roots, filepath.Join(profile, "Desktop"), filepath.Join(profile, "OneDrive", "Desktop"))
	}
	if local := strings.TrimSpace(b.lookup(localVariable)); local != "" {
		roots = append(roots, local)
	}
	if program := strings.TrimSpace(b.lookup(programVariable)); program != "" {
		roots = append(roots, program)
	}
	if profile != "" {
		roots = append(roots, filepath.Join(profile, "Downloads"))
	}
	return roots
}

func isTorBrowser(exe string) bool {
	info, err := os.Stat(exe)
	if err != nil || info.IsDir() {
		return false
	}

	marker, err := os.Stat(filepath.Join(filepath.Dir(exe), markerName))
	return err == nil && marker.IsDir()
}

func (b *Browser) Open(path, url string) error {
	accepting, err := b.desktop.Accepting(profileOf(path))
	if err != nil {
		return err
	}
	if accepting {
		return b.desktop.Start(path, allowRemote, newTab, url)
	}

	running, err := b.desktop.Running(path)
	if err != nil {
		return err
	}
	if running {
		return errors.New(errors.Conflict, "Tor Browser is open but was started without taking links from "+
			"other programs, so a new tab cannot be added to it; close Tor Browser and open the link again, "+
			"and every link after that opens as a new tab").
			WithDetail("code", ClosedToLinksCode)
	}
	return b.desktop.Start(path, allowRemote, url)
}

func profileOf(exe string) string {
	return filepath.Join(filepath.Dir(exe), markerName, "Data", "Browser", "profile.default")
}

func command(path string, args ...string) *exec.Cmd {
	built := exec.Command(path, args...)
	built.Dir = filepath.Dir(path)
	built.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	return built
}
