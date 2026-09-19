package reports_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/reports"
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

func (f *fixture) row(path, status, entity, skip string, counts [10]int, orphan bool) reports.PageAudit {
	row := reports.PageAudit{
		PageID: f.pages[path].ID, Path: path, Status: status, SkipReason: skip,
		Targets: counts[0], Required: counts[1], Satisfied: counts[2], Missing: counts[3], MissingRequired: counts[4],
		Blocked: counts[5], OffGraph: counts[6], Inbound: counts[7], Orphan: orphan,
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
		f.row("/coffee/", "published", "Coffee", "", [10]int{3, 0, 0, 2, 0, 1, 0, 1}, false),
		f.row("/coffee/espresso/", "published", "Espresso", "", [10]int{2, 1, 1, 1, 0, 0, 3, 1}, false),
		f.row("/coffee/filter/", "planned", "Filter", "", [10]int{2, 1, 1, 1, 1, 0, 0, 0}, true),
		f.row("/tea/", "exists", "", "unmapped", [10]int{0, 0, 0, 0, 0, 0, 0, 1}, false),
	}
	if !reflect.DeepEqual(got.Pages, want) {
		t.Fatalf("pages =\n%+v\nwant\n%+v", got.Pages, want)
	}

	totals := reports.LinkTotals{
		Pages: 4, Audited: 3, Targets: 7, Required: 2, Satisfied: 2, Missing: 4, MissingRequired: 1,
		Blocked: 1, OffGraph: 3, Orphans: 1,
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
		},
		{
			Relation: "sibling", TargetEntityID: filter.ID, TargetEntityName: "Filter",
			TargetPageID: f.pages["/coffee/filter/"].ID, TargetPath: "/coffee/filter/",
			AnchorsAllowed: []string{"Filter"}, Weight: 1, Depth: 1,
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
	}
	if !reflect.DeepEqual(blocked[2], beans) {
		t.Fatalf("the blocked child = %+v, want %+v", blocked[2], beans)
	}

	if tea := details["/tea/"]; len(tea.Required) != 0 || len(tea.Extra) != 0 || tea.TemplateID != "" {
		t.Fatalf("an unmapped page = %+v", tea)
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

func TestLinkAuditRebindsSelfToTheAuditedPage(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	pageRepo := sqlite.NewPageRepo(f.store)
	second := f.page(t, pageRepo, "/coffee/espresso-2/", pagemap.StatusPublished)
	f.map_(t, pageRepo, second, f.entities["Espresso"].ID)
	canonical := f.pages["/coffee/espresso/"]
	f.replaceLinks(t, second,
		f.storedLink(second, &second.ID, "/coffee/espresso-2/", "me", 0),
		f.storedLink(second, &canonical.ID, "/coffee/espresso/", "the other", 1),
	)

	detail, err := f.service.LinkAuditPage(t.Context(), reports.LinkAuditPageRequest{PageID: second.ID})
	if err != nil {
		t.Fatalf("LinkAuditPage: %v", err)
	}
	if detail.Page.OffGraph != 2 || len(detail.Extra) != 2 {
		t.Fatalf("detail = %+v", detail)
	}
	if detail.Extra[0].Kind != "self" || detail.Extra[1].Kind != "unknown_internal" {
		t.Fatalf("extra = %+v", detail.Extra)
	}
}
