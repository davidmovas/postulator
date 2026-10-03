package steps_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func seedEntityNamed(t *testing.T, h *syncHarness, name string) graph.Entity {
	t.Helper()

	record := graph.Entity{
		ID: id.New(), SiteID: h.siteID, Name: name, Kind: graph.KindHub, Source: graph.SourceImport,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := sqlite.NewEntityRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the entity %s: %v", name, err)
	}
	return record
}

func keepTerm(t *testing.T, h *syncHarness, held category.Term) {
	t.Helper()

	held.SiteID = h.siteID
	held.SeenAt = sqlitetest.Stamp
	if err := sqlite.NewCategoryTermRepo(h.store).Upsert(t.Context(), held); err != nil {
		t.Fatalf("keep the term of %s: %v", held.CategoryID, err)
	}
}

func termsKeptBy(t *testing.T, h *syncHarness) map[string]category.Term {
	t.Helper()

	listed, err := sqlite.NewCategoryTermRepo(h.store).ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the terms: %v", err)
	}
	held := make(map[string]category.Term, len(listed))
	for i := range listed {
		held[termKey(listed[i].CategoryID, listed[i].Taxonomy)] = listed[i]
	}
	return held
}

func TestSyncSiteMatchesTheCategoriesAlreadyOnTheSite(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 1)
	seedSite(t, h)
	drinks := h.server.SeedCategory(wptest.Category{Name: "Drinks"})
	coffee := h.server.SeedCategory(wptest.Category{Name: "coffee & more", Parent: drinks.ID})
	h.server.SeedCategory(wptest.Category{Name: "Coffee & More"})
	h.server.SeedCategory(wptest.Category{Name: "Ristretto"})
	h.server.SeedCategory(wptest.Category{Name: "Espresso", Parent: drinks.ID})
	shelf := h.server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Drinks"})[0]

	root := storedCategory(t, h.store, h.siteID, "Drinks", "")
	sub := storedCategory(t, h.store, h.siteID, "Coffee  &  More", root.ID)
	tea := storedCategory(t, h.store, h.siteID, "Tea", root.ID)
	juice := storedCategory(t, h.store, h.siteID, "Juice", "")
	lost := storedCategory(t, h.store, h.siteID, "Ristretto", tea.ID)
	seedEntityNamed(t, h, "Espresso")
	keepTerm(t, h, category.Term{CategoryID: root.ID, Taxonomy: category.TaxonomyCategory, TermID: drinks.ID, Name: "Drinks", RunID: "earlier-run"})
	keepTerm(t, h, category.Term{CategoryID: juice.ID, Taxonomy: category.TaxonomyCategory, TermID: 999, Name: "Juice", RunID: "earlier-run"})

	state := h.all(t)
	if len(state.Findings) != 0 {
		t.Errorf("the sync found %+v", state.Findings)
	}

	held := termsKeptBy(t, h)
	cases := []struct {
		name       string
		categoryID string
		taxonomy   category.Taxonomy
		termID     int64
		parent     int64
		runID      string
	}{
		{name: "a root kept from an earlier run", categoryID: root.ID, taxonomy: category.TaxonomyCategory, termID: drinks.ID, runID: "earlier-run"},
		{name: "a subcategory matched by its key under its parent", categoryID: sub.ID, taxonomy: category.TaxonomyCategory, termID: coffee.ID, parent: drinks.ID},
		{name: "a root matched among the store's categories", categoryID: root.ID, taxonomy: category.TaxonomyProductCategory, termID: shelf.ID},
		{name: "a category the site lacks", categoryID: tea.ID, taxonomy: category.TaxonomyCategory},
		{name: "a category under one the site lacks", categoryID: lost.ID, taxonomy: category.TaxonomyCategory},
		{name: "a category whose term was deleted", categoryID: juice.ID, taxonomy: category.TaxonomyCategory},
		{name: "a subcategory the store lacks", categoryID: sub.ID, taxonomy: category.TaxonomyProductCategory},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kept, found := held[termKey(tc.categoryID, tc.taxonomy)]
			if tc.termID == 0 {
				if found {
					t.Fatalf("the sync keeps %+v, want no term", kept)
				}
				return
			}
			if !found || kept.TermID != tc.termID || kept.ParentTermID != tc.parent || kept.RunID != tc.runID {
				t.Fatalf("the sync keeps %+v (%t), want %d under %d from %q", kept, found, tc.termID, tc.parent, tc.runID)
			}
		})
	}
	if len(held) != 3 {
		t.Errorf("the sync keeps %d terms, want three: %+v", len(held), held)
	}
}

func TestSyncSiteAsksForNoTermsWhenTheSiteHasNoCategories(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	seedSite(t, h)
	h.server.SeedCategory(wptest.Category{Name: "Drinks"})
	seedEntityNamed(t, h, "Drinks")

	h.all(t)
	for _, request := range h.server.Requests() {
		if request.Path == "/wp-json/wp/v2/categories" || request.Path == "/wp-json/wc/v3/products/categories" {
			t.Errorf("the sync asked %s %s of a site with no categories", request.Method, request.Path)
		}
	}
	if held := termsKeptBy(t, h); len(held) != 0 {
		t.Errorf("the sync keeps %+v", held)
	}
}

func TestSyncSiteNamesTheStoreItWasNotGiven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		missing func(*steps.Deps)
		says    string
	}{
		{name: "no category term store", missing: func(deps *steps.Deps) { deps.CategoryTerms = nil }, says: "term store"},
		{name: "no category reader", missing: func(deps *steps.Deps) { deps.Categories = nil }, says: "category reader"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newSyncHarness(t, 0)
			seedSite(t, h)
			tc.missing(&h.deps)

			sc := &run.StepContext{
				Run:       run.Run{ID: "run", SiteID: h.siteID, Kind: run.KindSync},
				Item:      run.Item{ID: "item", RunID: "run", SiteID: h.siteID, TargetID: h.siteID},
				Params:    map[string]any{},
				Artifacts: map[run.ArtifactKind]run.Artifact{},
				Check:     run.NewCheckpoint(),
			}
			_, err := steps.SyncSite(h.deps).Run(t.Context(), sc)
			if !errors.IsCode(err, errors.Internal) || !strings.Contains(err.Error(), tc.says) {
				t.Fatalf("SyncSite = %v, want an error naming the missing %s", err, tc.says)
			}
		})
	}
}

func TestSyncSiteSaysWhichCategoriesTheSiteWouldNotList(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	seedSite(t, h)
	drinks := h.server.SeedCategory(wptest.Category{Name: "Drinks"})
	root := storedCategory(t, h.store, h.siteID, "Drinks", "")
	h.deps.WordPress = oneClient{client: behind(t, h.server, func(w http.ResponseWriter, r *http.Request, forward http.Handler) {
		if r.URL.Path == "/wp-json/wc/v3/products/categories" {
			failWith(t, w, http.StatusForbidden)
			return
		}
		forward.ServeHTTP(w, r)
	})}

	state := h.all(t)
	if len(state.Findings) != 1 || state.Findings[0].Code != steps.CodeCategoriesUnread {
		t.Fatalf("findings = %+v, want one %s", state.Findings, steps.CodeCategoriesUnread)
	}
	held := termsKeptBy(t, h)
	if kept, found := held[termKey(root.ID, category.TaxonomyCategory)]; !found || kept.TermID != drinks.ID {
		t.Errorf("the sync keeps %+v, want the post categories matched whatever the store refused", held)
	}
}
