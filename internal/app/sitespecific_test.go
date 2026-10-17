package app_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const replicaStand = "docker/e2e/stands/replica"

var (
	fixtureClass = regexp.MustCompile(`class=\\?["']([^"'\\]+)`)
	fixtureTheme = regexp.MustCompile(`(?m)^\s*ensure_theme\s+(\S+)`)
)

func clientNames(t *testing.T, root string) []string {
	t.Helper()

	behaviour, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(replicaStand), "site-behaviour.php"))
	if err != nil {
		t.Fatalf("read the replica's site behaviour: %v", err)
	}
	provision, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(replicaStand), "provision.sh"))
	if err != nil {
		t.Fatalf("read the replica's provisioning: %v", err)
	}

	var names []string
	for _, match := range fixtureClass.FindAllStringSubmatch(string(behaviour), -1) {
		names = append(names, strings.Fields(match[1])...)
	}
	for _, match := range fixtureTheme.FindAllStringSubmatch(string(provision), -1) {
		names = append(names, match[1])
	}
	slices.Sort(names)
	names = slices.Compact(names)
	if len(names) == 0 {
		t.Fatal("the replica fixture names no class and no theme, so the scan would prove nothing")
	}
	return names
}

func TestNothingSiteSpecificLivesInTheSources(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	names := clientNames(t, root)
	patterns := make(map[string]*regexp.Regexp, len(names))
	for _, name := range names {
		patterns[name] = regexp.MustCompile(`(?i)(^|[^a-z0-9_-])` + regexp.QuoteMeta(name) + `($|[^a-z0-9_-])`)
	}

	for _, tree := range []string{"internal", "cmd", "wp-plugin"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "testdata" || entry.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for name, pattern := range patterns {
				if pattern.Match(body) {
					relative, _ := filepath.Rel(root, path)
					t.Errorf("%s names %q, which belongs to the client's site and lives only in %s", filepath.ToSlash(relative), name, replicaStand)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
}
