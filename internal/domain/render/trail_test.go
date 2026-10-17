package render_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func trail(source pagemap.TrailSource, crumbs ...string) pagemap.Trail {
	out := pagemap.Trail{Source: source, Crumbs: make([]pagemap.Crumb, 0, len(crumbs)/2)}
	for i := 0; i+1 < len(crumbs); i += 2 {
		out.Crumbs = append(out.Crumbs, pagemap.Crumb{Name: crumbs[i], Path: crumbs[i+1]})
	}
	return out
}

func TestExtractReadsEveryBreadcrumbShape(t *testing.T) {
	t.Parallel()

	const (
		healing = "/product-category/peptides/healing-peptides/"
		koffein = "/product-category/koffein/"
	)
	cases := []struct {
		name     string
		stand    string
		slug     string
		document string
		want     []pagemap.Trail
	}{
		{
			name: "the replica's hub: Rank Math's list, the fixture's list and its aria nav", stand: replica, slug: "bpc-157", document: "/bpc-157/",
			want: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailJSONLD, "Home", "/", "Peptides", "/peptides/", "Healing Peptides", "/healing-peptides/", "BPC-157", "/bpc-157/"),
				trail(pagemap.TrailNav, "Home", "/", "Peptides", "/peptides/", "Healing Peptides", "/healing-peptides/", "BPC-157", "/bpc-157/"),
			},
		},
		{
			name: "the replica's product: Rank Math's list and the theme's microdata nav", stand: replica, slug: "bpc-157-peptide", document: "/product/bpc-157-peptide/",
			want: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Healing Peptides", healing, "BPC-157 Peptide", "/product/bpc-157-peptide/"),
				trail(pagemap.TrailNav, "Home", "/", "Peptides", "/product-category/peptides/", "Healing Peptides", healing, "BPC-157 Peptide", "/product/bpc-157-peptide/"),
			},
		},
		{
			name: "vanilla's product: WooCommerce's list of item objects and its list-less nav", stand: vanilla, slug: "powder", document: "/product/powder/",
			want: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Koffein", koffein, "Powder", "/product/powder/"),
				trail(pagemap.TrailNav, "Home", "/", "Koffein", koffein, "Powder", "/product/powder/"),
			},
		},
		{
			name: "vanilla's term", stand: vanilla, slug: "koffein", document: koffein,
			want: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Koffein", koffein),
				trail(pagemap.TrailNav, "Home", "/", "Koffein", koffein),
			},
		},
		{
			name: "vanilla's guide carries no breadcrumb", stand: vanilla, slug: "guide", document: "/guide/",
			want: []pagemap.Trail{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := extractStand(t, tc.stand, tc.slug, tc.document)
			assertTrails(t, got.Trails, tc.want)
		})
	}
}

func TestExtractReadsSyntheticBreadcrumbs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		public string
		want   []pagemap.Trail
	}{
		{
			name: "positions out of order and as strings, a last crumb without a url, a url on the list item",
			public: `<html><head><script type="application/ld+json">{"@type":["BreadcrumbList","Thing"],"itemListElement":[
				{"@type":"ListItem","position":"3","name":"Shoes"},
				{"@type":"ListItem","position":1,"item":{"@type":"WebPage","@id":"_:home","url":"https://shop.test/","name":"Home"}},
				{"@type":"ListItem","position":"2","name":"Men &amp; Boys","url":"https://shop.test/men/"},
				{"@type":"ListItem","position":"4","name":"Elsewhere","item":"https://other.test/x/"}
			]}</script></head><body><main><p>x</p></main></body></html>`,
			want: []pagemap.Trail{
				trail(pagemap.TrailJSONLD, "Home", "/", "Men & Boys", "/men/", "Shoes", "", "Elsewhere", ""),
			},
		},
		{
			name: "a yoast list whose last crumb names the page without a url",
			public: `<html><head><script type="application/ld+json">{"@type":"BreadcrumbList","itemListElement":[
				{"@type":"ListItem","position":1,"name":"Home","item":"https://shop.test/"},
				{"@type":"ListItem","position":2,"name":"Shoes"}
			]}</script></head><body><main><p>x</p></main></body></html>`,
			want: []pagemap.Trail{trail(pagemap.TrailJSONLD, "Home", "/", "Shoes", "/men/shoes/")},
		},
		{
			name: "a list nav with a separator item and an unlinked middle crumb",
			public: `<html><body><nav aria-label="You are here: BREADCRUMBS"><ul><li><a href="/">Home</a></li><li aria-hidden="true">»</li>
				<li>Men</li><li><span aria-current="page">Shoes</span></li></ul></nav><main><p>x</p></main></body></html>`,
			want: []pagemap.Trail{trail(pagemap.TrailNav, "Home", "/", "Men", "", "Shoes", "/men/shoes/")},
		},
		{
			name: "a list nav whose current crumb is marked but not last",
			public: `<html><body><nav aria-label="breadcrumb"><ol><li><a href="/">Home</a> ›</li><li aria-current="page">Shoes</li>
				<li><a href="https://other.test/">Partner</a></li></ol></nav><main><p>x</p></main></body></html>`,
			want: []pagemap.Trail{trail(pagemap.TrailNav, "Home", "/", "Shoes", "/men/shoes/", "Partner", "")},
		},
		{
			name: "a list-less nav whose last crumb follows two separators",
			public: `<html><body><nav role="navigation" aria-label="Breadcrumb"><span><a href="/">Home</a></span> » <a href="/men/">Men</a>
				<span> » </span><span>Shoes &amp; Boots</span></nav><main><p>x</p></main></body></html>`,
			want: []pagemap.Trail{trail(pagemap.TrailNav, "Home", "/", "Men", "/men/", "Shoes & Boots", "/men/shoes/")},
		},
		{
			name: "a microdata nav whose last item is a link",
			public: `<html><body><nav itemscope itemtype="https://schema.org/BreadcrumbList"><a href="/">Home</a> / <a href="/men/shoes/">Shoes</a> /
				</nav><main><p>x</p></main></body></html>`,
			want: []pagemap.Trail{trail(pagemap.TrailNav, "Home", "/", "Shoes", "/men/shoes/")},
		},
		{
			name:   "a nav that is not a breadcrumb gives no trail",
			public: `<html><body><nav aria-label="Primary"><a href="/">Home</a> / <a href="/men/">Men</a></nav><main><p>x</p></main></body></html>`,
			want:   []pagemap.Trail{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := synthetic(t, render.Input{Public: tc.public, Body: block.Parse(""), Document: "/men/shoes/"})
			assertTrails(t, got.Trails, tc.want)
		})
	}
}

func assertTrails(t *testing.T, got, want []pagemap.Trail) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("trails %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i].Source != want[i].Source || !slices.Equal(got[i].Crumbs, want[i].Crumbs) {
			t.Errorf("trail %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}
