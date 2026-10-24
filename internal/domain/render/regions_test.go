package render_test

import (
	"slices"
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
			name: "the hub's item list names the fixture's section and the shortcode's grid is a listing by its product links, its own add-to-cart no link and its cards' categories navigation", stand: replica, slug: "bpc-157", document: "/bpc-157/",
			want: []render.Region{
				{Kind: render.RegionListing, Inside: true, Links: 8, Words: 67, Signal: render.SignalLinks},
				{Kind: render.RegionListing, Inside: true, Links: 2, Words: 10, Signal: render.SignalJSONLD},
			},
		},
		{
			name: "the replica's term lists its products inside the assumed root, the cards' links to the term itself no links", stand: replica, slug: "healing-peptides", document: "/product-category/peptides/healing-peptides/",
			want: []render.Region{{Kind: render.RegionListing, Inside: true, Links: 8, Words: 101, Signal: render.SignalLinks}},
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
				{Kind: render.RegionGenerated, Inside: true, Links: 0, Words: 4, Signal: ""},
				{Kind: render.RegionUnknown, Inside: false, Links: 1, Words: 2, Signal: ""},
			},
		},
		{
			name: "links typed by their rel outside the root, a listing outside the root holding a category link, a breadcrumb keeping its own",
			in: render.Input{
				Public: `<html><body>
					<header><a href="/">Shop</a></header>
					<main><article>
						<div><span>By <a href="/author/ann/" rel="author">Ann</a></span>
							<span>Tags: <a href="/tag/a/" rel="tag">A</a>, <a href="/tag/b/" rel="TAG">B</a></span>
							<a href="/elsewhere/" rel="nofollow">Elsewhere</a></div>
						<div><p>Own words with a <a href="/tag/c/" rel="tag">tag</a> link.</p></div>
						<section><ul><li><a href="/p/1/">P1</a> <a href="/p/1/">P1 image</a> <a href="/category/x/" rel="category tag">X</a></li>
							<li><a href="/p/2/">P2</a></li></ul></section>
					</article></main>
					<nav aria-label="Breadcrumb"><a href="/">Home</a> / <a href="/category/x/" rel="category">X</a> / Page</nav>
				</body></html>`,
				Filtered: `<p>Own words with a <a href="/tag/c/" rel="tag">tag</a> link.</p>`,
				Body:     block.Parse(`<p>Own words with a <a href="/tag/c/" rel="tag">tag</a> link.</p>`),
				Document: "/page/",
				Lookup:   productsUnder("/p/"),
			},
			want: []render.Region{
				{Kind: render.RegionNavigation, Inside: false, Links: 1, Words: 1, Signal: render.SignalLandmark},
				{Kind: render.RegionNavigation, Inside: false, Links: 1, Words: 2, Signal: render.SignalRel},
				{Kind: render.RegionNavigation, Inside: false, Links: 2, Words: 3, Signal: render.SignalRel},
				{Kind: render.RegionListing, Inside: false, Links: 3, Words: 4, Signal: render.SignalLinks},
				{Kind: render.RegionNavigation, Inside: false, Links: 1, Words: 1, Signal: render.SignalRel},
				{Kind: render.RegionBreadcrumb, Inside: false, Links: 2, Words: 3, Signal: render.SignalARIA},
				{Kind: render.RegionContent, Inside: true, Links: 1, Words: 6, Signal: ""},
				{Kind: render.RegionUnknown, Inside: false, Links: 1, Words: 1, Signal: ""},
			},
		},
		{
			name: "a rel-typed link the site generates inside the root is navigation, the content's own stays contextual",
			in: render.Input{
				Public: `<html><body><main><div><p>Own words with a <a href="/tag/c/" rel="tag">tag</a> link.</p>` +
					`<p>Filed under <a href="/category/x/" rel="category tag">X</a></p></div></main></body></html>`,
				Filtered: `<p>Own words with a <a href="/tag/c/" rel="tag">tag</a> link.</p><p>Filed under <a href="/category/x/" rel="category tag">X</a></p>`,
				Body:     block.Parse(`<p>Own words with a <a href="/tag/c/" rel="tag">tag</a> link.</p>`),
				Document: "/page/",
			},
			want: []render.Region{
				{Kind: render.RegionNavigation, Inside: true, Links: 1, Words: 1, Signal: render.SignalRel},
				{Kind: render.RegionContent, Inside: true, Links: 1, Words: 6, Signal: ""},
				{Kind: render.RegionGenerated, Inside: true, Links: 0, Words: 2, Signal: ""},
			},
		},
		{
			name: "a product without a description: its category by rel inside the assumed root",
			in: render.Input{
				Public: `<html><body><main><h1>First</h1><div><span>SKU: F1</span> <span>Category: <a href="/c/x/" rel="tag">X</a></span></div>` +
					`<p>Ships in a day.</p></main></body></html>`,
				Body:     block.Parse(""),
				Document: "/p/1/",
			},
			want: []render.Region{
				{Kind: render.RegionNavigation, Inside: true, Links: 1, Words: 2, Signal: render.SignalRel},
				{Kind: render.RegionGenerated, Inside: true, Links: 0, Words: 7, Signal: ""},
			},
		},
		{
			name: "landmark roles count as their elements: a sidebar's footer is no page footer, a widget in the page footer is the footer's",
			in: render.Input{
				Public: `<html><body>
					<div role="banner"><a href="/">Shop</a></div>
					<main><div><p>Own words of the page.</p></div></main>
					<div role="complementary"><h3>Recent</h3><a href="/r/">Recent post</a><footer><a href="/all/">All posts</a></footer></div>
					<footer><div role="complementary"><a href="/w/">Widget link</a></div><p>Rights</p></footer>
				</body></html>`,
				Filtered: `<p>Own words of the page.</p>`,
				Body:     block.Parse(`<p>Own words of the page.</p>`),
				Document: "/page/",
			},
			want: []render.Region{
				{Kind: render.RegionNavigation, Inside: false, Links: 1, Words: 1, Signal: render.SignalLandmark},
				{Kind: render.RegionNavigation, Inside: false, Links: 1, Words: 3, Signal: render.SignalLandmark},
				{Kind: render.RegionContent, Inside: true, Links: 0, Words: 5, Signal: ""},
				{Kind: render.RegionUnknown, Inside: false, Links: 2, Words: 5, Signal: ""},
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

func TestExtractFindsAProductListingWhereverItStands(t *testing.T) {
	t.Parallel()

	const description = `<p>A sealed vial of the first product.</p>`
	page := func(after string) string {
		return `<html><body><header><nav><a href="/">Home</a></nav></header><main><h1>First</h1><div>` + description + `</div>` +
			after + `</main><footer><a href="/legal/">Legal</a></footer></body></html>`
	}
	cases := []struct {
		name string
		in   render.Input
		want []render.Region
	}{
		{
			name: "related products after the description, each card linked twice and carrying its own add-to-cart",
			in: render.Input{
				Public: page(`<section><h2>Related</h2><ul><li><a href="/p/2/">Second</a> <a href="/p/2/">View</a> <a href="?add-to-cart=2">Add</a></li>` +
					`<li><a href="/p/3/">Third</a> <a href="/p/3/">View</a> <a href="/p/1/?add-to-cart=3">Add</a></li></ul></section>`),
				Filtered: description, Body: block.Parse(description),
			},
			want: []render.Region{{Kind: render.RegionListing, Inside: false, Links: 4, Words: 7, Signal: render.SignalLinks}},
		},
		{
			name: "the page's own add-to-cart links make no listing",
			in: render.Input{
				Public:   page(`<ul><li><a href="/p/1/?add-to-cart=1">Add</a></li><li><a href="/p/1/?add-to-cart=1">Add again</a></li><li><a href="/p/2/">Second</a></li></ul>`),
				Filtered: description, Body: block.Parse(description),
			},
			want: []render.Region{},
		},
		{
			name: "two links to two products are no listing",
			in: render.Input{
				Public:   page(`<ul><li><a href="/p/2/">Second</a></li><li><a href="/p/3/">Third</a></li></ul>`),
				Filtered: description, Body: block.Parse(description),
			},
			want: []render.Region{},
		},
		{
			name: "three links to one product are no listing",
			in: render.Input{
				Public:   page(`<div><a href="/p/2/">Second</a><a href="/p/2/">View</a><a href="/p/2/">Buy</a></div>`),
				Filtered: description, Body: block.Parse(description),
			},
			want: []render.Region{},
		},
		{
			name: "a menu of products stays navigation",
			in: render.Input{
				Public: `<html><body><header><nav><a href="/p/2/">Second</a><a href="/p/3/">Third</a><a href="/p/4/">Fourth</a></nav></header>` +
					`<main><div>` + description + `</div></main></body></html>`,
				Filtered: description, Body: block.Parse(description),
			},
			want: []render.Region{},
		},
		{
			name: "an archive's grid inside the assumed root",
			in: render.Input{
				Public:  page(`<ul><li><a href="/p/2/">Second</a> <a href="/p/2/">View</a></li><li><a href="/p/3/">Third</a> <a href="/p/3/">View</a></li></ul>`),
				Body:    block.Parse(""),
				Archive: true,
			},
			want: []render.Region{{Kind: render.RegionListing, Inside: true, Links: 4, Words: 4, Signal: render.SignalLinks}},
		},
		{
			name: "a grid of generated text inside the root",
			in: render.Input{
				Public: `<html><body><main><div>` + description +
					`<ul><li><a href="/p/2/">Second</a> <a href="/p/2/">View</a></li><li><a href="/p/3/">Third</a></li></ul></div></main></body></html>`,
				Filtered: description + `<ul><li><a href="/p/2/">Second</a> <a href="/p/2/">View</a></li><li><a href="/p/3/">Third</a></li></ul>`,
				Body:     block.Parse(description),
			},
			want: []render.Region{{Kind: render.RegionListing, Inside: true, Links: 3, Words: 3, Signal: render.SignalLinks}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.in.Document, tc.in.Lookup = "/p/1/", productsUnder("/p/")
			got := regionsOf(synthetic(t, tc.in), render.RegionListing)
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

func TestExtractLeavesOnlyTheGenuinelyUnknownLinks(t *testing.T) {
	t.Parallel()

	type link struct {
		href   string
		anchor string
		region render.RegionKind
	}
	const replicaSite = "http://localhost:8087"
	cases := []struct {
		name        string
		stand       string
		slug        string
		document    string
		archive     bool
		wantUnknown []link
		wantRel     int
		wantListing int
	}{
		{
			name: "the replica's post keeps its avatar link without a rel and the comments' external links", stand: replica, slug: "hello-world", document: "/hello-world/",
			wantUnknown: []link{
				{replicaSite + "/author/shopadmin/", "shopadmin", render.RegionUnknown},
				{"https://wordpress.org/", "A WordPress Commenter", render.RegionUnknown},
				{"https://gravatar.com/", "Gravatar", render.RegionUnknown},
			},
			wantRel: 2,
		},
		{
			name: "the replica's product: its category by rel, the related products a listing, its own links gone", stand: replica, slug: "bpc-157-peptide", document: "/product/bpc-157-peptide/",
			wantUnknown: []link{},
			wantRel:     3,
			wantListing: 5,
		},
		{
			name: "the replica's account keeps the link inside its dynamic form", stand: replica, slug: "my-account", document: "/my-account/",
			wantUnknown: []link{{replicaSite + "/my-account/lost-password/", "Lost your password?", render.RegionContentDynamic}},
		},
		{
			name: "the replica's cart lists what is new in store inside the assumed root, its own add-to-cart no link", stand: replica, slug: "cart", document: "/cart/",
			wantUnknown: []link{},
			wantListing: 6,
		},
		{
			name: "vanilla's term lists its products outside the root of its description", stand: vanilla, slug: "koffein", document: "/product-category/koffein/", archive: true,
			wantUnknown: []link{},
			wantListing: 6,
		},
		{
			name: "vanilla's shop lists its products inside the assumed root", stand: vanilla, slug: "shop", document: "/shop/", archive: true,
			wantUnknown: []link{},
			wantListing: 6,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			in := standInput(t, tc.stand, tc.slug, tc.document)
			in.Archive, in.Dynamic = tc.archive, technicalNames()
			got, err := render.Extract(in)
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			unknown := make([]link, 0)
			for _, l := range linksOfType(got, pagemap.LinkUnknown) {
				unknown = append(unknown, link{l.Href, l.Anchor, l.Region})
			}
			if !slices.Equal(unknown, tc.wantUnknown) {
				t.Errorf("unknown links %+v, want %+v", unknown, tc.wantUnknown)
			}
			rel := 0
			for _, region := range got.Regions {
				if region.Signal == render.SignalRel {
					rel += region.Links
				}
			}
			if rel != tc.wantRel {
				t.Errorf("%d links typed by their rel, want %d", rel, tc.wantRel)
			}
			if listed := len(linksOfType(got, pagemap.LinkListing)); listed != tc.wantListing {
				t.Errorf("%d listing links, want %d", listed, tc.wantListing)
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
