package block_test

import (
	"maps"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestALinkIsOursInOurBlockOrWhenWePlacedIt(t *testing.T) {
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
	if links[0].InOurBlock || links[0].Position != 0 || links[0].Anchor != "coffee" {
		t.Errorf("the client's link = %+v, want it at block 0 and his", links[0])
	}
	if !links[1].InOurBlock || links[1].Position != 1 {
		t.Errorf("our link = %+v, want it at block 1 and ours", links[1])
	}

	placed := map[block.LinkKey]bool{{To: "/coffee/", Anchor: "coffee"}: true}
	if !links[0].Ours("/coffee/", placed) {
		t.Error("a link we placed in the client's paragraph is not ours")
	}
	if links[0].Ours("/coffee/", map[block.LinkKey]bool{{To: "/coffee/", Anchor: "another"}: true}) {
		t.Error("a link was taken as ours by its target alone, though another anchor was placed")
	}
}

func TestTheLinksAlreadyOursAreHeldByTargetAndAnchor(t *testing.T) {
	t.Parallel()

	held := block.HeldOurs([]pagemap.PageLink{
		{ToURL: "/coffee/", AnchorText: "coffee", Source: pagemap.SourceRaw, Ours: true},
		{ToURL: "/tea/", AnchorText: "tea", Source: pagemap.SourceRaw, Ours: true},
		{ToURL: "/tea/", AnchorText: "green tea", Source: pagemap.SourceRaw},
		{ToURL: "/milk/", AnchorText: "milk", Source: pagemap.SourceRaw},
		{ToURL: "/sugar/", AnchorText: "sugar", Source: pagemap.SourceRender, Ours: true},
	})
	want := map[block.LinkKey]bool{{To: "/coffee/", Anchor: "coffee"}: true, {To: "/tea/", Anchor: "tea"}: true}
	if !maps.Equal(held, want) {
		t.Fatalf("HeldOurs = %v, want only the raw links marked ours, %v", held, want)
	}
}
