package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheReportNamesEachModelsEncodingAndTheFallback(t *testing.T) {
	t.Parallel()

	var table bytes.Buffer
	if err := execute(t.Context(), &table, false, ""); err != nil {
		t.Fatalf("execute: %v", err)
	}
	for _, want := range []string{"gpt-5.6-terra", "o200k_base", "gpt-image-2"} {
		if !strings.Contains(table.String(), want) {
			t.Errorf("the table lacks %q:\n%s", want, table.String())
		}
	}

	var encoded bytes.Buffer
	prompts := t.TempDir()
	if err := execute(t.Context(), &encoded, true, prompts); err != nil {
		t.Fatalf("execute -json -prompts: %v", err)
	}
	if _, err := os.Stat(filepath.Join(prompts, agentFile)); err != nil {
		t.Errorf("-prompts wrote no agent round: %v", err)
	}
	var decoded struct {
		Models []struct {
			Model    string `json:"model"`
			Encoding string `json:"encoding"`
			Fallback bool   `json:"fallback"`
		} `json:"models"`
	}
	if err := json.Unmarshal(encoded.Bytes(), &decoded); err != nil {
		t.Fatalf("the JSON does not decode: %v\n%s", err, encoded.String())
	}
	fallbacks := 0
	for _, row := range decoded.Models {
		if row.Encoding != "o200k_base" {
			t.Errorf("%s is counted as %s, want o200k_base", row.Model, row.Encoding)
		}
		if row.Fallback {
			fallbacks++
		}
	}
	if len(decoded.Models) != 4 || fallbacks != 1 {
		t.Errorf("the JSON names %d models with %d fallbacks, want the catalog's four and gpt-image-2's fallback", len(decoded.Models), fallbacks)
	}
}
