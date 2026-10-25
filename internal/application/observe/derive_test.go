package observe_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func (s *scenario) allPages() []pagemap.Page {
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	out := make([]pagemap.Page, 0, len(s.stores.pages))
	for id := range s.stores.pages {
		out = append(out, s.stores.pages[id])
	}
	return out
}

func TestASettingsEditDuringTheDerivationSurvives(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.stores.interleave = func() {
		s.stores.mu.Lock()
		defer s.stores.mu.Unlock()
		owner := s.stores.sites[siteID]
		owner.Settings.UserAgent = "Edited/2.0"
		owner.Settings.Observe.RenderConcurrency = 2
		s.stores.sites[siteID] = owner
	}

	s.run(t, observe.Options{})

	owner, err := s.stores.Get(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if owner.Settings.UserAgent != "Edited/2.0" || owner.Settings.Observe.RenderConcurrency != 2 {
		t.Errorf("settings after the walk = %+v, want the edit made while it ran", owner.Settings)
	}
	if owner.Settings.Observation.LastFinishedAt == nil || owner.Settings.Technical.Shop != shopID {
		t.Errorf("the walk recorded observation %+v technical %+v", owner.Settings.Observation, owner.Settings.Technical)
	}
}

func TestAPageEditDuringTheDerivationSurvives(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.stores.interleave = func() {
		s.stores.mu.Lock()
		defer s.stores.mu.Unlock()
		for id, page := range s.stores.pages {
			if page.Path == "/tb-500/" {
				page.Keywords = keyword.New([]keyword.Keyword{{Text: "edited while the walk ran"}})
				page.Title = "Edited"
				s.stores.pages[id] = page
			}
		}
	}

	s.run(t, observe.Options{})

	tb500 := s.pageAt(t, "/tb-500/")
	if len(tb500.Keywords) != 1 || tb500.Keywords[0].Text != "edited while the walk ran" || tb500.Title != "Edited" {
		t.Errorf("tb-500 after the walk = keywords %v title %q, want the edit made while it ran", tb500.Keywords, tb500.Title)
	}
	if peptides := s.pageAt(t, "/peptides/"); tb500.ParentPageID == nil || *tb500.ParentPageID != peptides.ID {
		t.Errorf("tb-500's parent = %v, want the derivation still written", tb500.ParentPageID)
	}
}

func TestTheFrontPageIsTheRootAndNotTechnical(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})

	home := s.pageAt(t, "/")
	if home.TechnicalReason != pagemap.TechnicalNone || home.ParentPageID != nil || home.ParentReason != pagemap.ParentNone {
		t.Errorf("the front page = technical %q parent %v by %s, want the root: not technical and no parent", home.TechnicalReason, home.ParentPageID, home.ParentReason)
	}
	if _, raised := s.finding(finding.TechnicalDetected, finding.SubjectPage(home.ID)); raised {
		t.Error("the front page raised technical_detected")
	}
	peptides := s.pageAt(t, "/peptides/")
	if peptides.ParentPageID == nil || *peptides.ParentPageID != home.ID || peptides.ParentReason != pagemap.ParentJSONLD || peptides.ParentMethod != pagemap.MethodHref {
		t.Errorf("peptides' parent = %v %s by %q, want the front page its trail names by its link", peptides.ParentPageID, peptides.ParentReason, peptides.ParentMethod)
	}
}

func TestArchivesAndStorePagesReadTruthfulRoots(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})

	want := map[string]render.RootState{
		"/shop/":                      render.RootArchive,
		"/product-category/peptides/": render.RootArchive,
		"/account/":                   render.RootDynamicOnly,
		"/peptides/":                  render.RootEmptyRaw,
		"/tb-500/":                    render.RootFound,
		"/product/bpc-157-peptide/":   render.RootFound,
	}
	for path, root := range want {
		if got := s.stores.renders[s.pageAt(t, path).ID].Root; got != root {
			t.Errorf("%s reads %s, want %s", path, got, root)
		}
	}
	summary, err := s.service.Summary(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Roots[string(render.RootNotFound)] != 0 {
		t.Errorf("roots = %v, want no page not found", summary.Roots)
	}
}

func TestACrumbFindsAPageByTheH1TheSiteShows(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.seedPage(t, pagemap.Page{ID: "planned-hub", Path: "/peptides/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned,
		Title: "Peptides for research", H1: "Research peptides"})
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == peptidesID {
			s.plugin.items[i].H1 = "The Peptide Shelf"
		}
	}
	s.serveTheSite(false)
	s.serve(
		publicPage{title: "The Peptide Shelf", path: "/peptides/", crumbs: []crumb{{name: "Home", path: "/"}, {name: "The Peptide Shelf", path: "/peptides/"}}, listing: []string{"/product/bpc-157-peptide/"}},
		tb500Page(false, crumb{name: "Home", path: "/"}, crumb{name: "The Peptide Shelf", bare: true}, crumb{name: "TB-500", path: "/tb-500/"}),
	)

	s.run(t, observe.Options{})

	peptides := s.pageAt(t, "/peptides/")
	if got := s.fieldOf(t, peptides.ID, field.H1); got.Current != "The Peptide Shelf" {
		t.Errorf("the hub's h1 field = %+v, want the H1 the site holds", got)
	}
	tb500 := s.pageAt(t, "/tb-500/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != peptides.ID || tb500.ParentMethod != pagemap.MethodName {
		t.Errorf("tb-500's parent = %v by %q, want the hub found by the H1 its fields hold, which neither its title nor the plan says",
			tb500.ParentPageID, tb500.ParentMethod)
	}
}

func TestAPageWinsItsNameOverAStoreCategoryAndSaysSo(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.serve(tb500Page(false, crumb{name: "Home", path: "/"}, crumb{name: "Peptides", bare: true}, crumb{name: "TB-500", path: "/tb-500/"}))

	s.run(t, observe.Options{})

	tb500 := s.pageAt(t, "/tb-500/")
	peptides := s.pageAt(t, "/peptides/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != peptides.ID || tb500.ParentMethod != pagemap.MethodName {
		t.Errorf("tb-500's parent = %v by %q, want the hub page by its name", tb500.ParentPageID, tb500.ParentMethod)
	}
	if _, ambiguous := s.finding(finding.ParentAmbiguous, finding.SubjectCrumb(tb500.ID, "Peptides")); ambiguous {
		t.Error("a page and a store category of one name raised parent_ambiguous")
	}
	raised, ok := s.finding(finding.ParentNameTwin, finding.SubjectCrumb(tb500.ID, "Peptides"))
	if !ok || raised.Status != finding.Open || raised.Severity != finding.Info {
		t.Fatalf("parent_name_twin for Peptides = %+v %t", raised, ok)
	}
	var subject struct {
		Name  string   `json:"name"`
		Page  string   `json:"page"`
		Terms []string `json:"terms"`
	}
	if err := json.Unmarshal(raised.Subject, &subject); err != nil {
		t.Fatalf("the subject: %v", err)
	}
	if subject.Name != "Peptides" || subject.Page != "/peptides/" || !slices.Equal(subject.Terms, []string{"/product-category/peptides/"}) {
		t.Errorf("the subject = %+v, want the hub taken over the store category", subject)
	}

	s.serveTheSite(false)
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{Full: true, RenderRescan: true})
	if cleared, _ := s.finding(finding.ParentNameTwin, finding.SubjectCrumb(tb500.ID, "Peptides")); cleared.Status != finding.Resolved {
		t.Errorf("parent_name_twin after the trail links its crumb = %s, want resolved", cleared.Status)
	}
}

func TestAnUnlinkedCrumbTwoPagesAnswerToIsAFindingNamingThem(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	const formsID = 77
	s.plugin.items = append(s.plugin.items, contentItem(formsID, "page", "/forms/peptides/", "Peptides", ""))
	s.plugin.raws[formsID] = rawOf("")
	s.serveTheSite(false)
	s.serve(
		publicPage{title: "Peptides", path: "/forms/peptides/"},
		tb500Page(false, crumb{name: "Home", path: "/"}, crumb{name: "Peptides", bare: true}, crumb{name: "TB-500", path: "/tb-500/"}),
	)

	s.run(t, observe.Options{})

	tb500 := s.pageAt(t, "/tb-500/")
	home := s.pageAt(t, "/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != home.ID || tb500.ParentReason != pagemap.ParentURL {
		t.Errorf("tb-500's parent = %v %s, want the URL's after the ambiguous crumb", tb500.ParentPageID, tb500.ParentReason)
	}
	raised, ok := s.finding(finding.ParentAmbiguous, finding.SubjectCrumb(tb500.ID, "Peptides"))
	if !ok || raised.Status != finding.Open {
		t.Fatalf("parent_ambiguous for Peptides = %+v %t", raised, ok)
	}
	var subject struct {
		Name       string   `json:"name"`
		Candidates []string `json:"candidates"`
	}
	if err := json.Unmarshal(raised.Subject, &subject); err != nil {
		t.Fatalf("the subject: %v", err)
	}
	slices.Sort(subject.Candidates)
	if subject.Name != "Peptides" || !slices.Equal(subject.Candidates, []string{"/forms/peptides/", "/peptides/"}) {
		t.Errorf("the subject = %+v, want the two pages and not the store category", subject)
	}
	if _, twin := s.finding(finding.ParentNameTwin, finding.SubjectCrumb(tb500.ID, "Peptides")); twin {
		t.Error("an ambiguous crumb raised parent_name_twin")
	}
}

func TestATwinFindingClearsWhenTheCycleIsBrokenThere(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.serve(
		bpcPage(false, crumb{name: "Home", path: "/"}, crumb{name: "Peptides", bare: true}, crumb{name: "BPC-157", path: "/bpc-157/"}),
		publicPage{title: "Peptides", path: "/peptides/", crumbs: []crumb{{name: "Home", path: "/"}, {name: "BPC-157", path: "/bpc-157/"}, {name: "Peptides", path: "/peptides/"}}},
	)

	s.run(t, observe.Options{})

	bpc := s.pageAt(t, "/bpc-157/")
	if bpc.ParentReason != pagemap.ParentURL {
		t.Errorf("bpc-157's parent reason = %s, want the URL's once the cycle is broken there", bpc.ParentReason)
	}
	if _, broken := s.finding(finding.ParentCycle, finding.SubjectPage(bpc.ID)); !broken {
		t.Fatal("the cycle was not broken at bpc-157")
	}
	if twin, raised := s.finding(finding.ParentNameTwin, finding.SubjectCrumb(bpc.ID, "Peptides")); raised && twin.Status == finding.Open {
		t.Error("parent_name_twin stays open on a page whose parent no longer comes from the name")
	}
}

func TestAParentCycleIsBrokenAtItsFirstPathAndNamed(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.serve(
		tb500Page(false, crumb{name: "Home", path: "/"}, crumb{name: "BPC-157", path: "/bpc-157/"}, crumb{name: "TB-500", path: "/tb-500/"}),
		bpcPage(false, crumb{name: "Home", path: "/"}, crumb{name: "TB-500", path: "/tb-500/"}, crumb{name: "BPC-157", path: "/bpc-157/"}),
	)

	s.run(t, observe.Options{})

	bpc := s.pageAt(t, "/bpc-157/")
	tb500 := s.pageAt(t, "/tb-500/")
	home := s.pageAt(t, "/")
	if tb500.ParentPageID == nil || *tb500.ParentPageID != bpc.ID {
		t.Errorf("tb-500's parent = %v, want bpc-157 by its trail", tb500.ParentPageID)
	}
	if bpc.ParentPageID == nil || *bpc.ParentPageID != home.ID || bpc.ParentReason != pagemap.ParentURL {
		t.Errorf("bpc-157's parent = %v %s, want the URL's once the cycle is broken there", bpc.ParentPageID, bpc.ParentReason)
	}
	if cycles := pagemap.Cycles(s.allPages()); len(cycles) != 0 {
		t.Errorf("the model keeps the cycles %v", cycles)
	}
	raised, ok := s.finding(finding.ParentCycle, finding.SubjectPage(bpc.ID))
	if !ok || raised.Status != finding.Open {
		t.Fatalf("parent_cycle on bpc-157 = %+v %t", raised, ok)
	}
	var subject struct {
		Paths []string `json:"paths"`
	}
	if err := json.Unmarshal(raised.Subject, &subject); err != nil || !slices.Equal(subject.Paths, []string{"/bpc-157/", "/tb-500/"}) {
		t.Errorf("the subject = %+v (%v), want both paths of the cycle", subject, err)
	}

	s.serveTheSite(false)
	s.clock.Set(start.Add(time.Hour))
	s.run(t, observe.Options{Full: true, RenderRescan: true})
	if cleared, _ := s.finding(finding.ParentCycle, finding.SubjectPage(bpc.ID)); cleared.Status != finding.Resolved {
		t.Errorf("parent_cycle once the trails no longer loop = %s, want resolved", cleared.Status)
	}
}
