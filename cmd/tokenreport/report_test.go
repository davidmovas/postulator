package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTheReportNamesEachModelsEncodingAndTheFallback(t *testing.T) {
	t.Parallel()

	var table bytes.Buffer
	if err := run(t.Context(), &table, false); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{"gpt-5.6-terra", "o200k_base", "gpt-image-2"} {
		if !strings.Contains(table.String(), want) {
			t.Errorf("the table lacks %q:\n%s", want, table.String())
		}
	}

	var encoded bytes.Buffer
	if err := run(t.Context(), &encoded, true); err != nil {
		t.Fatalf("run -json: %v", err)
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
