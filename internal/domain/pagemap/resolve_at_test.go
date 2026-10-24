package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestResolveAtAnswersARelativeHrefAgainstTheDocument(t *testing.T) {
	t.Parallel()

	cases := []struct {
		site, document, href string
		kind                 pagemap.LinkKind
		path                 string
	}{
		{"http://example.test:8087", "/x/y/", "a/", pagemap.LinkPath, "/x/y/a/"},
		{"http://example.test:8087", "/x/y", "a/", pagemap.LinkPath, "/x/a/"},
		{"http://example.test:8087", "/x/y/", "a/b", pagemap.LinkPath, "/x/y/a/b/"},
		{"http://example.test:8087", "/x/y/", "../z/", pagemap.LinkUnresolved, ""},
		{"http://example.test:8087", "/x/y/", "./a/", pagemap.LinkUnresolved, ""},
		{"http://example.test:8087", "/x/y/", "?page=2", pagemap.LinkSameDocument, ""},
		{"http://example.test:8087", "/x/y/", "#top", pagemap.LinkSameDocument, ""},
		{"http://example.test:8087", "/x/y/", "/abs/", pagemap.LinkPath, "/abs/"},
		{"http://example.test:8087", "/x/y/", "http://example.test:8087/abs/", pagemap.LinkPath, "/abs/"},
		{"http://example.test:8087", "/x/y/", "http://example.test:8088/abs/", pagemap.LinkExternal, ""},
		{"http://example.test:8087", "", "a/", pagemap.LinkPath, "/a/"},
		{"https://example.test/blog", "/blog/x/", "a/", pagemap.LinkPath, "/blog/x/a/"},
		{"https://example.test/blog", "", "a/", pagemap.LinkPath, "/blog/a/"},
	}
	for _, tc := range cases {
		path, kind := pagemap.NewSite(tc.site).ResolveAt(tc.document, tc.href)
		if kind != tc.kind || path != tc.path {
			t.Errorf("ResolveAt(%q, %q) on %s = %q %s, want %q %s", tc.document, tc.href, tc.site, path, kind, tc.path, tc.kind)
		}
	}
}

func TestResolveAtAgreesWithResolveAtTheBase(t *testing.T) {
	t.Parallel()

	for _, tc := range hrefTable(t) {
		if tc.Document != "" {
			continue
		}
		site := pagemap.NewSite(tc.Site)
		path, kind := site.Resolve(tc.Href)
		atPath, atKind := site.ResolveAt(site.Base, tc.Href)
		if path != atPath || kind != atKind {
			t.Errorf("Resolve(%q) = %q %s but ResolveAt(base) = %q %s", tc.Href, path, kind, atPath, atKind)
		}
	}
}
