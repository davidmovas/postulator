package observe_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func (s *scenario) underOnSite(id, parent int64) {
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == id {
			s.plugin.items[i].Parent = parent
		}
	}
}

func (s *scenario) expect(pageID string, trail ...string) {
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	page := s.stores.pages[pageID]
	page.PlannedTrail = trail
	s.stores.pages[pageID] = page
}

func (s *scenario) dismiss(code finding.Code, subject string) {
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	key := findingKey(siteID, code, subject)
	held := s.stores.findings[key]
	held.Status = finding.Dismissed
	s.stores.findings[key] = held
}

func (s *scenario) walkAgain(t *testing.T, after time.Duration) {
	t.Helper()
	s.clock.Set(start.Add(after))
	s.run(t, observe.Options{})
}

func (s *scenario) previewSubject(t *testing.T, pageID string) string {
	t.Helper()
	fields, err := fieldStore{s.stores}.ListBySite(t.Context(), siteID)
	if err != nil {
		t.Fatalf("list the fields: %v", err)
	}
	pages := s.allPages()
	site := pagemap.NewTrailSite(pagemap.NewIndexNamed(pages, pagemap.Names(fields)), nil, nil)
	diff, _ := pagemap.TrailDiffers(s.stores.pages[pageID], site)
	encoded, err := json.Marshal(diff)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(encoded)
}

func TestTheObservationRaisesATrailTheSiteDoesNotShowInThePreviewsBytes(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.underOnSite(tb500ID, peptidesID)
	s.seedPage(t, pagemap.Page{ID: "tb", Path: "/tb-500/", WPType: pagemap.WPPage, WPID: new(tb500ID), Status: pagemap.StatusPublished,
		PlannedTrail: []string{"Peptides", "BPC-157"}})

	s.run(t, observe.Options{})

	raised, ok := s.finding(finding.ParentDiffers, finding.SubjectPage("tb"))
	if !ok || raised.Status != finding.Open {
		t.Fatalf("parent_differs = %+v %t", raised, ok)
	}
	if want := `{"sheet":["Peptides","BPC-157"],"site":["Peptides"]}`; string(raised.Subject) != want {
		t.Errorf("subject = %s, want %s", raised.Subject, want)
	}
	if preview := s.previewSubject(t, "tb"); string(raised.Subject) != preview {
		t.Errorf("subject = %s, want the bytes the preview computes over the same model %s", raised.Subject, preview)
	}
	if !slices.Equal(raised.Options, []finding.Option{finding.AcceptSite, finding.AcceptWorkbook, finding.Ignore}) {
		t.Errorf("options = %v, want the site, the workbook or neither", raised.Options)
	}
	peptides := s.pageAt(t, "/peptides/")
	if tb := s.stores.pages["tb"]; tb.ParentPageID == nil || *tb.ParentPageID != peptides.ID || tb.ParentReason != pagemap.ParentWP {
		t.Errorf("tb-500's parent = %v %s, want the site's: the sheet's trail wins nothing on its own", tb.ParentPageID, tb.ParentReason)
	}
}

func TestATrailFindingClearsWhenTheSiteAgreesAndReopensWhenItDiffersAgain(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.underOnSite(tb500ID, peptidesID)
	s.seedPage(t, pagemap.Page{ID: "tb", Path: "/tb-500/", WPType: pagemap.WPPage, WPID: new(tb500ID), Status: pagemap.StatusPublished,
		PlannedTrail: []string{"Blends"}})
	s.run(t, observe.Options{})

	steps := []struct {
		name   string
		trail  []string
		status finding.Status
	}{
		{name: "the site shows the sheet's trail", trail: []string{"PEPTIDES"}, status: finding.Resolved},
		{name: "the sheet moves the page again", trail: []string{"Blends"}, status: finding.Open},
		{name: "the sheet expects nothing", status: finding.Resolved},
	}
	for at, step := range steps {
		s.expect("tb", step.trail...)
		s.walkAgain(t, time.Duration(at+1)*time.Hour)
		got, ok := s.finding(finding.ParentDiffers, finding.SubjectPage("tb"))
		if !ok || got.Status != step.status {
			t.Fatalf("%s: parent_differs = %+v %t, want %s", step.name, got, ok, step.status)
		}
		if step.status == finding.Resolved && (got.ResolvedBy != finding.ResolvedByObservation || got.Resolution != finding.ResolutionObservedClear) {
			t.Errorf("%s: cleared by %q as %q, want the observation's clear", step.name, got.ResolvedBy, got.Resolution)
		}
	}
}

func TestADismissedTrailFindingStaysDismissedOnTheNextWalk(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.underOnSite(tb500ID, peptidesID)
	s.seedPage(t, pagemap.Page{ID: "tb", Path: "/tb-500/", WPType: pagemap.WPPage, WPID: new(tb500ID), Status: pagemap.StatusPublished,
		PlannedTrail: []string{"Blends"}})
	s.run(t, observe.Options{})
	if raised, ok := s.finding(finding.ParentDiffers, finding.SubjectPage("tb")); !ok || raised.Status != finding.Open {
		t.Fatalf("parent_differs = %+v %t, want it raised before a person dismisses it", raised, ok)
	}
	s.dismiss(finding.ParentDiffers, finding.SubjectPage("tb"))

	s.walkAgain(t, time.Hour)
	s.clock.Set(start.Add(2 * time.Hour))
	s.run(t, observe.Options{Full: true})

	if got, _ := s.finding(finding.ParentDiffers, finding.SubjectPage("tb")); got.Status != finding.Dismissed {
		t.Fatalf("parent_differs after two more walks = %s (%s), want it kept dismissed while its facts stand", got.Status, got.Subject)
	}
}

func TestAProductTrailIsComparedWithItsCategories(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		trail   []string
		subject string
	}{
		{name: "the categories the store files it under", trail: []string{"Peptides", "Healing Peptides"}},
		{
			name: "a category the store does not file it under", trail: []string{"Peptides", "Blends"},
			subject: `{"sheet":["Peptides","Blends"],"site":["Peptides","Healing Peptides"]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newScenario(t)
			s.seedPage(t, pagemap.Page{ID: "vial", Path: "/product/bpc-157-peptide/", WPType: pagemap.WPProduct, WPID: new(productID),
				Status: pagemap.StatusPublished, PlannedTrail: tc.trail})

			s.run(t, observe.Options{})

			raised, ok := s.finding(finding.ParentDiffers, finding.SubjectPage("vial"))
			switch {
			case tc.subject == "" && ok && raised.Status == finding.Open:
				t.Fatalf("parent_differs = %s, want none: the store files the product as the sheet does", raised.Subject)
			case tc.subject == "":
			case !ok || raised.Status != finding.Open || string(raised.Subject) != tc.subject:
				t.Fatalf("parent_differs = %+v %t, want %s", raised, ok, tc.subject)
			case !slices.Equal(raised.Options, []finding.Option{finding.AcceptSite, finding.Ignore}):
				t.Fatalf("options = %v, want the store's categories or neither: the workbook files no product", raised.Options)
			}
		})
	}
}

func TestAWorkbookParentSourceKeepsTheTrailsNearestLevelAsTheModelParent(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.underOnSite(tb500ID, bpcID)
	s.seedPage(t, pagemap.Page{ID: "tb", Path: "/tb-500/", WPType: pagemap.WPPage, WPID: new(tb500ID), Status: pagemap.StatusPublished,
		ParentSource: pagemap.ParentFromWorkbook, PlannedTrail: []string{"Peptides"}})

	s.run(t, observe.Options{})
	s.walkAgain(t, time.Hour)

	peptides := s.pageAt(t, "/peptides/")
	tb := s.stores.pages["tb"]
	if tb.ParentPageID == nil || *tb.ParentPageID != peptides.ID || tb.ParentReason != pagemap.ParentWorkbook || tb.ParentMethod != pagemap.MethodName {
		t.Fatalf("tb-500's parent = %v %s by %q, want the page the trail's nearest level names, by the workbook", tb.ParentPageID, tb.ParentReason, tb.ParentMethod)
	}
	if got, ok := s.finding(finding.ParentDiffers, finding.SubjectPage("tb")); ok && got.Status == finding.Open {
		t.Errorf("parent_differs = %s, want none: the model now follows the sheet", got.Subject)
	}
}

func TestATieUnderTheWorkbookParentSourceKeepsTheDerivedParentAndNamesTheCandidates(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	const formsID = 77
	s.plugin.items = append(s.plugin.items, contentItem(formsID, "page", "/forms/peptides/", "Peptides", ""))
	s.plugin.raws[formsID] = rawOf("")
	s.underOnSite(tb500ID, bpcID)
	s.seedPage(t, pagemap.Page{ID: "tb", Path: "/tb-500/", WPType: pagemap.WPPage, WPID: new(tb500ID), Status: pagemap.StatusPublished,
		ParentSource: pagemap.ParentFromWorkbook, PlannedTrail: []string{"Peptides"}})

	s.run(t, observe.Options{})

	bpc := s.pageAt(t, "/bpc-157/")
	if tb := s.stores.pages["tb"]; tb.ParentPageID == nil || *tb.ParentPageID != bpc.ID || tb.ParentReason != pagemap.ParentWP {
		t.Fatalf("tb-500's parent = %v %s, want the derived parent kept on a tie", tb.ParentPageID, tb.ParentReason)
	}
	raised, ok := s.finding(finding.ParentAmbiguous, finding.SubjectCrumb("tb", "Peptides"))
	if !ok || raised.Status != finding.Open {
		t.Fatalf("parent_ambiguous = %+v %t", raised, ok)
	}
	if want := `{"candidates":["/forms/peptides/","/peptides/"],"name":"Peptides"}`; string(raised.Subject) != want {
		t.Errorf("subject = %s, want %s", raised.Subject, want)
	}
	if differs, held := s.finding(finding.ParentDiffers, finding.SubjectPage("tb")); !held || string(differs.Subject) != `{"sheet":["Peptides"],"site":["BPC-157"]}` {
		t.Errorf("parent_differs = %+v %t, want the sheet's level beside the site's", differs, held)
	}

	s.walkAgain(t, time.Hour)
	if again, _ := s.finding(finding.ParentAmbiguous, finding.SubjectCrumb("tb", "Peptides")); again.Status != finding.Open || again.ID != raised.ID {
		t.Errorf("parent_ambiguous on the next walk = %+v, want the same finding still open", again)
	}
}
