package render_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func TestExtractTypesEveryLink(t *testing.T) {
	t.Parallel()

	got := extractStand(t, replica, "tb-500", "/tb-500/")
	cases := []struct {
		name       string
		href       string
		wantCount  int
		wantType   pagemap.LinkType
		wantRegion render.RegionKind
		wantPos    int
	}{
		{"the body's link to its sibling", "/bpc-157/", 1, pagemap.LinkContextual, render.RegionContent, 1},
		{"the crumb to the parent", "http://localhost:8087/healing-peptides/", 1, pagemap.LinkBreadcrumb, render.RegionBreadcrumb, -1},
		{"the crumb to the hub", "http://localhost:8087/peptides/", 3, "", "", -1},
		{"the listed product", "http://localhost:8087/product/tb-500-peptide/", 1, pagemap.LinkListing, render.RegionListing, -1},
		{"the header and drawer menus", "http://localhost:8087/cart/", 2, pagemap.LinkNavigation, render.RegionNavigation, -1},
		{"the menus' own link to the page is no page link", "http://localhost:8087/tb-500/", 0, "", "", -1},
		{"the footer's credit", "https://creativethemes.com", 1, pagemap.LinkNavigation, render.RegionNavigation, -1},
		{"the skip link is no page link", "#main", 0, "", "", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			matched := make([]render.Link, 0)
			for _, link := range got.Links {
				if link.Href == tc.href {
					matched = append(matched, link)
				}
			}
			if len(matched) != tc.wantCount {
				t.Fatalf("links to %s: %+v, want %d", tc.href, matched, tc.wantCount)
			}
			for _, link := range matched {
				if tc.wantType != "" && (link.Type != tc.wantType || link.Region != tc.wantRegion) {
					t.Errorf("%+v, want type %s in region %s", link, tc.wantType, tc.wantRegion)
				}
				if link.Position != tc.wantPos {
					t.Errorf("%+v, want position %d", link, tc.wantPos)
				}
			}
		})
	}

	byType := map[pagemap.LinkType]int{}
	for _, link := range got.Links {
		byType[link.Type]++
	}
	want := map[pagemap.LinkType]int{
		pagemap.LinkContextual: 1, pagemap.LinkBreadcrumb: 3, pagemap.LinkListing: 1,
		pagemap.LinkNavigation: 19, pagemap.LinkUnknown: 0,
	}
	for typ, count := range want {
		if byType[typ] != count {
			t.Errorf("%s links: %d, want %d", typ, byType[typ], count)
		}
	}
}

func TestExtractResolvesARelativeHrefAgainstTheDocument(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		document string
		href     string
		wantPath string
		wantKind pagemap.LinkKind
	}{
		{"a relative href under a nested page", "/x/y/", "a/", "/x/y/a/", pagemap.LinkPath},
		{"a document without its trailing slash", "/x/y", "a/", "/x/a/", pagemap.LinkPath},
		{"a root-relative href", "/x/y/", "/c/", "/c/", pagemap.LinkPath},
		{"an absolute href on the site", "/x/y/", "https://shop.test/d/", "/d/", pagemap.LinkPath},
		{"another host", "/x/y/", "https://other.test/d/", "", pagemap.LinkExternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			public := `<html><body><main><p>Read <a href="` + tc.href + `">this</a> now.</p></main></body></html>`
			got := synthetic(t, render.Input{Public: public, Body: block.Parse(""), Document: tc.document})
			if len(got.Links) != 1 {
				t.Fatalf("links %+v, want one", got.Links)
			}
			link := got.Links[0]
			if link.Path != tc.wantPath || link.Kind != tc.wantKind || link.Anchor != "this" || link.Href != tc.href {
				t.Errorf("%+v, want %s (%s)", link, tc.wantPath, tc.wantKind)
			}
		})
	}
}

func TestExtractEmitsNoLinkToTheDocumentItself(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		href string
	}{
		{"a fragment", "#comments"},
		{"a bare query", "?add-to-cart=14"},
		{"the page's own path with a fragment", "/x/y/#respond"},
		{"the page's own absolute address", "https://shop.test/x/y/"},
		{"the page's own path with a query", "/x/y/?add-to-cart=14"},
		{"the page's own path in capitals", "/X/Y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			public := `<html><body><header><a href="` + tc.href + `">Self</a><a href="/home/">Home</a></header>` +
				`<main><p>Read <a href="` + tc.href + `">this</a> and <a href="/other/">that</a>.</p></main></body></html>`
			got := synthetic(t, render.Input{Public: public, Body: block.Parse(""), Document: "/x/y/"})
			paths := make([]string, 0, len(got.Links))
			for _, link := range got.Links {
				paths = append(paths, link.Path)
			}
			if len(got.Links) != 2 || paths[0] != "/home/" || paths[1] != "/other/" {
				t.Errorf("links %+v, want only /home/ and /other/", got.Links)
			}
			links := 0
			for _, region := range got.Regions {
				links += region.Links
			}
			if links != 2 {
				t.Errorf("regions %+v count %d links, want the 2 emitted", got.Regions, links)
			}
		})
	}
}

func TestExtractNamesTheMajorityURLForm(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		anchors string
		want    string
	}{
		{"absolute hrefs win", `<a href="https://shop.test/a/">a</a><a href="//shop.test/b/">b</a><a href="/c/">c</a>`, "absolute"},
		{"relative hrefs win a tie", `<a href="https://shop.test/a/">a</a><a href="/c/">c</a>`, "relative"},
		{"external links do not vote", `<a href="https://other.test/a/">a</a><a href="https://other.test/b/">b</a><a href="c/">c</a>`, "relative"},
		{"no internal link", `<a href="https://other.test/a/">a</a><a href="#x">x</a>`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			public := `<html><body><main><p>` + tc.anchors + `</p></main></body></html>`
			got := synthetic(t, render.Input{Public: public, Body: block.Parse(""), Document: "/x/"})
			if got.URLForm != tc.want {
				t.Errorf("url form %q, want %q", got.URLForm, tc.want)
			}
		})
	}
}

func TestExtractNamesAnImageLinkByItsAccessibleName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		anchor string
		want   string
	}{
		{"the text", `<a href="/a/">  Plain   text </a>`, "Plain text"},
		{"the aria label of an image link", `<a href="/a/" aria-label="Named by its label"><img src="x.png" alt="Alt"></a>`, "Named by its label"},
		{"the alt of an image link", `<a href="/a/"><img src="x.png" alt="Alt text"></a>`, "Alt text"},
		{"nothing to name", `<a href="/a/"><img src="x.png"></a>`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, render.Input{Public: `<html><body><main>` + tc.anchor + `</main></body></html>`, Body: block.Parse("")})
			if len(got.Links) != 1 || got.Links[0].Anchor != tc.want {
				t.Errorf("links %+v, want the anchor %q", got.Links, tc.want)
			}
		})
	}
}
