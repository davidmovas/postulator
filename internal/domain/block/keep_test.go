package block_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

func hashesOf(raw string) []string {
	body := block.Parse(raw)
	out := make([]string, 0, len(body.Blocks))
	for i := range body.Blocks {
		out = append(out, body.Blocks[i].Hash)
	}
	return out
}

func TestKeepPutsOurPartAfterTheClientsText(t *testing.T) {
	t.Parallel()

	const (
		client    = "<p>Step-by-step guides for handling research peptides.</p>\n\n<p>Ask before you mix.</p>"
		shortcode = `[products category="healing-peptides" limit="4"]`
		ours      = "<h2>Storage</h2><p>Keep them cold.</p>"
		earlier   = "<h2>Old</h2><p>What we wrote before.</p>"
	)
	cases := []struct {
		name    string
		current string
		written []string
		want    string
		kept    bool
	}{
		{name: "an empty body is ours alone", current: "", want: ours},
		{name: "the client's text first, ours after it", current: client, want: client + "\n\n" + ours, kept: true},
		{name: "ours before a lone shortcode, so the grid stays last", current: shortcode, want: ours + "\n\n" + shortcode, kept: true},
		{
			name: "ours after the text and before a trailing shortcode", current: client + "\n\n" + shortcode,
			want: client + "\n\n" + ours + "\n\n" + shortcode, kept: true,
		},
		{
			name: "what we wrote before is replaced, the client's text kept", current: client + "\n\n" + earlier,
			written: hashesOf(earlier), want: client + "\n\n" + ours, kept: true,
		},
		{name: "a body that is all ours is ours again", current: earlier, written: hashesOf(earlier), want: ours},
		{
			name: "the client's bytes stay as they were, line endings included", current: "<p>One\r\nline.</p>\r\n\r\n<p>Two.</p>",
			want: "<p>One\r\nline.</p>\r\n\r\n<p>Two.</p>\n\n" + ours, kept: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, kept := block.Keep(tc.current, tc.written, ours)
			if got != tc.want || kept != tc.kept {
				t.Errorf("Keep = %q, %t\nwant %q, %t", got, kept, tc.want, tc.kept)
			}
			if tc.kept && !strings.Contains(got, ours) {
				t.Errorf("our part is lost: %q", got)
			}
		})
	}
}

func TestTheClientsBlocksAreWhatWeDidNotWrite(t *testing.T) {
	t.Parallel()

	const ours = "<p>Ours.</p>"
	cases := []struct {
		name    string
		current string
		written []string
		want    bool
	}{
		{name: "nothing", current: "", want: false},
		{name: "blank lines only", current: "\n\n  \n", want: false},
		{name: "the client's paragraph", current: "<p>Theirs.</p>", want: true},
		{name: "only ours", current: ours, written: hashesOf(ours), want: false},
		{name: "a lone shortcode is the client's though it carries no text", current: "[products limit=\"4\"]", want: true},
		{name: "a gallery is the client's", current: "<!-- wp:gallery {\"ids\":[4,5]} -->\n<figure></figure>\n<!-- /wp:gallery -->", want: true},
		{name: "theirs beside ours", current: "<p>Theirs.</p>\n\n" + ours, written: hashesOf(ours), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := block.HasClient(tc.current, tc.written); got != tc.want {
				t.Errorf("HasClient(%q) = %t, want %t", tc.current, got, tc.want)
			}
		})
	}
}

func TestABlockEditorBodyKeepsItsFreeformClientContentAcrossRuns(t *testing.T) {
	t.Parallel()

	const (
		paragraph = "<!-- wp:paragraph -->\n<p>Theirs.</p>\n<!-- /wp:paragraph -->"
		form      = `[contact-form-7 id="1"]`
		first     = "<h3>Storage</h3><p>Keep them cold.</p>"
		second    = "<h3>Storage</h3><p>Keep them frozen.</p>"
	)
	current := paragraph + "\n\n" + form
	stored, kept := block.Keep(current, nil, first)
	if !kept || !strings.HasPrefix(stored, paragraph) || !strings.HasSuffix(stored, form) ||
		!strings.Contains(stored, "<!-- wp:html -->\n"+first+"\n<!-- /wp:html -->") {
		t.Fatalf("the first run stored %q, want the client's paragraph, ours as its own block, then the form", stored)
	}
	written := block.Ours(stored, block.Client(current, nil), block.Part(current, nil, first))
	if len(written) != 1 {
		t.Fatalf("ours = %v, want our one block and nothing of the client's", written)
	}

	again, _ := block.Keep(stored, written, second)
	if !strings.HasPrefix(again, paragraph) || !strings.HasSuffix(again, form) || strings.Contains(again, first) ||
		strings.Count(again, second) != 1 {
		t.Errorf("the second run stored %q, want the client's paragraph and form kept and one copy of ours", again)
	}
}

func TestTheClientsBlocksAreWhatIsLeftOfOurs(t *testing.T) {
	t.Parallel()

	const theirs = "<p>Theirs.</p>\n\n"
	ours := "<p>Ours.</p>"
	if got := block.Client(theirs+ours, hashesOf(ours)); got != theirs {
		t.Errorf("Client = %q, want only the client's block", got)
	}
	if got := block.Client(ours, hashesOf(ours)); got != "" {
		t.Errorf("Client of our own body = %q, want nothing", got)
	}
}

func TestOursAreTheBlocksTheClientDidNotHave(t *testing.T) {
	t.Parallel()

	const (
		client = "<p>Theirs.</p>"
		part   = "<h2>Storage</h2><p>Keep them cold.</p>"
	)
	stored := client + "\n\n" + part
	got := block.Ours(stored, client, part)
	if len(got) == 0 {
		t.Fatal("no block counted as ours")
	}
	for _, hash := range got {
		if hash == block.Hash("<p>Theirs.</p>") {
			t.Errorf("the client's paragraph was counted as ours")
		}
	}
	if all := block.Ours(stored, "", stored); len(all) != len(hashesOf(stored)) {
		t.Errorf("with no client text every block is ours: %v", all)
	}
}

func TestARewriteKeepsTheClientsNonTextBlocks(t *testing.T) {
	t.Parallel()

	const (
		intro     = "<p>His intro.</p>"
		outro     = "<p>His outro.</p>"
		shortcode = `[products limit="4"]`
		banner    = `<div class="banner">Sale today</div>`
		ours      = "<h2>Storage</h2>\n\n<p>Keep them cold.</p>"
		earlier   = "<p>What we wrote before.</p>"
		html      = "<!-- wp:html -->\n" + ours + "\n<!-- /wp:html -->"
		paragraph = "<!-- wp:paragraph -->\n<p>His editor text.</p>\n<!-- /wp:paragraph -->"
		cart      = "<!-- wp:woocommerce/cart /-->"
	)
	cases := []struct {
		name    string
		current string
		written []string
		want    string
		theirs  string
	}{
		{name: "an empty body is ours alone", current: "", want: ours},
		{
			name: "ours takes the first text block and his other text goes", current: intro + "\n\n" + shortcode + "\n\n" + outro + "\n\n" + banner,
			want: ours + "\n\n" + shortcode + "\n\n" + banner, theirs: shortcode + banner,
		},
		{
			name: "ours takes the place of what we wrote before", current: shortcode + "\n\n" + earlier + "\n\n" + outro,
			written: []string{block.Hash(earlier)}, want: shortcode + "\n\n" + ours, theirs: shortcode,
		},
		{
			name: "a body with no text keeps everything and takes ours before the grid", current: shortcode,
			want: ours + "\n\n" + shortcode, theirs: shortcode,
		},
		{
			name: "an editor body takes ours as one html block", current: paragraph + "\n\n" + cart,
			want: html + "\n\n" + cart, theirs: cart,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw, theirs, part, err := block.Rewrite(tc.current, tc.written, ours)
			if err != nil {
				t.Fatalf("Rewrite: %v", err)
			}
			if raw != tc.want {
				t.Errorf("the body is\n%q\nwant\n%q", raw, tc.want)
			}
			if strings.ReplaceAll(theirs, "\n", "") != tc.theirs {
				t.Errorf("theirs = %q, want %q", theirs, tc.theirs)
			}
			if !strings.Contains(raw, strings.TrimSpace(part)) {
				t.Errorf("our part %q is not in the body %q", part, raw)
			}
		})
	}
}
