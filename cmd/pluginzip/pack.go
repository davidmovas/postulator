package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"
)

var epoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

func collect(source string) ([]string, error) {
	var names []string
	err := filepath.WalkDir(source, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", source, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s holds no files", source)
	}
	slices.Sort(names)
	return names, nil
}

func write(source, root string, names []string, sink *bytes.Buffer) error {
	archive := zip.NewWriter(sink)
	for _, name := range names {
		header := &zip.FileHeader{Name: path.Join(root, name), Method: zip.Deflate, Modified: epoch}
		header.SetMode(0o644)

		entry, err := archive.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("create entry %s: %w", name, err)
		}
		body, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := entry.Write(body); err != nil {
			return fmt.Errorf("write entry %s: %w", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close archive: %w", err)
	}
	return nil
}

func run(source, out string) error {
	names, err := collect(source)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	if err := write(source, filepath.Base(filepath.Clean(source)), names, &buffer); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(out), err)
	}
	if err := os.WriteFile(out, buffer.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}
