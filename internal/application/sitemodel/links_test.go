package sitemodel_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func aSection() *sitemodeltest.Memory {
	contextual := func(id, from, to, anchor string, ours bool) pagemap.PageLink {
		return pagemap.PageLink{
			ID: id, SiteID: siteID, FromPageID: from, ToPageID: new(to), ToURL: "/" + to + "/", AnchorText: anchor,
			Type: pagemap.LinkContextual, Source: pagemap.SourceRaw, Ours: ours,
		}
	}
	audit := func(id, path string, missing int, orphan bool) reports.PageAudit {
		return reports.PageAudit{PageID: id, Path: path, OnSite: true, Required: 1, MissingRequired: missing, Orphan: orphan}
	}
	return &sitemodeltest.Memory{
		Site: site.Site{ID: siteID, Name: "Shop", BaseURL: "https://shop.example.com", Settings: site.DefaultSettings()},
		Pages: []pagemap.Page{
			sitePage("home", "/", "Home", 1, pagemap.WPPage, pagetype.Entity, ""),
			carrying(sitePage("serums", "/serums/", "Serums", 2, pagemap.WPPage, pagetype.Hub, "home"), "e-serums"),
			carrying(sitePage("vitc", "/serums/vitamin-c/", "Vitamin C", 3, pagemap.WPPage, pagetype.Entity, "serums"), "e-vitc"),
			carrying(sitePage("retinol", "/serums/retinol/", "Retinol", 4, pagemap.WPPage, pagetype.Entity, "serums"), "e-retinol"),
			carrying(sitePage("oils", "/oils/", "Oils", 5, pagemap.WPPage, pagetype.Hub, "home"), "e-oils"),
		},
		Entities: []graph.Entity{
			graphEntity("e-serums", "Serums", "serums"), graphEntity("e-vitc", "Vitamin C", "vitc"),
			graphEntity("e-retinol", "Retinol", "retinol"), graphEntity("e-oils", "Oils", "oils"),
		},
		Edges: []graph.Edge{edge("p-vitc", "e-vitc", "e-serums")},
		Types: presetSet(),
		Links: []pagemap.PageLink{
			contextual("l1", "vitc", "serums", "serums", false),
			contextual("l2", "retinol", "vitc", "vitamin C", true),
			contextual("l3", "oils", "serums", "our serums", false),
		},
		Audit: reports.LinkAuditResponse{SiteID: siteID, Pages: []reports.PageAudit{
			audit("home", "/", 0, false), audit("serums", "/serums/", 0, false), audit("vitc", "/serums/vitamin-c/", 0, false),
			audit("retinol", "/serums/retinol/", 1, true), audit("oils", "/oils/", 0, false),
		}},
		Owed: map[string][]reports.RequiredLink{
			"vitc": {{Relation: "up", Required: true, TargetEntityID: "e-serums", TargetPageID: "serums", TargetPath: "/serums/",
				Satisfied: true, State: string(reports.LinkPlaced), AnchorsAllowed: []string{"Serums"}}},
			"retinol": {{Relation: "up", Required: true, TargetEntityID: "e-serums", TargetPageID: "serums", TargetPath: "/serums/",
				State: string(reports.LinkMissing), AnchorsAllowed: []string{"Serums", "face serums"}}},
		},
	}
}

func readLinks(t *testing.T, m *sitemodeltest.Memory, req sitemodel.LinksRequest) sitemodel.LinksAnswer {
	t.Helper()

	req.SiteID = m.Site.ID
	got, err := m.Service().Links(t.Context(), req)
	if err != nil {
		t.Fatalf("Links(%+v): %v", req, err)
	}
	return got
}

func TestASectionReadsItsPagesTotalsAndOrphans(t *testing.T) {
	t.Parallel()

	m := aSection()
	section := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksSection, PageID: "serums", Limit: 2})
	if section.Root == nil || section.Root.Path != "/serums/" || section.Totals == nil || section.Totals.Pages != 3 ||
		section.Totals.MissingRequired != 1 || section.Totals.Orphans != 1 {
		t.Fatalf("the section = root %+v totals %+v", section.Root, section.Totals)
	}
	if section.Pages == nil || len(section.Pages.Items) != 2 || !section.Pages.HasMore || section.Pages.Items[0].Path != "/serums/" {
		t.Fatalf("the section's first page = %+v", section.Pages)
	}
	rest := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksSection, PageID: "serums", Limit: 2, Cursor: section.Pages.Next})
	if len(rest.Pages.Items) != 1 || rest.Pages.HasMore || rest.Pages.Items[0].Path != "/serums/vitamin-c/" {
		t.Fatalf("the section's second page = %+v", rest.Pages)
	}

	orphans := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksOrphans, PageID: "serums"})
	if len(orphans.Pages.Items) != 1 || orphans.Pages.Items[0].Path != "/serums/retinol/" {
		t.Fatalf("the section's orphans = %+v", orphans.Pages.Items)
	}
	whole := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksSection})
	if whole.Root != nil || whole.Totals.Pages != 5 {
		t.Fatalf("the whole site's section = root %+v, %d pages", whole.Root, whole.Totals.Pages)
	}
}

func TestASectionsLinkMapSaysWhoLinksToWhom(t *testing.T) {
	t.Parallel()

	m := aSection()
	var links []sitemodel.SectionLink
	req := sitemodel.LinksRequest{View: sitemodel.LinksMap, PageID: "serums", Limit: 1}
	for {
		got := readLinks(t, m, req)
		links = append(links, got.Links.Items...)
		if !got.Links.HasMore {
			break
		}
		req.Cursor = got.Links.Next
	}
	if len(links) != 2 {
		t.Fatalf("the section's links = %+v, want the two its own pages hold", links)
	}
	if first := links[0]; first.FromPath != "/serums/retinol/" || first.ToPath != "/serums/vitamin-c/" || first.Anchor != "vitamin C" ||
		!first.Ours || !first.Inside {
		t.Errorf("the first link = %+v", first)
	}

	page := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksPage, PageID: "serums"})
	if page.Page == nil || page.Page.Page.Path != "/serums/" || page.Inbound == nil {
		t.Fatalf("the page's links = %+v", page)
	}
	from := make([]string, 0)
	for _, in := range page.Inbound.Items {
		from = append(from, in.FromPath+" "+in.Anchor)
	}
	if !slices.Equal(from, []string{"/oils/ our serums", "/serums/vitamin-c/ serums"}) {
		t.Errorf("the links into the hub = %v, want each with its anchor, the section's edge included", from)
	}
}

func TestASectionSaysWhatItOwesAndWhatToDoAboutIt(t *testing.T) {
	t.Parallel()

	m := aSection()
	owed := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksOwed, PageID: "serums"})
	if len(owed.Owed.Items) != 2 {
		t.Fatalf("the section owes %+v", owed.Owed.Items)
	}
	missing := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksOwed, PageID: "serums", Missing: true})
	if len(missing.Owed.Items) != 1 || missing.Owed.Items[0].FromPath != "/serums/retinol/" || missing.Owed.Items[0].TargetPath != "/serums/" {
		t.Fatalf("the section's missing links = %+v", missing.Owed.Items)
	}

	suggested := readLinks(t, m, sitemodel.LinksRequest{View: sitemodel.LinksSuggest, PageID: "serums"})
	kinds := map[string]sitemodel.Suggestion{}
	for _, held := range suggested.Suggestions.Items {
		kinds[held.Kind] = held
	}
	if edgeOf := kinds[sitemodel.SuggestParentEdge]; edgeOf.EntityID != "e-retinol" || edgeOf.ToEntityID != "e-serums" ||
		edgeOf.FromPath != "/serums/retinol/" || edgeOf.ToPath != "/serums/" {
		t.Errorf("the missing parent edge = %+v", edgeOf)
	}
	if link := kinds[sitemodel.SuggestLink]; link.FromPageID != "retinol" || link.ToPageID != "serums" ||
		!slices.Equal(link.Anchors, []string{"Serums", "face serums"}) || link.Relation != "up" {
		t.Errorf("the link to place = %+v", link)
	}
	if len(suggested.Suggestions.Items) != 2 {
		t.Errorf("the suggestions = %+v, want the parent edge and the one missing link", suggested.Suggestions.Items)
	}
}

func TestAPagesLinkPlanIsTheOneTheWriterReads(t *testing.T) {
	t.Parallel()

	m := aSection()
	plan, err := m.Service().LinkContextOf(t.Context(), siteID, "vitc")
	if err != nil {
		t.Fatalf("LinkContextOf: %v", err)
	}
	if _, owed := plan.ByPageID("serums"); !owed || plan.PageID != "vitc" {
		t.Errorf("the plan of vitamin C = %+v, want its parent hub among its targets", plan)
	}
	if _, err = m.Service().LinkContextOf(t.Context(), "another-site", "vitc"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("another site's page = %v, want NotFound", err)
	}
}

func TestALinksViewRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	m := aSection()
	if _, err := m.Service().Links(t.Context(), sitemodel.LinksRequest{SiteID: siteID, View: sitemodel.LinksPage}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("the page view without a page answered %v", err)
	}
	if _, err := m.Service().Links(t.Context(), sitemodel.LinksRequest{SiteID: siteID, View: "everything"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown view answered %v", err)
	}
}
