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

func TestASpliceOverSeveralChangedBlocksKeepsEverythingBetweenThem(t *testing.T) {
	t.Parallel()

	const (
		first  = "<p>Tea &amp; coffee&nbsp;now.</p>"
		ours   = "<!-- wp:html -->\n<p>Espresso is pulled under pressure.</p>\n<!-- /wp:html -->"
		second = "<p>Milk &amp; sugar&nbsp;later.</p>"
	)
	before := first + "\n\n" + ours + "\n\n" + second
	after := rerendered(t, rerendered(t, before, "coffee", `<a href="/coffee/">coffee</a>`), "sugar", `<a href="/sugar/">sugar</a>`)

	raw, gotOurs := block.Splice(before, after, []string{block.Hash(ours)})
	if !strings.Contains(raw, "\n\n"+ours+"\n\n") {
		t.Errorf("Splice = %q, want our untouched block between the two linked ones byte for byte", raw)
	}
	if !slices.Equal(gotOurs, []string{block.Hash(ours)}) {
		t.Errorf("ours = %v, want our untouched block still ours and the client's linked blocks his", gotOurs)
	}
}

func TestOursAreOnlyTheBlocksOfOurPart(t *testing.T) {
	t.Parallel()

	const (
		theirs = "<p>Tea & coffee, roasted daily.</p>"
		part   = "<p>Espresso is pulled under pressure.</p>"
	)
	cases := []struct {
		name   string
		stored string
		client string
		want   []string
	}{
		{name: "a whole body of ours", stored: part, want: []string{block.Hash(part)}},
		{name: "ours beside the client's", stored: theirs + "\n\n" + part, client: theirs, want: []string{block.Hash(part)}},
		{
			name: "a client block the save changed stays his", stored: "<p>Tea &amp; coffee, roasted daily.</p>\n\n" + part,
			client: theirs, want: []string{block.Hash(part)},
		},
		{name: "an unclosed client tag that swallowed ours is no block of ours", stored: "<div>Client text\n\n" + part, client: "<div>Client text"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := block.Ours(tc.stored, tc.client, part); !slices.Equal(got, tc.want) && (len(got) != 0 || len(tc.want) != 0) {
				t.Errorf("Ours = %v, want %v", got, tc.want)
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
