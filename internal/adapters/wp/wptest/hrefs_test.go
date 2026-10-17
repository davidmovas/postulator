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
			Href     string `json:"href"`
			Kind     string `json:"kind"`
			Path     string `json:"path"`
			Relative bool   `json:"relative"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatalf("decode the href table: %v", err)
	}

	for _, tc := range table.Cases {
		t.Run(tc.Site+" "+tc.Href, func(t *testing.T) {
			t.Parallel()

			path, listed := internalTarget(hostOf(tc.Site), tc.Href)
			if listed != (tc.Kind == "path") {
				t.Fatalf("internalTarget(%q) listed = %t, want the kind %s", tc.Href, listed, tc.Kind)
			}
			if listed && !tc.Relative && path != tc.Path {
				t.Errorf("internalTarget(%q) = %q, want %q", tc.Href, path, tc.Path)
			}
		})
	}
}
