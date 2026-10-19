package sitemodel_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func audited() *memory {
	m := site1()
	m.audit = reports.LinkAuditResponse{
		SiteID: siteID,
		Totals: reports.LinkTotals{Audited: 2, Targets: 3, Satisfied: 1, Missing: 1, MissingRequired: 1, Pending: 1, Orphans: 1, OffGraph: 2},
		Pages: []reports.PageAudit{
			{PageID: "bpc", Path: "/bpc-157/", MissingRequired: 1, Missing: 1},
			{PageID: "far", Path: "/far-away/", Orphan: true},
		},
	}
	return m
}

func TestTheBriefReadsTheWholeSiteInOneRead(t *testing.T) {
	t.Parallel()

	m := audited()
	brief, err := m.service().Brief(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Brief: %v", err)
	}
	if m.reads != 1 {
		t.Errorf("the brief read in %d units, want one", m.reads)
	}
	if brief.Name != "Shop" || brief.Overview.Pages != 5 || brief.Overview.Empty != 4 || brief.Overview.Unmapped != 3 {
		t.Errorf("brief = %+v, want five pages, four holding no text and three with no entity", brief.Overview)
	}
	if brief.Links.Orphans != 1 || brief.Links.MissingRequired != 1 || brief.Links.OffGraph != 2 {
		t.Errorf("links = %+v, want the audit's totals", brief.Links)
	}
	if len(brief.Overview.Findings) != 1 || brief.Overview.Findings[0].Code != "parent_unresolved" {
		t.Errorf("findings = %+v, want the open one only", brief.Overview.Findings)
	}
	if _, err := m.service().Brief(t.Context(), "elsewhere"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Brief of an unknown site = %v, want NotFound", err)
	}
}

func explore(t *testing.T, m *memory, req sitemodel.ExploreRequest) sitemodel.ExploreResponse {
	t.Helper()

	req.SiteID = siteID
	got, err := m.service().Explore(t.Context(), req)
	if err != nil {
		t.Fatalf("Explore(%+v): %v", req, err)
	}
	return got
}

func TestEveryViewAnswersFromTheModel(t *testing.T) {
	t.Parallel()

	m := audited()
	page := explore(t, m, sitemodel.ExploreRequest{View: sitemodel.ViewPage, Path: "peptides"})
	if page.Page == nil || page.Page.Page.PageID != "hub" || page.Page.Entity == nil || page.Page.Entity.Name != "Peptides" ||
		len(page.Page.Products) != 1 || len(page.Page.Findings) != 1 || page.Page.Findings[0].Code != "parent_unresolved" {
		t.Errorf("page = %+v, want the hub found by its path with its card, its product and its open finding", page.Page)
	}

	top := explore(t, m, sitemodel.ExploreRequest{View: sitemodel.ViewTree})
	if top.Tree == nil || top.Tree.Parent != nil || paths(top.Tree.Nodes.Items, nodeRef) != "/ /far-away/ /product/tb-500/" {
		t.Errorf("the top of the tree = %+v", top.Tree)
	}
	if top.Tree.Nodes.Items[0].Descendants != 2 {
		t.Errorf("the front page holds %d pages below it, want 2", top.Tree.Nodes.Items[0].Descendants)
	}
	under := explore(t, m, sitemodel.ExploreRequest{View: sitemodel.ViewTree, PageID: "hub"})
	if under.Tree.Parent == nil || under.Tree.Parent.Path != "/peptides/" || paths(under.Tree.Nodes.Items, nodeRef) != "/bpc-157/" {
		t.Errorf("under the hub = %+v", under.Tree)
	}

	links := explore(t, m, sitemodel.ExploreRequest{View: sitemodel.ViewLinks, Path: "/bpc-157/"})
	if links.Links == nil || links.Links.Page.PageID != "bpc" || links.Links.Page.MissingRequired != 1 {
		t.Errorf("links = %+v, want the audit of BPC-157", links.Links)
	}

	gaps := explore(t, m, sitemodel.ExploreRequest{View: sitemodel.ViewGaps, Gap: "orphan"})
	if gaps.Gaps == nil || paths(gaps.Gaps.Items, gapRef) != "/far-away/" {
		t.Errorf("orphans = %+v, want the far page", gaps.Gaps)
	}

	pack := explore(t, m, sitemodel.ExploreRequest{View: sitemodel.ViewPack, PageID: "hub", Budget: 100000})
	if pack.Pack == nil || pack.Pack.Field != "body" || pack.Pack.Budget != 100000 || !strings.Contains(pack.Pack.Text, "Peptides") {
		t.Errorf("pack = %+v, want the hub's body pack as the writer reads it", pack.Pack)
	}
}

func TestTheTreeAndTheGapsPageByCursor(t *testing.T) {
	t.Parallel()

	m := audited()
	seen := make([]string, 0)
	var cursor paging.Cursor
	for range 4 {
		got := explore(t, m, sitemodel.ExploreRequest{View: sitemodel.ViewGaps, Cursor: cursor, Limit: 2})
		seen = append(seen, paths(got.Gaps.Items, gapRef))
		if !got.Gaps.HasMore {
			break
		}
		cursor = got.Gaps.Next
	}
	if want := "/ /bpc-157/|/far-away/ /peptides/|/product/tb-500/"; strings.Join(seen, "|") != want {
		t.Errorf("pages of gaps = %q, want %q", strings.Join(seen, "|"), want)
	}
}

func TestExploreRefusesWhatItCannotAnswer(t *testing.T) {
	t.Parallel()

	other := site1()
	other.pages[3].SiteID = "another-site"
	cases := []struct {
		name string
		m    *memory
		req  sitemodel.ExploreRequest
		code errors.Code
	}{
		{name: "an unknown view", m: audited(), req: sitemodel.ExploreRequest{View: "everything"}, code: errors.Invalid},
		{name: "a page view with no page", m: audited(), req: sitemodel.ExploreRequest{View: sitemodel.ViewPage}, code: errors.Invalid},
		{name: "a links view with no page", m: audited(), req: sitemodel.ExploreRequest{View: sitemodel.ViewLinks}, code: errors.Invalid},
		{name: "a path no page has", m: audited(), req: sitemodel.ExploreRequest{View: sitemodel.ViewPage, Path: "/nowhere/"}, code: errors.NotFound},
		{name: "a page of another site", m: other, req: sitemodel.ExploreRequest{View: sitemodel.ViewPage, PageID: "far"}, code: errors.NotFound},
		{name: "an unknown gap", m: audited(), req: sitemodel.ExploreRequest{View: sitemodel.ViewGaps, Gap: "ugly"}, code: errors.Invalid},
		{name: "a cursor of another order", m: audited(), req: sitemodel.ExploreRequest{View: sitemodel.ViewTree, Cursor: "bm90IGEgY3Vyc29y"}, code: errors.Invalid},
		{name: "a field the writer never writes", m: audited(), req: sitemodel.ExploreRequest{View: sitemodel.ViewPack, PageID: "tb", Field: "price"}, code: errors.Invalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.req.SiteID = siteID
			if _, err := tc.m.service().Explore(t.Context(), tc.req); !errors.IsCode(err, tc.code) {
				t.Errorf("Explore = %v, want %s", err, tc.code)
			}
		})
	}
}

func nodeRef(node contextpack.Node) contextpack.Ref {
	return node.Page
}

func gapRef(row contextpack.GapRow) contextpack.Ref {
	return row.Page
}

func paths[T any](rows []T, ref func(T) contextpack.Ref) string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, ref(row).Path)
	}
	return strings.Join(out, " ")
}
