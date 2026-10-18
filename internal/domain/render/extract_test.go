package render_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func TestExtractFindsTheContentRootOnEveryStandPage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		stand    string
		slug     string
		document string
		archive  bool
		dynamic  []string
		want     render.RootState
	}{
		{"the replica's hub with an empty raw", replica, "peptides", "/peptides/", false, nil, render.RootEmptyRaw},
		{"the replica's hub whose raw is one shortcode", replica, "bpc-157", "/bpc-157/", false, nil, render.RootDynamicOnly},
		{"the replica's page of plain text", replica, "tb-500", "/tb-500/", false, nil, render.RootFound},
		{"the replica's product", replica, "bpc-157-peptide", "/product/bpc-157-peptide/", false, nil, render.RootFound},
		{"the replica's post", replica, "hello-world", "/hello-world/", false, technicalNames(), render.RootFound},
		{"the replica's term with no description", replica, "healing-peptides", "/product-category/peptides/healing-peptides/", true, technicalNames(), render.RootArchive},
		{"the replica's shop", replica, "shop", "/shop/", true, technicalNames(), render.RootArchive},
		{"the replica's shop not known as one has an empty raw", replica, "shop", "/shop/", false, technicalNames(), render.RootEmptyRaw},
		{"the replica's cart block counted technical", replica, "cart", "/cart/", false, technicalNames(), render.RootDynamicOnly},
		{"the replica's cart block not counted technical holds text the render never shows", replica, "cart", "/cart/", false, nil, render.RootNotFound},
		{"the replica's account shortcode", replica, "my-account", "/my-account/", false, technicalNames(), render.RootDynamicOnly},
		{"vanilla's guide", vanilla, "guide", "/guide/", false, nil, render.RootFound},
		{"vanilla's nested page", vanilla, "setup", "/guide/setup/", false, nil, render.RootFound},
		{"vanilla's product whose description repeats its excerpt", vanilla, "powder", "/product/powder/", false, nil, render.RootFound},
		{"vanilla's term read as a page of its description", vanilla, "koffein", "/product-category/koffein/", false, nil, render.RootFound},
		{"vanilla's term", vanilla, "koffein", "/product-category/koffein/", true, technicalNames(), render.RootArchive},
		{"vanilla's shop", vanilla, "shop", "/shop/", true, technicalNames(), render.RootArchive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			in := standInput(t, tc.stand, tc.slug, tc.document)
			in.Archive, in.Dynamic = tc.archive, tc.dynamic
			got, err := render.Extract(in)
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			if got.Root != tc.want {
				t.Errorf("root %q, want %q", got.Root, tc.want)
			}
			sum := sha256.Sum256([]byte(in.Public))
			if got.Hash != hex.EncodeToString(sum[:]) {
				t.Errorf("hash %q is not the sha256 of the public render", got.Hash)
			}
		})
	}
}

func TestExtractPlacesTheRootWhereTheContentsLinksAre(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		in       render.Input
		wantPath string
		wantPos  int
	}{
		{
			name: "the filtered text is found twice and the occurrence with its link wins",
			in: render.Input{
				Public:   `<html><body><main><p>See Koffein.</p><div><p>See <a href="/koffein/">Koffein</a>.</p></div></main></body></html>`,
				Filtered: `<p>See <a href="/koffein/">Koffein</a>.</p>`,
				Body:     block.Parse(`<p>See <a href="/koffein/">Koffein</a>.</p>`),
				Document: "/powder/",
			},
			wantPath: "/koffein/",
			wantPos:  0,
		},
		{
			name: "without the filtered content the raw blocks find the root",
			in: render.Input{
				Public:   `<html><body><header><a href="/">Home</a></header><main><h1>Title</h1><div><p>First block of text.</p><p>Second <a href="/b/">block</a> here.</p></div></main></body></html>`,
				Body:     block.Parse("First block of text.\n\nSecond <a href=\"/b/\">block</a> here."),
				Document: "/a/",
			},
			wantPath: "/b/",
			wantPos:  1,
		},
		{
			name: "a filtered block the theme renders otherwise still lets the others place the root",
			in: render.Input{
				Public: `<html><body><main><h1>Hub</h1><div><nav aria-label="Breadcrumb"><a href="/">Home</a> / Hub</nav>` +
					`<ul><li>Card one by the theme</li><li>Card two by the theme</li></ul><p>Read the <a href="/guide/">guide</a> first.</p></div></main></body></html>`,
				Filtered: `<div><nav aria-label="Breadcrumb"><a href="/">Home</a> / Hub</nav>` +
					`<ul><li>Card one</li><li>Card two</li></ul><p>Read the <a href="/guide/">guide</a> first.</p></div>`,
				Body:     block.Parse(`[cards]` + "\n\n" + `<p>Read the <a href="/guide/">guide</a> first.</p>`),
				Document: "/hub/",
			},
			wantPath: "/guide/",
			wantPos:  1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, tc.in)
			if got.Root != render.RootFound {
				t.Fatalf("root %q, want found", got.Root)
			}
			contextual := linksOfType(got, pagemap.LinkContextual)
			if len(contextual) != 1 {
				t.Fatalf("contextual links %+v, want one", contextual)
			}
			if contextual[0].Path != tc.wantPath || contextual[0].Position != tc.wantPos {
				t.Errorf("contextual link %+v, want %s at block %d", contextual[0], tc.wantPath, tc.wantPos)
			}
		})
	}
}

func TestExtractStatesTheRootTruthfully(t *testing.T) {
	t.Parallel()

	const page = `<html><body><header><a href="/">Shop</a></header><main><h1>Koffein</h1><div><p>Strong beans, ground fine.</p></div>` +
		`<form><p>Showing all results</p></form></main></body></html>`
	cases := []struct {
		name          string
		in            render.Input
		want          render.RootState
		wantContent   int
		wantGenerated int
	}{
		{
			name: "an archive keeps its root where its description landed",
			in: render.Input{
				Public: page, Filtered: `<p>Strong beans, ground fine.</p>`, Body: block.Parse(`<p>Strong beans, ground fine.</p>`),
				Archive: true,
			},
			want: render.RootArchive, wantContent: 4, wantGenerated: 0,
		},
		{
			name:          "an archive without a description assumes the landmark",
			in:            render.Input{Public: page, Body: block.Parse(""), Archive: true},
			want:          render.RootArchive,
			wantGenerated: 8,
		},
		{
			name:          "an empty raw whose root cannot be located",
			in:            render.Input{Public: page, Body: block.Parse("")},
			want:          render.RootEmptyRaw,
			wantGenerated: 8,
		},
		{
			name:          "a raw of one shortcode whose expansion cannot be located",
			in:            render.Input{Public: page, Filtered: `<div><form><p>Log in</p></form></div>`, Body: block.Parse(`[account_form]`)},
			want:          render.RootDynamicOnly,
			wantGenerated: 8,
		},
		{
			name: "a block the site counts technical holds text the render never shows",
			in: render.Input{
				Public:  page,
				Body:    block.Parse(`<!-- wp:shop/basket --><div><h2>Your basket is empty</h2></div><!-- /wp:shop/basket -->`),
				Dynamic: []string{"shop/basket"},
			},
			want:          render.RootDynamicOnly,
			wantGenerated: 8,
		},
		{
			name: "the same block the site does not count technical is text that cannot be found",
			in: render.Input{
				Public: page,
				Body:   block.Parse(`<!-- wp:shop/basket --><div><h2>Your basket is empty</h2></div><!-- /wp:shop/basket -->`),
			},
			want:          render.RootNotFound,
			wantGenerated: 8,
		},
		{
			name: "a paragraph holding only a shortcode the site counts technical",
			in: render.Input{
				Public:  page,
				Body:    block.Parse(`<!-- wp:paragraph --><p>[account_form]</p><!-- /wp:paragraph -->`),
				Dynamic: []string{"account_form"},
			},
			want:          render.RootDynamicOnly,
			wantGenerated: 8,
		},
		{
			name: "a technical block whose text is found is still dynamic",
			in: render.Input{
				Public:   page,
				Filtered: `<div><p>Strong beans, ground fine.</p></div>`,
				Body:     block.Parse(`<!-- wp:shop/notice --><div><p>Strong beans, ground fine.</p></div><!-- /wp:shop/notice -->`),
				Dynamic:  []string{"shop/notice"},
			},
			want:          render.RootDynamicOnly,
			wantGenerated: 4,
		},
		{
			name:          "text that cannot be found",
			in:            render.Input{Public: page, Body: block.Parse(`<p>Words the page never shows.</p>`)},
			want:          render.RootNotFound,
			wantGenerated: 8,
		},
		{
			name:        "text that is found",
			in:          render.Input{Public: page, Body: block.Parse(`<p>Strong beans, ground fine.</p>`)},
			want:        render.RootFound,
			wantContent: 4,
		},
		{
			name:          "an empty name in the technical list names no block",
			in:            render.Input{Public: page, Body: block.Parse(`<p>Strong beans, ground fine.</p>`), Dynamic: []string{""}},
			want:          render.RootFound,
			wantContent:   4,
			wantGenerated: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.in.Document = "/koffein/"
			got := synthetic(t, tc.in)
			if got.Root != tc.want {
				t.Errorf("root %q, want %q", got.Root, tc.want)
			}
			if got.WordsContent != tc.wantContent || got.WordsGenerated != tc.wantGenerated {
				t.Errorf("content %d and generated %d words, want %d and %d", got.WordsContent, got.WordsGenerated, tc.wantContent, tc.wantGenerated)
			}
		})
	}
}

func TestBodyStateIsTheStateABodyAloneGives(t *testing.T) {
	t.Parallel()

	const basket = `<!-- wp:shop/basket --><div><h2>Your basket is empty</h2></div><!-- /wp:shop/basket -->`
	cases := []struct {
		name    string
		raw     string
		archive bool
		dynamic []string
		want    render.RootState
	}{
		{name: "an archive whatever its body", raw: "<p>A description.</p>", archive: true, want: render.RootArchive},
		{name: "an empty body", raw: "", want: render.RootEmptyRaw},
		{name: "a body of one shortcode", raw: "[account_form]", want: render.RootDynamicOnly},
		{name: "a block the site counts technical", raw: basket, dynamic: []string{"shop/basket"}, want: render.RootDynamicOnly},
		{name: "the same block the site does not count technical", raw: basket, want: render.RootNotFound},
		{name: "a paragraph holding only a technical shortcode", raw: `<!-- wp:paragraph --><p>[account_form]</p><!-- /wp:paragraph -->`, dynamic: []string{"account_form"}, want: render.RootDynamicOnly},
		{name: "a body of text", raw: "<p>The page's own words.</p>", dynamic: []string{"account_form"}, want: render.RootNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := block.Parse(tc.raw)
			if got := render.BodyState(body, tc.archive, tc.dynamic); got != tc.want {
				t.Errorf("BodyState = %q, want %q", got, tc.want)
			}
			extracted := synthetic(t, render.Input{
				Public: `<html><body><main><p>Words of another page.</p></main></body></html>`,
				Body:   body, Archive: tc.archive, Dynamic: tc.dynamic, Document: "/a/",
			})
			if extracted.Root != tc.want {
				t.Errorf("an extraction whose root is not located = %q, want the body's %q", extracted.Root, tc.want)
			}
		})
	}
}

func TestARecordKeepsWhereItsVisitLanded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		final string
		want  string
	}{
		{"a visit that moved", "/cart/", `"finalPath":"/cart/"`},
		{"a visit that stayed", "", `"finalPath":""`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(render.Record{PageID: "p", FinalPath: tc.final})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(encoded), tc.want) {
				t.Errorf("%s, want it to carry %s", encoded, tc.want)
			}
		})
	}
}

func TestExtractSurvivesAnErrorPage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		public       string
		raw          string
		wantRoot     render.RootState
		wantLinks    int
		wantHeadings int
	}{
		{"a forbidden page instead of the page's text", readGolden(t, "errors", "forbidden.html"), "<p>The page's own words.</p>", render.RootNotFound, 1, 1},
		{"a forbidden page instead of an empty page", readGolden(t, "errors", "forbidden.html"), "", render.RootEmptyRaw, 1, 1},
		{"an empty answer instead of the page's text", "", "<p>The page's own words.</p>", render.RootNotFound, 0, 0},
		{"an empty answer instead of an empty page", "", "", render.RootEmptyRaw, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, render.Input{Public: tc.public, Body: block.Parse(tc.raw), Document: "/a/"})
			if got.Root != tc.wantRoot {
				t.Errorf("root %q, want %q", got.Root, tc.wantRoot)
			}
			if len(got.Trails) != 0 || len(got.JSONLD) != 0 {
				t.Errorf("trails %v, json-ld %v, want none", got.Trails, got.JSONLD)
			}
			if len(got.Links) != tc.wantLinks || len(got.Headings) != tc.wantHeadings {
				t.Errorf("links %+v and headings %+v, want %d and %d", got.Links, got.Headings, tc.wantLinks, tc.wantHeadings)
			}
			if got.WordsContent != 0 || got.Builder != "" || got.Robots != "" {
				t.Errorf("content words %d, builder %q, robots %q, want nothing", got.WordsContent, got.Builder, got.Robots)
			}
			if got.Trails == nil || got.Links == nil || got.Regions == nil || got.Headings == nil || got.JSONLD == nil {
				t.Error("an extraction keeps empty collections, never nil ones")
			}
		})
	}
}

func TestExtractKeepsTheJSONLDByType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		stand    string
		slug     string
		document string
		want     map[string]int
	}{
		{
			name: "the replica's hub", stand: replica, slug: "bpc-157", document: "/bpc-157/",
			want: map[string]int{
				"BreadcrumbList": 2, "ItemList": 1, "CollectionPage": 1, "WebPage": 1,
				"Person": 2, "Organization": 1, "WebSite": 1, "Article": 1,
			},
		},
		{
			name: "vanilla's product", stand: vanilla, slug: "powder", document: "/product/powder/",
			want: map[string]int{"BreadcrumbList": 1, "Product": 1},
		},
		{
			name: "vanilla's guide", stand: vanilla, slug: "guide", document: "/guide/",
			want: map[string]int{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := extractStand(t, tc.stand, tc.slug, tc.document)
			if len(got.JSONLD) != len(tc.want) {
				t.Errorf("types %v, want %v", typeCounts(got.JSONLD), tc.want)
			}
			for typ, count := range tc.want {
				if len(got.JSONLD[typ]) != count {
					t.Errorf("%s: %d objects, want %d", typ, len(got.JSONLD[typ]), count)
				}
				for _, raw := range got.JSONLD[typ] {
					var object map[string]json.RawMessage
					if err := json.Unmarshal(raw, &object); err != nil {
						t.Errorf("%s: %v", typ, err)
					}
					if _, ok := object["@type"]; !ok {
						t.Errorf("%s: the object as found lost its @type: %s", typ, raw)
					}
				}
			}
		})
	}
}

func TestExtractReadsJSONLDInEveryShape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		script string
		want   map[string]int
	}{
		{"an array of objects", `[{"@type":"WebPage"},{"@type":"Product"}]`, map[string]int{"WebPage": 1, "Product": 1}},
		{"a graph inside an array", `[{"@graph":[{"@type":"WebPage"}]}]`, map[string]int{"WebPage": 1}},
		{"an array @type", `{"@type":["Person","Organization"]}`, map[string]int{"Person": 1, "Organization": 1}},
		{"a broken script", `{"@type":`, map[string]int{}},
		{"an object without a type", `{"name":"nothing"}`, map[string]int{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			public := `<html><head><script type="application/ld+json">` + tc.script + `</script></head><body><p>x</p></body></html>`
			got := synthetic(t, render.Input{Public: public, Body: block.Parse("")})
			counts := typeCounts(got.JSONLD)
			if len(counts) != len(tc.want) {
				t.Fatalf("types %v, want %v", counts, tc.want)
			}
			for typ, count := range tc.want {
				if counts[typ] != count {
					t.Errorf("%s: %d, want %d", typ, counts[typ], count)
				}
			}
		})
	}
}

func typeCounts(jsonld map[string][]json.RawMessage) map[string]int {
	out := make(map[string]int, len(jsonld))
	for typ, objects := range jsonld {
		out[typ] = len(objects)
	}
	return out
}

func linksOfType(e render.Extraction, typ pagemap.LinkType) []render.Link {
	out := make([]render.Link, 0)
	for _, link := range e.Links {
		if link.Type == typ {
			out = append(out, link)
		}
	}
	return out
}
