package block_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
)

func rerendered(t *testing.T, raw, from, to string) string {
	t.Helper()

	doc, err := content.Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	rendered, err := doc.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(rendered, from) {
		t.Fatalf("the render %q holds no %q", rendered, from)
	}
	return strings.Replace(rendered, from, to, 1)
}

func TestASpliceKeepsTheBytesOfEveryBlockItDidNotChange(t *testing.T) {
	t.Parallel()

	const (
		theirs = "<p>Tea &amp; coffee&nbsp;now, <em class='x'>freshly</em> roasted.</p>"
		ours   = "<p>Espresso is pulled under pressure.</p>"
	)
	before := theirs + "\n\n" + ours
	cases := []struct {
		name     string
		from, to string
		written  []string
		want     string
		wantOurs []string
	}{
		{
			name: "a link placed in our block keeps it ours", from: "under pressure", to: `under <a href="/pressure/">pressure</a>`,
			written:  []string{block.Hash(ours)},
			want:     theirs + "\n\n" + `<p>Espresso is pulled under <a href="/pressure/">pressure</a>.</p>`,
			wantOurs: []string{block.Hash(`<p>Espresso is pulled under <a href="/pressure/">pressure</a>.</p>`)},
		},
		{
			name: "a link placed in the client's block leaves it his", from: "roasted.", to: `<a href="/roast/">roasted</a>.`,
			written:  []string{block.Hash(ours)},
			wantOurs: []string{block.Hash(ours)},
		},
		{
			name: "a sentence added as a block of its own is ours", from: "pressure.</p>", to: "pressure.</p>\n\n<p>See the guide.</p>",
			written:  []string{block.Hash(ours)},
			want:     before + "\n\n<p>See the guide.</p>",
			wantOurs: []string{block.Hash(ours), block.Hash("<p>See the guide.</p>")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw, gotOurs := block.Splice(before, rerendered(t, before, tc.from, tc.to), tc.written)
			if tc.want != "" && raw != tc.want {
				t.Errorf("Splice =\n%q\nwant\n%q", raw, tc.want)
			}
			if !strings.Contains(raw, "Tea &amp; coffee&nbsp;now, <em class='x'>") && tc.from != "roasted." {
				t.Errorf("the client's untouched block lost its bytes: %q", raw)
			}
			slices.Sort(gotOurs)
			slices.Sort(tc.wantOurs)
			if !slices.Equal(gotOurs, tc.wantOurs) {
				t.Errorf("ours = %v, want %v", gotOurs, tc.wantOurs)
			}
		})
	}
}

func TestASpliceOfAnUnchangedBodyIsTheBodyItself(t *testing.T) {
	t.Parallel()

	const before = "<!-- wp:paragraph -->\n<p>One &amp; two.</p>\n<!-- /wp:paragraph -->\n\n[products limit=\"4\"]"
	raw, ours := block.Splice(before, rerendered(t, before, "One", "One"), nil)
	if raw != before || len(ours) != 0 {
		t.Errorf("Splice = %q, %v; want the body byte for byte and nothing ours", raw, ours)
	}
}
