package wpplugin_test

import (
	"bufio"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	wpplugin "github.com/davidmovas/postulator/wp-plugin"
)

var seenByTheAdminOnly = regexp.MustCompile(`^ \* Plugin Name: Postulator Companion$|Postulator\\Companion|'postulator/v1'|'_postulator_[a-z_]+'`)

func TestThePluginNamesTheProgramOnlyWhereTheAdminSeesIt(t *testing.T) {
	t.Parallel()

	scanned := 0
	err := fs.WalkDir(wpplugin.Files, wpplugin.Root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		file, err := wpplugin.Files.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()

		scanned++
		lines := bufio.NewScanner(file)
		for number := 1; lines.Scan(); number++ {
			rest := seenByTheAdminOnly.ReplaceAllString(lines.Text(), "")
			if strings.Contains(strings.ToLower(rest), "postulator") {
				t.Errorf("%s:%d names the program where a visitor can meet it: %s", path, number, strings.TrimSpace(lines.Text()))
			}
		}
		return lines.Err()
	})
	if err != nil {
		t.Fatalf("walk the plugin tree: %v", err)
	}
	if scanned == 0 {
		t.Fatal("the plugin tree holds no file")
	}
}
