package sitemodel_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func audited() *sitemodeltest.Memory {
	m := site1()
	m.Audit = reports.LinkAuditResponse{
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
	brief, err := m.Service().Brief(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Brief: %v", err)
	}
	if m.Reads() != 1 {
		t.Errorf("the brief read in %d units, want one", m.Reads())
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
	if _, err := m.Service().Brief(t.Context(), "elsewhere"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Brief of an unknown site = %v, want NotFound", err)
	}
}

func TestThePackTextIsWhatTheWriterReadsForAField(t *testing.T) {
	t.Parallel()

	m := audited()
	page, err := m.Service().Page(t.Context(), sitemodel.PageRequest{SiteID: siteID, Path: "peptides"})
	if err != nil || page.Page.PageID != "hub" || page.Entity == nil || page.Entity.Name != "Peptides" ||
		len(page.Products) != 1 || len(page.Findings) != 1 || page.Findings[0].Code != "parent_unresolved" {
		t.Errorf("page = %+v, %v, want the hub found by its path with its card, its product and its open finding", page, err)
	}

	pack, err := m.Service().PackText(t.Context(), sitemodel.PackTextRequest{SiteID: siteID, PageID: "hub", Budget: 100000})
	if err != nil || pack.Field != "body" || pack.Budget != 100000 || !strings.Contains(pack.Text, "Peptides") {
		t.Errorf("pack = %+v, %v, want the hub's body pack as the writer reads it", pack, err)
	}
}

func TestAPageIsNamedByItsIdOrItsPathOnItsOwnSite(t *testing.T) {
	t.Parallel()

	other := site1()
	other.Pages[3].SiteID = "another-site"
	cases := []struct {
		name string
		m    *sitemodeltest.Memory
		req  sitemodel.PackTextRequest
		code errors.Code
	}{
		{name: "no page named", m: audited(), req: sitemodel.PackTextRequest{}, code: errors.Invalid},
		{name: "a path no page has", m: audited(), req: sitemodel.PackTextRequest{Path: "/nowhere/"}, code: errors.NotFound},
		{name: "a page of another site", m: other, req: sitemodel.PackTextRequest{PageID: "far"}, code: errors.NotFound},
		{name: "a field the writer never writes", m: audited(), req: sitemodel.PackTextRequest{PageID: "tb", Field: "price"}, code: errors.Invalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.req.SiteID = siteID
			if _, err := tc.m.Service().PackText(t.Context(), tc.req); !errors.IsCode(err, tc.code) {
				t.Errorf("PackText = %v, want %s", err, tc.code)
			}
		})
	}
}
