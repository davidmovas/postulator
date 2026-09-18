package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

func TestCollect(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files map[string]string
		want  []string
		fails bool
	}{
		{
			name:  "sorted and slash separated",
			files: map[string]string{"z.php": "z", "includes/b.php": "b", "a.php": "a"},
			want:  []string{"a.php", "includes/b.php", "z.php"},
		},
		{
			name:  "single file",
			files: map[string]string{"only.php": "x"},
			want:  []string{"only.php"},
		},
		{
			name:  "empty tree is an error",
			files: map[string]string{},
			fails: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := collect(writeTree(t, testCase.files))
			if testCase.fails {
				if err == nil {
					t.Fatalf("collect: no error, want one")
				}
				return
			}
			if err != nil {
				t.Fatalf("collect: %v", err)
			}
			if len(got) != len(testCase.want) {
				t.Fatalf("collect = %v, want %v", got, testCase.want)
			}
			for i, name := range testCase.want {
				if got[i] != name {
					t.Errorf("collect[%d] = %q, want %q", i, got[i], name)
				}
			}
		})
	}
}

func TestWriteIsDeterministic(t *testing.T) {
	t.Parallel()

	root := writeTree(t, map[string]string{
		"postulator-companion.php": "<?php\n",
		"includes/http.php":        "<?php\n",
	})
	names, err := collect(root)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}

	var first, second bytes.Buffer
	if err := write(root, "postulator-companion", names, &first); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := write(root, "postulator-companion", names, &second); err != nil {
		t.Fatalf("write second: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("two runs produced %d and %d bytes that differ", first.Len(), second.Len())
	}

	archive, err := zip.NewReader(bytes.NewReader(first.Bytes()), int64(first.Len()))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	want := []string{"postulator-companion/includes/http.php", "postulator-companion/postulator-companion.php"}
	if len(archive.File) != len(want) {
		t.Fatalf("archive holds %d entries, want %d", len(archive.File), len(want))
	}
	for i, entry := range archive.File {
		if entry.Name != want[i] {
			t.Errorf("entry %d = %q, want %q", i, entry.Name, want[i])
		}
		if !entry.Modified.Equal(epoch) {
			t.Errorf("entry %q modified %s, want %s", entry.Name, entry.Modified, epoch)
		}
	}
}

func TestRunPackagesTheRealPlugin(t *testing.T) {
	t.Parallel()

	out := filepath.Join(t.TempDir(), "postulator-companion.zip")
	if err := run(filepath.Join("..", "..", "wp-plugin", "postulator-companion"), out); err != nil {
		t.Fatalf("run: %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read %s: %v", out, err)
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}

	found := false
	for _, entry := range archive.File {
		if entry.Name == "postulator-companion/postulator-companion.php" {
			found = true
		}
	}
	if !found {
		t.Fatalf("archive does not hold the plugin main file")
	}
}

func TestRunRejectsAMissingSource(t *testing.T) {
	t.Parallel()

	if err := run(filepath.Join(t.TempDir(), "absent"), filepath.Join(t.TempDir(), "out.zip")); err == nil {
		t.Fatalf("run: no error, want one")
	}
}
