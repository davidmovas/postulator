package content_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func mustParse(t *testing.T, fragment string) *content.Document {
	t.Helper()

	doc, err := content.Parse(fragment)
	if err != nil {
		t.Fatalf("Parse(%q): %v", fragment, err)
	}
	return doc
}

func bodyOf(t *testing.T, doc *content.Document) string {
	t.Helper()

	body, err := doc.Render()
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return body
}

func TestDocumentReadsItsShape(t *testing.T) {
	t.Parallel()

	doc := mustParse(t, `<h1>Espresso</h1><p>A <strong>short</strong> intro to <a href="/coffee/">coffee</a>.</p>`+
		`<h2>Beans</h2><p>More text.</p>`)

	if got := len(doc.Paragraphs()); got != 2 {
		t.Errorf("Paragraphs = %d, want 2", got)
	}
	if got := len(doc.Headings()); got != 2 {
		t.Errorf("Headings = %d, want 2", got)
	}

	links := doc.Links()
	if len(links) != 1 || links[0].Href != "/coffee/" || links[0].Anchor != "coffee" {
		t.Fatalf("Links = %+v", links)
	}
	if !strings.Contains(doc.Text(), "A short intro to coffee.") {
		t.Errorf("Text = %q", doc.Text())
	}
	if got := len(doc.Words()); got != 9 {
		t.Errorf("Words = %d (%v)", got, doc.Words())
	}
	if body := bodyOf(t, doc); !strings.HasPrefix(body, "<h1>Espresso</h1>") {
		t.Errorf("Render = %q", body)
	}
}

func TestRenderSaysSoRatherThanAnsweringAnEmptyBody(t *testing.T) {
	t.Parallel()

	cases := []struct {
		mutate   func(*content.Document)
		name     string
		fragment string
		want     string
	}{
		{
			name:     "a readable body",
			fragment: "<h1>Espresso</h1><p>One shot.</p>",
			want:     "<h1>Espresso</h1><p>One shot.</p>",
		},
		{
			name:     "a body that cannot be rendered",
			fragment: "<p>One shot.</p>",
			mutate:   func(d *content.Document) { d.Paragraphs()[0].Type = html.ErrorNode },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc := mustParse(t, tc.fragment)
			if tc.mutate != nil {
				tc.mutate(doc)
			}

			rendered, err := doc.Render()
			if tc.mutate == nil {
				if err != nil {
					t.Fatalf("Render: %v", err)
				}
				if rendered != tc.want {
					t.Fatalf("Render = %q, want %q", rendered, tc.want)
				}
				return
			}
			if !errors.IsCode(err, errors.Internal) {
				t.Fatalf("Render = %q, %v; want an internal error rather than a silent empty body", rendered, err)
			}
			if rendered != "" {
				t.Fatalf("Render = %q, want nothing beside the error", rendered)
			}
		})
	}
}

func TestTextNodesSkipWhatTheCallerRefuses(t *testing.T) {
	t.Parallel()

	doc := mustParse(t, `<p>outside <code>inside</code> outside again</p>`)

	var all []string
	for node := range doc.TextNodes(nil) {
		all = append(all, strings.TrimSpace(node.Data))
	}
	if len(all) != 3 {
		t.Fatalf("text nodes = %v", all)
	}

	skipped := make([]string, 0)
	for node := range doc.TextNodes(func(n *html.Node) bool {
		return n.Parent != nil && n.Parent.Data == "code"
	}) {
		skipped = append(skipped, strings.TrimSpace(node.Data))
	}
	if len(skipped) != 2 || skipped[0] != "outside" {
		t.Fatalf("skipped text nodes = %v", skipped)
	}

	stopped := 0
	for range doc.TextNodes(nil) {
		stopped++
		break
	}
	if stopped != 1 {
		t.Fatalf("breaking out of the iterator yielded %d nodes", stopped)
	}
}

func TestHashIgnoresFormattingButNotContent(t *testing.T) {
	t.Parallel()

	spaced := mustParse(t, "<p>  Coffee   is   good  </p>\n<p>Second</p>")
	tight := mustParse(t, "<p>Coffee is good</p><p>Second</p>")
	if spaced.Hash() != tight.Hash() {
		t.Error("whitespace must not change the hash")
	}

	reordered := mustParse(t, `<p class="lead" id="one">Coffee is good</p>`)
	swapped := mustParse(t, `<p id="one" class="lead">Coffee is good</p>`)
	if reordered.Hash() != swapped.Hash() {
		t.Error("attribute order must not change the hash")
	}

	other := mustParse(t, "<p>Tea is good</p><p>Second</p>")
	if tight.Hash() == other.Hash() {
		t.Error("different text must change the hash")
	}
	if len(tight.Hash()) != 64 {
		t.Errorf("Hash = %q, want a sha256 digest", tight.Hash())
	}
}

func TestParseRefusesNothingButReportsBadInput(t *testing.T) {
	t.Parallel()

	doc := mustParse(t, "<p>unclosed")
	if body := bodyOf(t, doc); body != "<p>unclosed</p>" {
		t.Errorf("Render = %q", body)
	}
	empty := mustParse(t, "")
	if body := bodyOf(t, empty); body != "" || empty.Text() != "" {
		t.Errorf("the empty document = %q", body)
	}
}

func TestAttrAndTextOf(t *testing.T) {
	t.Parallel()

	doc := mustParse(t, `<p><a href="/x/" rel="nofollow">click <em>here</em></a></p>`)
	link := doc.Links()[0]

	if content.Attr(link.Node, "rel") != "nofollow" || content.Attr(link.Node, "target") != "" {
		t.Errorf("Attr = %q, %q", content.Attr(link.Node, "rel"), content.Attr(link.Node, "target"))
	}
	if content.TextOf(link.Node) != "click here" {
		t.Errorf("TextOf = %q", content.TextOf(link.Node))
	}
}

func TestAssembleBuildsABody(t *testing.T) {
	t.Parallel()

	draft := content.ContentDraft{
		Title: "Espresso guide",
		H1:    "Espresso & you",
		Sections: []content.DraftSection{
			{Heading: "Beans", HTML: "<p>Pick a roast.</p>"},
			{HTML: "<p>No heading here.</p>"},
		},
		Summary: "A guide.",
	}

	doc, err := content.Assemble(draft)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	want := "<h1>Espresso &amp; you</h1><h2>Beans</h2><p>Pick a roast.</p><p>No heading here.</p>"
	if body := bodyOf(t, doc); body != want {
		t.Fatalf("Assemble = %q, want %q", body, want)
	}
}

func TestAssembleRefusesAnIncompleteDraft(t *testing.T) {
	t.Parallel()

	complete := content.ContentDraft{
		Title: "t", H1: "h", Sections: []content.DraftSection{{Heading: "s", HTML: "<p>x</p>"}},
	}

	cases := []struct {
		name   string
		mutate func(*content.ContentDraft)
	}{
		{name: "no title", mutate: func(d *content.ContentDraft) { d.Title = " " }},
		{name: "no h1", mutate: func(d *content.ContentDraft) { d.H1 = "" }},
		{name: "no sections", mutate: func(d *content.ContentDraft) { d.Sections = nil }},
		{
			name:   "an empty section",
			mutate: func(d *content.ContentDraft) { d.Sections = []content.DraftSection{{Heading: "s"}} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			draft := complete
			tc.mutate(&draft)
			if _, err := content.Assemble(draft); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Assemble = %v, want an invalid error", err)
			}
		})
	}
}

func TestInsertAfterSection(t *testing.T) {
	t.Parallel()

	const body = "<h1>Espresso</h1><h2>About</h2><p>One.</p><h2>Brewing</h2><p>Two.</p>"
	const figure = `<figure><img src="/a.png" alt="a"/></figure>`

	cases := []struct {
		name  string
		body  string
		index int
		want  string
	}{
		{
			name:  "between the first and the second section",
			body:  body,
			index: 0,
			want:  `<h1>Espresso</h1><h2>About</h2><p>One.</p><figure><img src="/a.png" alt="a"/></figure><h2>Brewing</h2><p>Two.</p>`,
		},
		{
			name:  "after the last section it lands at the end",
			body:  body,
			index: 1,
			want:  `<h1>Espresso</h1><h2>About</h2><p>One.</p><h2>Brewing</h2><p>Two.</p><figure><img src="/a.png" alt="a"/></figure>`,
		},
		{
			name:  "a body without headings takes it at the end",
			body:  "<p>One.</p>",
			index: 0,
			want:  `<p>One.</p><figure><img src="/a.png" alt="a"/></figure>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := content.Parse(tc.body)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if err = doc.InsertAfterSection(tc.index, figure); err != nil {
				t.Fatalf("InsertAfterSection: %v", err)
			}
			if got := bodyOf(t, doc); got != tc.want {
				t.Errorf("Render =\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}

func TestInsertAfterSectionRefusesNonsense(t *testing.T) {
	t.Parallel()

	doc, err := content.Parse("<p>One.</p>")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	cases := []struct {
		name     string
		index    int
		fragment string
	}{
		{name: "a negative section", index: -1, fragment: "<p>x</p>"},
		{name: "nothing to insert", index: 0, fragment: "   "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := doc.InsertAfterSection(tc.index, tc.fragment); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Invalid, err)
			}
		})
	}
}
