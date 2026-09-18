package plugin

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"path"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	wpplugin "github.com/davidmovas/postulator/wp-plugin"
)

const Filename = "postulator-companion.zip"

var epoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

func Package() ([]byte, error) {
	tree, err := fs.Sub(wpplugin.Files, wpplugin.Root)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "open the embedded companion plugin")
	}
	return Pack(tree, wpplugin.Root)
}

func Pack(tree fs.FS, root string) ([]byte, error) {
	names, err := collect(tree)
	if err != nil {
		return nil, err
	}

	var sink bytes.Buffer
	archive := zip.NewWriter(&sink)
	for _, name := range names {
		header := &zip.FileHeader{Name: path.Join(root, name), Method: zip.Deflate, Modified: epoch}
		header.SetMode(0o644)

		entry, createErr := archive.CreateHeader(header)
		if createErr != nil {
			return nil, errors.Wrap(createErr, errors.Internal, "create the archive entry "+name)
		}
		body, readErr := fs.ReadFile(tree, name)
		if readErr != nil {
			return nil, errors.Wrap(readErr, errors.External, "read the plugin file "+name)
		}
		if _, writeErr := entry.Write(body); writeErr != nil {
			return nil, errors.Wrap(writeErr, errors.Internal, "write the archive entry "+name)
		}
	}
	if err = archive.Close(); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "close the plugin archive")
	}
	return sink.Bytes(), nil
}

func collect(tree fs.FS) ([]string, error) {
	names := make([]string, 0)
	err := fs.WalkDir(tree, ".", func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		names = append(names, current)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, errors.External, "scan the plugin tree")
	}
	if len(names) == 0 {
		return nil, errors.New(errors.NotFound, "the plugin tree holds no file")
	}
	slices.Sort(names)
	return names, nil
}
