package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

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
