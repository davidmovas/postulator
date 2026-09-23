package plugin_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp/plugin"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var epoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

func tree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return root
}

func entries(t *testing.T, archive []byte) []*zip.File {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("read the archive: %v", err)
	}
	return reader.File
}

func TestPackIsSortedPrefixedAndDeterministic(t *testing.T) {
	t.Parallel()

	root := tree(t, map[string]string{
		"z.php":                    "z",
		"includes/http.php":        "http",
		"postulator-companion.php": "<?php\n",
	})

	first, err := plugin.Pack(os.DirFS(root), "postulator-companion")
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	second, err := plugin.Pack(os.DirFS(root), "postulator-companion")
	if err != nil {
		t.Fatalf("Pack again: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("two runs produced different bytes")
	}

	want := []string{
		"postulator-companion/includes/http.php",
		"postulator-companion/postulator-companion.php",
		"postulator-companion/z.php",
	}
	files := entries(t, first)
	if len(files) != len(want) {
		t.Fatalf("the archive holds %d entries, want %d", len(files), len(want))
	}
	for i, name := range want {
		if files[i].Name != name {
			t.Errorf("entry %d = %q, want %q", i, files[i].Name, name)
		}
		if !files[i].Modified.Equal(epoch) {
			t.Errorf("entry %q modified %s, want %s", files[i].Name, files[i].Modified, epoch)
		}
		if files[i].Mode().Perm() != 0o644 {
			t.Errorf("entry %q mode %s", files[i].Name, files[i].Mode())
		}
	}
}

func TestPackRefusesAnEmptyOrMissingTree(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		dir  string
		want errors.Code
	}{
		{name: "empty", dir: t.TempDir(), want: errors.NotFound},
		{name: "absent", dir: filepath.Join(t.TempDir(), "gone"), want: errors.External},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := plugin.Pack(os.DirFS(tc.dir), "postulator-companion"); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}

func TestPackageCarriesTheEmbeddedPlugin(t *testing.T) {
	t.Parallel()

	archive, err := plugin.Package()
	if err != nil {
		t.Fatalf("Package: %v", err)
	}

	found := false
	for _, entry := range entries(t, archive) {
		if entry.Name == "postulator-companion/postulator-companion.php" {
			found = true
		}
	}
	if !found {
		t.Fatal("the archive does not hold the plugin main file")
	}

	again, err := plugin.Package()
	if err != nil {
		t.Fatalf("Package again: %v", err)
	}
	if !bytes.Equal(archive, again) {
		t.Fatal("two packages of the embedded plugin differ")
	}
}
