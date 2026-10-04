package graph_test

import (
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
