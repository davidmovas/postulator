package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
		fail bool
	}{
		{name: "root", raw: "/", want: "/"},
		{name: "bare word", raw: "about", want: "/about/"},
		{name: "lowercased", raw: "/About/Us/", want: "/about/us/"},
		{name: "full url loses host query and fragment", raw: "https://Example.com/Shop/Bags?x=1#top", want: "/shop/bags/"},
		{name: "duplicate slashes collapse", raw: "//a///b//", want: "/a/b/"},
		{name: "query on a bare path", raw: "/a/?page=2", want: "/a/"},
		{name: "surrounding whitespace trimmed", raw: "  /a/b  ", want: "/a/b/"},
		{name: "unicode kept", raw: "/Обувь/", want: "/обувь/"},
		{name: "percent encoding kept", raw: "/a%20b/", want: "/a%20b/"},
		{name: "escape case is lowered", raw: "/a%2Fb/", want: "/a%2fb/"},
		{name: "no file extension exception", raw: "/Shop/index.HTML", want: "/shop/index.html/"},
		{name: "absolute url with no path is the root", raw: "https://example.com", want: "/"},
		{name: "koffein example", raw: "https://example.com/Koffein/Powder", want: "/koffein/powder/"},
		{name: "empty", raw: "   ", fail: true},
		{name: "inner whitespace", raw: "/a b/", fail: true},
		{name: "dot segment", raw: "/a/./b/", fail: true},
		{name: "parent segment", raw: "/a/../b/", fail: true},
		{name: "control character", raw: "/a\tb/", fail: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := pagemap.NormalizePath(tc.raw)
			if tc.fail {
				if !errors.IsCode(err, errors.Invalid) {
					t.Fatalf("code = %q, want INVALID (got %q)", errors.CodeOf(err), got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizePath: %v", err)
			}
			if got != tc.want {
				t.Errorf("NormalizePath = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParentPathAndSlug(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path   string
		parent string
		slug   string
	}{
		{path: "/", parent: "", slug: ""},
		{path: "/a/", parent: "/", slug: "a"},
		{path: "/a/b/", parent: "/a/", slug: "b"},
		{path: "/a/b/c/", parent: "/a/b/", slug: "c"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			if got := pagemap.ParentPath(tc.path); got != tc.parent {
				t.Errorf("ParentPath = %q, want %q", got, tc.parent)
			}
			if got := pagemap.Slug(tc.path); got != tc.slug {
				t.Errorf("Slug = %q, want %q", got, tc.slug)
			}
		})
	}
}

func TestParentPathBindingExamples(t *testing.T) {
	t.Parallel()

	if got := pagemap.ParentPath("/koffein/powder/"); got != "/koffein/" {
		t.Errorf("ParentPath = %q, want /koffein/", got)
	}
	if got := pagemap.ParentPath("/"); got != "" {
		t.Errorf("ParentPath of the root = %q, want empty", got)
	}
}

func TestInternalPath(t *testing.T) {
	t.Parallel()

	const host = "shop.example.com"
	cases := []struct {
		name     string
		href     string
		path     string
		internal bool
	}{
		{name: "relative path", href: "/Shop/Bags?x=1#top", path: "/shop/bags/", internal: true},
		{name: "relative without leading slash", href: "shoes/", path: "/shoes/", internal: true},
		{name: "same host any case", href: "https://Shop.Example.com/Sale/", path: "/sale/", internal: true},
		{name: "scheme ignored", href: "http://shop.example.com/sale/", path: "/sale/", internal: true},
		{name: "absolute url with no path", href: "https://shop.example.com", path: "/", internal: true},
		{name: "network path reference", href: "//shop.example.com/x/", path: "/x/", internal: true},
		{name: "same document", href: "#top", path: "", internal: true},
		{name: "query only", href: "?page=2", path: "", internal: true},
		{name: "www is another host", href: "https://www.shop.example.com/x/", path: "", internal: false},
		{name: "port is part of the host", href: "https://shop.example.com:8443/x/", path: "", internal: false},
		{name: "foreign host", href: "https://other.example.com/x/", path: "", internal: false},
		{name: "mailto", href: "mailto:hello@shop.example.com", path: "", internal: false},
		{name: "javascript", href: "javascript:void(0)", path: "", internal: false},
		{name: "control character", href: "/a\tb/", path: "", internal: false},
		{name: "dot segments", href: "/a/../b/", path: "", internal: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, internal := pagemap.InternalPath(tc.href, host)
			if path != tc.path || internal != tc.internal {
				t.Errorf("InternalPath = %q, %v; want %q, %v", path, internal, tc.path, tc.internal)
			}
		})
	}
}

func TestNewSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		baseURL string
		want    pagemap.Site
	}{
		{name: "root install", baseURL: "https://shop.example.com", want: pagemap.Site{Scheme: "https", Host: "shop.example.com", Base: "/"}},
		{name: "trailing slash", baseURL: "https://shop.example.com/", want: pagemap.Site{Scheme: "https", Host: "shop.example.com", Base: "/"}},
		{name: "subdirectory install", baseURL: "https://h.example.com/blog", want: pagemap.Site{Scheme: "https", Host: "h.example.com", Base: "/blog/"}},
		{name: "subdirectory with a slash", baseURL: "https://h.example.com/Blog/", want: pagemap.Site{Scheme: "https", Host: "h.example.com", Base: "/blog/"}},
		{name: "plain http on a port", baseURL: "http://localhost:8089", want: pagemap.Site{Scheme: "http", Host: "localhost:8089", Base: "/"}},
		{name: "nothing at all", baseURL: "", want: pagemap.Site{Base: "/"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := pagemap.NewSite(tc.baseURL); got != tc.want {
				t.Errorf("NewSite(%q) = %+v, want %+v", tc.baseURL, got, tc.want)
			}
		})
	}
}

func TestSiteURLDoesNotDoubleTheBasePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{name: "root install", baseURL: "https://shop.example.com", path: "/shop/", want: "https://shop.example.com/shop/"},
		{name: "root home", baseURL: "https://shop.example.com", path: "/", want: "https://shop.example.com/"},
		{name: "subdirectory path already carries the base", baseURL: "https://h.example.com/blog", path: "/blog/shop/", want: "https://h.example.com/blog/shop/"},
		{name: "subdirectory home", baseURL: "https://h.example.com/blog", path: "/blog/", want: "https://h.example.com/blog/"},
		{name: "subdirectory path without the base", baseURL: "https://h.example.com/blog", path: "/shop/", want: "https://h.example.com/blog/shop/"},
		{name: "a sibling prefix is not the base", baseURL: "https://h.example.com/blog", path: "/blogging/", want: "https://h.example.com/blog/blogging/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := pagemap.NewSite(tc.baseURL).URL(tc.path); got != tc.want {
				t.Errorf("URL(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestSiteResolve(t *testing.T) {
	t.Parallel()

	root := pagemap.NewSite("https://shop.example.com")
	nested := pagemap.NewSite("https://h.example.com/blog")

	cases := []struct {
		name string
		site pagemap.Site
		href string
		path string
		kind pagemap.LinkKind
	}{
		{name: "a bare path is completed", site: root, href: "/shop", path: "/shop/", kind: pagemap.LinkPath},
		{name: "casing is folded", site: root, href: "/Shop/", path: "/shop/", kind: pagemap.LinkPath},
		{name: "a query is dropped", site: root, href: "/shop/?utm=x", path: "/shop/", kind: pagemap.LinkPath},
		{name: "a fragment is dropped", site: root, href: "/shop/#top", path: "/shop/", kind: pagemap.LinkPath},
		{name: "the own host over https", site: root, href: "https://shop.example.com/shop/", path: "/shop/", kind: pagemap.LinkPath},
		{name: "the own host over http", site: root, href: "http://shop.example.com/shop/", path: "/shop/", kind: pagemap.LinkPath},
		{name: "www is another host", site: root, href: "https://www.shop.example.com/shop/", kind: pagemap.LinkExternal},
		{name: "an escaped slash stays escaped", site: root, href: "/a%2Fb/", path: "/a%2fb/", kind: pagemap.LinkPath},
		{name: "a percent escaped accent", site: root, href: "/caf%C3%A9/", path: "/caf%c3%a9/", kind: pagemap.LinkPath},
		{name: "a raw accent escapes the same way", site: root, href: "/café/", path: "/caf%c3%a9/", kind: pagemap.LinkPath},
		{name: "a relative href joins the base", site: root, href: "shop/", path: "/shop/", kind: pagemap.LinkPath},
		{name: "a fragment names this document", site: root, href: "#faq", kind: pagemap.LinkSameDocument},
		{name: "a query names this document", site: root, href: "?utm=x", kind: pagemap.LinkSameDocument},
		{name: "an empty href names this document", site: root, href: "", kind: pagemap.LinkSameDocument},
		{name: "a mail link is external", site: root, href: "mailto:hello@shop.example.com", kind: pagemap.LinkExternal},
		{name: "an unrelated host is external", site: root, href: "https://other.example.org/shop/", kind: pagemap.LinkExternal},
		{name: "a dot segment resolves to nothing", site: root, href: "/a/../b/", kind: pagemap.LinkUnresolved},
		{name: "the subdirectory path as written", site: nested, href: "/blog/shop/", path: "/blog/shop/", kind: pagemap.LinkPath},
		{name: "the subdirectory path absolute", site: nested, href: "https://h.example.com/blog/shop/", path: "/blog/shop/", kind: pagemap.LinkPath},
		{name: "a relative href joins the subdirectory base", site: nested, href: "shop/", path: "/blog/shop/", kind: pagemap.LinkPath},
		{name: "the server root of a subdirectory install", site: nested, href: "https://h.example.com", path: "/", kind: pagemap.LinkPath},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path, kind := tc.site.Resolve(tc.href)
			if path != tc.path || kind != tc.kind {
				t.Errorf("Resolve(%q) = %q, %s; want %q, %s", tc.href, path, kind, tc.path, tc.kind)
			}
		})
	}
}
