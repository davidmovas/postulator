package render_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
)

const (
	replica = "replica"
	vanilla = "vanilla"
)

var standURLs = map[string]string{
	replica: "http://localhost:8087",
	vanilla: "http://localhost:8088",
}

var standPaths = map[string]map[string]pagemap.WPType{
	replica: {
		"/":                              pagemap.WPPage,
		"/peptides/":                     pagemap.WPPage,
		"/bpc-157/":                      pagemap.WPPage,
		"/tb-500/":                       pagemap.WPPage,
		"/healing-peptides/":             pagemap.WPPage,
		"/sample-page/":                  pagemap.WPPage,
		"/hello-world/":                  pagemap.WPPost,
		"/shop/":                         pagemap.WPPage,
		"/cart/":                         pagemap.WPPage,
		"/checkout/":                     pagemap.WPPage,
		"/my-account/":                   pagemap.WPPage,
		"/product/bpc-157-peptide/":      pagemap.WPProduct,
		"/product/tb-500-peptide/":       pagemap.WPProduct,
		"/product/bpc-157-nasal-spray/":  pagemap.WPProduct,
		"/product/bacteriostatic-water/": pagemap.WPProduct,
		"/product-category/peptides/":    pagemap.WPProductCategory,
		"/product-category/peptides/healing-peptides/": pagemap.WPProductCategory,
	},
	vanilla: {
		"/":                          pagemap.WPPage,
		"/guide/":                    pagemap.WPPage,
		"/guide/setup/":              pagemap.WPPage,
		"/sample-page/":              pagemap.WPPage,
		"/shop/":                     pagemap.WPPage,
		"/cart/":                     pagemap.WPPage,
		"/checkout/":                 pagemap.WPPage,
		"/my-account/":               pagemap.WPPage,
		"/product/powder/":           pagemap.WPProduct,
		"/product/coffee-beans/":     pagemap.WPProduct,
		"/product-category/koffein/": pagemap.WPProductCategory,
	},
}

func defaultBuilders() []render.Builder {
	rules := site.DefaultSettings().Builders
	out := make([]render.Builder, 0, len(rules))
	for _, rule := range rules {
		out = append(out, render.Builder{
			Name: rule.Name, DataAttributes: rule.DataAttributes,
			ClassPrefixes: rule.ClassPrefixes, ShortcodePrefixes: rule.ShortcodePrefixes,
		})
	}
	return out
}

func technicalNames() []string {
	defaults := site.DefaultSettings()
	return append(slices.Clone(defaults.TechnicalBlocks), defaults.TechnicalShortcodes...)
}

func lookupOf(stand string) func(string) (pagemap.WPType, bool) {
	known := standPaths[stand]
	return func(path string) (pagemap.WPType, bool) {
		wpType, ok := known[path]
		return wpType, ok
	}
}

func readGolden(t *testing.T, parts ...string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, parts...)...))
	if err != nil {
		t.Fatalf("read the golden %v: %v", parts, err)
	}
	return string(body)
}

func readOptional(t *testing.T, parts ...string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, parts...)...))
	if errors.Is(err, fs.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatalf("read the golden %v: %v", parts, err)
	}
	return string(body)
}

func standInput(t *testing.T, stand, slug, document string) render.Input {
	t.Helper()
	return render.Input{
		Public:   readGolden(t, stand, slug+".public.html"),
		Filtered: readOptional(t, stand, slug+".filtered.html"),
		Body:     block.Parse(readGolden(t, stand, slug+".raw.html")),
		Site:     pagemap.NewSite(standURLs[stand]),
		Document: document,
		Builders: defaultBuilders(),
		Lookup:   lookupOf(stand),
	}
}

func extractStand(t *testing.T, stand, slug, document string) render.Extraction {
	t.Helper()
	extraction, err := render.Extract(standInput(t, stand, slug, document))
	if err != nil {
		t.Fatalf("extract %s/%s: %v", stand, slug, err)
	}
	return extraction
}

func synthetic(t *testing.T, in render.Input) render.Extraction {
	t.Helper()
	if in.Site == (pagemap.Site{}) {
		in.Site = pagemap.NewSite("https://shop.test")
	}
	extraction, err := render.Extract(in)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return extraction
}
