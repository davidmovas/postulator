package contextpack_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
)

func titles[T any](rows []T, title func(T) string) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, title(row))
	}
	return out
}

func refTitle(ref contextpack.Ref) string { return ref.Title }

func productTitle(product contextpack.Product) string { return product.Title }

func targetPath(target contextpack.Target) string { return target.Path }

func TestASummaryFallsFromMetaToExcerptToBodyToStructure(t *testing.T) {
	t.Parallel()

	s := replicaLike().snapshot(t)
	cases := []struct {
		page   string
		text   string
		source contextpack.SummarySource
	}{
		{page: "guides", text: "Guides for handling research peptides.", source: contextpack.FromMeta},
		{page: "bpc-5", text: "Five milligrams of BPC-157, lyophilised.", source: contextpack.FromExcerpt},
		{page: "bpc", text: "BPC-157 is a peptide studied for tissue repair.", source: contextpack.FromBody},
		{page: "healing", text: "Hub for Healing Peptides under Peptides; lists BPC-157 5 mg, TB-500", source: contextpack.FromStructure},
		{page: "home", text: "Entity for Home; its children are Lab Guides, Peptides", source: contextpack.FromStructure},
	}
	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			t.Parallel()

			page, _ := s.Index.ByID(tc.page)
			if got := s.Summary(page); got.Text != tc.text || got.Source != tc.source {
				t.Errorf("Summary(%s) = %+v, want %q from %s", tc.page, got, tc.text, tc.source)
			}
		})
	}
}

func TestASummaryReadsAFieldsTextNotItsMarkup(t *testing.T) {
	t.Parallel()

	f := replicaLike()
	f.fields["bpc-5"] = values("bpc-5", "product_short_description", "<p>Single boiler, <strong>58&nbsp;mm</strong>.</p>\n<p>Second.</p>")
	f.fields["guides"] = values("guides", "seo_description", "Guides &amp; handling notes.")
	s := f.snapshot(t)
	cases := map[string]string{"bpc-5": "Single boiler, 58 mm. Second.", "guides": "Guides & handling notes."}
	for pageID, want := range cases {
		page, _ := s.Index.ByID(pageID)
		if got := s.Summary(page).Text; got != want {
			t.Errorf("Summary(%s) = %q, want %q", pageID, got, want)
		}
	}
}

func TestAnEmptyPageIsSummarisedFromItsStructure(t *testing.T) {
	t.Parallel()

	f := replicaLike()
	for i := range f.pages {
		if f.pages[i].ID == "bpc" {
			f.pages[i].ParentPageID = new("healing")
		}
	}
	s := f.snapshot(t)
	page, _ := s.Index.ByID("healing")
	want := "Hub for Healing Peptides under Peptides; lists BPC-157 5 mg, TB-500; its children are BPC-157"
	if got := s.Summary(page); got.Text != want || got.Source != contextpack.FromStructure {
		t.Errorf("Summary = %+v, want %q", got, want)
	}
}

func TestListedProductsComeInOneOrderWhateverOrderTheLinksWereStoredIn(t *testing.T) {
	t.Parallel()

	stored := replicaLike()
	reversed := replicaLike()
	reversed.links[3], reversed.links[4] = reversed.links[4], reversed.links[3]
	orders := make([][]string, 0, 2)
	for _, f := range []fixture{stored, reversed} {
		s := f.snapshot(t)
		facts, err := s.Facts("healing", planFor(t, s, "healing"))
		if err != nil {
			t.Fatalf("Facts: %v", err)
		}
		if facts.ListedFrom != "listing" || len(facts.Products) != 2 {
			t.Fatalf("products from %q = %d, want the healing hub's two listed products", facts.ListedFrom, len(facts.Products))
		}
		got := make([]string, 0, len(facts.Products))
		for _, product := range facts.Products {
			got = append(got, product.Path)
		}
		orders = append(orders, got)
	}
	if !slices.Equal(orders[0], orders[1]) {
		t.Errorf("products = %v stored one way and %v the other, want one order", orders[0], orders[1])
	}
}

func TestTheNeighboursAreTheContentBesideAPage(t *testing.T) {
	t.Parallel()

	f := replicaLike()
	cart := typed(under(onSitePage("cart", "/cart/", "Cart", 40, pagemap.WPPage), "home", pagemap.ParentURL), pagetype.Technical)
	cart.TechnicalReason = pagemap.TechnicalCart
	twin := under(f.pages[5], "home", pagemap.ParentURL)
	post := typed(under(onSitePage("post", "/hello-world/", "Hello world!", 41, pagemap.WPPost), "home", pagemap.ParentURL), pagetype.Entity)
	f.pages = append(f.pages[:5], f.pages[6:]...)
	f.pages = append(f.pages, cart, twin, post)
	s := f.snapshot(t)

	facts, err := s.Facts("hub", planFor(t, s, "hub"))
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	if got := titles(facts.Siblings, refTitle); !slices.Equal(got, []string{"Lab Guides", "Hello world!"}) &&
		!slices.Equal(got, []string{"Hello world!", "Lab Guides"}) {
		t.Errorf("siblings = %v, want the content beside the hub, never the cart or a store category", got)
	}
	home, err := s.Facts("home", planFor(t, s, "home"))
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	for _, child := range home.Children {
		if child.Preset == pagetype.Technical || child.Preset == pagetype.StoreCategory {
			t.Errorf("the front page's children hold %s (%s), want content only", child.Path, child.Preset)
		}
	}
	term, err := s.Facts("term", planFor(t, s, "term"))
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	for _, sibling := range term.Siblings {
		if sibling.Preset == pagetype.Technical {
			t.Errorf("a store category's siblings hold %s, want no technical page", sibling.Path)
		}
	}
}

func TestTheFactsOfAHub(t *testing.T) {
	t.Parallel()

	s := replicaLike().snapshot(t)
	facts, err := s.Facts("hub", planFor(t, s, "hub"))
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}

	if facts.Page.Type != "Hub" || facts.Page.Preset != "hub" || !facts.Page.OnSite {
		t.Errorf("page = %+v, want a hub on the site", facts.Page)
	}
	if got := titles(facts.Trail, refTitle); !slices.Equal(got, []string{"Home"}) {
		t.Errorf("trail = %v", got)
	}
	if got := titles(facts.Children, refTitle); !slices.Equal(got, []string{"Healing Peptides", "BPC-157"}) {
		t.Errorf("children = %v, want the one sharing the hub's keyword first", got)
	}
	if got := titles(facts.Siblings, refTitle); !slices.Equal(got, []string{"Lab Guides"}) {
		t.Errorf("siblings = %v", got)
	}
	if got := titles(facts.Targets, targetPath); !slices.Equal(got, []string{"/bpc-157/", "/healing-peptides/"}) {
		t.Errorf("targets = %v", got)
	}
	if facts.Targets[1].Summary.Source != contextpack.FromStructure || facts.Targets[1].Summary.Text == "" {
		t.Errorf("the empty child's summary = %+v, want one built from its structure", facts.Targets[1].Summary)
	}
	if facts.Card == nil || facts.Card.Definition == "" || len(facts.Card.Facts) != 1 {
		t.Errorf("card = %+v", facts.Card)
	}
	if len(facts.Keywords) != 2 || facts.Keywords[0].Text != "peptides" {
		t.Errorf("keywords = %+v, want them by volume", facts.Keywords)
	}
	if len(facts.Inbound) != 1 || facts.Inbound[0].Type != pagemap.LinkContextual || facts.Inbound[0].Page.Title != "BPC-157" {
		t.Errorf("inbound = %+v, want BPC-157's contextual link", facts.Inbound)
	}
	if len(facts.Outbound) != 3 || facts.Outbound[0].Type != pagemap.LinkListing {
		t.Errorf("outbound = %+v, want the three listing links", facts.Outbound)
	}
	if facts.Root != render.RootEmptyRaw || facts.Generated != 48 || len(facts.Regions) != 3 {
		t.Errorf("render = %s %d %d", facts.Root, facts.Generated, len(facts.Regions))
	}
	body, held := facts.Field("body")
	if !held || !body.Scope.Enabled || body.Scope.Instruction == "" || body.Current != "" {
		t.Errorf("body = %+v, want the enabled, empty body with its instruction", body)
	}
	if title, held := facts.Field("title"); !held || title.Current != "Peptides" {
		t.Errorf("title = %+v, want the observed title", title)
	}
	if _, held := facts.Field("product_description"); held {
		t.Error("a hub carries no product description")
	}
	if len(facts.Notes) != 1 {
		t.Errorf("notes = %+v", facts.Notes)
	}
}

func TestAHubListsItsTwinsProductsWhenItsOwnListingIsSmaller(t *testing.T) {
	t.Parallel()

	larger := replicaLike()
	larger.links = append(larger.links, listing("l9", "hub", "ghk"), listing("l10", "hub", "bpc-5"), listing("l11", "hub", "tb-500"))

	cases := []struct {
		name     string
		fixture  fixture
		from     string
		products []string
		twin     bool
	}{
		{
			name: "the twin's four products over the hub's three", fixture: replicaLike(), from: "twin", twin: true,
			products: []string{"BPC-157 10 mg", "BPC-157 5 mg", "GHK-Cu", "TB-500"},
		},
		{
			name: "the hub's own listing when it holds as many", fixture: larger, from: "listing", twin: true,
			products: []string{"BPC-157 10 mg", "BPC-157 5 mg", "GHK-Cu", "TB-500"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := tc.fixture.snapshot(t)
			facts, err := s.Facts("hub", planFor(t, s, "hub"))
			if err != nil {
				t.Fatalf("Facts: %v", err)
			}
			if got := titles(facts.Products, productTitle); facts.ListedFrom != tc.from || !slices.Equal(got, tc.products) {
				t.Errorf("products from %q = %v, want from %q %v", facts.ListedFrom, got, tc.from, tc.products)
			}
			if (facts.Twin != nil) != tc.twin || (facts.Twin != nil && facts.Twin.PageID != "term") {
				t.Errorf("twin = %+v", facts.Twin)
			}
			if facts.Products[0].Price == "" || facts.Products[0].SKU == "" {
				t.Errorf("product = %+v, want its facts", facts.Products[0])
			}
		})
	}

	s := replicaLike().snapshot(t)
	term, err := s.Facts("term", planFor(t, s, "term"))
	if err != nil {
		t.Fatalf("Facts(term): %v", err)
	}
	if term.ListedFrom != "term" || len(term.Products) != 4 || term.Twin == nil || term.Twin.PageID != "hub" {
		t.Errorf("the category lists %d from %q with twin %+v, want its four products and the hub", len(term.Products), term.ListedFrom, term.Twin)
	}
}

func TestTheNeighborhoodIsEveryPageThePackReads(t *testing.T) {
	t.Parallel()

	s := replicaLike().snapshot(t)
	got := s.Neighborhood("hub", planFor(t, s, "hub"))
	want := []string{"bpc", "bpc-10", "bpc-5", "ghk", "guides", "healing", "home", "hub", "tb-500", "term"}
	if !slices.Equal(got, want) {
		t.Errorf("Neighborhood = %v, want %v", got, want)
	}
	if got := s.Neighborhood("missing", planFor(t, s, "hub")); len(got) != 0 {
		t.Errorf("an unknown page has neighbors %v", got)
	}
}

func TestAnUnknownPageHasNoFacts(t *testing.T) {
	t.Parallel()

	s := replicaLike().snapshot(t)
	if _, err := s.Facts("missing", planFor(t, s, "hub")); err == nil {
		t.Error("Facts of an unknown page must fail")
	}
}

func TestOurLinksFollowTheSitesContextualFormInTheSnapshot(t *testing.T) {
	t.Parallel()

	stored := func(id, to, path string, absolute bool) pagemap.PageLink {
		link := contextual(id, "x", to, path, id)
		link.Absolute = absolute
		return link
	}
	relative := []pagemap.PageLink{
		stored("a", "y", "/bpc-157/", false), stored("b", "z", "/tb-500/", false), stored("c", "y", "/bpc-157/", true),
	}
	absolute := []pagemap.PageLink{stored("a", "y", "/bpc-157/", true), stored("b", "z", "/tb-500/", true)}
	ours := []pagemap.PageLink{stored("a", "y", "/bpc-157/", true), stored("b", "z", "/tb-500/", true)}
	for i := range ours {
		ours[i].Ours = true
	}
	navigation := []pagemap.PageLink{stored("a", "y", "/", true)}
	navigation[0].Type = pagemap.LinkNavigation

	cases := []struct {
		name    string
		setting site.URLForm
		links   []pagemap.PageLink
		renders []string
		want    site.URLForm
	}{
		{name: "the setting wins", setting: site.URLFormAbsolute, links: relative, want: site.URLFormAbsolute},
		{name: "the client's links are mostly relative", setting: site.URLFormAuto, links: relative, want: site.URLFormRelative},
		{name: "the client's links are absolute", setting: site.URLFormAuto, links: absolute, want: site.URLFormAbsolute},
		{name: "our own links are not the client's form", setting: site.URLFormAuto, links: ours, renders: []string{"relative"}, want: site.URLFormRelative},
		{name: "no contextual link falls to the theme", setting: site.URLFormAuto, links: navigation, renders: []string{"absolute", "absolute", "relative"}, want: site.URLFormAbsolute},
		{name: "nothing at all is relative", setting: site.URLFormAuto, want: site.URLFormRelative},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := contextpack.LinkForm(tc.setting, tc.links, tc.renders); got != tc.want {
				t.Errorf("LinkForm = %s, want %s", got, tc.want)
			}
		})
	}
}
