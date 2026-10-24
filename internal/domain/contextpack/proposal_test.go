package contextpack_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func storeWithoutEntities(t *testing.T) contextpack.Snapshot {
	t.Helper()

	home := onSitePage("home", "/", "Home", 1, pagemap.WPPage)
	hub := typed(mapped(under(onSitePage("hub", "/peptides/", "Peptides", 2, pagemap.WPPage), "home", pagemap.ParentJSONLD), "e-peptides"), pagetype.Hub)
	bpc := typed(mapped(under(onSitePage("bpc", "/bpc-157/", "BPC-157", 3, pagemap.WPPage), "hub", pagemap.ParentJSONLD), "e-bpc"), pagetype.Entity)
	term := typed(onSitePage("term", "/product-category/peptides/", "Peptides", 20, pagemap.WPProductCategory), pagetype.StoreCategory)
	product := func(id, path, name string, wpID int64, parent string) pagemap.Page {
		page := typed(onSitePage(id, path, name, wpID, pagemap.WPProduct), pagetype.Product)
		if parent != "" {
			page = under(page, parent, pagemap.ParentJSONLD)
		}
		return page
	}
	five := product("five", "/product/bpc-157-5mg/", "BPC-157 5 mg", 30, "term")
	kit := product("kit", "/product/bpc-kit/", "BPC-157 5 mg", 31, "term")
	carried := product("carried", "/product/bpc-157/", "BPC-157", 32, "term")
	sampler := product("sampler", "/product/peptides-sampler/", "Peptides", 33, "term")
	blends := typed(under(onSitePage("blends", "/blends/", "Blends", 5, pagemap.WPPage), "home", pagemap.ParentJSONLD), pagetype.Hub)
	blend := product("blend", "/product/blend-a/", "Blend A", 34, "blends")
	lonely := product("lonely", "/product/lonely/", "Lonely Peptide", 35, "")
	untitled := product("untitled", "/product/untitled/", "", 36, "")
	cart := onSitePage("cart", "/cart/", "Cart", 6, pagemap.WPPage)
	cart.TechnicalReason = pagemap.TechnicalCart
	planned := pagemap.Page{
		ID: "planned", SiteID: siteID, Path: "/peptides/guide/", Slug: "guide", WPType: pagemap.WPPage,
		Status: pagemap.StatusPlanned, Title: "Guide", CreatedAt: stamp, UpdatedAt: stamp,
	}

	f := fixture{
		pages: []pagemap.Page{home, hub, bpc, term, five, kit, carried, sampler, blends, blend, lonely, untitled, cart, planned},
		entities: []graph.Entity{
			{ID: "e-peptides", SiteID: siteID, Name: "Peptides", CanonicalPageID: new("hub"), Source: graph.SourceUser},
			{ID: "e-bpc", SiteID: siteID, Name: "BPC-157", ScopeID: new("e-peptides"), CanonicalPageID: new("bpc"), Source: graph.SourceUser},
			{ID: "e-lonely", SiteID: siteID, Name: "Lonely Peptide", Source: graph.SourceUser},
		},
		edges: []graph.Edge{parentEdge("g1", "e-bpc", "e-peptides")},
		fields: map[string][]field.Value{
			"five": values("five", "attributes",
				`[{"name":"Strength","options":["5 mg"],"visible":true,"variation":false},{"name":"Form","options":["Lyophilised powder","Vial"],"visible":true,"variation":true}]`),
		},
	}
	return f.snapshot(t)
}

func TestTheSiteProposesAnEntityForEveryPageThatNeedsOne(t *testing.T) {
	t.Parallel()

	proposals := storeWithoutEntities(t).Map(nil).EntityProposals(nil)
	paths := make([]string, 0, len(proposals))
	byPath := make(map[string]contextpack.EntityProposal, len(proposals))
	for i := range proposals {
		paths = append(paths, proposals[i].Path)
		byPath[proposals[i].Path] = proposals[i]
	}
	want := []string{
		"/product/lonely/", "/product/untitled/", "/blends/",
		"/product/blend-a/", "/product/bpc-157-5mg/", "/product/bpc-157/", "/product/bpc-kit/", "/product/peptides-sampler/",
	}
	if !slices.Equal(paths, want) {
		t.Fatalf("proposals = %v, want %v: by depth, then path; no root, store category, technical, mapped or planned page", paths, want)
	}

	cases := map[string]contextpack.EntityProposal{
		"/product/lonely/":   {Name: "Lonely Peptide", Label: "Lonely Peptide", ExistingEntityID: "e-lonely"},
		"/product/untitled/": {Clash: contextpack.ClashNoName},
		"/blends/":           {Name: "Blends", Label: "Blends"},
		"/product/blend-a/":  {Name: "Blend A", Label: "Blend A", Parent: "/blends/"},
		"/product/bpc-157-5mg/": {
			Name: "BPC-157 5 mg", Label: "BPC-157 5 mg", Parent: "/peptides/", ParentEntityID: "e-peptides",
			Facts: []graph.Fact{{Label: "Strength", Value: "5 mg"}, {Label: "Form", Value: "Lyophilised powder, Vial"}},
		},
		"/product/bpc-157/": {Name: "BPC-157", Label: "BPC-157", Parent: "/peptides/", ParentEntityID: "e-peptides", Clash: contextpack.ClashCarried},
		"/product/bpc-kit/": {
			Name: "BPC-157 5 mg", Label: "BPC-157 5 mg", Parent: "/peptides/", ParentEntityID: "e-peptides", Clash: contextpack.ClashInBatch,
		},
		"/product/peptides-sampler/": {Name: "Peptides", Label: "Peptides Peptides", Parent: "/peptides/", ParentEntityID: "e-peptides"},
	}
	for path, expected := range cases {
		got := byPath[path]
		expected.PageID, expected.Path = got.PageID, path
		if got.PageID == "" || got.Name != expected.Name || got.Label != expected.Label || got.Parent != expected.Parent ||
			got.ParentEntityID != expected.ParentEntityID || got.ExistingEntityID != expected.ExistingEntityID ||
			got.Clash != expected.Clash || !slices.Equal(got.Facts, expected.Facts) {
			t.Errorf("%s = %+v, want %+v", path, got, expected)
		}
	}
}

func TestTheSiteProposesOnlyForTheChosenPages(t *testing.T) {
	t.Parallel()

	proposals := storeWithoutEntities(t).Map(nil).EntityProposals([]string{"blend", "term", "five"})
	paths := make([]string, 0, len(proposals))
	for i := range proposals {
		paths = append(paths, proposals[i].Path)
	}
	if want := []string{"/product/blend-a/", "/product/bpc-157-5mg/"}; !slices.Equal(paths, want) {
		t.Fatalf("proposals = %v, want %v: a store category is never proposed for", paths, want)
	}
	if proposals[0].Parent != "" || proposals[0].ParentEntityID != "" {
		t.Errorf("a product whose parent page is not in the batch and carries no entity = %+v, want no parent", proposals[0])
	}
}
