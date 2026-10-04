package graph_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

func TestAKeyFoldsOnlyWhatTheStoreFolds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want string
	}{
		{name: "ascii letters are lowered", text: "BPC-157 Liquid", want: "bpc-157 liquid"},
		{name: "the ends are trimmed", text: "  Liquid\t", want: "liquid"},
		{name: "inner spaces are kept as written", text: "Liquid  Form", want: "liquid  form"},
		{name: "an accented capital is kept", text: "CAFÉ", want: "cafÉ"},
		{name: "a sharp s is kept", text: "Straße", want: "straße"},
		{name: "the kelvin sign is not a k", text: "Kelvin", want: "Kelvin"},
		{name: "an entity stays escaped", text: "Tools &amp; Kits", want: "tools &amp; kits"},
		{name: "an empty text", text: "   ", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := graph.Key(tc.text); got != tc.want {
				t.Fatalf("Key(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestDistinctKeepsEachTextOnceAsFirstWritten(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		lists [][]string
		want  []string
	}{
		{name: "nothing", want: []string{}},
		{name: "blanks are dropped and the rest trimmed", lists: [][]string{{" ", " running shoes ", ""}}, want: []string{"running shoes"}},
		{name: "a repeat in another case keeps the first spelling", lists: [][]string{{"Trail Shoes", "trail shoes", "TRAIL SHOES "}}, want: []string{"Trail Shoes"}},
		{name: "texts the store tells apart are both kept", lists: [][]string{{"Café", "CAFÉ"}}, want: []string{"Café", "CAFÉ"}},
		{name: "a second list adds only what the first lacks", lists: [][]string{{"/shop/", "/blog/"}, {"/Shop/", "/news/"}}, want: []string{"/shop/", "/blog/", "/news/"}},
		{name: "an empty first list", lists: [][]string{nil, {"Liquid", "liquid"}}, want: []string{"Liquid"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := graph.Distinct(tc.lists...)
			if got == nil || !slices.Equal(got, tc.want) {
				t.Fatalf("Distinct(%q) = %#v, want %q", tc.lists, got, tc.want)
			}
		})
	}
}
