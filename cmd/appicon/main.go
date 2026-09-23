package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var iconSizes = []int{16, 24, 32, 48, 64, 128, 256}

const (
	windowsIcon = "build/windows/icon.ico"
	plateIcon   = "build/appicon.png"
	windowIcon  = "cmd/postulator/appicon.png"
	webMark     = "frontend/public/appmark.svg"

	plateIconSize  = 512
	windowIconSize = 256
)

func main() {
	root := flag.String("root", ".", "repository root the artifacts are written under")
	flag.Parse()

	written, err := run(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, path := range written {
		fmt.Println(path)
	}
}

func run(root string) ([]string, error) {
	icon, err := encodeICO(iconSizes)
	if err != nil {
		return nil, err
	}
	plate, err := encodePNG(render(plateIconSize))
	if err != nil {
		return nil, err
	}
	window, err := encodePNG(render(windowIconSize))
	if err != nil {
		return nil, err
	}

	artifacts := []struct {
		path  string
		bytes []byte
	}{
		{path: windowsIcon, bytes: icon},
		{path: plateIcon, bytes: plate},
		{path: windowIcon, bytes: window},
		{path: webMark, bytes: vector()},
	}

	written := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		target := filepath.Join(root, filepath.FromSlash(artifact.path))
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, fmt.Errorf("create %s: %w", filepath.Dir(target), err)
		}
		if err = os.WriteFile(target, artifact.bytes, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", target, err)
		}
		written = append(written, artifact.path)
	}
	return written, nil
}
