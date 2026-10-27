package runs_test

import (
	stderrors "errors"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func briefFixture(t *testing.T) *fixture {
	t.Helper()

	f := newFixture(t)
	f.specs.siteID = f.siteID
	return f
}

func TestARunStartCarriesItsBriefAndThePagesDirectives(t *testing.T) {
	t.Parallel()

	f := briefFixture(t)
	f.links.owed[f.pages[0]] = []string{f.pages[1]}
	brief := run.Brief{Goal: "Sell the starter kit", Tone: "plain", Reference: "Kept frozen at −20 °C."}
	directive := run.Directive{
		PageID: f.pages[0], Notes: "Lead with storage.", MustLink: []run.MustLink{{PageID: f.pages[1], Anchor: "the child page"}},
	}
	if _, err := f.service.Start(t.Context(), runs.StartRequest{
		SiteID: f.siteID, PageIDs: f.pages, Brief: &brief, Directives: []run.Directive{directive},
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if f.engine.queued.Brief != brief {
		t.Errorf("the queued run's brief = %+v, want %+v", f.engine.queued.Brief, brief)
	}
	if len(f.engine.directives) != 1 || f.engine.directives[0].Notes != "Lead with storage." || len(f.links.asked) == 0 {
		t.Errorf("the directives handed to the engine = %+v, the link plans read %v", f.engine.directives, f.links.asked)
	}
	if len(f.edges.added) != 0 {
		t.Errorf("a connected must-link target added edges %+v", f.edges.added)
	}
}

func TestAMustLinkTargetTheGraphDoesNotConnectIsRefusedOrBecomesAnEdge(t *testing.T) {
	t.Parallel()

	f := briefFixture(t)
	req := runs.StartRequest{SiteID: f.siteID, PageIDs: f.pages, Directives: []run.Directive{{
		PageID: f.pages[0], MustLink: []run.MustLink{{PageID: f.pages[1], Anchor: "the child page"}},
	}}}
	for name, call := range map[string]func() error{
		"start":    func() error { _, err := f.service.Start(t.Context(), req); return err },
		"estimate": func() error { _, err := f.service.Estimate(t.Context(), req); return err },
	} {
		err := call()
		var refused *errors.Error
		if !stderrors.As(err, &refused) || refused.Code != errors.Invalid || refused.Details["field"] != "directives.mustLink" ||
			!strings.Contains(err.Error(), "/"+f.pages[1]+"/") {
			t.Fatalf("%s with a must-link the graph does not connect = %v", name, err)
		}
	}
	if f.engine.queued.ID != "" || len(f.edges.added) != 0 {
		t.Fatalf("a refused start queued %+v and added %+v", f.engine.queued, f.edges.added)
	}

	req.AddEdges = true
	if _, err := f.service.Start(t.Context(), req); err != nil {
		t.Fatalf("Start adding the edge: %v", err)
	}
	if len(f.edges.added) != 1 {
		t.Fatalf("the start added %+v, want one related edge", f.edges.added)
	}
	added := f.edges.added[0]
	if added.FromEntityID != "entity-"+f.pages[0] || added.ToEntityID != "entity-"+f.pages[1] || added.Kind != string(graph.EdgeRelated) ||
		added.Status != string(graph.StatusApproved) || added.Weight != 1 {
		t.Errorf("the edge = %+v, want an approved related edge between the two pages' entities", added)
	}
	if f.engine.queued.ID == "" {
		t.Error("the start that added its edge queued nothing")
	}
}

func TestAMustLinkAnEdgeAssumedFromAnEarlierStepConnectsIsNeitherRefusedNorAddedTwice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		assumed func(pages []string) []runs.AssumedEdge
		refused bool
	}{
		{name: "from the page to its target", assumed: func(pages []string) []runs.AssumedEdge {
			return []runs.AssumedEdge{{FromEntityID: "entity-" + pages[0], ToEntityID: "entity-" + pages[1]}}
		}},
		{name: "from the target to the page", assumed: func(pages []string) []runs.AssumedEdge {
			return []runs.AssumedEdge{{FromEntityID: "entity-" + pages[1], ToEntityID: "entity-" + pages[0]}}
		}},
		{name: "between two other entities", refused: true, assumed: func([]string) []runs.AssumedEdge {
			return []runs.AssumedEdge{{FromEntityID: "entity-elsewhere", ToEntityID: "entity-nowhere"}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := briefFixture(t)
			req := runs.StartRequest{SiteID: f.siteID, PageIDs: f.pages, Assumed: tc.assumed(f.pages), Directives: []run.Directive{{
				PageID: f.pages[0], MustLink: []run.MustLink{{PageID: f.pages[1], Anchor: "the child page"}},
			}}}
			_, err := f.service.Estimate(t.Context(), req)
			if refused := errors.IsCode(err, errors.Invalid); refused != tc.refused {
				t.Fatalf("Estimate = %v, want refused %v", err, tc.refused)
			}
			if tc.refused {
				return
			}
			if _, err = f.service.Start(t.Context(), req); err != nil {
				t.Fatalf("Start: %v", err)
			}
			if f.engine.queued.ID == "" || len(f.edges.added) != 0 {
				t.Errorf("the start queued %q and added %+v, want it queued with no edge of its own", f.engine.queued.ID, f.edges.added)
			}
		})
	}
}

func TestADirectiveTheRunCannotCarryIsRefused(t *testing.T) {
	t.Parallel()

	f := briefFixture(t)
	cases := map[string]runs.StartRequest{
		"a page outside the run": {SiteID: f.siteID, PageIDs: f.pages[:1], Directives: []run.Directive{{PageID: f.pages[1], Notes: "x"}}},
		"a goal too long":        {SiteID: f.siteID, PageIDs: f.pages, Brief: &run.Brief{Goal: strings.Repeat("g", 1001)}},
	}
	for name, req := range cases {
		if _, err := f.service.Start(t.Context(), req); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s = %v, want INVALID", name, err)
		}
	}
}
