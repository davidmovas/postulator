package block_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestALinkIsOursInOurBlockOrWhenWePlacedItThere(t *testing.T) {
	t.Parallel()

	const (
		theirs = `<p>See the <a href="/coffee/">coffee</a> range.</p>`
		ours   = `<p>Read about <a href="/coffee/espresso/">espresso</a>.</p>`
		grid   = `[products limit="4"]`
	)
	links := block.LinksOf(theirs+"\n\n"+ours+"\n\n"+grid, []string{block.Hash(ours)})
	if len(links) != 2 {
		t.Fatalf("LinksOf = %+v, want the two links of the text blocks", links)
	}
	if links[0].InOurBlock || links[0].Position != 0 || links[0].Anchor != "coffee" || links[0].BlockHash != block.Hash(theirs) {
		t.Errorf("the client's link = %+v, want it at block 0, in his block, and his", links[0])
	}
	if !links[1].InOurBlock || links[1].Position != 1 || links[1].BlockHash != block.Hash(ours) {
		t.Errorf("our link = %+v, want it at block 1, in our block, and ours", links[1])
	}

	inHisBlock := map[block.PlacedKey]bool{{To: "/coffee/", Anchor: "coffee", BlockHash: block.Hash(theirs)}: true}
	if !links[0].Ours("/coffee/", inHisBlock) {
		t.Error("a link we placed in the client's paragraph is not ours")
	}
	inAnother := map[block.PlacedKey]bool{{To: "/coffee/", Anchor: "coffee", BlockHash: block.Hash(ours)}: true}
	if links[0].Ours("/coffee/", inAnother) {
		t.Error("a link was taken as ours by its target and anchor alone, though it was placed in another block")
	}
	otherAnchor := map[block.PlacedKey]bool{{To: "/coffee/", Anchor: "another", BlockHash: block.Hash(theirs)}: true}
	if links[0].Ours("/coffee/", otherAnchor) {
		t.Error("a link was taken as ours by its target and block alone, though another anchor was placed")
	}
}

func TestTheLinksAlreadyOursAreHeldByTheirBlock(t *testing.T) {
	t.Parallel()

	const (
		first  = `<p>See the <a href="/coffee/">coffee</a> range.</p>`
		second = `<p>Our <a href="/tea/">tea</a> and <a href="/tea/">green tea</a>, with <a href="/milk/">milk</a>.</p>`
		again  = `<p>Also <a href="/coffee/">coffee</a> again.</p>`
	)
	last := block.LinksOf(first+"\n\n"+second+"\n\n"+again, nil)
	resolve := func(href string) (string, bool) { return href, true }
	held := block.HeldOurs([]pagemap.PageLink{
		{ToURL: "/tea/", AnchorText: "tea", Source: pagemap.SourceRaw, Ours: true, BlockHash: block.Hash(second)},
		{ToURL: "/milk/", AnchorText: "milk", Source: pagemap.SourceRaw, Ours: true},
		{ToURL: "/coffee/", AnchorText: "coffee", Source: pagemap.SourceRaw, Ours: true},
		{ToURL: "/water/", AnchorText: "water", Source: pagemap.SourceRaw, Ours: true},
		{ToURL: "/tea/", AnchorText: "green tea", Source: pagemap.SourceRaw, BlockHash: block.Hash(second)},
		{ToURL: "/sugar/", AnchorText: "sugar", Source: pagemap.SourceRender, Ours: true, BlockHash: block.Hash(first)},
	}, last, resolve, []block.Moved{{From: block.Hash(second), To: "the-block-after-our-write"}})
	want := map[block.PlacedKey]bool{
		{To: "/tea/", Anchor: "tea", BlockHash: "the-block-after-our-write"}:   true,
		{To: "/milk/", Anchor: "milk", BlockHash: "the-block-after-our-write"}: true,
	}
	if !maps.Equal(held, want) {
		t.Fatalf("HeldOurs = %v\nwant the keyed raw mark and the legacy mark one block holds, both moved with their block, %v", held, want)
	}
}

func TestLineageFollowsTheBlocksThatChangedBetweenTwoBodies(t *testing.T) {
	t.Parallel()

	const (
		client   = "<p>Fresh roasted daily.</p>"
		para     = "<p>We sell espresso and filter coffee.</p>"
		linked   = `<p>We sell <a href="/espresso/">espresso</a> and filter coffee.</p>`
		sentence = "<p>Read more about filter coffee.</p>"
		grid     = `[products limit="4"]`
	)
	before := client + "\n\n" + para + "\n\n" + grid
	after := client + "\n\n" + linked + "\n\n" + sentence + "\n\n" + grid

	if got, want := block.Lineage(before, after, nil), []block.Moved{{From: block.Hash(para), To: block.Hash(linked)}}; !slices.Equal(got, want) {
		t.Errorf("Lineage forward = %v, want the linked paragraph alone, %v", got, want)
	}
	if got, want := block.Lineage(after, before, []string{block.Hash(sentence)}), []block.Moved{{From: block.Hash(linked), To: block.Hash(para)}}; !slices.Equal(got, want) {
		t.Errorf("Lineage back = %v, want the paragraph put back alone, %v", got, want)
	}
}
