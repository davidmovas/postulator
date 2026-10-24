package app_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	replicaStand = "docker/e2e/stands/replica"
	classReader  = "internal/domain/render/builder.go"
	classLiteral = "class"
)

var (
	fixtureClass    = regexp.MustCompile(`class=\\?["']([^"'\\]+)`)
	fixtureTheme    = regexp.MustCompile(`(?m)^\s*ensure_theme\s+(\S+)`)
	selectorLiteral = regexp.MustCompile(`(?i)(class=|classname|queryselector|^\.[a-z_-]|^#[a-z_-])`)
	extractionTrees = []string{"internal/domain/render", "internal/application/observe"}
)

func clientNames(t *testing.T, root string) []string {
	t.Helper()

	muPlugin, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(replicaStand), "site-behaviour.php"))
	if err != nil {
		t.Fatalf("read the replica's mu-plugin: %v", err)
	}
	provision, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(replicaStand), "provision.sh"))
	if err != nil {
		t.Fatalf("read the replica's provisioning: %v", err)
	}

	var names []string
	for _, match := range fixtureClass.FindAllStringSubmatch(string(muPlugin), -1) {
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
					shown, relErr := filepath.Rel(root, path)
					if relErr != nil {
						shown = path
					}
					t.Errorf("%s names %q, which belongs to the client's site and lives only in %s", filepath.ToSlash(shown), name, replicaStand)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
}

type stringLiteral struct {
	value string
	line  int
}

func stringLiterals(t *testing.T, path string) []stringLiteral {
	t.Helper()

	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var found []stringLiteral
	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, unquoteErr := strconv.Unquote(literal.Value)
		if unquoteErr != nil {
			t.Fatalf("unquote %s at %s: %v", literal.Value, files.Position(literal.Pos()), unquoteErr)
		}
		found = append(found, stringLiteral{value: value, line: files.Position(literal.Pos()).Line})
		return true
	})
	return found
}

func TestTheExtractionNamesNoSelector(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	scanned := 0
	for _, tree := range extractionTrees {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(tree)), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			shown, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			shown = filepath.ToSlash(shown)
			scanned++
			for _, literal := range stringLiterals(t, path) {
				if selectorLiteral.MatchString(literal.value) {
					t.Errorf("%s:%d holds %q, a selector; the extraction finds the content by its text, never by a selector",
						shown, literal.line, literal.value)
				}
				if literal.value == classLiteral && shown != classReader {
					t.Errorf("%s:%d reads the class attribute; only %s reads class tokens, against the rules it is given",
						shown, literal.line, classReader)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", tree, err)
		}
	}
	if scanned == 0 {
		t.Fatal("no extraction source was scanned, so the test would prove nothing")
	}
}
