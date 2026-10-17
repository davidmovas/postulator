package block_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
)

type wantBlock struct {
	kind    block.BlockKind
	name    string
	text    string
	dynamic bool
}

func golden(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read the golden %s: %v", name, err)
	}
	return string(raw)
}

func assertBlocks(t *testing.T, body block.Body, want []wantBlock) {
	t.Helper()
	if len(body.Blocks) != len(want) {
		for i := range body.Blocks {
			t.Logf("block %d %s %q dynamic=%t text=%q", i, body.Blocks[i].Kind, body.Blocks[i].Name, body.Blocks[i].Dynamic, body.Blocks[i].Text())
		}
		t.Fatalf("got %d blocks, want %d", len(body.Blocks), len(want))
	}
	for i, w := range want {
		got := body.Blocks[i]
		if got.Index != i {
			t.Errorf("block %d carries index %d", i, got.Index)
		}
		if got.Kind != w.kind {
			t.Errorf("block %d kind = %s, want %s", i, got.Kind, w.kind)
		}
		if got.Name != w.name {
			t.Errorf("block %d name = %q, want %q", i, got.Name, w.name)
		}
		if got.Dynamic != w.dynamic {
			t.Errorf("block %d dynamic = %t, want %t", i, got.Dynamic, w.dynamic)
		}
		if got.Text() != w.text {
			t.Errorf("block %d text = %q, want %q", i, got.Text(), w.text)
		}
		if got.Hash == "" || got.Hash != block.Hash(got.Raw) {
			t.Errorf("block %d hash %q is not the hash of its raw", i, got.Hash)
		}
	}
}

func TestParseSplitsClassicContentOnBlankLinesAndBlockElements(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		file   string
		blocks []wantBlock
	}{
		"two paragraphs of bare text": {
			file: "classic_home.html",
			blocks: []wantBlock{
				{kind: block.Paragraph, text: "Research peptides for the laboratory, shipped cold and sealed under nitrogen."},
				{kind: block.Paragraph, text: "Browse the peptides or read about BPC-157 and TB-500."},
			},
		},
		"paragraph runs, a wrapper, a list, a heading, bare text, a shortcode and a quote": {
			file: "classic_mixed.html",
			blocks: []wantBlock{
				{kind: block.Paragraph, text: "First second"},
				{kind: block.Paragraph, text: "Third"},
				{kind: block.Foreign, name: "div", text: "inner deeper"},
				{kind: block.List, name: "ul", text: "a b"},
				{kind: block.Heading, name: "h2", text: "Heading"},
				{kind: block.Paragraph, text: "Bare text with emphasis"},
				{kind: block.Shortcode, name: "products", dynamic: true},
				{kind: block.Quote, name: "blockquote", text: "Quoted"},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := block.Parse(golden(t, tc.file))
			if body.Kind != block.Classic {
				t.Fatalf("kind = %s, want classic", body.Kind)
			}
			assertBlocks(t, body, tc.blocks)
		})
	}
}

func TestParseReadsBlockEditorDelimitersAndAttributes(t *testing.T) {
	t.Parallel()

	body := block.Parse(golden(t, "blocks_setup.html"))
	if body.Kind != block.Blocks {
		t.Fatalf("kind = %s, want blocks", body.Kind)
	}
	assertBlocks(t, body, []wantBlock{
		{kind: block.Heading, name: "core/heading", text: "Grinding"},
		{kind: block.Paragraph, name: "core/paragraph", text: "Grind the beans just before you brew them, and keep the rest sealed."},
	})
	if string(body.Blocks[0].Attributes) != "" {
		t.Errorf("a heading without attributes carries %q", body.Blocks[0].Attributes)
	}

	nested := block.Parse(golden(t, "blocks_nested.html"))
	assertBlocks(t, nested, []wantBlock{
		{kind: block.WP, name: "core/group", text: "Inside the group"},
		{kind: block.WP, name: "woocommerce/cart", dynamic: true},
		{kind: block.Heading, name: "core/heading", text: "After"},
		{kind: block.List, name: "core/list", text: "One"},
		{kind: block.Shortcode, name: "core/shortcode", dynamic: true},
	})
	if string(nested.Blocks[0].Attributes) != `{"layout":{"type":"constrained"}}` {
		t.Errorf("group attributes = %s", nested.Blocks[0].Attributes)
	}
	if string(nested.Blocks[2].Attributes) != `{"level":3}` {
		t.Errorf("heading attributes = %s", nested.Blocks[2].Attributes)
	}
	wantNames := []string{"core/group", "woocommerce/cart", "core/heading", "core/list", "core/shortcode"}
	if got := nested.BlockNames(); !equalStrings(got, wantNames) {
		t.Errorf("block names = %v, want %v", got, wantNames)
	}
	if got := nested.Shortcodes(); !equalStrings(got, []string{"products"}) {
		t.Errorf("shortcodes = %v, want [products]", got)
	}
	if nested.DynamicOnly() || nested.Empty() {
		t.Error("a body with text is neither dynamic only nor empty")
	}
}

func TestParseKeepsAShortcodeAtomic(t *testing.T) {
	t.Parallel()

	body := block.Parse(golden(t, "shortcode_hub.html"))
	assertBlocks(t, body, []wantBlock{{kind: block.Shortcode, name: "products", dynamic: true}})
	if !body.DynamicOnly() {
		t.Error("a lone shortcode is dynamic only")
	}
	if body.Empty() {
		t.Error("a lone shortcode is not empty")
	}
	if got := body.Shortcodes(); !equalStrings(got, []string{"products"}) {
		t.Errorf("shortcodes = %v", got)
	}
	if body.Text() != "" {
		t.Errorf("a shortcode has no text, got %q", body.Text())
	}

	enclosing := block.Parse("[note]Keep cold[/note]\n\nPlain words.")
	assertBlocks(t, enclosing, []wantBlock{
		{kind: block.Shortcode, name: "note", dynamic: true},
		{kind: block.Paragraph, text: "Plain words."},
	})
	if enclosing.DynamicOnly() {
		t.Error("a body with a paragraph beside a shortcode is not dynamic only")
	}
}

func TestAnEmptyBodyIsEmptyAndNotDynamicOnly(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "  \n\n ", "<p></p>"} {
		body := block.Parse(raw)
		if !body.Empty() {
			t.Errorf("Parse(%q) is not empty", raw)
		}
		if body.DynamicOnly() {
			t.Errorf("Parse(%q) is dynamic only", raw)
		}
		if body.Serialize() != raw {
			t.Errorf("Parse(%q) serializes to %q", raw, body.Serialize())
		}
	}
}

func TestTextBlocksListTheBlocksThatCarryWords(t *testing.T) {
	t.Parallel()

	body := block.Parse(golden(t, "classic_mixed.html"))
	want := []string{"First second", "Third", "inner deeper", "a b", "Heading", "Bare text with emphasis", "Quoted"}
	if got := body.TextBlocks(); !equalStrings(got, want) {
		t.Errorf("text blocks = %q, want %q", got, want)
	}
	if body.Text() != "First second Third inner deeper a b Heading Bare text with emphasis Quoted" {
		t.Errorf("text = %q", body.Text())
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
