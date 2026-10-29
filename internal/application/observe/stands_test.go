package observe_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

func capturedStores(t *testing.T, stand string) (s *stores, siteID string) {
	t.Helper()

	held, err := sitemodeltest.Read(stand)
	if err != nil {
		t.Fatalf("read the %s capture: %v", stand, err)
	}
	s = newStores(held.Site)
	for i := range held.Pages {
		s.pages[held.Pages[i].ID] = held.Pages[i]
	}
	for i := range held.Renders {
		s.renders[held.Renders[i].PageID] = held.Renders[i]
	}
	for pageID, values := range held.Fields {
		s.fields[pageID] = map[field.Key]field.Value{}
		for i := range values {
			s.fields[pageID][values[i].Key] = values[i]
		}
	}
	s.entities, s.edges = slices.Clone(held.Entities), slices.Clone(held.Edges)
	s.types.types[held.Site.ID], s.types.scopes = slices.Clone(held.Types), slices.Clone(held.Scopes)
	return s, held.Site.ID
}

func TestTheCapturedStandsBuildTheTopicalMapTheirBreadcrumbsDraw(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		built []string
		left  map[string]int
	}{
		"replica": {
			built: []string{
				"BPC-157 Capsules under Healing Peptides", "BPC-157 Nasal Spray under Healing Peptides",
				"Bacteriostatic Water under Peptides", "Sample Page under the root", "TB-500 Peptide under Healing Peptides",
			},
			left: map[string]int{"unknown_link": 1},
		},
		"vanilla": {
			built: []string{"Coffee Beans under Koffein", "Koffein under the root", "Powder under Koffein"},
			left:  map[string]int{"no_breadcrumbs": 2},
		},
	}
	for stand, tc := range cases {
		t.Run(stand, func(t *testing.T) {
			t.Parallel()

			s, siteID := capturedStores(t, stand)
			service := observe.New(s.deps(clock.NewFake(start)))
			state := observe.State{SiteID: siteID, Stage: observe.StageTopics, StartedAt: start}
			for state.Stage == observe.StageTopics {
				var err error
				if state, err = service.Step(context.Background(), state, observe.Readers{}, 2); err != nil {
					t.Fatalf("step: %v", err)
				}
			}

			names := make(map[string]string, len(s.entities))
			for i := range s.entities {
				names[s.entities[i].ID] = s.entities[i].Name
			}
			built := make([]string, 0)
			for i := range s.entities {
				if entity := s.entities[i]; entity.Source == graph.SourceSite {
					parent := "the root"
					if entity.ScopeID != nil {
						parent = names[*entity.ScopeID]
					}
					built = append(built, entity.Name+" under "+parent)
				}
			}
			slices.Sort(built)
			if !slices.Equal(built, tc.built) {
				t.Errorf("the %s built\n%v\nwant\n%v", stand, built, tc.built)
			}
			if !maps.Equal(state.Counts.UnplacedBy, tc.left) {
				t.Errorf("the %s left pages for %v, want %v", stand, state.Counts.UnplacedBy, tc.left)
			}
		})
	}
}
