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
