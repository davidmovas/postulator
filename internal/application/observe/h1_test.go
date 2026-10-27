package observe_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func (s *scenario) takeVisits() []string {
	s.visitor.mu.Lock()
	defer s.visitor.mu.Unlock()
	visits := slices.Clone(s.visitor.visits)
	s.visitor.visits = nil
	return visits
}

func (s *scenario) serveTitlesByTheTheme() {
	s.serveTheSite(false)
	for i := range s.plugin.items {
		id := s.plugin.items[i].ID
		if rendered, ok := s.plugin.rendered[id]; ok {
			rendered.Content = strings.Replace(rendered.Content, "<h1>"+s.plugin.items[i].Title+"</h1>", "", 1)
			s.plugin.rendered[id] = rendered
		}
	}
}

func (s *scenario) renderOf(t *testing.T, pageID string) render.Record {
	t.Helper()
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	record, ok := s.stores.renders[pageID]
	if !ok {
		t.Fatalf("page %s has no render", pageID)
	}
	return record
}

func (s *scenario) h1FindingsRaised() int {
	count := 0
	open := s.openFindings(finding.FieldChangedOnSite)
	for i := range open {
		if open[i].PageID != nil && open[i].SubjectKey == finding.SubjectPageField(*open[i].PageID, string(field.H1)) {
			count++
		}
	}
	return count
}

func TestEveryPageTakesTheH1ItsRenderShows(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTitlesByTheTheme()

	s.run(t, observe.Options{Full: true, RenderRescan: true})

	for _, want := range []struct{ path, h1 string }{
		{"/tb-500/", "TB-500"},
		{"/bpc-157/", "BPC-157"},
		{"/peptides/", "Peptides"},
		{"/product/bpc-157-peptide/", "BPC-157 Peptide"},
		{"/product-category/peptides/healing-peptides/", "Healing Peptides"},
	} {
		page := s.pageAt(t, want.path)
		if got := s.fieldOf(t, page.ID, field.H1); got.Current != want.h1 {
			t.Errorf("%s: the h1 field is %q, want the render's %q", want.path, got.Current, want.h1)
		}
		if page.H1 != want.h1 {
			t.Errorf("%s: the page's h1 is %q, want it to follow the observed %q", want.path, page.H1, want.h1)
		}
		if record := s.renderOf(t, page.ID); record.H1 != want.h1 || record.H1Source != render.H1Theme {
			t.Errorf("%s: the render keeps %q from %q, want %q from the theme", want.path, record.H1, record.H1Source, want.h1)
		}
	}
	if raised := s.h1FindingsRaised(); raised != 0 {
		t.Errorf("%d field_changed_on_site findings on the h1, want none: we never write it", raised)
	}

	tb := s.pageAt(t, "/tb-500/")
	before := s.fieldOf(t, tb.ID, field.H1)
	s.walkAgain(t, time.Hour)
	if after := s.fieldOf(t, tb.ID, field.H1); after.Current != "TB-500" || after.CurrentHash != before.CurrentHash {
		t.Errorf("an incremental walk moved the h1 to %+v, want it kept as %+v", after, before)
	}
	if page := s.pageAt(t, "/tb-500/"); page.H1 != "TB-500" {
		t.Errorf("an incremental walk moved the page's h1 to %q", page.H1)
	}
}

func TestAPageWhoseRenderCannotBeFetchedTakesTheContentsH1(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTitlesByTheTheme()
	for i := range s.plugin.items {
		if s.plugin.items[i].ID == tb500ID {
			s.plugin.items[i].H1 = "TB-500 in the content"
		}
	}
	s.visitor.failed[baseURL+"/tb-500/"] = true
	s.visitor.failed[baseURL+"/peptides/"] = true

	s.run(t, observe.Options{Full: true, RenderRescan: true})

	if got := s.fieldOf(t, s.pageAt(t, "/tb-500/").ID, field.H1); got.Current != "TB-500 in the content" {
		t.Errorf("the h1 without a render is %q, want the plugin's from the content", got.Current)
	}
	if got := s.fieldOf(t, s.pageAt(t, "/peptides/").ID, field.H1); got.Current != "" {
		t.Errorf("the h1 with neither a render nor a heading in the content is %q, want none", got.Current)
	}
}

func TestAPlannedH1StaysBesideTheObservedOne(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTitlesByTheTheme()
	planned := s.seedPage(t, pagemap.Page{ID: "planned-tb", Path: "/tb-500/", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned, Title: "TB-500", H1: "TB-500 Research Peptide"})

	s.run(t, observe.Options{Full: true, RenderRescan: true})
	s.walkAgain(t, time.Hour)

	page := s.pageAt(t, "/tb-500/")
	if page.ID != planned.ID || page.H1 != "TB-500 Research Peptide" {
		t.Errorf("the planned h1 became %q on %s, want the workbook's kept", page.H1, page.ID)
	}
	if got := s.fieldOf(t, planned.ID, field.H1); got.Current != "TB-500" {
		t.Errorf("the observed h1 is %q, want the render's", got.Current)
	}
	if f, ok := s.finding(finding.PageH1Differs, finding.SubjectPage(planned.ID)); !ok || f.Status != finding.Open {
		t.Errorf("page_h1_differs = %+v %t, want the planned and the visible H1 told apart", f, ok)
	}
}

func TestARenderStoredBeforeTheH1IsFetchedAgainOnceOnAnOrdinaryWalk(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTitlesByTheTheme()
	s.run(t, observe.Options{Full: true, RenderRescan: true})

	tb, bpc := s.pageAt(t, "/tb-500/"), s.pageAt(t, "/bpc-157/")
	s.stores.mu.Lock()
	for _, pageID := range []string{tb.ID, bpc.ID} {
		record := s.stores.renders[pageID]
		record.H1, record.H1Source = "", render.H1Unknown
		s.stores.renders[pageID] = record
		values := s.stores.fields[pageID]
		h1 := field.NewValue(pageID, field.H1, "", start)
		values[field.H1] = h1
		page := s.stores.pages[pageID]
		page.H1 = ""
		s.stores.pages[pageID] = page
	}
	printed, absent := s.stores.renders[tb.ID], s.stores.renders[bpc.ID]
	printed.ThemeH1, absent.ThemeH1 = render.ThemeH1Printed, render.ThemeH1Absent
	s.stores.renders[tb.ID], s.stores.renders[bpc.ID] = printed, absent
	s.stores.mu.Unlock()
	s.takeVisits()

	s.walkAgain(t, time.Hour)

	if visits := s.takeVisits(); !slices.Equal(visits, []string{baseURL + "/tb-500/"}) {
		t.Errorf("the walk visited %v, want only the page whose theme prints an H1 the store never kept", visits)
	}
	if got := s.fieldOf(t, tb.ID, field.H1); got.Current != "TB-500" {
		t.Errorf("the h1 after the walk is %q, want the theme's", got.Current)
	}
	if page := s.pageAt(t, "/tb-500/"); page.H1 != "TB-500" {
		t.Errorf("the page's h1 after the walk is %q, want it to follow", page.H1)
	}

	s.walkAgain(t, 2*time.Hour)
	if visits := s.takeVisits(); len(visits) != 0 {
		t.Errorf("a third walk visited %v, want none: the H1 is stored now", visits)
	}
}
