package observe_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (s *scenario) typeOf(t *testing.T, path string) (pagetype.Preset, pagemap.TypeSource, pagemap.TypeReason) {
	t.Helper()
	page := s.pageAt(t, path)
	if page.PageTypeID == nil {
		return "", page.TypeSource, page.TypeReason
	}
	set, err := s.stores.types.Set(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	kind, found := set.ByID(*page.PageTypeID)
	if !found {
		t.Fatalf("%s carries the unknown type %s", path, *page.PageTypeID)
	}
	return kind.Preset, page.TypeSource, page.TypeReason
}

func TestTheFirstObservationSeedsTheTypesAndTypesEveryPage(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})

	set, err := s.stores.types.Set(t.Context(), siteID)
	if err != nil || len(set.Types()) != len(pagetype.Presets()) {
		t.Fatalf("the site's types = %d %v, want the presets seeded", len(set.Types()), err)
	}
	want := map[string]struct {
		preset pagetype.Preset
		reason pagemap.TypeReason
	}{
		"/product/bpc-157-peptide/":   {pagetype.Product, pagemap.TypeByProduct},
		"/product-category/peptides/": {pagetype.StoreCategory, pagemap.TypeByStoreCategory},
		"/shop/":                      {pagetype.Catalog, pagemap.TypeByShop},
		"/account/":                   {pagetype.Technical, pagemap.TypeByTechnical},
		"/peptides/":                  {pagetype.Hub, pagemap.TypeByListing},
		"/tb-500/":                    {pagetype.Hub, pagemap.TypeByListing},
		"/":                           {pagetype.Entity, pagemap.TypeByDefault},
	}
	for path, expected := range want {
		preset, source, reason := s.typeOf(t, path)
		if preset != expected.preset || source != pagemap.TypeSourceAuto || reason != expected.reason {
			t.Errorf("%s = %s %s %s, want %s auto %s", path, preset, source, reason, expected.preset, expected.reason)
		}
	}
}

func TestARegularPageWaitsForItsRenderBeforeItIsTyped(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.visitor.failed[baseURL+"/peptides/"] = true

	s.run(t, observe.Options{})
	if preset, source, _ := s.typeOf(t, "/peptides/"); preset != "" || source != pagemap.TypeSourceNone {
		t.Fatalf("a hub whose render failed = %s %s, want it left untyped", preset, source)
	}
	if preset, _, _ := s.typeOf(t, "/shop/"); preset != pagetype.Catalog {
		t.Errorf("the shop = %s, want it typed by its identity without waiting", preset)
	}

	delete(s.visitor.failed, baseURL+"/peptides/")
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{Full: true, RenderRescan: true})
	if preset, _, reason := s.typeOf(t, "/peptides/"); preset != pagetype.Hub || reason != pagemap.TypeByListing {
		t.Errorf("the hub once rendered = %s %s, want Hub by its listing", preset, reason)
	}
}

func TestATypedPageIsNeverRetypedByTheObservation(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})

	home := s.pageAt(t, "/")
	tb500 := s.pageAt(t, "/tb-500/")
	s.stores.setType(home.ID, siteID+"-hub", pagemap.TypeByUser)
	s.serve(publicPage{title: "TB-500", path: "/tb-500/", content: "<p>TB-500 is a synthetic fragment of thymosin beta-4.</p>"})

	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{Full: true, RenderRescan: true})
	if preset, source, reason := s.typeOf(t, "/"); preset != pagetype.Hub || source != pagemap.TypeSourceManual || reason != pagemap.TypeByUser {
		t.Errorf("the front page a person typed = %s %s %s, want it kept", preset, source, reason)
	}
	if preset, source, _ := s.typeOf(t, "/tb-500/"); preset != pagetype.Hub || source != pagemap.TypeSourceAuto {
		t.Errorf("tb-500 typed before = %s %s, want its first type kept", preset, source)
	}
	if tb500.PageTypeID == nil {
		t.Fatal("tb-500 was not typed by the first walk")
	}
}

func TestAManualTypeSetDuringTheWalkSurvives(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})

	tb500 := s.pageAt(t, "/tb-500/")
	s.touch(tb500ID, start.Add(30*time.Minute))
	s.stores.interleave = func() {
		s.stores.setType(tb500.ID, siteID+"-entity", pagemap.TypeByUser)
	}
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})

	if preset, source, reason := s.typeOf(t, "/tb-500/"); preset != pagetype.Entity || source != pagemap.TypeSourceManual || reason != pagemap.TypeByUser {
		t.Errorf("tb-500 after a person typed it during the walk = %s %s %s, want the person's type", preset, source, reason)
	}
}

func TestACustomFieldAScopeNamesIsReadAndAnAbsentOneIsNotAskedAgain(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.plugin.custom = map[int64]map[string]string{peptidesID: {"spec_sheet": "Lyophilized powder, 5 mg"}}
	for _, key := range []string{"spec_sheet", "dosage"} {
		s.stores.types.add(field.Scope{
			PageTypeID: siteID + "-hub", Key: field.Instance(field.Meta, key), Enabled: true, Label: key,
			Provider: field.ProviderPlugin, Links: field.NoLinks(),
		})
	}
	if err := s.stores.types.Seed(t.Context(), siteID); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	s.run(t, observe.Options{})
	peptides := s.pageAt(t, "/peptides/")
	if got := s.fieldOf(t, peptides.ID, field.Instance(field.Meta, "spec_sheet")); got.Current != "Lyophilized powder, 5 mg" || got.Base != got.Current {
		t.Errorf("spec_sheet = %+v, want the site's value as current and base", got)
	}
	if got := s.fieldOf(t, peptides.ID, field.Instance(field.Meta, "dosage")); got.Current != "" || got.ObservedAt.IsZero() {
		t.Errorf("an absent key = %+v, want it stored empty", got)
	}
	asked := len(s.plugin.askedFor(peptidesID))
	if asked == 0 {
		t.Fatal("the plugin was never asked for the custom fields")
	}

	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})
	if again := len(s.plugin.askedFor(peptidesID)); again != asked {
		t.Errorf("an unchanged page was asked for its custom fields again: %d keys, then %d", asked, again)
	}

	s.stores.types.add(field.Scope{
		PageTypeID: siteID + "-hub", Key: field.Instance(field.Meta, "storage"), Enabled: true, Label: "Storage",
		Provider: field.ProviderStore, Links: field.NoLinks(),
	})
	s.plugin.custom[peptidesID]["storage"] = "Keep frozen"
	before := len(s.plugin.askedFor(peptidesID))
	s.clock.Set(start.Add(2 * time.Hour))
	s.run(t, observe.Options{})
	if got := s.fieldOf(t, peptides.ID, field.Instance(field.Meta, "storage")); got.Current != "Keep frozen" {
		t.Errorf("a key added to the scope = %+v, want it read on the next observation", got)
	}
	if last := s.plugin.askedFor(peptidesID)[before:]; !slices.Equal(last, []string{"storage"}) {
		t.Errorf("the keys asked on that walk = %v, want only the new one", last)
	}
}
