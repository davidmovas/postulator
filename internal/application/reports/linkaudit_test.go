package reports_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (f *fixture) storedLink(from pagemap.Page, toPageID *string, toURL, anchor string, order int) pagemap.PageLink {
	return pagemap.PageLink{
		ID: id.New(), SiteID: f.siteID, FromPageID: from.ID, ToPageID: toPageID, ToURL: toURL,
		AnchorText: anchor, Origin: pagemap.OriginObserved,
		ObservedAt: sqlitetest.Stamp.Add(time.Duration(order) * time.Second),
	}
}

func (f *fixture) replaceLinks(t *testing.T, from pagemap.Page, links ...pagemap.PageLink) {
	t.Helper()

	if err := sqlite.NewPageLinkRepo(f.store).ReplaceForPage(t.Context(), from.ID, links); err != nil {
		t.Fatalf("replace the links of %s: %v", from.Path, err)
	}
}

func auditFixture(t *testing.T) *fixture {
	t.Helper()

	f := newFixture(t)
	beans := f.entity(t, sqlite.NewEntityRepo(f.store), "Beans", 0.9, nil)
	f.edge(t, sqlite.NewEdgeRepo(f.store), beans.ID, f.entities["Coffee"].ID, graph.EdgeParent, graph.StatusApproved)

	espresso := f.pages["/coffee/espresso/"]
	coffee := f.pages["/coffee/"]
	tea := f.pages["/tea/"]
	f.replaceLinks(t, espresso,
		f.storedLink(espresso, &coffee.ID, "/coffee/", "Coffee", 0),
		f.storedLink(espresso, nil, "https://example.org/", "elsewhere", 1),
		f.storedLink(espresso, &tea.ID, "/tea/", "tea", 2),
		f.storedLink(espresso, &espresso.ID, "/coffee/espresso/", "here", 3),
	)
	return f
}

func (f *fixture) row(path, status, entity, skip string, counts [10]int, orphan, onSite bool) reports.PageAudit {
	row := reports.PageAudit{
		PageID: f.pages[path].ID, Path: path, Status: status, SkipReason: skip, OnSite: onSite,
		Targets: counts[0], Required: counts[1], Satisfied: counts[2], Missing: counts[3], MissingRequired: counts[4],
		Blocked: counts[5], OffGraph: counts[6], Inbound: counts[7], Pending: counts[8], Unpublished: counts[9],
		Orphan: orphan,
	}
	if entity != "" {
		row.EntityID = f.entities[entity].ID
		row.EntityName = entity
	}
	return row
}

func TestLinkAuditAuditsEveryMappedPage(t *testing.T) {
	t.Parallel()

	f := auditFixture(t)
	got, err := f.service.LinkAudit(t.Context(), reports.LinkAuditRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("LinkAudit: %v", err)
	}

	want := []reports.PageAudit{
		f.row("/coffee/", "published", "Coffee", "", [10]int{3, 0, 0, 1, 0, 1, 0, 1, 1, 0}, false, true),
		f.row("/coffee/espresso/", "published", "Espresso", "", [10]int{2, 1, 1, 0, 0, 0, 3, 1, 1, 0}, false, true),
		f.row("/coffee/filter/", "planned", "Filter", "", [10]int{2, 1, 1, 0, 0, 0, 0, 0, 1, 0}, false, false),
		f.row("/tea/", "exists", "", "unmapped", [10]int{0, 0, 0, 0, 0, 0, 0, 1, 0, 0}, false, true),
	}
	if !reflect.DeepEqual(got.Pages, want) {
		t.Fatalf("pages =\n%+v\nwant\n%+v", got.Pages, want)
	}

	totals := reports.LinkTotals{
		Pages: 4, Audited: 3, Targets: 7, Required: 2, Satisfied: 2, Missing: 1, MissingRequired: 0,
		Blocked: 1, OffGraph: 3, Orphans: 0, Pending: 3, Unpublished: 0,
	}
	if got.Totals != totals {
		t.Fatalf("totals = %+v, want %+v", got.Totals, totals)
	}
	if got.SiteID != f.siteID {
		t.Fatalf("siteId = %s", got.SiteID)
	}

	policy := got.Policy
	if policy.ID == "" || policy.Name != "Default" || !policy.ForbidExternal || !policy.ForbidSelf ||
		policy.AnchorStrategy != "prefer_user" {
		t.Fatalf("policy = %+v", policy)
	}
}

func TestLinkAuditPageAgreesWithTheSiteAudit(t *testing.T) {
	t.Parallel()

	f := auditFixture(t)
	whole, err := f.service.LinkAudit(t.Context(), reports.LinkAuditRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("LinkAudit: %v", err)
	}

	details := make(map[string]reports.LinkAuditPageResponse, len(whole.Pages))
	for _, row := range whole.Pages {
		detail, detailErr := f.service.LinkAuditPage(t.Context(), reports.LinkAuditPageRequest{PageID: row.PageID})
		if detailErr != nil {
			t.Fatalf("LinkAuditPage(%s): %v", row.Path, detailErr)
		}
		if !reflect.DeepEqual(detail.Page, row) {
			t.Fatalf("the detail of %s answers %+v, the site audit %+v", row.Path, detail.Page, row)
		}
		details[row.Path] = detail
	}

	espresso := details["/coffee/espresso/"]
	if espresso.TemplateID != f.templateID {
		t.Fatalf("templateId = %s, want %s", espresso.TemplateID, f.templateID)
	}
	rules := template.LinkRules{
		UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 20, MaxPerTarget: 1,
		ParentLinkWithinParagraphs: 2, ChildrenSection: true,
	}
	if espresso.Rules != rules {
		t.Fatalf("rules = %+v, want %+v", espresso.Rules, rules)
	}

	coffee, filter := f.entities["Coffee"], f.entities["Filter"]
	wantRequired := []reports.RequiredLink{
		{
			Relation: "up", Required: true, TargetEntityID: coffee.ID, TargetEntityName: "Coffee",
			TargetPageID: f.pages["/coffee/"].ID, TargetPath: "/coffee/", Satisfied: true, Anchor: "Coffee",
			AnchorAllowed: true, AnchorsAllowed: []string{"Coffee"}, Weight: 1, Depth: 1,
			State: string(reports.LinkPlaced), TargetOnSite: true,
		},
		{
			Relation: "sibling", TargetEntityID: filter.ID, TargetEntityName: "Filter",
			TargetPageID: f.pages["/coffee/filter/"].ID, TargetPath: "/coffee/filter/",
			AnchorsAllowed: []string{"Filter"}, Weight: 1, Depth: 1, State: string(reports.LinkAwaitingTarget),
		},
	}
	if !reflect.DeepEqual(espresso.Required, wantRequired) {
		t.Fatalf("required =\n%+v\nwant\n%+v", espresso.Required, wantRequired)
	}
	wantExtra := []reports.ExtraLink{
		{ToURL: "https://example.org/", Anchor: "elsewhere", Kind: "external", Origin: "observed"},
		{ToURL: "/tea/", ToPageID: f.pages["/tea/"].ID, Anchor: "tea", Kind: "unknown_internal", Origin: "observed"},
		{ToURL: "/coffee/espresso/", ToPageID: f.pages["/coffee/espresso/"].ID, Anchor: "here", Kind: "self", Origin: "observed"},
	}
	if !reflect.DeepEqual(espresso.Extra, wantExtra) {
		t.Fatalf("extra =\n%+v\nwant\n%+v", espresso.Extra, wantExtra)
	}

	blocked := details["/coffee/"].Required
	if len(blocked) != 3 || blocked[0].TargetEntityName != "Espresso" || blocked[1].TargetEntityName != "Filter" {
		t.Fatalf("the hub's targets = %+v", blocked)
	}
	beans := reports.RequiredLink{
		Relation: "down", TargetEntityID: f.entities["Beans"].ID, TargetEntityName: "Beans",
		AnchorsAllowed: []string{}, Weight: 0.9, Depth: 1, BlockedReason: "no_canonical_page",
		State: string(reports.LinkBlocked),
	}
	if !reflect.DeepEqual(blocked[2], beans) {
		t.Fatalf("the blocked child = %+v, want %+v", blocked[2], beans)
	}

	if tea := details["/tea/"]; len(tea.Required) != 0 || len(tea.Extra) != 0 || tea.TemplateID != "" {
		t.Fatalf("an unmapped page = %+v", tea)
	}
}

func TestTheAuditSaysWhyALinkIsAbsent(t *testing.T) {
	t.Parallel()

	f := auditFixture(t)
	coffee, filter := f.pages["/coffee/"], f.pages["/coffee/filter/"]
	f.replaceLinks(t, coffee, f.storedLink(coffee, &filter.ID, "/coffee/filter/", "Filter", 0))

	cases := []struct {
		page   string
		states map[string]reports.LinkState
		onSite bool
	}{
		{
			page:   "/coffee/",
			states: map[string]reports.LinkState{"/coffee/espresso/": reports.LinkMissing, "/coffee/filter/": reports.LinkTargetUnpublished, "": reports.LinkBlocked},
			onSite: true,
		},
		{
			page:   "/coffee/espresso/",
			states: map[string]reports.LinkState{"/coffee/": reports.LinkPlaced, "/coffee/filter/": reports.LinkAwaitingTarget},
			onSite: true,
		},
		{
			page:   "/coffee/filter/",
			states: map[string]reports.LinkState{"/coffee/": reports.LinkAwaitingPage, "/coffee/espresso/": reports.LinkPlaced},
		},
	}

	for _, tc := range cases {
		t.Run(tc.page, func(t *testing.T) {
			t.Parallel()

			detail, err := f.service.LinkAuditPage(t.Context(), reports.LinkAuditPageRequest{PageID: f.pages[tc.page].ID})
			if err != nil {
				t.Fatalf("LinkAuditPage: %v", err)
			}
			if detail.Page.OnSite != tc.onSite {
				t.Fatalf("onSite = %v, want %v", detail.Page.OnSite, tc.onSite)
			}
			if len(detail.Required) != len(tc.states) {
				t.Fatalf("required = %+v, want %d rows", detail.Required, len(tc.states))
			}
			for _, row := range detail.Required {
				if want := tc.states[row.TargetPath]; row.State != string(want) {
					t.Errorf("%s -> %q = %q, want %q", tc.page, row.TargetPath, row.State, want)
				}
			}
		})
	}

	coffeeRow, err := f.service.LinkAuditPage(t.Context(), reports.LinkAuditPageRequest{PageID: coffee.ID})
	if err != nil {
		t.Fatalf("LinkAuditPage: %v", err)
	}
	if coffeeRow.Page.Unpublished != 1 || coffeeRow.Page.Missing != 1 || coffeeRow.Page.Satisfied != 1 {
		t.Fatalf("the hub = %+v, want one placed link to a page not on the site and one missing", coffeeRow.Page)
	}
}

func TestLinkAuditSkipsAPageWithoutATemplate(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	other := sqlitetest.Site(t, f.store, "bare")
	page := sqlitetest.Page(t, f.store, other.ID, "/alone/")
	entity := sqlitetest.Entity(t, f.store, other.ID, "Alone")
	page.EntityID = &entity.ID
	if err := sqlite.NewPageRepo(f.store).Update(t.Context(), page); err != nil {
		t.Fatalf("map the page: %v", err)
	}

	got, err := f.service.LinkAudit(t.Context(), reports.LinkAuditRequest{SiteID: other.ID})
	if err != nil {
		t.Fatalf("LinkAudit: %v", err)
	}
	if len(got.Pages) != 1 || got.Pages[0].SkipReason != "no_template" || got.Pages[0].Targets != 0 {
		t.Fatalf("pages = %+v", got.Pages)
	}
	if got.Totals.Audited != 0 || got.Totals.Pages != 1 {
		t.Fatalf("totals = %+v", got.Totals)
	}
}

func TestLinkAuditOrphansAgreeWithTheOverview(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	overview, err := f.service.SiteOverview(t.Context(), reports.SiteOverviewRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("SiteOverview: %v", err)
	}
	audit, err := f.service.LinkAudit(t.Context(), reports.LinkAuditRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("LinkAudit: %v", err)
	}
	if audit.Totals.Orphans != overview.Pages.Orphans || audit.Totals.Orphans == 0 {
		t.Fatalf("audit orphans = %d, overview orphans = %d", audit.Totals.Orphans, overview.Pages.Orphans)
	}
}

func TestLinkAuditRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cases := []struct {
		name string
		id   string
		code errors.Code
	}{
		{"a blank id", " ", errors.Invalid},
		{"an unknown id", id.New(), errors.NotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, siteErr := f.service.LinkAudit(t.Context(), reports.LinkAuditRequest{SiteID: tc.id})
			if !errors.IsCode(siteErr, tc.code) {
				t.Fatalf("LinkAudit = %v, want %s", siteErr, tc.code)
			}
			_, pageErr := f.service.LinkAuditPage(t.Context(), reports.LinkAuditPageRequest{PageID: tc.id})
			if !errors.IsCode(pageErr, tc.code) {
				t.Fatalf("LinkAuditPage = %v, want %s", pageErr, tc.code)
			}
		})
	}
}

func (f *fixture) plan(t *testing.T, page pagemap.Page, entityID string, rules template.LinkRules) content.LinkContext {
	t.Helper()

	entities, err := sqlite.NewEntityRepo(f.store).ListBySite(t.Context(), f.siteID)
	if err != nil {
		t.Fatalf("list the entities: %v", err)
	}
	edges, err := sqlite.NewEdgeRepo(f.store).ListBySite(t.Context(), f.siteID)
	if err != nil {
		t.Fatalf("list the edges: %v", err)
	}
	pages, err := sqlite.NewPageRepo(f.store).ListBySite(t.Context(), f.siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	owner, err := sqlite.NewSiteRepo(f.store).Get(t.Context(), f.siteID)
	if err != nil {
		t.Fatalf("read the site: %v", err)
	}
	built, err := graph.New(entities, edges)
	if err != nil {
		t.Fatalf("graph.New: %v", err)
	}

	return content.PlanLinks(built, pagemap.NewIndex(pages), content.Subject{
		Site: pagemap.NewSite(owner.BaseURL), PageID: page.ID, PagePath: page.Path, EntityID: entityID,
	}, template.LinkPolicy{Rules: rules, ForbidExternal: true, ForbidSelf: true}).Context
}

func TestASecondPageOfAnEntityIsAuditedAndValidatedAlike(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	pageRepo := sqlite.NewPageRepo(f.store)
	canonical := f.pages["/coffee/espresso/"]

	seconds := []pagemap.Page{
		f.page(t, pageRepo, "/coffee/espresso-2/", pagemap.StatusPublished),
		f.page(t, pageRepo, "/reviews/gaggia/", pagemap.StatusPublished),
		f.page(t, pageRepo, "/reviews/rancilio/", pagemap.StatusPublished),
	}
	for i := range seconds {
		f.map_(t, pageRepo, seconds[i], f.entities["Espresso"].ID)
		f.replaceLinks(t, seconds[i],
			f.storedLink(seconds[i], &seconds[i].ID, seconds[i].Path, "me", 0),
			f.storedLink(seconds[i], &canonical.ID, "/coffee/espresso/", "Espresso", 1),
			f.storedLink(seconds[i], nil, "https://shop.example.com/coffee/espresso/", "Espresso", 2),
		)
	}

	for _, second := range seconds {
		t.Run(second.Path, func(t *testing.T) {
			detail, err := f.service.LinkAuditPage(t.Context(), reports.LinkAuditPageRequest{PageID: second.ID})
			if err != nil {
				t.Fatalf("LinkAuditPage: %v", err)
			}

			row, found := requiredFor(detail.Required, canonical.ID)
			if !found || row.Required || row.Relation != "up" || !row.Satisfied {
				t.Fatalf("the canonical page is %+v, want an optional up target already satisfied", row)
			}
			if detail.Page.OffGraph != 1 || len(detail.Extra) != 1 || detail.Extra[0].Kind != "self" {
				t.Fatalf("off-graph = %+v, want the link to the page itself alone", detail.Extra)
			}

			lc := f.plan(t, second, f.entities["Espresso"].ID, detail.Rules)
			body := `<p>Read the <a href="/coffee/espresso/">Espresso</a> page, the ` +
				`<a href="https://shop.example.com/coffee/espresso/">Espresso</a> page again and ` +
				`<a href="` + second.Path + `">this one</a>.</p>`
			doc, err := content.Parse(body)
			if err != nil {
				t.Fatalf("parse the body: %v", err)
			}

			report := content.Compliance(doc, lc, template.LinkPolicy{
				Rules: detail.Rules, ForbidExternal: true, ForbidSelf: true,
			}, second.ID)
			selfLinks, offGraph := 0, 0
			for _, item := range report.Items {
				switch item.Code {
				case content.CodeSelfLink:
					selfLinks++
				case content.CodeExternalLink, content.CodeUnknownInternal:
					offGraph++
				}
			}
			if selfLinks != 1 || offGraph != 0 {
				t.Fatalf("Compliance found %d self links and %d off-graph links, want 1 and 0 (%+v)",
					selfLinks, offGraph, report.Items)
			}
		})
	}
}

func requiredFor(rows []reports.RequiredLink, pageID string) (reports.RequiredLink, bool) {
	for i := range rows {
		if rows[i].TargetPageID == pageID {
			return rows[i], true
		}
	}
	return reports.RequiredLink{}, false
}
