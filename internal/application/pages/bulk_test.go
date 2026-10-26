package pages_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type bulkSite struct {
	harness
	ids      map[string]string
	category pagetype.Type
	guide    pagetype.Type
	product  pagetype.Type
}

func newBulkSite(t *testing.T) bulkSite {
	t.Helper()
	h := newHarness(t)
	b := bulkSite{harness: h, ids: map[string]string{}}
	types := sqlite.NewPageTypeRepo(h.store)
	for _, held := range []*pagetype.Type{&b.category, &b.guide, &b.product} {
		*held = pagetype.Type{ID: id.New(), SiteID: h.siteID, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	}
	b.category.Name, b.category.Preset = "Category", pagetype.Category
	b.guide.Name, b.guide.Preset = "Guide", pagetype.Guide
	b.product.Name, b.product.Preset = "Product", pagetype.Product
	for _, kind := range []pagetype.Type{b.category, b.guide, b.product} {
		if err := types.Insert(t.Context(), kind); err != nil {
			t.Fatalf("Insert %s: %v", kind.Name, err)
		}
	}
	repo := sqlite.NewPageRepo(h.store)
	for _, path := range []string{"/", "/peptides/", "/peptides/healing/", "/peptides/healing/bpc/", "/lab-guides/", "/product/bpc/"} {
		page := sqlitetest.Page(t, h.store, h.siteID, path)
		if path == "/product/bpc/" {
			page.WPType = pagemap.WPProduct
			if err := repo.Update(t.Context(), page); err != nil {
				t.Fatalf("make the product: %v", err)
			}
		}
		b.ids[path] = page.ID
	}
	trail := func(paths ...string) string {
		out := ""
		for _, path := range paths {
			out += b.ids[path] + "/"
		}
		return out
	}
	row := func(path, typeID string, above ...string) contextpack.MapRow {
		return contextpack.MapRow{
			PageID: b.ids[path], SiteID: h.siteID, Path: path, Trail: trail(append(above, path)...), Row: true, TypeID: typeID,
			Title: path, TitleKey: path, PathKey: path,
		}
	}
	rows := []contextpack.MapRow{
		row("/", ""), row("/peptides/", b.category.ID, "/"), row("/peptides/healing/", b.category.ID, "/", "/peptides/"),
		row("/peptides/healing/bpc/", b.guide.ID, "/", "/peptides/", "/peptides/healing/"), row("/lab-guides/", b.guide.ID, "/"),
		row("/product/bpc/", b.product.ID, "/", "/peptides/"),
	}
	if err := sqlite.NewPageMapRepo(h.store).Write(t.Context(), h.siteID, rows, nil, 1); err != nil {
		t.Fatalf("Write the map: %v", err)
	}
	return b
}

func (b bulkSite) paths(ids []string) []string {
	byID := make(map[string]string, len(b.ids))
	for path, pageID := range b.ids {
		byID[pageID] = path
	}
	out := make([]string, 0, len(ids))
	for _, pageID := range ids {
		out = append(out, byID[pageID])
	}
	slices.Sort(out)
	return out
}

func TestASelectionIsResolvedFromItsIDsSubtreesFilterAndRuleLessItsExceptions(t *testing.T) {
	t.Parallel()
	b := newBulkSite(t)
	b.classifier.matches["rule-13"] = []string{b.ids["/lab-guides/"]}

	cases := map[string]struct {
		sel  pages.Selection
		want []string
	}{
		"ids":                {pages.Selection{PageIDs: []string{b.ids["/"], b.ids["/lab-guides/"]}}, []string{"/", "/lab-guides/"}},
		"a subtree":          {pages.Selection{Subtrees: []pages.Subtree{{PageID: b.ids["/peptides/"]}}}, []string{"/peptides/", "/peptides/healing/", "/peptides/healing/bpc/", "/product/bpc/"}},
		"a subtree's guides": {pages.Selection{Subtrees: []pages.Subtree{{PageID: b.ids["/peptides/"], TypeID: b.guide.ID}}}, []string{"/peptides/healing/bpc/"}},
		"a type, seen as two": {pages.Selection{Filter: &pages.PageFilter{PageTypeID: b.category.ID}, Seen: 2},
			[]string{"/peptides/", "/peptides/healing/"}},
		"a rule's matches": {pages.Selection{RuleID: "rule-13"}, []string{"/lab-guides/"}},
		"less exceptions": {pages.Selection{Subtrees: []pages.Subtree{{PageID: b.ids["/peptides/"]}}, Except: []string{b.ids["/product/bpc/"]}},
			[]string{"/peptides/", "/peptides/healing/", "/peptides/healing/bpc/"}},
		"less a branch": {pages.Selection{
			Subtrees: []pages.Subtree{{PageID: b.ids["/peptides/"]}}, ExceptSubtrees: []pages.Subtree{{PageID: b.ids["/peptides/healing/"]}},
		}, []string{"/peptides/", "/product/bpc/"}},
	}
	for name, tc := range cases {
		tc.sel.SiteID = b.siteID
		got, err := b.service.Resolve(t.Context(), tc.sel)
		if err != nil || !slices.Equal(b.paths(got.PageIDs), tc.want) {
			t.Errorf("%s = %v %v, want %v", name, b.paths(got.PageIDs), err, tc.want)
		}
		named := make([]string, 0, len(got.Paths))
		for _, pageID := range got.PageIDs {
			if path, held := got.Paths[pageID]; held && b.ids[path] == pageID {
				named = append(named, path)
			}
		}
		slices.Sort(named)
		if len(got.Paths) != len(got.PageIDs) || !slices.Equal(named, tc.want) {
			t.Errorf("%s names its pages %v, want each one's path %v", name, got.Paths, tc.want)
		}
	}
	if _, err := b.service.Resolve(t.Context(), pages.Selection{SiteID: b.siteID, Filter: &pages.PageFilter{PageTypeID: b.category.ID}, Seen: 3}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("a filter whose matches moved since they were seen = %v, want CONFLICT", err)
	}
	other := sqlitetest.Site(t, b.store, "other")
	stranger := sqlitetest.Page(t, b.store, other.ID, "/elsewhere/")
	if _, err := b.service.Resolve(t.Context(), pages.Selection{SiteID: b.siteID, PageIDs: []string{stranger.ID}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("another site's page = %v, want INVALID", err)
	}
	counted, err := b.service.Count(t.Context(), pages.CountRequest{SiteID: b.siteID, PageFilter: pages.PageFilter{Under: b.ids["/peptides/"]}})
	if err != nil || counted.Pages != 4 {
		t.Errorf("Count under the hub = %+v %v, want its four pages", counted, err)
	}
}

func TestAChosenTypeIsSetByAPersonExceptWhereTheRoleCannotHoldThePage(t *testing.T) {
	t.Parallel()
	b := newBulkSite(t)
	open, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: b.siteID, PageID: new(b.ids["/lab-guides/"]), Code: finding.TypeByDefault,
		SubjectKey: finding.SubjectPage(b.ids["/lab-guides/"]), Subject: []byte(`{}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("finding.New: %v", err)
	}
	findings := sqlite.NewFindingRepo(b.store)
	if err = findings.Upsert(t.Context(), open); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	typed, err := b.service.SetTypes(t.Context(), pages.SetTypesRequest{
		Selection: pages.Selection{SiteID: b.siteID, Subtrees: []pages.Subtree{{PageID: b.ids["/"]}}}, PageTypeID: &b.guide.ID,
	})
	if err != nil || typed.Typed != 5 || typed.Refused != 1 {
		t.Fatalf("SetTypes = %+v %v, want five pages typed and the product refused", typed, err)
	}
	repo := sqlite.NewPageRepo(b.store)
	stored, err := repo.Get(t.Context(), b.ids["/lab-guides/"])
	if err != nil || stored.TypeSource != pagemap.TypeSourceManual || stored.TypeReason != pagemap.TypeByUser || *stored.PageTypeID != b.guide.ID {
		t.Fatalf("a typed page = %+v %v, want the person's Guide", stored, err)
	}
	if product, getErr := repo.Get(t.Context(), b.ids["/product/bpc/"]); getErr != nil || product.PageTypeID != nil {
		t.Errorf("the product = %+v %v, want it left as it was", product.PageTypeID, getErr)
	}
	if left, listErr := findings.OpenBySite(t.Context(), b.siteID, finding.TypeByDefault); listErr != nil || len(left) != 0 {
		t.Errorf("type_by_default after a person typed the page = %+v %v, want it closed", left, listErr)
	}

	if _, err = b.service.SetType(t.Context(), pages.SetTypeRequest{PageID: b.ids["/product/bpc/"], PageTypeID: b.guide.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("one product typed as a guide = %v, want INVALID", err)
	}

	back, err := b.service.SetTypes(t.Context(), pages.SetTypesRequest{
		Selection: pages.Selection{SiteID: b.siteID, PageIDs: []string{b.ids["/lab-guides/"]}},
	})
	if err != nil || back.Typed != 1 || len(b.classifier.reclassified) != 1 || b.classifier.reclassified[0][0] != b.ids["/lab-guides/"] {
		t.Fatalf("back to the rules = %+v %v (%v), want the page handed to the rules", back, err, b.classifier.reclassified)
	}
	if stored, err = repo.Get(t.Context(), b.ids["/lab-guides/"]); err != nil || stored.PageTypeID != nil || stored.TypeSource != pagemap.TypeSourceNone {
		t.Fatalf("the page handed back = %+v %v, want it untyped for the rules", stored, err)
	}
	if len(b.recorder.Events()) == 0 {
		t.Error("no event told the window")
	}
}

func TestATemplateIsSetOnTheChosenPagesOrTheyFollowTheirType(t *testing.T) {
	t.Parallel()
	b := newBulkSite(t)
	shipped := sqlitetest.Template(t, b.store, "Hub")

	set, err := b.service.SetTemplates(t.Context(), pages.SetTemplatesRequest{
		Selection: pages.Selection{SiteID: b.siteID, Filter: &pages.PageFilter{PageTypeID: b.category.ID}}, TemplateID: &shipped.ID,
	})
	if err != nil || set.Changed != 2 {
		t.Fatalf("SetTemplates = %+v %v", set, err)
	}
	cleared, err := b.service.SetTemplates(t.Context(), pages.SetTemplatesRequest{
		Selection: pages.Selection{SiteID: b.siteID, PageIDs: []string{b.ids["/peptides/"]}},
	})
	if err != nil || cleared.Changed != 1 {
		t.Fatalf("following the type = %+v %v", cleared, err)
	}
	stored, err := sqlite.NewPageRepo(b.store).Get(t.Context(), b.ids["/peptides/healing/"])
	if err != nil || stored.TemplateID == nil || *stored.TemplateID != shipped.ID {
		t.Fatalf("the other category = %+v %v", stored.TemplateID, err)
	}
	missing := "no-such-template"
	if _, err = b.service.SetTemplates(t.Context(), pages.SetTemplatesRequest{
		Selection: pages.Selection{SiteID: b.siteID, PageIDs: []string{b.ids["/"]}}, TemplateID: &missing,
	}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("an unknown template = %v, want NOT_FOUND", err)
	}

	assigned, err := b.service.AssignTemplate(t.Context(), pages.AssignTemplateRequest{
		SiteID: b.siteID, PageIDs: []string{b.ids["/"], b.ids["/peptides/healing/"]}, TemplateID: shipped.ID,
	})
	if err != nil || assigned.Changed != 1 {
		t.Fatalf("AssignTemplate = %+v %v, want the one page without it", assigned, err)
	}
	if _, err = b.service.AssignTemplate(t.Context(), pages.AssignTemplateRequest{
		SiteID: b.siteID, PageIDs: []string{"no-such-page"}, TemplateID: shipped.ID,
	}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("assigning to an unknown page = %v, want NOT_FOUND", err)
	}
}
