package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const generatedFile = "frontend/src/generated/vocab.ts"

func main() {
	root := flag.String("root", "", "module root the Go sources are read from")
	out := flag.String("out", "", "path of the generated TypeScript module")
	flag.Parse()

	module, err := moduleRoot(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vocabgen:", err)
		os.Exit(1)
	}

	target := *out
	if target == "" {
		target = filepath.Join(module, filepath.FromSlash(generatedFile))
	}

	if err = write(module, target); err != nil {
		fmt.Fprintln(os.Stderr, "vocabgen:", err)
		os.Exit(1)
	}
}

func write(module, path string) error {
	blocks, err := collect(module)
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	if renderErr := render(&buffer, blocks); renderErr != nil {
		return renderErr
	}
	if dirErr := os.MkdirAll(filepath.Dir(path), 0o750); dirErr != nil {
		return dirErr
	}
	return os.WriteFile(path, buffer.Bytes(), 0o600)
}
