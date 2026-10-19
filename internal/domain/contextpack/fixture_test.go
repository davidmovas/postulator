package contextpack_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
)

const siteID = "site-1"

var stamp = time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)

func volume(text string, monthly int) keyword.Keyword {
	return keyword.Keyword{Text: text, Volume: &monthly}
}

type fixture struct {
	pages    []pagemap.Page
	entities []graph.Entity
	edges    []graph.Edge
	links    []pagemap.PageLink
	terms    map[string][]pagemap.Term
	fields   map[string][]field.Value
	bodies   map[string]block.Snapshot
	renders  map[string]render.Record
}

func typeIDOf(preset pagetype.Preset) string {
	return "type-" + string(preset)
}

func types() pagetype.Set {
	presets := pagetype.Presets()
	kinds := make([]pagetype.Type, 0, len(presets))
	scopes := make([]field.Scope, 0, len(presets)*4)
	for _, preset := range presets {
		id := typeIDOf(preset)
		kinds = append(kinds, pagetype.Type{ID: id, SiteID: siteID, Name: preset.DefaultName(), Preset: preset, CreatedAt: stamp, UpdatedAt: stamp})
		scopes = append(scopes, pagetype.DefaultScopes(preset, id)...)
	}
	for i := range scopes {
		if scopes[i].PageTypeID == typeIDOf(pagetype.Hub) && scopes[i].Key == field.Body {
			scopes[i].Instruction = "Introduce the category and send the reader to its products."
			scopes[i].MinLength, scopes[i].MaxLength = 1200, 6000
		}
	}
	return pagetype.NewSet(kinds, scopes)
}

func typed(page pagemap.Page, preset pagetype.Preset) pagemap.Page {
	page.PageTypeID = new(typeIDOf(preset))
	page.TypeSource, page.TypeReason = pagemap.TypeSourceAuto, pagemap.TypeByListing
	return page
}

func onSitePage(id, path, title string, wpID int64, wpType pagemap.WPType) pagemap.Page {
	return pagemap.Page{
		ID: id, SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: wpType, WPID: new(wpID),
		Status: pagemap.StatusPublished, Observed: pagemap.Observed{Title: title, H1: title}, CreatedAt: stamp, UpdatedAt: stamp,
	}
}

func under(page pagemap.Page, parent string, reason pagemap.ParentReason) pagemap.Page {
	page.ParentPageID, page.ParentReason = new(parent), reason
	return page
}

func mapped(page pagemap.Page, entityID string) pagemap.Page {
	page.EntityID = new(entityID)
	return page
}

func listing(id, from, to string) pagemap.PageLink {
	return pagemap.PageLink{ID: id, SiteID: siteID, FromPageID: from, ToPageID: new(to), ToURL: "/" + to + "/", Type: pagemap.LinkListing, Source: pagemap.SourceRender, ObservedAt: stamp}
}

func contextual(id, from, to, href, anchor string) pagemap.PageLink {
	return pagemap.PageLink{ID: id, SiteID: siteID, FromPageID: from, ToPageID: new(to), ToURL: href, AnchorText: anchor, Type: pagemap.LinkContextual, Source: pagemap.SourceRaw, ObservedAt: stamp}
}

func values(pageID string, pairs ...string) []field.Value {
	out := make([]field.Value, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, field.NewValue(pageID, field.Key(pairs[i]), pairs[i+1], stamp))
	}
	return out
}

func catalogTerm(termID int64) []pagemap.Term {
	return []pagemap.Term{{Taxonomy: "product_cat", TermID: termID, Name: "Peptides", Slug: "peptides"}}
}

func parentEdge(id, child, parent string) graph.Edge {
	return graph.Edge{ID: id, SiteID: siteID, FromEntityID: child, ToEntityID: parent, Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved}
}

func replicaLike() fixture {
	home := onSitePage("home", "/", "Home", 1, pagemap.WPPage)
	hub := typed(mapped(under(onSitePage("hub", "/peptides/", "Peptides", 2, pagemap.WPPage), "home", pagemap.ParentJSONLD), "e-peptides"), pagetype.Hub)
	hub.Keywords = keyword.New([]keyword.Keyword{volume("research peptides", 3000), volume("peptides", 12000)})
	hub.Notes = []note.Note{{Label: "Tone", Text: "Plain and factual, for laboratory buyers."}}
	bpc := typed(mapped(under(onSitePage("bpc", "/bpc-157/", "BPC-157", 3, pagemap.WPPage), "hub", pagemap.ParentJSONLD), "e-bpc"), pagetype.Entity)
	healing := typed(mapped(under(onSitePage("healing", "/healing-peptides/", "Healing Peptides", 4, pagemap.WPPage), "hub", pagemap.ParentJSONLD), "e-healing"), pagetype.Hub)
	guides := typed(under(onSitePage("guides", "/lab-guides/", "Lab Guides", 5, pagemap.WPPage), "home", pagemap.ParentJSONLD), pagetype.Entity)
	term := typed(onSitePage("term", "/product-category/peptides/", "Peptides", 20, pagemap.WPProductCategory), pagetype.StoreCategory)
	five := typed(onSitePage("bpc-5", "/product/bpc-157-5mg/", "BPC-157 5 mg", 30, pagemap.WPProduct), pagetype.Product)
	ten := typed(onSitePage("bpc-10", "/product/bpc-157-10mg/", "BPC-157 10 mg", 31, pagemap.WPProduct), pagetype.Product)
	tb := typed(onSitePage("tb-500", "/product/tb-500/", "TB-500", 32, pagemap.WPProduct), pagetype.Product)
	ghk := typed(onSitePage("ghk", "/product/ghk-cu/", "GHK-Cu", 33, pagemap.WPProduct), pagetype.Product)

	return fixture{
		pages: []pagemap.Page{home, hub, bpc, healing, guides, term, five, ten, tb, ghk},
		entities: []graph.Entity{
			{ID: "e-peptides", SiteID: siteID, Name: "Peptides", CanonicalPageID: new("hub"), Source: graph.SourceUser, Score: 1,
				Definition: "Short chains of amino acids sold for laboratory research.",
				Facts:      []graph.Fact{{Label: "Storage", Value: "Frozen, sealed under nitrogen"}},
				Anchors:    []graph.Anchor{{Text: "research peptides", Source: graph.AnchorUser, Weight: 1}}},
			{ID: "e-bpc", SiteID: siteID, Name: "BPC-157", CanonicalPageID: new("bpc"), Source: graph.SourceUser, Score: 0.6,
				Anchors: []graph.Anchor{{Text: "BPC-157", Source: graph.AnchorUser, Weight: 1}}},
			{ID: "e-healing", SiteID: siteID, Name: "Healing Peptides", CanonicalPageID: new("healing"), Source: graph.SourceUser, Score: 0.5,
				Anchors: []graph.Anchor{{Text: "healing peptides", Source: graph.AnchorUser, Weight: 1}}},
		},
		edges: []graph.Edge{parentEdge("g1", "e-bpc", "e-peptides"), parentEdge("g2", "e-healing", "e-peptides")},
		links: []pagemap.PageLink{
			listing("l1", "hub", "bpc-5"), listing("l2", "hub", "bpc-10"), listing("l3", "hub", "tb-500"),
			listing("l4", "healing", "bpc-5"), listing("l5", "healing", "tb-500"),
			contextual("c1", "bpc", "hub", "/peptides/", "peptides"),
			contextual("c2", "guides", "bpc", "/bpc-157/", "BPC-157"),
		},
		terms: map[string][]pagemap.Term{
			"bpc-5": catalogTerm(20), "bpc-10": catalogTerm(20), "tb-500": catalogTerm(20), "ghk": catalogTerm(20),
		},
		fields: map[string][]field.Value{
			"bpc-5":   values("bpc-5", "price", "45.00", "sku", "BPC-5", "stock_status", "instock", "product_short_description", "Five milligrams of BPC-157, lyophilised."),
			"bpc-10":  values("bpc-10", "price", "80.00", "sku", "BPC-10", "stock_status", "instock"),
			"tb-500":  values("tb-500", "price", "60.00", "sku", "TB-5", "stock_status", "outofstock"),
			"ghk":     values("ghk", "price", "30.00", "sku", "GHK-50"),
			"hub":     values("hub", "title", "Peptides", "seo_title", "Research peptides | Lab"),
			"guides":  values("guides", "seo_description", "Guides for handling research peptides."),
			"healing": values("healing", "title", "Healing Peptides"),
		},
		bodies: map[string]block.Snapshot{
			"hub":     {PageID: "hub", Side: block.Current, Raw: "", Kind: block.Classic, ObservedAt: stamp},
			"healing": {PageID: "healing", Side: block.Current, Raw: "[products limit=\"4\"]", Kind: block.Classic, ObservedAt: stamp},
			"bpc":     {PageID: "bpc", Side: block.Current, Raw: "<p>BPC-157 is a peptide studied for tissue repair. It is sold for research.</p>", Kind: block.Classic, ObservedAt: stamp},
		},
		renders: map[string]render.Record{
			"hub": {PageID: "hub", ExtractedAt: stamp, Extraction: render.Extraction{
				Root: render.RootEmptyRaw, WordsGenerated: 48,
				Regions: []render.Region{
					{Kind: render.RegionBreadcrumb, Inside: true, Links: 1, Signal: render.SignalARIA},
					{Kind: render.RegionListing, Inside: true, Links: 3, Signal: render.SignalJSONLD},
					{Kind: render.RegionNavigation, Links: 9, Signal: render.SignalLandmark},
				},
			}},
		},
	}
}

func (f fixture) snapshot(t *testing.T) contextpack.Snapshot {
	t.Helper()

	g, err := graph.New(f.entities, f.edges)
	if err != nil {
		t.Fatalf("graph.New: %v", err)
	}
	return contextpack.Snapshot{
		Site: pagemap.NewSite("https://shop.example.com"), Index: pagemap.NewIndex(f.pages), Types: types(), Graph: g,
		Terms: f.terms, Fields: f.fields, Bodies: f.bodies, Renders: f.renders, Links: f.links,
	}
}

func planFor(t *testing.T, s contextpack.Snapshot, pageID string) content.LinkContext {
	t.Helper()

	page, found := s.Index.ByID(pageID)
	if !found {
		t.Fatalf("no page %s", pageID)
	}
	subject := content.Subject{Site: s.Site, PageID: page.ID, PagePath: page.Path}
	if page.EntityID != nil {
		subject.EntityID = *page.EntityID
	}
	return content.PlanLinks(s.Graph, s.Index, subject, s.Types.Policy(page)).Context
}
