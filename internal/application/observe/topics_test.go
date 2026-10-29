package observe_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
)

type builtEntity struct {
	name, parent, page string
	source             graph.Source
}

func (s *scenario) theGraph() (entities []builtEntity, edges []string) {
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	names := make(map[string]string, len(s.stores.entities))
	for i := range s.stores.entities {
		names[s.stores.entities[i].ID] = s.stores.entities[i].Name
	}
	for i := range s.stores.entities {
		entity := s.stores.entities[i]
		held := builtEntity{name: entity.Name, source: entity.Source}
		if entity.ScopeID != nil {
			held.parent = names[*entity.ScopeID]
		}
		if entity.CanonicalPageID != nil {
			held.page = s.stores.pages[*entity.CanonicalPageID].Path
		}
		entities = append(entities, held)
	}
	slices.SortFunc(entities, func(a, b builtEntity) int { return strings.Compare(a.name, b.name) })
	for i := range s.stores.edges {
		edge := s.stores.edges[i]
		edges = append(edges, names[edge.FromEntityID]+" > "+names[edge.ToEntityID]+" "+string(edge.Kind)+" "+string(edge.Status)+" "+string(edge.Source))
	}
	slices.Sort(edges)
	return entities, edges
}

func (s *scenario) mappedTo(t *testing.T, path string) string {
	t.Helper()
	page := s.pageAt(t, path)
	if page.EntityID == nil {
		return ""
	}
	s.stores.mu.Lock()
	defer s.stores.mu.Unlock()
	for i := range s.stores.entities {
		if s.stores.entities[i].ID == *page.EntityID {
			return s.stores.entities[i].Name
		}
	}
	return "?"
}

var builtFromTheTrails = []builtEntity{
	{name: "BPC-157", parent: "Healing Peptides", page: "/bpc-157/", source: graph.SourceSite},
	{name: "Healing Peptides", parent: "Peptides", source: graph.SourceSite},
	{name: "Peptides", page: "/peptides/", source: graph.SourceSite},
	{name: "TB-500", parent: "Peptides", page: "/tb-500/", source: graph.SourceSite},
}

var edgesFromTheTrails = []string{
	"BPC-157 > Healing Peptides parent approved site",
	"Healing Peptides > Peptides parent approved site",
	"TB-500 > Peptides parent approved site",
}

func TestTheSyncBuildsTheTopicalMapFromTheBreadcrumbs(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	state := s.run(t, observe.Options{})
	entities, edges := s.theGraph()
	if !slices.Equal(entities, builtFromTheTrails) {
		t.Fatalf("the built entities =\n%+v\nwant, the dead Healing Peptides crumb a topic under Peptides\n%+v", entities, builtFromTheTrails)
	}
	if !slices.Equal(edges, edgesFromTheTrails) {
		t.Errorf("the built edges = %v, want %v", edges, edgesFromTheTrails)
	}
	for path, name := range map[string]string{"/peptides/": "Peptides", "/tb-500/": "TB-500", "/bpc-157/": "BPC-157", "/product/bpc-157-peptide/": ""} {
		if got := s.mappedTo(t, path); got != name {
			t.Errorf("%s carries %q, want %q", path, got, name)
		}
	}
	if got := s.withoutEntity(); !slices.Equal(got, []string{"/product/bpc-157-peptide/"}) {
		t.Errorf("the pages without an entity = %v, want only the product whose short trail skips Peptides", got)
	}
	counts := state.Counts
	if counts.Entities != 4 || counts.Placed != 3 || counts.Unplaced != 1 || counts.UnplacedBy["disagrees"] != 1 {
		t.Errorf("the counts = %+v, want 4 entities, 3 pages placed and the product left as disagreeing", counts)
	}
	s.stores.mu.Lock()
	published := slices.Clone(s.stores.published)
	s.stores.mu.Unlock()
	if !slices.Contains(published, events.GraphChanged) {
		t.Errorf("published %v, want graph.changed once the map was built", published)
	}
}

func TestASecondSyncWritesNothingAndADeletionSticks(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.run(t, observe.Options{})

	s.clock.Set(start.Add(time.Hour))
	again := s.run(t, observe.Options{})
	if entities, edges := s.theGraph(); !slices.Equal(entities, builtFromTheTrails) || !slices.Equal(edges, edgesFromTheTrails) {
		t.Fatalf("a second sync left the graph as %+v %v", entities, edges)
	}
	if again.Counts.Entities != 0 || again.Counts.Placed != 0 {
		t.Errorf("a second sync counted %+v, want nothing written", again.Counts)
	}

	tb500 := s.pageAt(t, "/tb-500/")
	if err := (graphStore{s.stores}).Delete(t.Context(), *tb500.EntityID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	s.clock.Set(start.Add(2 * time.Hour))
	after := s.run(t, observe.Options{})
	if got := s.mappedTo(t, "/tb-500/"); got != "" {
		t.Errorf("the sync brought back the entity a person deleted: %q", got)
	}
	if got := s.withoutEntity(); !slices.Equal(got, []string{"/product/bpc-157-peptide/", "/tb-500/"}) {
		t.Errorf("the pages without an entity = %v, want the deleted one's page open again", got)
	}
	if after.Counts.UnplacedBy["removed"] != 1 {
		t.Errorf("the counts = %+v, want the deleted one said as removed", after.Counts)
	}
}

func TestABuildStoppedMidwayResumesWhereItStopped(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)

	state, err := s.service.Begin(context.Background(), siteID, s.readers, observe.Options{})
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for state.Stage != observe.StageTopics || state.TopicsAfter == "" {
		if state, err = s.service.Step(context.Background(), state, s.readers, 1); err != nil {
			t.Fatalf("step: %v", err)
		}
		if state.Done() {
			t.Fatal("the walk ended before its topical map took more than one step")
		}
	}

	restarted := observe.New(s.stores.deps(s.clock))
	for !state.Done() {
		if state, err = restarted.Step(context.Background(), state, s.readers, 1); err != nil {
			t.Fatalf("step after the restart: %v", err)
		}
	}
	if entities, edges := s.theGraph(); !slices.Equal(entities, builtFromTheTrails) || !slices.Equal(edges, edgesFromTheTrails) {
		t.Errorf("the graph after a restart mid-build = %+v %v", entities, edges)
	}
}

func TestAPageThatNeedsAnEntityAndHasNoBreadcrumbsIsAFindingWithItsPath(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.serveTheSite(false)
	s.serve(tb500Page(false))

	s.run(t, observe.Options{})
	tb500 := s.pageAt(t, "/tb-500/")
	raised, found := s.finding(finding.PageWithoutEntity, finding.SubjectPage(tb500.ID))
	if !found || raised.Status != finding.Open || string(raised.Subject) != `{"path":"/tb-500/"}` {
		t.Errorf("the finding of a page with no breadcrumbs = %+v, found %v", raised, found)
	}
	if got := s.mappedTo(t, "/tb-500/"); got != "" {
		t.Errorf("a page with no breadcrumbs carries %q", got)
	}
}
