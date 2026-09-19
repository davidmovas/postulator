package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := moduleRoot("")
	if err != nil {
		t.Fatalf("moduleRoot: %v", err)
	}
	return root
}

func TestGeneratedFileIsInSync(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	generated := filepath.Join(t.TempDir(), "vocab.ts")
	if err := write(root, generated); err != nil {
		t.Fatalf("write: %v", err)
	}

	fresh, err := os.ReadFile(generated)
	if err != nil {
		t.Fatalf("read the generated file: %v", err)
	}

	committed, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(generatedFile)))
	if err != nil {
		t.Fatalf("read the committed file: %v", err)
	}

	if bytes.Contains(committed, []byte("\r\n")) {
		t.Fatalf("%s carries CRLF; .gitattributes pins it to LF", generatedFile)
	}
	if !bytes.Equal(fresh, committed) {
		t.Fatalf("%s is out of date; run `task vocab`", generatedFile)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	first, err := collect(root)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	second, err := collect(root)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	var one, two bytes.Buffer
	if err = render(&one, first); err != nil {
		t.Fatalf("render: %v", err)
	}
	if err = render(&two, second); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.Equal(one.Bytes(), two.Bytes()) {
		t.Fatal("two renders of the same sources differ")
	}
}

func TestEveryCatalogEntryResolves(t *testing.T) {
	t.Parallel()

	blocks, err := collect(repoRoot(t))
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(blocks) != len(catalog()) {
		t.Fatalf("collect produced %d blocks for %d entries", len(blocks), len(catalog()))
	}

	for _, entry := range blocks {
		if entry.aliasExport != "" {
			continue
		}
		if len(entry.values) == 0 {
			t.Fatalf("%s resolved to nothing", entry.export)
		}
		for _, derived := range entry.derived {
			for _, value := range derived.values {
				if !slices.Contains(entry.values, value) {
					t.Fatalf("%s carries %q, which %s does not declare", derived.export, value, entry.export)
				}
			}
		}
	}
}

func TestTheDerivedGroupingsReadTheirGoPredicates(t *testing.T) {
	t.Parallel()

	blocks, err := collect(repoRoot(t))
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	want := map[string][]string{
		"activeRunStatuses":            {"pending", "running", "waiting", "paused"},
		"terminalRunStatuses":          {"completed", "failed", "cancelled"},
		"purgeableArtifactKinds":       {"draft", "body_html", "images"},
		"toolRisksNeedingConfirmation": {"write", "dangerous"},
		"blockingImportFindingCodes": {
			"bad_path", "unknown_parent", "unknown_related", "self_edge", "cycle",
		},
	}

	seen := make(map[string][]string)
	for _, entry := range blocks {
		for _, derived := range entry.derived {
			seen[derived.export] = derived.values
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("the catalog derives %v", seen)
	}
	for export, values := range want {
		if !slices.Equal(seen[export], values) {
			t.Fatalf("%s = %v, want %v", export, seen[export], values)
		}
	}
}

func TestConstValuesKeepsDeclarationOrderAcrossFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	write("b_second.go", "package sample\n\nconst (\n\tThird Shade = \"third\"\n)\n")
	write("a_first.go", "package sample\n\ntype Shade string\n\nconst (\n\tSecond Shade = \"second\"\n"+
		"\tFirst Shade = \"first\"\n)\n\nconst Other Tint = \"ignored\"\n")
	write("c_third_test.go", "package sample\n\nconst Fourth Shade = \"fourth\"\n")

	values, err := constValues(dir, "Shade")
	if err != nil {
		t.Fatalf("constValues: %v", err)
	}
	if !slices.Equal(values, []string{"second", "first", "third"}) {
		t.Fatalf("constValues = %v", values)
	}
}

func TestConstValuesRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		typ  string
		code errors.Code
	}{
		{
			name: "no constant of the type",
			body: "package sample\n\nconst Other Tint = \"ignored\"\n",
			typ:  "Shade",
			code: errors.NotFound,
		},
		{
			name: "a constant that is not a string literal",
			body: "package sample\n\nconst First Shade = 1\n",
			typ:  "Shade",
			code: errors.Invalid,
		},
		{
			name: "a source the parser rejects",
			body: "package sample\n\nconst First Shade =\n",
			typ:  "Shade",
			code: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(tc.body), 0o600); err != nil {
				t.Fatalf("write the sample: %v", err)
			}
			if _, err := constValues(dir, tc.typ); !errors.IsCode(err, tc.code) {
				t.Fatalf("constValues = %v, want %s", err, tc.code)
			}
		})
	}

	if _, err := constValues(filepath.Join(t.TempDir(), "absent"), "Shade"); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("constValues of an absent package = %v", err)
	}
}

func TestRenderRefusesAnEmptyCatalog(t *testing.T) {
	t.Parallel()

	if err := render(&bytes.Buffer{}, nil); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("render of nothing = %v", err)
	}
}

func TestRenderWrapsOnlyWhatDoesNotFit(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := render(&out, []block{
		{export: "shorts", tsType: "Short", values: []string{"a", "b"}},
		{
			export: "longs", tsType: "Long",
			values: []string{strings.Repeat("x", 60), strings.Repeat("y", 60)},
		},
	}); err != nil {
		t.Fatalf("render: %v", err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, `export const shorts = ["a", "b"] as const;`) {
		t.Fatalf("a short list must stay on one line:\n%s", rendered)
	}
	if !strings.Contains(rendered, "export const longs = [\n    \""+strings.Repeat("x", 60)+"\",\n") {
		t.Fatalf("a long list must wrap:\n%s", rendered)
	}
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "\r") {
			t.Fatal("the generated module must carry no carriage return")
		}
	}
}

func TestModuleRootWalksUpToGoMod(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	nested, err := moduleRoot(filepath.Join(root, "internal", "domain", "run"))
	if err != nil || nested != root {
		t.Fatalf("moduleRoot = %q, %v; want %q", nested, err, root)
	}
	if _, err = moduleRoot(filepath.Dir(filepath.VolumeName(root) + string(filepath.Separator))); err == nil {
		t.Fatal("moduleRoot outside the module must fail")
	}
}
