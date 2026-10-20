package block_test

import (
	stderrors "errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	classicBody = "<p>One</p>\n\n<p>Two</p>\n\n[products limit=\"2\"]\n\n<p>Three</p>"
	editorOne   = "<!-- wp:paragraph -->\n<p>One</p>\n<!-- /wp:paragraph -->"
	editorShort = "<!-- wp:shortcode -->\n[products limit=\"4\"]\n<!-- /wp:shortcode -->"
	editorTwo   = "<!-- wp:paragraph -->\n<p>Two</p>\n<!-- /wp:paragraph -->"
	ourHTML     = "<!-- wp:html -->\n<p>Ours</p>\n<!-- /wp:html -->"
	crlfBody    = "Bare one\r\n\r\nBare two\r\n\r\n<p>Three</p>"
)

const editorBody = editorOne + "\n\n" + editorShort + "\n\n" + editorTwo

type operation func(block.Body) (block.Body, error)

func replace(i int, raw string) operation {
	return func(b block.Body) (block.Body, error) { return b.Replace(i, raw) }
}

func insertAfter(i int, raw string) operation {
	return func(b block.Body) (block.Body, error) { return b.InsertAfter(i, raw) }
}

func remove(i int) operation {
	return func(b block.Body) (block.Body, error) { return b.Delete(i) }
}

func rewrite(from, to int, raw string) operation {
	return func(b block.Body) (block.Body, error) { return b.RewriteRange(from, to, raw) }
}

func placeLink(i int, anchor, href string, nth int) operation {
	return func(b block.Body) (block.Body, error) {
		return b.PlaceLink(i, block.Link{Anchor: anchor, Href: href, Nth: nth})
	}
}

func reasonOf(t *testing.T, err error) string {
	t.Helper()

	var kerr *errors.Error
	if !stderrors.As(err, &kerr) || kerr.Code != errors.Invalid {
		t.Fatalf("err = %v, want INVALID", err)
	}
	if reason, ok := kerr.Details["reason"].(string); ok {
		return reason
	}
	return ""
}

func coreOf(raw string) string {
	return strings.Trim(raw, " \t\r\n")
}

func TestEachOperationWritesOnlyWhatItChanges(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		body  string
		apply operation
		want  string
	}{
		{
			name: "a classic block is replaced", body: classicBody, apply: replace(1, "<p>Deux</p>"),
			want: "<p>One</p>\n\n<p>Deux</p>\n\n[products limit=\"2\"]\n\n<p>Three</p>",
		},
		{
			name: "a classic block is added after a middle one", body: classicBody, apply: insertAfter(1, "<p>New</p>"),
			want: "<p>One</p>\n\n<p>Two</p>\n\n<p>New</p>\n\n[products limit=\"2\"]\n\n<p>Three</p>",
		},
		{
			name: "a classic block is added after the last", body: classicBody, apply: insertAfter(3, "<p>New</p>"),
			want: classicBody + "\n\n<p>New</p>",
		},
		{
			name: "a classic block is added before the first", body: classicBody, apply: insertAfter(-1, "<p>New</p>"),
			want: "<p>New</p>\n\n" + classicBody,
		},
		{
			name: "a middle classic block goes with its separator", body: classicBody, apply: remove(1),
			want: "<p>One</p>\n\n[products limit=\"2\"]\n\n<p>Three</p>",
		},
		{
			name: "the first classic block goes", body: classicBody, apply: remove(0),
			want: "<p>Two</p>\n\n[products limit=\"2\"]\n\n<p>Three</p>",
		},
		{
			name: "the last classic block goes with the separator before it", body: classicBody, apply: remove(3),
			want: "<p>One</p>\n\n<p>Two</p>\n\n[products limit=\"2\"]",
		},
		{
			name: "a rewrite takes the first text block and keeps the shortcode", body: classicBody,
			apply: rewrite(0, 4, "<p>Ours</p>"), want: "<p>Ours</p>\n\n[products limit=\"2\"]",
		},
		{
			name: "a rewrite of part of the body leaves the rest", body: classicBody,
			apply: rewrite(2, 4, "<p>Ours</p>"),
			want:  "<p>One</p>\n\n<p>Two</p>\n\n[products limit=\"2\"]\n\n<p>Ours</p>",
		},
		{
			name: "a link is written into its block", body: classicBody, apply: placeLink(1, "Two", "/two/", 0),
			want: "<p>One</p>\n\n<p><a href=\"/two/\">Two</a></p>\n\n[products limit=\"2\"]\n\n<p>Three</p>",
		},
		{
			name: "an editor block is added", body: editorBody, apply: insertAfter(0, ourHTML),
			want: editorOne + "\n\n" + ourHTML + "\n\n" + editorShort + "\n\n" + editorTwo,
		},
		{
			name: "an editor block goes", body: editorBody, apply: remove(1),
			want: editorOne + "\n\n" + editorTwo,
		},
		{
			name: "an editor rewrite keeps the shortcode block", body: editorBody, apply: rewrite(0, 3, ourHTML),
			want: ourHTML + "\n\n" + editorShort,
		},
		{
			name: "an editor block is replaced", body: editorBody, apply: replace(2, ourHTML),
			want: editorOne + "\n\n" + editorShort + "\n\n" + ourHTML,
		},
		{
			name: "a link keeps the editor delimiters", body: editorBody, apply: placeLink(2, "two", "/two/", 0),
			want: editorOne + "\n\n" + editorShort + "\n\n" +
				"<!-- wp:paragraph -->\n<p><a href=\"/two/\">Two</a></p>\n<!-- /wp:paragraph -->",
		},
		{
			name: "a CRLF body is separated by its own line ends", body: crlfBody, apply: insertAfter(0, "Middle"),
			want: "Bare one\r\n\r\nMiddle\r\n\r\nBare two\r\n\r\n<p>Three</p>",
		},
		{
			name: "a CRLF block goes with its separator", body: crlfBody, apply: remove(1),
			want: "Bare one\r\n\r\n<p>Three</p>",
		},
		{
			name: "a delete never fuses its neighbors", body: "a<p>b</p>c", apply: remove(1),
			want: "a\n\nc",
		},
		{
			name: "the only classic block goes", body: "<p>Alone</p>", apply: remove(0),
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := block.Parse(tc.body)
			got, err := tc.apply(body)
			if err != nil {
				t.Fatalf("the operation was refused: %v", err)
			}
			if got.Serialize() != tc.want {
				t.Fatalf("the body is\n%q\nwant\n%q", got.Serialize(), tc.want)
			}
			if body.Serialize() != tc.body {
				t.Fatalf("the operation changed the body it was given: %q", body.Serialize())
			}
			if got.Kind != body.Kind && len(got.Blocks) > 0 {
				t.Fatalf("the body turned %s from %s", got.Kind, body.Kind)
			}
		})
	}
}

func TestAnOperationThatWouldDamageTheBodyIsRefused(t *testing.T) {
	t.Parallel()

	const faq = "<!-- wp:acme/faq {\"question\":\"Why coffee?\"} -->\n<div class=\"faq\"><p>Why coffee?</p></div>\n<!-- /wp:acme/faq -->"
	cases := []struct {
		name   string
		body   string
		apply  operation
		reason string
	}{
		{name: "an editor body keeps its last delimited block", body: editorOne, apply: remove(0), reason: block.ReasonBlockNotEditable},
		{
			name: "an editor body never reads back as classic", body: "<p>Freeform</p>\n\n" + editorOne,
			apply: remove(1), reason: block.ReasonBlockNotEditable,
		},
		{name: "a classic body takes no delimiter", body: classicBody, apply: replace(0, editorOne), reason: block.ReasonBlockNotEditable},
		{name: "a classic body gains no delimiter", body: classicBody, apply: insertAfter(0, editorOne), reason: block.ReasonBlockNotEditable},
		{
			name: "two freeform chunks would fuse", body: "<p>A</p>\n\n" + editorOne + "\n\n<p>B</p>\n\n" + editorTwo,
			apply: remove(1), reason: block.ReasonBlockNotEditable,
		},
		{name: "an empty block is no block", body: classicBody, apply: replace(0, " \n "), reason: block.ReasonBlockNotEditable},
		{name: "a replace past the end", body: classicBody, apply: replace(4, "<p>x</p>"), reason: block.ReasonBlockOutOfRange},
		{name: "an insert before nothing", body: classicBody, apply: insertAfter(-2, "<p>x</p>"), reason: block.ReasonBlockOutOfRange},
		{name: "a delete past the end", body: classicBody, apply: remove(-1), reason: block.ReasonBlockOutOfRange},
		{name: "a range turned around", body: classicBody, apply: rewrite(2, 1, "<p>x</p>"), reason: block.ReasonBlockOutOfRange},
		{name: "a range with no text block", body: classicBody, apply: rewrite(2, 3, "<p>x</p>"), reason: block.ReasonNothingToRewrite},
		{name: "a link inside a heading", body: "<h2>Coffee</h2>", apply: placeLink(0, "Coffee", "/c/", 0), reason: block.ReasonAnchorNotFound},
		{name: "a link past its occurrences", body: "<p>coffee</p>", apply: placeLink(0, "coffee", "/c/", 1), reason: block.ReasonAnchorNotFound},
		{name: "a link inside a shortcode", body: "[products category=\"coffee\"]", apply: placeLink(0, "coffee", "/c/", 0), reason: block.ReasonBlockNotEditable},
		{name: "a link inside a dynamic block", body: "<!-- wp:woocommerce/cart /-->", apply: placeLink(0, "cart", "/c/", 0), reason: block.ReasonBlockNotEditable},
		{name: "a link in a block whose attributes hold its text", body: faq, apply: placeLink(0, "coffee", "/c/", 0), reason: block.ReasonBlockNotEditable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := block.Parse(tc.body)
			_, err := tc.apply(body)
			if got := reasonOf(t, err); got != tc.reason {
				t.Fatalf("the reason is %q, want %q", got, tc.reason)
			}
			if body.Serialize() != tc.body {
				t.Fatalf("a refused operation changed the body: %q", body.Serialize())
			}
		})
	}
}

func TestEachOperationIsUndoneByItsInverse(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{classicBody, editorBody, crlfBody} {
		body := block.Parse(raw)
		added := "<p>Added</p>"
		if body.Kind == block.Blocks {
			added = ourHTML
		}
		for i := -1; i < len(body.Blocks); i++ {
			inserted, err := body.InsertAfter(i, added)
			if err != nil {
				t.Fatalf("InsertAfter(%d) of %q: %v", i, raw, err)
			}
			undone, err := inserted.Delete(i + 1)
			if err != nil {
				t.Fatalf("Delete(%d) of %q: %v", i+1, inserted.Serialize(), err)
			}
			if undone.Serialize() != raw {
				t.Errorf("an insert after %d undone is\n%q\nwant\n%q", i, undone.Serialize(), raw)
			}
		}
		for i := range body.Blocks {
			replaced, err := body.Replace(i, added)
			if err != nil {
				t.Fatalf("Replace(%d) of %q: %v", i, raw, err)
			}
			undone, err := replaced.Replace(i, body.Blocks[i].Raw)
			if err != nil {
				t.Fatalf("Replace(%d) back: %v", i, err)
			}
			if undone.Serialize() != raw {
				t.Errorf("a replace of %d undone is\n%q\nwant\n%q", i, undone.Serialize(), raw)
			}
		}
	}
}

func TestAnAlignedParagraphStaysValid(t *testing.T) {
	t.Parallel()

	const aligned = "<!-- wp:paragraph {\"align\":\"center\"} -->\n<p class=\"has-text-align-center\">We roast coffee daily.</p>\n<!-- /wp:paragraph -->"
	body := block.Parse(aligned + "\n\n" + editorTwo)
	linked, err := body.PlaceLink(0, block.Link{Anchor: "coffee", Href: "/coffee/"})
	if err != nil {
		t.Fatalf("PlaceLink: %v", err)
	}
	held := linked.Blocks[0]
	want := "<!-- wp:paragraph {\"align\":\"center\"} -->\n<p class=\"has-text-align-center\">We roast <a href=\"/coffee/\">coffee</a> daily.</p>\n<!-- /wp:paragraph -->"
	if coreOf(held.Raw) != want {
		t.Fatalf("the aligned paragraph is\n%q\nwant\n%q", held.Raw, want)
	}
	if held.Name != "core/paragraph" || string(held.Attributes) != "{\"align\":\"center\"}" {
		t.Fatalf("the block is %s with %s, want core/paragraph keeping its align", held.Name, held.Attributes)
	}
	if linked.Blocks[1].Raw != body.Blocks[1].Raw {
		t.Fatalf("the next block changed: %q", linked.Blocks[1].Raw)
	}
}

func TestABlockWhoseAttributesCannotBeKeptIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"its attributes hold the anchor": "<!-- wp:acme/faq {\"question\":\"Why coffee?\"} -->\n<div class=\"faq\"><p>Why coffee?</p></div>\n<!-- /wp:acme/faq -->",
		"an inner block's attributes hold it": "<!-- wp:group -->\n<div class=\"wp-block-group\"><!-- wp:acme/note {\"label\":[\"coffee\"]} -->\n" +
			"<p>Note on coffee</p>\n<!-- /wp:acme/note --></div>\n<!-- /wp:group -->",
		"its attributes cannot be read": "<!-- wp:acme/note {\"label\": } -->\n<p>Note on coffee</p>\n<!-- /wp:acme/note -->",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := block.Parse(raw).PlaceLink(0, block.Link{Anchor: "coffee", Href: "/coffee/"})
			if got := reasonOf(t, err); got != block.ReasonBlockNotEditable {
				t.Fatalf("the reason is %q, want %q", got, block.ReasonBlockNotEditable)
			}
		})
	}
}

func TestPlaceLinkWritesTheAnchorsBytesAlone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		raw    string
		anchor string
		href   string
		nth    int
		want   string
	}{
		{
			name: "entities stay as written", raw: "<p>Tea &amp; coffee&nbsp;now</p>", anchor: "Tea & coffee", href: "/tea/",
			want: "<p><a href=\"/tea/\">Tea &amp; coffee</a>&nbsp;now</p>",
		},
		{
			name: "links, code and headings are passed over", raw: "<p>See <a href=\"/old/\">coffee</a> and <code>coffee</code>, then coffee.</p>",
			anchor: "coffee", href: "/c/",
			want: "<p>See <a href=\"/old/\">coffee</a> and <code>coffee</code>, then <a href=\"/c/\">coffee</a>.</p>",
		},
		{
			name: "the second occurrence", raw: "<p>coffee and Coffee</p>", anchor: "coffee", href: "/c/", nth: 1,
			want: "<p>coffee and <a href=\"/c/\">Coffee</a></p>",
		},
		{
			name: "a shortcode's attributes are passed over", raw: "<p>[products category=\"coffee\"] then coffee</p>",
			anchor: "coffee", href: "/c/", want: "<p>[products category=\"coffee\"] then <a href=\"/c/\">coffee</a></p>",
		},
		{
			name: "the href is escaped", raw: "<p>coffee</p>", anchor: "coffee", href: "/a?b=1&c=\"2\"",
			want: "<p><a href=\"/a?b=1&amp;c=&#34;2&#34;\">coffee</a></p>",
		},
		{
			name: "a word inside a longer word is no anchor", raw: "<p>coffees and coffee</p>", anchor: "coffee", href: "/c/",
			want: "<p>coffees and <a href=\"/c/\">coffee</a></p>",
		},
		{
			name: "bare classic text stays one block", raw: "Bare words about coffee", anchor: "coffee", href: "/c/",
			want: "Bare words about <a href=\"/c/\">coffee</a>",
		},
		{
			name: "a CRLF before the anchor is kept", raw: "<p>Fresh\r\ncoffee daily</p>", anchor: "daily", href: "/c/",
			want: "<p>Fresh\r\ncoffee <a href=\"/c/\">daily</a></p>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			linked, err := block.Parse(tc.raw).PlaceLink(0, block.Link{Anchor: tc.anchor, Href: tc.href, Nth: tc.nth})
			if err != nil {
				t.Fatalf("PlaceLink: %v", err)
			}
			if got := linked.Serialize(); got != tc.want {
				t.Fatalf("the block is\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}

func TestWhichBlocksAreProse(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"<p>Plain prose</p>":                     true,
		"Bare words":                             true,
		"<h2>A heading</h2>":                     true,
		"<ul><li>one</li></ul>":                  true,
		"<blockquote><p>quoted</p></blockquote>": true,
		"<p>With <strong>emphasis</strong> and <a href=\"/x/\">a link</a></p>": true,
		"<div class=\"wrap\">a client's wrapper</div>":                         false,
		"[products limit=\"2\"]":                                               false,
		"<p>Call us on [phone]</p>":                                            false,
		"<p><img src=\"/a.jpg\"> a caption</p>":                                false,
		"<p><iframe src=\"/map\"></iframe> the map</p>":                        false,
		"<table><tr><td>cell</td></tr></table>":                                false,
		editorOne:                                                              true,
		"<!-- wp:heading -->\n<h2 class=\"wp-block-heading\">Heading</h2>\n<!-- /wp:heading -->":                                                  true,
		"<!-- wp:group -->\n<div class=\"wp-block-group\"><!-- wp:paragraph -->\n<p>Nested</p>\n<!-- /wp:paragraph --></div>\n<!-- /wp:group -->": true,
		"<!-- wp:group -->\n<div class=\"wp-block-group\"><!-- wp:image -->\n<figure><img src=\"/a.jpg\"/></figure>\n<!-- /wp:image -->" +
			"<!-- wp:paragraph -->\n<p>Beside a picture</p>\n<!-- /wp:paragraph --></div>\n<!-- /wp:group -->": false,
		"<!-- wp:html -->\n<p>Custom markup</p>\n<!-- /wp:html -->":             false,
		"<!-- wp:acme/note -->\n<p>A plugin's note</p>\n<!-- /wp:acme/note -->": false,
		"<!-- wp:woocommerce/cart /-->":                                         false,
		editorShort:                                                             false,
		"<p>Freeform prose</p>\n\n" + editorTwo:                                 true,
		"<div class=\"wrap\">Freeform wrapper</div>\n\n" + editorTwo:            false,
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			if got := block.Parse(raw).Blocks[0].Prose(); got != want {
				t.Fatalf("Prose() = %t, want %t", got, want)
			}
		})
	}
}

func TestSerializeKeepsEveryUntouchedBlock(t *testing.T) {
	t.Parallel()

	for _, file := range []string{"classic_home.html", "classic_mixed.html", "blocks_setup.html", "blocks_nested.html", "shortcode_hub.html"} {
		body := block.Parse(golden(t, file))
		added := "<p>Ours</p>"
		if body.Kind == block.Blocks {
			added = ourHTML
		}
		for i := range body.Blocks {
			others := cores(body.Blocks[:i], body.Blocks[i+1:])
			all := cores(body.Blocks)
			for name, apply := range map[string]operation{
				"replace": replace(i, added), "delete": remove(i), "insert": insertAfter(i, added),
			} {
				got, err := apply(body)
				if err != nil {
					if name == "delete" && body.Kind == block.Blocks {
						continue
					}
					t.Fatalf("%s %s at %d: %v", file, name, i, err)
				}
				kept := others
				if name == "insert" {
					kept = all
				}
				if missing := keptInOrder(got, kept); missing != "" {
					t.Errorf("%s %s at %d lost or moved %q", file, name, i, missing)
				}
			}
		}
	}
}

func cores(groups ...[]block.Block) []string {
	out := make([]string, 0)
	for _, blocks := range groups {
		for i := range blocks {
			out = append(out, coreOf(blocks[i].Raw))
		}
	}
	return out
}

func keptInOrder(body block.Body, kept []string) string {
	at := 0
	for _, want := range kept {
		for at < len(body.Blocks) && coreOf(body.Blocks[at].Raw) != want {
			at++
		}
		if at == len(body.Blocks) {
			return want
		}
		at++
	}
	return ""
}

var classicParts = []string{
	"<p>A paragraph with <a href=\"/x/\">a link</a> about grinding.</p>",
	"Bare words about roasting on their own line",
	"<h2>A heading on brewing</h2>",
	"<ul>\n<li>one filter</li>\n<li>two kettle</li>\n</ul>",
	"<div class=\"wrap\"><p>inside the kettle</p></div>",
	"[products limit=\"2\" category=\"beans\"]",
	"[note]kept beans[/note]",
	"<blockquote>quoted roasting</blockquote>",
	"<p>Tea &amp; coffee&nbsp;now, <em>freshly</em> roasted coffee.</p>",
	"<p>Use <code>grind</code> to grind the beans finely.</p>",
	"<p>First<br>second grinding</p>",
}

var editorParts = []string{
	"<!-- wp:paragraph -->\n<p>Block paragraph about grinding</p>\n<!-- /wp:paragraph -->",
	"<!-- wp:paragraph {\"align\":\"center\"} -->\n<p class=\"has-text-align-center\">Centered roasting notes</p>\n<!-- /wp:paragraph -->",
	"<!-- wp:heading {\"level\":2} -->\n<h2 class=\"wp-block-heading\">Block heading on brewing</h2>\n<!-- /wp:heading -->",
	"<!-- wp:group -->\n<div class=\"wp-block-group\"><!-- wp:paragraph -->\n<p>Nested kettle</p>\n<!-- /wp:paragraph --></div>\n<!-- /wp:group -->",
	"<!-- wp:list -->\n<ul class=\"wp-block-list\"><!-- wp:list-item -->\n<li>One filter</li>\n<!-- /wp:list-item --></ul>\n<!-- /wp:list -->",
	"<!-- wp:paragraph -->\n<p>Tea &amp; coffee&nbsp;now, <em>freshly</em> roasted coffee.</p>\n<!-- /wp:paragraph -->",
	"<!-- wp:woocommerce/cart /-->",
	"<!-- wp:shortcode -->\n[products limit=\"4\" category=\"beans\"]\n<!-- /wp:shortcode -->",
	"<!-- wp:image {\"id\":7} -->\n<figure class=\"wp-block-image\"><img src=\"/a.jpg\" alt=\"beans\"/></figure>\n<!-- /wp:image -->",
	"<p>Freeform roasting chunk</p>",
}

var classicAdditions = []string{"<p>Ours added here</p>", "<p>Ours one</p>\n\n<p>Ours two</p>", "<h2>Our heading</h2>", "Our bare sentence"}

var editorAdditions = []string{ourHTML, ourHTML + "\n\n<!-- wp:paragraph -->\n<p>Ours two</p>\n<!-- /wp:paragraph -->"}

func randomBody(source *rand.Rand, editor bool) string {
	parts := classicParts
	if editor {
		parts = editorParts
	}
	var builder strings.Builder
	if source.IntN(4) == 0 {
		builder.WriteString(separators[source.IntN(len(separators))])
	}
	count := 1 + source.IntN(6)
	for i := range count {
		if i > 0 {
			builder.WriteString(separators[source.IntN(len(separators))])
		}
		part := parts[source.IntN(len(parts))]
		if editor && i == 0 {
			part = parts[source.IntN(len(parts)-1)]
		}
		builder.WriteString(part)
	}
	if source.IntN(4) == 0 {
		builder.WriteString(separators[source.IntN(len(separators))])
	}
	return builder.String()
}

func randomWord(source *rand.Rand, text string) string {
	words := make([]string, 0)
	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return r < 'A' || (r > 'Z' && r < 'a') || r > 'z' }) {
		if len(word) >= 4 {
			words = append(words, word)
		}
	}
	if len(words) == 0 {
		return ""
	}
	return words[source.IntN(len(words))]
}

type expectation struct {
	apply  operation
	kept   []string
	blocks int
	linkAt int
}

func randomOperation(source *rand.Rand, body block.Body) expectation {
	additions := classicAdditions
	if body.Kind == block.Blocks {
		additions = editorAdditions
	}
	added := additions[source.IntN(len(additions))]
	addedBlocks := len(block.Parse(added).Blocks)
	n := len(body.Blocks)
	if n == 0 {
		return expectation{apply: insertAfter(-1, added), blocks: addedBlocks, linkAt: -1}
	}
	i := source.IntN(n)
	switch source.IntN(5) {
	case 0:
		return expectation{apply: replace(i, added), kept: cores(body.Blocks[:i], body.Blocks[i+1:]), blocks: n - 1 + addedBlocks, linkAt: -1}
	case 1:
		at := source.IntN(n+1) - 1
		return expectation{apply: insertAfter(at, added), kept: cores(body.Blocks), blocks: n + addedBlocks, linkAt: -1}
	case 2:
		return expectation{apply: remove(i), kept: cores(body.Blocks[:i], body.Blocks[i+1:]), blocks: n - 1, linkAt: -1}
	case 3:
		to := i + 1 + source.IntN(n-i)
		kept := cores(body.Blocks[:i])
		prose := 0
		for k := i; k < to; k++ {
			if body.Blocks[k].Prose() {
				prose++
				continue
			}
			kept = append(kept, coreOf(body.Blocks[k].Raw))
		}
		kept = append(kept, cores(body.Blocks[to:])...)
		return expectation{apply: rewrite(i, to, added), kept: kept, blocks: n - prose + addedBlocks, linkAt: -1}
	default:
		return expectation{
			apply:  placeLink(i, randomWord(source, body.Blocks[i].Text()), "/target/", 0),
			kept:   cores(body.Blocks[:i], body.Blocks[i+1:]),
			blocks: n, linkAt: i,
		}
	}
}

func TestEveryOperationKeepsTheBytesItDidNotTouch(t *testing.T) {
	t.Parallel()

	source := rand.New(rand.NewPCG(31, 7))
	applied := 0
	for round := range 400 {
		raw := randomBody(source, source.IntN(2) == 0)
		body := block.Parse(raw)
		for step := range 1 + source.IntN(4) {
			op := randomOperation(source, body)
			before := body.Serialize()
			next, err := op.apply(body)
			if body.Serialize() != before {
				t.Fatalf("round %d step %d: the operation changed the body it was given", round, step)
			}
			if err != nil {
				reasonOf(t, err)
				continue
			}
			if len(body.Blocks) > 0 && next.Kind != body.Kind {
				t.Fatalf("round %d step %d: %q turned %s into %q", round, step, before, body.Kind, next.Serialize())
			}
			reparsed := block.Parse(next.Serialize())
			if len(reparsed.Blocks) != len(next.Blocks) {
				t.Fatalf("round %d step %d: %q parses back to %d blocks, not %d", round, step, next.Serialize(), len(reparsed.Blocks), len(next.Blocks))
			}
			for k := range next.Blocks {
				if reparsed.Blocks[k].Raw != next.Blocks[k].Raw || reparsed.Blocks[k].Hash != next.Blocks[k].Hash {
					t.Fatalf("round %d step %d: block %d parses back as %q, not %q", round, step, k, reparsed.Blocks[k].Raw, next.Blocks[k].Raw)
				}
			}
			if missing := keptInOrder(next, op.kept); missing != "" {
				t.Fatalf("round %d step %d: %q became %q and lost or moved %q", round, step, before, next.Serialize(), missing)
			}
			if len(next.Blocks) != op.blocks {
				t.Fatalf("round %d step %d: %q became %q with %d blocks, want %d", round, step, before, next.Serialize(), len(next.Blocks), op.blocks)
			}
			if op.linkAt >= 0 {
				if unlinked := withoutOurLink(next.Blocks[op.linkAt].Raw); coreOf(unlinked) != coreOf(body.Blocks[op.linkAt].Raw) {
					t.Fatalf("round %d step %d: the link changed more than its own bytes: %q from %q", round, step, next.Blocks[op.linkAt].Raw, body.Blocks[op.linkAt].Raw)
				}
			}
			body = next
			applied++
		}
	}
	if applied < 500 {
		t.Fatalf("only %d operations were applied, too few to hold the property", applied)
	}
}

func TestPlaceLinkAgreesWithTheSplice(t *testing.T) {
	t.Parallel()

	source := rand.New(rand.NewPCG(5, 11))
	linkPolicy := field.LinkPolicy{Links: true, Rules: field.LinkRules{MaxLinks: 5, MaxPerTarget: 1}, AnchorStrategy: field.AnchorPreferUser}
	agreed := 0
	for round := range 400 {
		raw := randomBody(source, source.IntN(2) == 0)
		anchor := randomWord(source, block.Parse(raw).Text())
		if anchor == "" {
			continue
		}
		goal := content.LinkTarget{PageID: "t", EntityID: "t", URL: "/target/", Anchors: []string{anchor}, Relation: content.RelationDown, Weight: 1, Depth: 1}
		lc := content.LinkContext{PageID: "self", PageURL: "/self/", EntityID: "self", Targets: []content.LinkTarget{goal}}
		doc, err := content.Parse(raw)
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		placement := content.InsertTarget(doc, lc, linkPolicy, goal)
		if len(placement.Placed) == 0 || placement.Decisions[0].Outcome != content.OutcomeInserted {
			continue
		}
		rendered, err := doc.Render()
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		spliced, _ := block.Splice(raw, rendered, nil)
		at, single := changedBlock(block.Diff(raw, spliced, nil))
		if !single {
			continue
		}
		placed, err := block.Parse(raw).PlaceLink(at, block.Link{Anchor: placement.Placed[0].Anchor, Href: lc.Href(goal)})
		if err != nil {
			t.Fatalf("round %d: the linker placed %q in block %d of %q and PlaceLink refused: %v", round, anchor, at, raw, err)
		}
		if !block.SameRender(placed.Serialize(), spliced) {
			t.Fatalf("round %d: PlaceLink wrote\n%q\nand the splice\n%q", round, placed.Serialize(), spliced)
		}
		agreed++
	}
	if agreed < 120 {
		t.Fatalf("only %d placements were compared, too few to hold the agreement", agreed)
	}
}

func withoutOurLink(raw string) string {
	before, rest, opened := strings.Cut(raw, "<a href=\"/target/\">")
	anchor, after, closed := strings.Cut(rest, "</a>")
	if !opened || !closed {
		return raw
	}
	return before + anchor + after
}

func changedBlock(changes []block.Change) (at int, single bool) {
	at, removed, added, old := -1, 0, 0, 0
	for i := range changes {
		switch changes[i].Change {
		case block.ChangeAdded:
			added++
		case block.ChangeRemoved:
			at = old
			removed++
			old++
		default:
			old++
		}
	}
	return at, removed == 1 && added == 1
}
