package wptest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTheFakeClassifiesTheSharedHrefTable(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "wp-plugin", "testdata", "hrefs.json"))
	if err != nil {
		t.Fatalf("read the href table: %v", err)
	}
	var table struct {
		Cases []struct {
			Site     string `json:"site"`
			Document string `json:"document"`
			Href     string `json:"href"`
			Kind     string `json:"kind"`
			Path     string `json:"path"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatalf("decode the href table: %v", err)
	}

	for _, tc := range table.Cases {
		document := tc.Document
		if document == "" {
			document = "/"
		}
		t.Run(tc.Site+" "+document+" "+tc.Href, func(t *testing.T) {
			t.Parallel()

			path, listed := internalTarget(hostOf(tc.Site), document, tc.Href)
			if listed != (tc.Kind == "path") {
				t.Fatalf("internalTarget(%q, %q) listed = %t, want the kind %s", document, tc.Href, listed, tc.Kind)
			}
			if listed && path != tc.Path {
				t.Errorf("internalTarget(%q, %q) = %q, want %q", document, tc.Href, path, tc.Path)
			}
		})
	}
}
