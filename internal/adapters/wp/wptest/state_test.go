package wptest

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		title string
		want  string
	}{
		{name: "plain", title: "Powder", want: "powder"},
		{name: "spaces collapse", title: "Koffein   Powder", want: "koffein-powder"},
		{name: "punctuation collapses", title: "Koffein: Powder!", want: "koffein-powder"},
		{name: "trims", title: "  Powder  ", want: "powder"},
		{name: "non ascii dropped", title: "Grüner Tee", want: "gr-ner-tee"},
		{name: "empty falls back", title: "!!!", want: "item"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := slugify(tc.title); got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}

func TestContentHashMatchesTheFrozenDigests(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{name: "paragraph", raw: "<p>Koffein ist ein Alkaloid.</p>", want: "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"},
		{name: "powder", raw: "<p>Powder</p>", want: "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := contentHash(tc.raw); got != tc.want {
				t.Errorf("contentHash(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestItemPath(t *testing.T) {
	t.Parallel()

	server := &Server{items: map[int64]*Item{
		1: {ID: 1, Slug: "koffein"},
		2: {ID: 2, Slug: "powder", Parent: 1},
		3: {ID: 3, Slug: "loop", Parent: 3},
	}}

	cases := []struct {
		name string
		id   int64
		want string
	}{
		{name: "top level", id: 1, want: "/koffein/"},
		{name: "nested under its parent", id: 2, want: "/koffein/powder/"},
		{name: "a cycle stops at the depth limit", id: 3, want: "/" + strings.Repeat("loop/", maxPathDepth+1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := server.itemPath(server.items[tc.id]); got != tc.want {
				t.Errorf("itemPath = %q, want %q", got, tc.want)
			}
		})
	}
}
