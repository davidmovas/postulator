package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/application/events"
)

const generatedFile = "frontend/src/generated/events.ts"

func main() {
	out := flag.String("out", filepath.FromSlash(generatedFile), "path of the generated TypeScript module")
	flag.Parse()

	if err := write(*out); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func write(path string) error {
	var buffer bytes.Buffer
	if err := render(&buffer, events.NewRegistry().Entries()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, buffer.Bytes(), 0o600)
}
