package content_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
)

func TestAResearchReadsItsFactsWithoutTheMarkup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want []string
	}{
		{name: "nothing found", text: "", want: []string{}},
		{name: "blank lines only", text: "\n  \n", want: []string{}},
		{
			name: "bullets, numbers and headings",
			text: "- one\n* two\n1. three\n  4) four  \n### Heading\nplain",
			want: []string{"one", "two", "three", "four", "Heading", "plain"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			research := content.Research{Text: tc.text}
			if got := research.Facts(); !slices.Equal(got, tc.want) {
				t.Errorf("Facts(%q) = %q, want %q", tc.text, got, tc.want)
			}
			if research.Empty() != (len(tc.want) == 0) {
				t.Errorf("Empty() = %t for %q", research.Empty(), tc.text)
			}
		})
	}
}
