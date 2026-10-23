package content_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
)

func TestABlockKeepsItsHeadingsAndLosesItsMarkupAndLinks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		given  string
		keepH1 bool
		want   string
	}{
		{name: "a section heading stays", given: `<h2>How to store it</h2>`, want: `<h2>How to store it</h2>`},
		{name: "an h1 the theme prints is demoted", given: `<h1>Peptides</h1>`, want: `<h2>Peptides</h2>`},
		{name: "an h1 the page owes stays", given: `<h1>Peptides</h1>`, keepH1: true, want: `<h1>Peptides</h1>`},
		{
			name: "classes, ids, styles and data go", given: `<p class="lead" id="x" style="color:red" data-k="v">Text</p>`,
			want: `<p>Text</p>`,
		},
		{name: "a link becomes its text", given: `<p>See <a href="/elsewhere/">this page</a>.</p>`, want: `<p>See this page.</p>`},
		{name: "a script goes", given: `<p>Text</p><script>alert(1)</script>`, want: `<p>Text</p>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, _ := content.SanitizeBlock(tc.given, tc.keepH1)
			if strings.TrimSpace(got) != tc.want {
				t.Errorf("SanitizeBlock(%q) = %q, want %q", tc.given, got, tc.want)
			}
		})
	}
}
