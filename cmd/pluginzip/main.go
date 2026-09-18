package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/wp/plugin"
)

func main() {
	source := flag.String("source", "wp-plugin/postulator-companion", "plugin directory to package")
	out := flag.String("out", "bin/"+plugin.Filename, "zip archive to write")
	flag.Parse()

	if err := run(*source, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(*out)
}

func run(source, out string) error {
	archive, err := plugin.Pack(os.DirFS(source), filepath.Base(filepath.Clean(source)))
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(out), err)
	}
	if err = os.WriteFile(out, archive, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}
