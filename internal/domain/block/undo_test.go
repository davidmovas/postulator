package block_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

const (
	clientIntro   = "<!-- wp:paragraph -->\n<p>The client's intro.</p>\n<!-- /wp:paragraph -->"
	clientBanner  = "<!-- wp:shortcode -->\n[banner id=\"7\"]\n<!-- /wp:shortcode -->"
	clientOutro   = "<!-- wp:paragraph -->\n<p>The client's outro.</p>\n<!-- /wp:paragraph -->"
	clientMiddle  = "<!-- wp:paragraph -->\n<p>The client's middle paragraph.</p>\n<!-- /wp:paragraph -->"
	oursSection   = "<!-- wp:html -->\n<h2>Ours</h2><p>Our new section.</p>\n<!-- /wp:html -->"
	oursRewrite   = "<!-- wp:html -->\n<p>Our rewrite of the middle.</p>\n<!-- /wp:html -->"
	clientLateAdd = "<!-- wp:paragraph -->\n<p>Added by the client after the run.</p>\n<!-- /wp:paragraph -->"
)

func joined(parts ...string) string {
	return strings.Join(parts, "\n\n")
}

func TestUndoPutsBackEachBlockTheRunChangedAndKeepsTheClientsLaterEdits(t *testing.T) {
	t.Parallel()

	before := joined(clientIntro, clientMiddle, clientBanner, clientOutro)
	after := joined(clientIntro, oursRewrite, clientBanner, clientOutro, oursSection)
	changes := block.Diff(before, after, []string{})

	cases := []struct {
		name   string
		live   string
		want   string
		undone int
		kept   int
	}{
		{name: "nothing changed since", live: after, want: before, undone: 2},
		{
			name: "the client edited his outro since", live: joined(clientIntro, oursRewrite, clientBanner, clientLateAdd, oursSection),
			want: joined(clientIntro, clientMiddle, clientBanner, clientLateAdd), undone: 2,
		},
		{
			name:   "the client edited our rewrite since",
			live:   joined(clientIntro, "<!-- wp:html -->\n<p>The client reworded our rewrite.</p>\n<!-- /wp:html -->", clientBanner, clientOutro, oursSection),
			want:   joined(clientIntro, "<!-- wp:html -->\n<p>The client reworded our rewrite.</p>\n<!-- /wp:html -->", clientBanner, clientOutro),
			undone: 1, kept: 1,
		},
		{name: "the client removed our section", live: joined(clientIntro, oursRewrite, clientBanner, clientOutro), want: joined(clientIntro, clientMiddle, clientBanner, clientOutro), undone: 1, kept: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			undone := block.Undo(tc.live, changes)
			if undone.Body != tc.want {
				t.Errorf("Undo =\n%q\nwant\n%q", undone.Body, tc.want)
			}
			if undone.Undone != tc.undone || undone.Kept != tc.kept {
				t.Errorf("undone %d kept %d, want %d and %d", undone.Undone, undone.Kept, tc.undone, tc.kept)
			}
		})
	}
}

func TestUndoOfAWriteThatAddedNothingLeavesTheBodyAsItIs(t *testing.T) {
	t.Parallel()

	body := joined(clientIntro, clientBanner)
	undone := block.Undo(body, block.Diff(body, body, []string{}))
	if undone.Body != body || undone.Undone != 0 || undone.Kept != 0 {
		t.Errorf("Undo = %+v, want the body untouched", undone)
	}
}

func TestUndoOfAClassicBodyPutsItsParagraphsBack(t *testing.T) {
	t.Parallel()

	before := "<p>One.</p>\n\n<p>Two.</p>"
	after := "<p>One.</p>\n\n<p>Our two.</p>\n\n<p>Ours.</p>"
	if got := block.Undo(after, block.Diff(before, after, []string{})); got.Body != before {
		t.Errorf("Undo = %q, want %q", got.Body, before)
	}
}
