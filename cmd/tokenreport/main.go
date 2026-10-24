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
	prompts := flag.String("prompts", "", "also write the steps' and the agent's prompts as plain text into this directory, such as out/prompts")
	flag.Parse()

	if err := execute(context.Background(), os.Stdout, *asJSON, *prompts); err != nil {
		fmt.Fprintln(os.Stderr, "tokenreport:", err)
		os.Exit(1)
	}
}

func execute(ctx context.Context, out io.Writer, asJSON bool, prompts string) error {
	built, err := build(ctx)
	if err != nil {
		return err
	}
	if prompts != "" {
		commit, err := commitOf(".")
		if err != nil {
			return err
		}
		if err := writePromptFiles(ctx, prompts, built.Pages, commit); err != nil {
			return err
		}
	}
	if asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(built)
	}
	return built.write(out)
}
