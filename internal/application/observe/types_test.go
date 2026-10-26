package observe_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
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

func (s *scenario) ruleOf(t *testing.T, path string) int {
	t.Helper()
	page := s.pageAt(t, path)
	rules, err := s.stores.types.Rules(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Rules: %v", err)
	}
	for i := range rules {
		if rules[i].ID == page.TypeRuleID {
			return rules[i].Position
		}
	}
	return 0
}

func TestTheFirstObservationTypesEveryPageByTheSitesRules(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})

	set, err := s.stores.types.Set(t.Context(), siteID)
	if err != nil || len(set.Types()) != len(pagetype.Seeded()) {
		t.Fatalf("the site's types = %d %v, want the presets seeded", len(set.Types()), err)
	}
	want := map[string]struct {
		preset pagetype.Preset
		rule   int
	}{
		"/account/":                   {pagetype.Technical, 1},
		"/product/bpc-157-peptide/":   {pagetype.Product, 2},
		"/product-category/peptides/": {pagetype.StoreCategory, 3},
		"/shop/":                      {pagetype.Catalog, 4},
		"/":                           {pagetype.Landing, 5},
		"/peptides/":                  {pagetype.Landing, 6},
		"/tb-500/":                    {pagetype.Category, 7},
	}
	for path, expected := range want {
		preset, source, reason := s.typeOf(t, path)
		if preset != expected.preset || source != pagemap.TypeSourceAuto || reason != pagemap.TypeByRule || s.ruleOf(t, path) != expected.rule {
			t.Errorf("%s = %s %s %s by rule %d, want %s by rule %d", path, preset, source, reason, s.ruleOf(t, path), expected.preset, expected.rule)
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
	if preset, _, reason := s.typeOf(t, "/peptides/"); preset != pagetype.Landing || reason != pagemap.TypeByRule || s.ruleOf(t, "/peptides/") != 6 {
		t.Errorf("the hub once rendered = %s %s, want Landing by its listing at the top level", preset, reason)
	}
}

func TestAPageTheRulesPlacedFollowsThemAndAPersonsOrAnOlderTypeStays(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})

	home, shop := s.pageAt(t, "/"), s.pageAt(t, "/shop/")
	s.stores.setType(home.ID, siteID+"-category", pagemap.TypeByUser)
	s.stores.setLegacyType(shop.ID, siteID+"-catalog", pagemap.TypeByShop)
	s.serve(publicPage{title: "TB-500", path: "/tb-500/", content: "<p>TB-500 is a synthetic fragment of thymosin beta-4.</p>"})

	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{Full: true, RenderRescan: true})
	if preset, source, reason := s.typeOf(t, "/"); preset != pagetype.Category || source != pagemap.TypeSourceManual || reason != pagemap.TypeByUser {
		t.Errorf("the front page a person typed = %s %s %s, want it kept", preset, source, reason)
	}
	if preset, _, reason := s.typeOf(t, "/shop/"); preset != pagetype.Catalog || reason != pagemap.TypeByShop {
		t.Errorf("the shop typed before 2.6 = %s %s, want it left until a person applies the rules", preset, reason)
	}
	tb500 := s.pageAt(t, "/tb-500/")
	if preset, source, reason := s.typeOf(t, "/tb-500/"); preset != pagetype.Info || source != pagemap.TypeSourceAuto || reason != pagemap.TypeByUnmatched || tb500.TypeRuleID != "" {
		t.Errorf("tb-500 without its listing = %s %s %s, want the rules to place it again: no rule matches, so Info", preset, source, reason)
	}
	if held, _ := s.finding(finding.TypeByDefault, finding.SubjectPage(tb500.ID)); held.Status != finding.Open {
		t.Errorf("type_by_default on a page no rule matched = %+v, want it open for the client to review", held)
	}

	s.stores.types.changeRule(siteID, 10, func(rule *pagetype.Rule) {
		rule.Conditions.URLs = append(rule.Conditions.URLs, pagetype.URLMatch{Mode: pagetype.URLSegment, Value: "tb"})
	})
	s.clock.Set(start.Add(2 * time.Hour))
	s.run(t, observe.Options{Full: true})
	if preset, _, reason := s.typeOf(t, "/tb-500/"); preset != pagetype.Guide || reason != pagemap.TypeByRule || s.ruleOf(t, "/tb-500/") != 10 {
		t.Errorf("tb-500 after its rule changed = %s %s by rule %d, want the next sync to follow the rule", preset, reason, s.ruleOf(t, "/tb-500/"))
	}
	if held, _ := s.finding(finding.TypeByDefault, finding.SubjectPage(tb500.ID)); held.Status == finding.Open {
		t.Errorf("type_by_default once a rule matched = %+v, want it closed", held)
	}
}

func TestAPageAPersonTypedAsRegularIsNeverDetectedTechnicalAgain(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})

	account := s.pageAt(t, "/account/")
	if account.TechnicalReason != pagemap.TechnicalShortcode {
		t.Fatalf("the account page = %q, want it detected technical first", account.TechnicalReason)
	}
	shop := s.pageAt(t, "/shop/")
	s.stores.setType(account.ID, siteID+"-guide", pagemap.TypeByUser)
	s.stores.setType(shop.ID, siteID+"-technical", pagemap.TypeByUser)

	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{Full: true, RenderRescan: true})
	if after := s.pageAt(t, "/account/"); after.TechnicalReason != pagemap.TechnicalNone {
		t.Errorf("the account page a person typed Entity = %q, want no technical reason", after.TechnicalReason)
	}
	if held, _ := s.finding(finding.TechnicalDetected, finding.SubjectPage(account.ID)); held.Status == finding.Open {
		t.Errorf("technical_detected on the page a person typed = %+v, want it closed", held)
	}
	if after := s.pageAt(t, "/shop/"); after.TechnicalReason != pagemap.TechnicalShop {
		t.Errorf("the shop a person typed Technical = %q, want its detection kept", after.TechnicalReason)
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
		s.stores.setType(tb500.ID, siteID+"-guide", pagemap.TypeByUser)
	}
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{})

	if preset, source, reason := s.typeOf(t, "/tb-500/"); preset != pagetype.Guide || source != pagemap.TypeSourceManual || reason != pagemap.TypeByUser {
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
			PageTypeID: siteID + "-landing", Key: field.Instance(field.Meta, key), Enabled: true, Label: key,
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
		PageTypeID: siteID + "-landing", Key: field.Instance(field.Meta, "storage"), Enabled: true, Label: "Storage",
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
