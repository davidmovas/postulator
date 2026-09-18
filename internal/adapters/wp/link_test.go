package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "empty is the root", path: "", want: "/"},
		{name: "root stays the root", path: "/", want: "/"},
		{name: "wraps in slashes", path: "koffein/powder", want: "/koffein/powder/"},
		{name: "keeps a trailing slash", path: "/koffein/powder/", want: "/koffein/powder/"},
		{name: "collapses duplicates", path: "//koffein///powder//", want: "/koffein/powder/"},
		{name: "no file extension exception", path: "/koffein/powder.html", want: "/koffein/powder.html/"},
		{name: "lowercases ascii", path: "/Koffein/Powder/", want: "/koffein/powder/"},
		{name: "does not decode percent escapes", path: "/koffein/gr%C3%BCner-tee", want: "/koffein/gr%c3%bcner-tee/"},
		{name: "leaves non ascii bytes alone", path: "/koffein/Grüner-Tee", want: "/koffein/grüner-tee/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := wp.NormalizePath(tc.path); got != tc.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestInternalPath(t *testing.T) {
	t.Parallel()

	const host = "example.com"

	cases := []struct {
		name     string
		href     string
		want     string
		internal bool
	}{
		{name: "relative", href: "koffein/powder", want: "/koffein/powder/", internal: true},
		{name: "absolute path", href: "/koffein/", want: "/koffein/", internal: true},
		{name: "same host over https", href: "https://example.com/koffein/", want: "/koffein/", internal: true},
		{name: "same host over http", href: "http://example.com/koffein/", want: "/koffein/", internal: true},
		{name: "host case is ignored", href: "https://EXAMPLE.COM/Koffein/", want: "/koffein/", internal: true},
		{name: "protocol relative", href: "//example.com/koffein/", want: "/koffein/", internal: true},
		{name: "host root", href: "https://example.com", want: "/", internal: true},
		{name: "query is stripped", href: "/koffein/?utm=1", want: "/koffein/", internal: true},
		{name: "fragment is stripped", href: "/koffein/#top", want: "/koffein/", internal: true},
		{name: "percent escapes are not decoded", href: "/gr%C3%BCner-tee", want: "/gr%c3%bcner-tee/", internal: true},
		{name: "www is a different host", href: "https://www.example.com/koffein/"},
		{name: "another host", href: "https://other.example/koffein/"},
		{name: "a port makes it another host", href: "https://example.com:8080/koffein/"},
		{name: "mail is not a link", href: "mailto:hello@example.com"},
		{name: "telephone is not a link", href: "tel:+123"},
		{name: "a bare fragment is not a link", href: "#top"},
		{name: "a bare query is not a link", href: "?utm=1"},
		{name: "empty", href: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, internal := wp.InternalPath(host, tc.href)
			if internal != tc.internal {
				t.Fatalf("InternalPath(%q) internal = %t, want %t", tc.href, internal, tc.internal)
			}
			if got != tc.want {
				t.Errorf("InternalPath(%q) = %q, want %q", tc.href, got, tc.want)
			}
		})
	}
}
