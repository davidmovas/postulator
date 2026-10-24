package observe_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/finding"
)

func (s *scenario) carry(t *testing.T, path string, entityID *string) {
	t.Helper()
	page := s.pageAt(t, path)
	page.EntityID = entityID
	s.stores.mu.Lock()
	s.stores.pages[page.ID] = page
	s.stores.mu.Unlock()
}

func (s *scenario) withoutEntity() []string {
	open := s.openFindings(finding.PageWithoutEntity)
	out := make([]string, 0, len(open))
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	for i := range open {
		if open[i].PageID != nil {
			out = append(out, s.stores.pages[*open[i].PageID].Path)
		}
	}
	slices.Sort(out)
	return out
}

func TestAWritablePageNoEntityCarriesIsAFindingUntilOneDoes(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	s.run(t, observe.Options{})
	want := []string{"/bpc-157/", "/peptides/", "/product/bpc-157-peptide/", "/tb-500/"}
	if got := s.withoutEntity(); !slices.Equal(got, want) {
		t.Fatalf("the pages without an entity = %v, want %v", got, want)
	}
	peptides := s.pageAt(t, "/peptides/")
	raised, _ := s.finding(finding.PageWithoutEntity, finding.SubjectPage(peptides.ID))
	if string(raised.Subject) != `{"path":"/peptides/"}` || !slices.Equal(raised.Options, []finding.Option{finding.ProposeEntity, finding.Ignore}) {
		t.Errorf("the finding = %s %v, want the path alone and its two options", raised.Subject, raised.Options)
	}

	s.carry(t, "/peptides/", new("entity-peptides"))
	tb500 := s.pageAt(t, "/tb-500/")
	s.dismiss(finding.PageWithoutEntity, finding.SubjectPage(tb500.ID))
	for i := range s.plugin.items {
		if s.plugin.items[i].Path == "/tb-500/" {
			s.plugin.items[i].Title = "TB-500 Fragment"
		}
	}
	s.touch(tb500WPID(s), start.Add(time.Hour))
	s.walkAgain(t, 2*time.Hour)
	if got := s.withoutEntity(); !slices.Equal(got, []string{"/bpc-157/", "/product/bpc-157-peptide/"}) {
		t.Fatalf("after mapping /peptides/ and dismissing /tb-500/ = %v", got)
	}
	if cleared, _ := s.finding(finding.PageWithoutEntity, finding.SubjectPage(peptides.ID)); cleared.Status != finding.Resolved || cleared.ResolvedBy != finding.ResolvedByObservation {
		t.Errorf("the mapped page's finding = %s by %s, want resolved by the observation", cleared.Status, cleared.ResolvedBy)
	}
	if kept, _ := s.finding(finding.PageWithoutEntity, finding.SubjectPage(tb500.ID)); kept.Status != finding.Dismissed {
		t.Errorf("a dismissed finding whose page was retitled = %s, want it still dismissed", kept.Status)
	}

	s.carry(t, "/peptides/", nil)
	s.walkAgain(t, 3*time.Hour)
	if got := s.withoutEntity(); !slices.Equal(got, []string{"/bpc-157/", "/peptides/", "/product/bpc-157-peptide/"}) {
		t.Errorf("after unmapping /peptides/ = %v, want its finding open again", got)
	}
}

func tb500WPID(s *scenario) int64 {
	for i := range s.plugin.items {
		if s.plugin.items[i].Path == "/tb-500/" {
			return s.plugin.items[i].ID
		}
	}
	return 0
}
