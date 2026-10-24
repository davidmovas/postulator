package pagemap_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type hrefCase struct {
	Site     string `json:"site"`
	Document string `json:"document"`
	Href     string `json:"href"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
}

func hrefTable(t *testing.T) []hrefCase {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "wp-plugin", "testdata", "hrefs.json"))
	if err != nil {
		t.Fatalf("read the href table: %v", err)
	}
	var table struct {
		Cases []hrefCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatalf("decode the href table: %v", err)
	}
	if len(table.Cases) == 0 {
		t.Fatal("the href table holds no case")
	}
	return table.Cases
}

func TestResolveAnswersTheSharedHrefTable(t *testing.T) {
	t.Parallel()

	for _, tc := range hrefTable(t) {
		t.Run(tc.Site+" "+tc.Href, func(t *testing.T) {
			t.Parallel()

			site := pagemap.NewSite(tc.Site)
			path, kind := site.Resolve(tc.Href)
			if tc.Document != "" {
				path, kind = site.ResolveAt(tc.Document, tc.Href)
			}
			if string(kind) != tc.Kind {
				t.Fatalf("Resolve(%q) kind = %s, want %s", tc.Href, kind, tc.Kind)
			}
			if tc.Kind == string(pagemap.LinkPath) && path != tc.Path {
				t.Errorf("Resolve(%q) path = %q, want %q", tc.Href, path, tc.Path)
			}
		})
	}
}
