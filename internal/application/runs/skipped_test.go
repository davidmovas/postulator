package runs_test

import (
	stderrors "errors"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (f *fixture) storeCategory(pageID, path string) {
	f.specs.siteID = f.siteID
	f.types.set = pagetype.NewSet([]pagetype.Type{
		{ID: "store", SiteID: f.siteID, Name: "Store category", Preset: pagetype.StoreCategory},
	}, nil)
	typeID, entityID := "store", "entity-"+pageID
	if f.mapping.known == nil {
		f.mapping.known = map[string]pagemap.Page{}
	}
	f.mapping.known[pageID] = pagemap.Page{ID: pageID, SiteID: f.siteID, Path: path, PageTypeID: &typeID, EntityID: &entityID}
	f.specs.unassigned[pageID] = path
}

func TestARunLeavesOutAPageNoTemplateAnswersFor(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.storeCategory(fixture.pages[1], "/shop/")
	request := runs.StartRequest{SiteID: fixture.siteID, PageIDs: fixture.pages}
	want := []run.Skipped{{PageID: fixture.pages[1], Path: "/shop/", Reason: run.SkipNoTemplate, PageType: "Store category"}}

	estimated, err := fixture.service.Estimate(t.Context(), request)
	if err != nil || !slices.Equal(estimated.Skipped, want) {
		t.Fatalf("Estimate = %+v, %v; want %+v left out", estimated.Skipped, err, want)
	}
	started, err := fixture.service.Start(t.Context(), request)
	if err != nil {
		t.Fatalf("Start = %v, want the run over the page a template answers for", err)
	}
	if !slices.Equal(started.Skipped, want) || !slices.Equal(fixture.engine.queued.Skipped, want) {
		t.Fatalf("left out %+v, the run records %+v; want %+v", started.Skipped, fixture.engine.queued.Skipped, want)
	}
	if got := fixture.engine.queued.Targets; !slices.Equal(got, fixture.pages[:1]) {
		t.Fatalf("the run targets %v, want the page a template answers for alone", got)
	}
}

func TestAPickedTemplateAnswersForAPageWithoutOneOfItsOwn(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.specs.siteID = fixture.siteID
	fixture.specs.unassigned[fixture.pages[1]] = "/hub/child/"
	fixture.specs.byTemplate = map[string]template.TemplateSpec{"picked": {Recipe: pickedRecipe()}}

	started, err := fixture.service.Start(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: fixture.pages, TemplateID: "picked",
	})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	if len(started.Skipped) != 0 || !slices.Equal(fixture.engine.queued.Targets, fixture.pages) {
		t.Fatalf("left out %+v, targets %v; a picked template writes every chosen page", started.Skipped, fixture.engine.queued.Targets)
	}
}

func TestAPageLeftOutTakesItsDirectiveAndItsRewriteWithIt(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.storeCategory(fixture.pages[1], "/shop/")

	_, err := fixture.service.Start(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: fixture.pages, Rewrite: fixture.pages[1:],
		Directives: []run.Directive{{PageID: fixture.pages[1], Notes: "Lead with the shop."}, {PageID: fixture.pages[0], Notes: "Lead with the hub."}},
	})
	if err != nil {
		t.Fatalf("Start = %v, want the directive and the rewrite of the page left out dropped with it", err)
	}
	if len(fixture.engine.directives) != 1 || fixture.engine.directives[0].PageID != fixture.pages[0] {
		t.Fatalf("the run carries the directives %+v, want the hub's alone", fixture.engine.directives)
	}
	for _, step := range fixture.engine.queued.Recipe {
		if _, rewrites := step.Params[run.ParamRewrite]; rewrites {
			t.Fatalf("the step %s still rewrites a page the run left out: %+v", step.Name, step.Params)
		}
	}
}

func TestARunThatLeavesEveryPageOutSaysWhyForEach(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.storeCategory(fixture.pages[0], "/shop/")
	fixture.mapping.unmapped[fixture.pages[1]] = "/cart/"

	_, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: fixture.pages})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Start = %v, want invalid", err)
	}
	var refusal *errors.Error
	if !stderrors.As(err, &refusal) || refusal.Details["field"] != "pageIds" {
		t.Fatalf("the refusal = %v, want it on pageIds", err)
	}
	if skipped, ok := refusal.Details["skipped"].([]run.Skipped); !ok || len(skipped) != 2 {
		t.Fatalf("the refusal carries %v, want both pages left out", refusal.Details["skipped"])
	}
	for _, said := range []string{"/shop/", "/cart/", "Page types", "Graph"} {
		if !strings.Contains(err.Error(), said) {
			t.Errorf("the refusal %q does not say %q", err.Error(), said)
		}
	}
	if fixture.engine.queued.SiteID != "" {
		t.Fatalf("a run with nothing to write reached the engine: %+v", fixture.engine.queued)
	}
}

func TestAParentTheRunMustWriteFirstWithoutATemplateIsNamed(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.chain(
		chain{id: "bikes", path: "/bikes/", mapped: true},
		chain{id: "cargo", path: "/bikes/cargo/", parent: "bikes", mapped: true},
	)
	fixture.specs.unassigned["bikes"] = "/bikes/"

	_, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: []string{"cargo"}})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Start = %v, want invalid", err)
	}
	for _, said := range []string{"/bikes/", "/bikes/cargo/", "template"} {
		if !strings.Contains(err.Error(), said) {
			t.Errorf("the refusal %q does not say %q", err.Error(), said)
		}
	}
}
