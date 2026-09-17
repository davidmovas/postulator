package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	path := flag.String("profile", "coverage.out", "path to the coverage profile produced by go test -coverprofile")
	root := flag.String("root", ".", "module root, scanned to tell an empty core tree from an uncovered one")
	core := flag.Float64("core", 80, "minimum percentage of covered statements in internal/domain and internal/application")
	total := flag.Float64("total", 70, "minimum percentage of covered statements across the module")
	flag.Parse()

	if err := run(*path, *root, gates{core: *core, total: *total}); err != nil {
		fmt.Fprintln(os.Stderr, "covergate:", err)
		os.Exit(1)
	}
}

func run(path, root string, thresholds gates) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	parsed, err := parse(file)
	if err != nil {
		return err
	}

	coreSource, err := coreSourcePresent(root)
	if err != nil {
		return err
	}

	outcome := evaluate(parsed, thresholds, coreSource)
	fmt.Println(outcome)
	if !outcome.pass() {
		return fmt.Errorf("coverage gate not met")
	}
	return nil
}
