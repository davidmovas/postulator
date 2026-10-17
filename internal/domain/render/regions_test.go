package render_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func regionsOf(e render.Extraction, kind render.RegionKind) []render.Region {
	out := make([]render.Region, 0)
	for _, region := range e.Regions {
		if region.Kind == kind {
			out = append(out, region)
		}
	}
	return out
}

func TestExtractNamesTheListingFromTheItemList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		stand    string
		slug     string
		document string
		want     []render.Region
	}{
		{
			name: "tb-500 lists one product and still has a listing", stand: replica, slug: "tb-500", document: "/tb-500/",
			want: []render.Region{{Kind: render.RegionListing, Inside: true, Links: 1, Words: 6, Signal: render.SignalJSONLD}},
		},
		{
			name: "the hub's item list names the fixture's section and the shortcode's grid is a listing by its product links", stand: replica, slug: "bpc-157", document: "/bpc-157/",
			want: []render.Region{
				{Kind: render.RegionListing, Inside: true, Links: 12, Words: 73, Signal: render.SignalLinks},
				{Kind: render.RegionListing, Inside: true, Links: 2, Words: 10, Signal: render.SignalJSONLD},
			},
		},
		{
			name: "the replica's term lists its products inside the assumed root", stand: replica, slug: "healing-peptides", document: "/product-category/peptides/healing-peptides/",
			want: []render.Region{{Kind: render.RegionListing, Inside: true, Links: 12, Words: 101, Signal: render.SignalLinks}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := regionsOf(extractStand(t, tc.stand, tc.slug, tc.document), render.RegionListing)
			if len(got) != len(tc.want) {
				t.Fatalf("listings %+v, want %+v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("listing %d: %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestExtractNamesEveryRegion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   render.Input
		want []render.Region
	}{
		{
			name: "landmarks, an article's header that is no banner, a nav inside main, generated text and a product grid",
			in: render.Input{
				Public: `<html><body>
					<header><a href="/">Shop</a><nav><a href="/a/">A</a></nav></header>
					<nav aria-label="Drawer"><a href="/b/">B</a></nav>
					<main><article><header><h1>Title</h1></header>
						<div><nav aria-label="Contents"><a href="#one">One</a></nav>
							<p>Our own words with a <a href="/a/">link</a>.</p>
							<p>Injected promo text.</p>
							<ul><li><a href="/p/1/">P1</a></li><li><a href="/p/2/">P2</a></li><li><a href="/p/3/">P3</a></li></ul>
						</div>
						<aside><a href="/z/">Z</a></aside>
					</article></main>
					<div role="contentinfo"><a href="/legal/">Legal</a></div>
				</body></html>`,
				Filtered: `<nav aria-label="Contents"><a href="#one">One</a></nav><p>Our own words with a <a href="/a/">link</a>.</p><p>Injected promo text.</p>` +
					`<ul><li><a href="/p/1/">P1</a></li><li><a href="/p/2/">P2</a></li><li><a href="/p/3/">P3</a></li></ul>`,
				Body:     block.Parse(`<p>Our own words with a <a href="/a/">link</a>.</p>`),
				Document: "/page/",
				Lookup: func(path string) (pagemap.WPType, bool) {
					if len(path) > 3 && path[:3] == "/p/" {
						return pagemap.WPProduct, true
					}
					return pagemap.WPPage, true
				},
			},
			want: []render.Region{
				{Kind: render.RegionNavigation, Inside: false, Links: 2, Words: 2, Signal: render.SignalLandmark},
				{Kind: render.RegionNavigation, Inside: false, Links: 1, Words: 1, Signal: render.SignalLandmark},
				{Kind: render.RegionListing, Inside: true, Links: 3, Words: 3, Signal: render.SignalLinks},
				{Kind: render.RegionNavigation, Inside: false, Links: 1, Words: 1, Signal: render.SignalLandmark},
				{Kind: render.RegionContent, Inside: true, Links: 1, Words: 6, Signal: ""},
				{Kind: render.RegionGenerated, Inside: true, Links: 1, Words: 4, Signal: ""},
				{Kind: render.RegionUnknown, Inside: false, Links: 1, Words: 2, Signal: ""},
			},
		},
		{
			name: "a shortcode's expansion is the content's own dynamic region",
			in: render.Input{
				Public:   `<html><body><main><div><p>Expanded by the shortcode.</p><p>Second line.</p></div></main></body></html>`,
				Filtered: `<p>Expanded by the shortcode.</p><p>Second line.</p>`,
				Body:     block.Parse(`[gallery ids="1,2"]`),
				Document: "/page/",
			},
			want: []render.Region{
				{Kind: render.RegionContentDynamic, Inside: true, Links: 0, Words: 6, Signal: ""},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, tc.in)
			if len(got.Regions) != len(tc.want) {
				t.Fatalf("regions %+v, want %+v", got.Regions, tc.want)
			}
			for i := range tc.want {
				if got.Regions[i] != tc.want[i] {
					t.Errorf("region %d: %+v, want %+v", i, got.Regions[i], tc.want[i])
				}
			}
		})
	}
}

func TestExtractCountsGeneratedWords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		stand         string
		slug          string
		document      string
		wantContent   int
		wantGenerated int
	}{
		{"the empty hub is all generated", replica, "peptides", "/peptides/", 0, 16},
		{"tb-500 carries its own words and the fixture's", replica, "tb-500", "/tb-500/", 31, 12},
		{"vanilla's nested page is all its own", vanilla, "setup", "/guide/setup/", 14, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := extractStand(t, tc.stand, tc.slug, tc.document)
			if got.WordsContent != tc.wantContent || got.WordsGenerated != tc.wantGenerated {
				t.Errorf("content %d and generated %d, want %d and %d", got.WordsContent, got.WordsGenerated, tc.wantContent, tc.wantGenerated)
			}
		})
	}
}

func TestExtractMarksGeneratedHeadings(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   func(t *testing.T) render.Input
		want []render.Heading
	}{
		{
			name: "tb-500's listing heading is the fixture's",
			in:   func(t *testing.T) render.Input { return standInput(t, replica, "tb-500", "/tb-500/") },
			want: []render.Heading{{Level: 2, Text: "TB-500 products", Generated: true}},
		},
		{
			name: "vanilla's heading is the raw's own",
			in:   func(t *testing.T) render.Input { return standInput(t, vanilla, "setup", "/guide/setup/") },
			want: []render.Heading{{Level: 2, Text: "Grinding", Generated: false}},
		},
		{
			name: "the guide has none inside its root",
			in:   func(t *testing.T) render.Input { return standInput(t, vanilla, "guide", "/guide/") },
			want: []render.Heading{},
		},
		{
			name: "a root that is itself a heading",
			in: func(*testing.T) render.Input {
				return render.Input{
					Public:   `<html><body><main><h1>Title</h1><h2>Only a <em>heading</em></h2><h3></h3></main></body></html>`,
					Filtered: `<h2>Only a <em>heading</em></h2>`,
					Body:     block.Parse(`<h2>Only a <em>heading</em></h2>`),
				}
			},
			want: []render.Heading{{Level: 2, Text: "Only a heading", Generated: false}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, tc.in(t))
			if len(got.Headings) != len(tc.want) {
				t.Fatalf("headings %+v, want %+v", got.Headings, tc.want)
			}
			for i := range tc.want {
				if got.Headings[i] != tc.want[i] {
					t.Errorf("heading %d: %+v, want %+v", i, got.Headings[i], tc.want[i])
				}
			}
		})
	}
}
