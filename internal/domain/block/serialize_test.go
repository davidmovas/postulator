package block_test

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

var pieces = []string{
	"<p>A paragraph with <a href=\"/x/\">a link</a>.</p>",
	"Bare words on their own line",
	"<h2>A heading</h2>",
	"<ul>\n<li>one</li>\n<li>two</li>\n</ul>",
	"<div class=\"wrap\"><p>inside</p></div>",
	"[products limit=\"2\"]",
	"[note]kept[/note]",
	"<blockquote>quoted</blockquote>",
	"<!-- wp:paragraph -->\n<p>Block paragraph</p>\n<!-- /wp:paragraph -->",
	"<!-- wp:heading {\"level\":2} -->\n<h2 class=\"wp-block-heading\">Block heading</h2>\n<!-- /wp:heading -->",
	"<!-- wp:woocommerce/cart /-->",
	"<!-- wp:group -->\n<div class=\"wp-block-group\"><!-- wp:paragraph -->\n<p>Nested</p>\n<!-- /wp:paragraph --></div>\n<!-- /wp:group -->",
}

var separators = []string{"\n", "\n\n", "\r\n\r\n", "\n\n\n", "  \n\n  ", "\n \n"}

func TestSerializeReturnsTheBytesParsed(t *testing.T) {
	t.Parallel()

	for _, file := range []string{"classic_home.html", "classic_mixed.html", "blocks_setup.html", "blocks_nested.html", "shortcode_hub.html"} {
		raw := golden(t, file)
		if got := block.Parse(raw).Serialize(); got != raw {
			t.Errorf("%s does not round trip:\n%q\n%q", file, got, raw)
		}
	}

	source := rand.New(rand.NewPCG(17, 23))
	for round := range 200 {
		var builder strings.Builder
		if source.IntN(3) == 0 {
			builder.WriteString(separators[source.IntN(len(separators))])
		}
		count := 1 + source.IntN(5)
		for i := range count {
			if i > 0 {
				builder.WriteString(separators[source.IntN(len(separators))])
			}
			builder.WriteString(pieces[source.IntN(len(pieces))])
		}
		if source.IntN(3) == 0 {
			builder.WriteString(separators[source.IntN(len(separators))])
		}
		raw := builder.String()
		if got := block.Parse(raw).Serialize(); got != raw {
			t.Fatalf("round %d does not round trip:\n%q\n%q", round, got, raw)
		}
	}
}

func TestHashIsStableAcrossLineEndingsAndOuterWhitespace(t *testing.T) {
	t.Parallel()

	if block.Hash("<p>a</p>\r\n<p>b</p>") != block.Hash("<p>a</p>\n<p>b</p>") {
		t.Error("a CRLF changes the hash")
	}
	if block.Hash("  <p>a</p>\n\n") != block.Hash("<p>a</p>") {
		t.Error("outer whitespace changes the hash")
	}
	if block.Hash("<p>a</p>") == block.Hash("<p>b</p>") {
		t.Error("different content hashes alike")
	}
	if len(block.Hash("x")) != 64 {
		t.Errorf("the hash is not hex sha256: %q", block.Hash("x"))
	}
	body := block.Parse("<p>a</p>\n\n<p>b</p>")
	if body.Hash() != block.Hash("<p>a</p>\n\n<p>b</p>") {
		t.Error("a body's hash is not the hash of its raw")
	}
	if body.Blocks[0].Hash == body.Blocks[1].Hash {
		t.Error("two blocks with different text share a hash")
	}
}

func TestBodyHashIsTheSitesHashOfTheExactBytes(t *testing.T) {
	t.Parallel()

	if block.BodyHash("<p>a</p>\r\n") == block.BodyHash("<p>a</p>\n") {
		t.Error("a CRLF does not change the body hash, so a body the site holds with CRLF would never match")
	}
	if block.BodyHash(" <p>a</p>") == block.BodyHash("<p>a</p>") {
		t.Error("outer whitespace does not change the body hash")
	}
	if got := block.BodyHash(""); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("the body hash of an empty body is %q, want sha256 of nothing as the site answers it", got)
	}
	if len(block.BodyHash("x")) != 64 {
		t.Errorf("the body hash is not hex sha256: %q", block.BodyHash("x"))
	}
}
