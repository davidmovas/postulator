package tor

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	remotePrefix = "Mozilla_"
	remoteSuffix = "_RemoteWindow"
	classLength  = 1024
	pathLength   = windows.MAX_LONG_PATH
)

type system struct{}

func takesLinksFor(class, profile string) bool {
	lowered := strings.ToLower(class)
	ending := "_" + strings.ToLower(filepath.Clean(profile)) + strings.ToLower(remoteSuffix)
	return strings.HasPrefix(lowered, strings.ToLower(remotePrefix)) && strings.HasSuffix(lowered, ending)
}

func (system) Accepting(profile string) (bool, error) {
	found := false
	visit := syscall.NewCallback(func(window windows.HWND, _ uintptr) uintptr {
		name := make([]uint16, classLength)
		copied, err := windows.GetClassName(window, &name[0], int32(len(name)))
		if err == nil && copied > 0 && takesLinksFor(windows.UTF16ToString(name[:copied]), profile) {
			found = true
			return 0
		}
		return 1
	})
	if err := windows.EnumWindows(visit, nil); err != nil && !found {
		return false, errors.Wrap(err, errors.External, "read the windows open on this desktop")
	}
	return found, nil
}

func (system) Running(exe string) (running bool, err error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false, errors.Wrap(err, errors.External, "list the processes on this machine")
	}
	defer func() {
		if closeErr := windows.CloseHandle(snapshot); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, errors.External, "release the list of processes")
		}
	}()

	wanted := filepath.Clean(exe)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for walked := windows.Process32First(snapshot, &entry); walked == nil; walked = windows.Process32Next(snapshot, &entry) {
		if !strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), filepath.Base(wanted)) {
			continue
		}
		if image, known := imageOf(entry.ProcessID); known && strings.EqualFold(filepath.Clean(image), wanted) {
			return true, nil
		}
	}
	return false, nil
}

func imageOf(pid uint32) (image string, known bool) {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", false
	}
	defer func() {
		if windows.CloseHandle(process) != nil {
			image, known = "", false
		}
	}()

	buffer := make([]uint16, pathLength)
	size := uint32(len(buffer))
	if err = windows.QueryFullProcessImageName(process, 0, &buffer[0], &size); err != nil {
		return "", false
	}
	return windows.UTF16ToString(buffer[:size]), true
}

func (system) Start(path string, args ...string) error {
	started := command(path, args...)
	if err := started.Start(); err != nil {
		return errors.Wrap(err, errors.External, "start Tor Browser")
	}
	return errors.Wrap(started.Process.Release(), errors.External, "let Tor Browser run on its own")
}
