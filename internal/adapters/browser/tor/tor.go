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

	folderName     = "Tor Browser"
	executableName = "firefox.exe"
	markerName     = "TorBrowser"

	profileVariable = "USERPROFILE"
	localVariable   = "LOCALAPPDATA"
	programVariable = "PROGRAMFILES"
)

type Lookup func(name string) string

type Browser struct {
	lookup Lookup
}

func New(lookup Lookup) *Browser {
	if lookup == nil {
		lookup = os.Getenv
	}
	return &Browser{lookup: lookup}
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
	started := command(path, url)
	if err := started.Start(); err != nil {
		return errors.Wrap(err, errors.External, "start Tor Browser")
	}
	return errors.Wrap(started.Process.Release(), errors.External, "let Tor Browser run on its own")
}

func command(path, url string) *exec.Cmd {
	built := exec.Command(path, url)
	built.Dir = filepath.Dir(path)
	built.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	return built
}
