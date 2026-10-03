package steps_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func seedCategoryEntity(t *testing.T, h *syncHarness, name string, scope *string, flagged bool) graph.Entity {
	t.Helper()

	record := graph.Entity{
		ID: id.New(), SiteID: h.siteID, Name: name, Kind: graph.KindCategory, SiteCategory: flagged, ScopeID: scope,
		Source: graph.SourceImport, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := sqlite.NewEntityRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the entity %s: %v", name, err)
	}
	return record
}

func keepTerm(t *testing.T, h *syncHarness, held graph.Term) {
	t.Helper()

	held.SiteID = h.siteID
	held.SeenAt = sqlitetest.Stamp
	if err := sqlite.NewTermRepo(h.store).Upsert(t.Context(), held); err != nil {
		t.Fatalf("keep the term of %s: %v", held.EntityID, err)
	}
}

func termsKeptBy(t *testing.T, h *syncHarness) map[string]graph.Term {
	t.Helper()

	listed, err := sqlite.NewTermRepo(h.store).ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the terms: %v", err)
	}
	held := make(map[string]graph.Term, len(listed))
	for i := range listed {
		held[termKey(listed[i].EntityID, listed[i].Taxonomy)] = listed[i]
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
	shelf := h.server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Drinks"})[0]

	root := seedCategoryEntity(t, h, "Drinks", nil, true)
	sub := seedCategoryEntity(t, h, "Coffee & More", &root.ID, true)
	tea := seedCategoryEntity(t, h, "Tea", &root.ID, true)
	juice := seedCategoryEntity(t, h, "Juice", nil, true)
	seedCategoryEntity(t, h, "Espresso", &sub.ID, false)
	keepTerm(t, h, graph.Term{EntityID: root.ID, Taxonomy: graph.TaxonomyCategory, TermID: drinks.ID, Name: "Drinks", RunID: "earlier-run"})
	keepTerm(t, h, graph.Term{EntityID: juice.ID, Taxonomy: graph.TaxonomyCategory, TermID: 999, Name: "Juice", RunID: "earlier-run"})

	state := h.all(t)
	if len(state.Findings) != 0 {
		t.Errorf("the sync found %+v", state.Findings)
	}

	held := termsKeptBy(t, h)
	cases := []struct {
		name     string
		entityID string
		taxonomy graph.Taxonomy
		termID   int64
		parent   int64
		runID    string
	}{
		{name: "a root kept from an earlier run", entityID: root.ID, taxonomy: graph.TaxonomyCategory, termID: drinks.ID, runID: "earlier-run"},
		{name: "a subcategory matched under its parent", entityID: sub.ID, taxonomy: graph.TaxonomyCategory, termID: coffee.ID, parent: drinks.ID},
		{name: "a root matched among the store's categories", entityID: root.ID, taxonomy: graph.TaxonomyProductCategory, termID: shelf.ID},
		{name: "a category the site lacks", entityID: tea.ID, taxonomy: graph.TaxonomyCategory},
		{name: "a category whose term was deleted", entityID: juice.ID, taxonomy: graph.TaxonomyCategory},
		{name: "a subcategory the store lacks", entityID: sub.ID, taxonomy: graph.TaxonomyProductCategory},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kept, found := held[termKey(tc.entityID, tc.taxonomy)]
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

func TestSyncSiteAsksForNoTermsWhenTheGraphFilesNothing(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	seedSite(t, h)
	h.server.SeedCategory(wptest.Category{Name: "Drinks"})
	seedCategoryEntity(t, h, "Drinks", nil, false)

	h.all(t)
	for _, request := range h.server.Requests() {
		if request.Path == "/wp-json/wp/v2/categories" || request.Path == "/wp-json/wc/v3/products/categories" {
			t.Errorf("the sync asked %s %s of a graph that files nothing", request.Method, request.Path)
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
		{name: "no term store", missing: func(deps *steps.Deps) { deps.Terms = nil }, says: "term store"},
		{name: "no entity reader", missing: func(deps *steps.Deps) { deps.Entities = nil }, says: "entity reader"},
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
	root := seedCategoryEntity(t, h, "Drinks", nil, true)
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
	if kept, found := held[termKey(root.ID, graph.TaxonomyCategory)]; !found || kept.TermID != drinks.ID {
		t.Errorf("the sync keeps %+v, want the post categories matched whatever the store refused", held)
	}
}
