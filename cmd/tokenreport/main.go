package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	asJSON := flag.Bool("json", false, "print the report as JSON instead of tables")
	flag.Parse()

	if err := execute(context.Background(), os.Stdout, *asJSON); err != nil {
		fmt.Fprintln(os.Stderr, "tokenreport:", err)
		os.Exit(1)
	}
}

func execute(ctx context.Context, out io.Writer, asJSON bool) error {
	built, err := build(ctx)
	if err != nil {
		return err
	}
	if asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(built)
	}
	return built.write(out)
}
