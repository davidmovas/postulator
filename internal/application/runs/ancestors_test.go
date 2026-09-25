package runs_test

import (
	stderrors "errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type chain struct {
	path   string
	id     string
	parent string
	wpID   *int64
	mapped bool
}

func (f *fixture) chain(links ...chain) {
	f.specs.siteID = f.siteID
	f.specs.spec = template.TemplateSpec{Recipe: []template.StepSpec{
		{Name: "generate_body", Enabled: true}, {Name: "publish", Enabled: true},
	}}
	f.mapping.known = make(map[string]pagemap.Page, len(links))
	for _, link := range links {
		page := pagemap.Page{ID: link.id, Path: link.path, WPID: link.wpID}
		if link.parent != "" {
			parent := link.parent
			page.ParentPageID = &parent
		}
		if link.mapped {
			entity := "entity-" + link.id
			page.EntityID = &entity
		}
		f.mapping.known[link.id] = page
	}
}

func onSite() *int64 {
	wpID := int64(40)
	return &wpID
}

func TestStartAddsTheAncestorsThatAreNotOnTheSite(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.chain(
		chain{id: "bikes", path: "/bikes/", mapped: true},
		chain{id: "cargo", path: "/bikes/cargo/", parent: "bikes", mapped: true},
		chain{id: "max", path: "/bikes/cargo/max/", parent: "cargo", mapped: true},
	)

	resp, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: []string{"max"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if got := fixture.engine.queued.Targets; !slices.Equal(got, []string{"max", "cargo", "bikes"}) {
		t.Fatalf("the run targets %v, want the page and both parents that are not on the site", got)
	}
	want := []runs.AddedPage{
		{PageID: "cargo", Path: "/bikes/cargo/", NeededBy: "/bikes/cargo/max/"},
		{PageID: "bikes", Path: "/bikes/", NeededBy: "/bikes/cargo/max/"},
	}
	if !slices.Equal(resp.Added, want) {
		t.Fatalf("Added = %+v, want %+v", resp.Added, want)
	}

	estimated, err := fixture.service.Estimate(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: []string{"max"}})
	if err != nil || !slices.Equal(estimated.Added, want) {
		t.Fatalf("Estimate = %+v, %v; want the same parents named before the run starts", estimated.Added, err)
	}
}

func TestStartStopsAtTheFirstAncestorOnTheSite(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.chain(
		chain{id: "bikes", path: "/bikes/", mapped: true},
		chain{id: "cargo", path: "/bikes/cargo/", parent: "bikes", wpID: onSite(), mapped: true},
		chain{id: "max", path: "/bikes/cargo/max/", parent: "cargo", mapped: true},
	)

	resp, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: []string{"max"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(resp.Added) != 0 || !slices.Equal(fixture.engine.queued.Targets, []string{"max"}) {
		t.Fatalf("Added = %+v, targets %v; a parent already on the site is not written again",
			resp.Added, fixture.engine.queued.Targets)
	}
}

func TestStartLeavesAnAncestorAnotherRunIsWriting(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	parentID := fixture.pages[0]
	fixture.chain(
		chain{id: parentID, path: "/hub/", mapped: true},
		chain{id: "espresso", path: "/hub/espresso/", parent: parentID, mapped: true},
	)
	_, item := fixture.seedRun(t, run.StatusRunning)
	item.Status = run.StatusRunning
	if ok, err := fixture.items.Persist(t.Context(), item, item.AdvanceSeq); err != nil || !ok {
		t.Fatalf("put the parent in flight: %v, %v", ok, err)
	}

	resp, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: []string{"espresso"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(resp.Added) != 0 || !slices.Equal(fixture.engine.queued.Targets, []string{"espresso"}) {
		t.Fatalf("Added = %+v, targets %v; a parent another run is writing is waited for, not written twice",
			resp.Added, fixture.engine.queued.Targets)
	}
}

func TestStartRefusesAnAncestorItCannotWrite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		links []chain
		names string
	}{
		{
			name: "the parent is mapped to nothing",
			links: []chain{
				{id: "bikes", path: "/bikes/"},
				{id: "cargo", path: "/bikes/cargo/", parent: "bikes", mapped: true},
			},
			names: "/bikes/",
		},
		{
			name:  "the page map holds no parent",
			links: []chain{{id: "cargo", path: "/bikes/cargo/", mapped: true}},
			names: "/bikes/",
		},
		{
			name: "the link disagrees with the path",
			links: []chain{
				{id: "trikes", path: "/trikes/", mapped: true},
				{id: "cargo", path: "/bikes/cargo/", parent: "trikes", mapped: true},
			},
			names: "/trikes/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			fixture.chain(tc.links...)

			_, err := fixture.service.Start(t.Context(), runs.StartRequest{SiteID: fixture.siteID, PageIDs: []string{"cargo"}})
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Start = %v, want invalid", err)
			}
			var refusal *errors.Error
			if !stderrors.As(err, &refusal) || refusal.Details["field"] != "pageIds" {
				t.Fatalf("the refusal = %v, want it on pageIds", err)
			}
			if !strings.Contains(err.Error(), tc.names) || !strings.Contains(err.Error(), "/bikes/cargo/") {
				t.Fatalf("the refusal %q does not name the parent and the page that needs it", err.Error())
			}
			if fixture.engine.queued.SiteID != "" {
				t.Fatalf("a run whose parent cannot be written reached the engine: %+v", fixture.engine.queued)
			}
		})
	}
}

func TestARunThatDoesNotPublishAddsNoAncestor(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.chain(
		chain{id: "bikes", path: "/bikes/", mapped: true},
		chain{id: "cargo", path: "/bikes/cargo/", parent: "bikes", mapped: true},
	)

	resp, err := fixture.service.Start(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: []string{"cargo"}, Kind: string(run.KindRelink),
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(resp.Added) != 0 || !slices.Equal(fixture.engine.queued.Targets, []string{"cargo"}) {
		t.Fatalf("a relink added %+v", resp.Added)
	}
}

func pickedRecipe() []template.StepSpec {
	return []template.StepSpec{
		{Name: "resolve_context", Enabled: true}, {Name: "generate_body", Enabled: true}, {Name: "publish", Enabled: true},
	}
}

func TestStartAssignsThePickedTemplateToTheChosenPagesOnly(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.chain(
		chain{id: "bikes", path: "/bikes/", mapped: true},
		chain{id: "cargo", path: "/bikes/cargo/", parent: "bikes", mapped: true},
		chain{id: "max", path: "/bikes/cargo/max/", parent: "cargo", mapped: true},
	)
	fixture.specs.byTemplate = map[string]template.TemplateSpec{"picked": {Recipe: pickedRecipe()}}

	if _, err := fixture.service.Start(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: []string{"max"}, TemplateID: "picked",
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if len(fixture.assigner.calls) != 1 {
		t.Fatalf("assigned %d times, want once", len(fixture.assigner.calls))
	}
	call := fixture.assigner.calls[0]
	if call.SiteID != fixture.siteID || call.TemplateID != "picked" || !slices.Equal(call.PageIDs, []string{"max"}) {
		t.Fatalf("assigned %+v, want the picked template on the chosen page alone", call)
	}
	if fixture.specs.handed["max"] != "picked" || fixture.specs.handed["cargo"] != "" || fixture.specs.handed["bikes"] != "" {
		t.Fatalf("resolved with %v, want the parents the run added on their own templates", fixture.specs.handed)
	}
	if want := map[string]string{"max": "picked"}; !maps.Equal(fixture.engine.assigned, want) {
		t.Fatalf("the estimate priced %v, want %v", fixture.engine.assigned, want)
	}
	queued := fixture.engine.queued
	if queued.TemplateID != "picked" || queued.TemplateVersion != 7 || len(queued.Recipe) != len(pickedRecipe()) {
		t.Fatalf("queued %s@%d with %v, want the picked template's recipe", queued.TemplateID, queued.TemplateVersion,
			queued.Recipe)
	}
}

func TestEstimateAssignsNothing(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.chain(chain{id: "max", path: "/max/", mapped: true})
	fixture.specs.byTemplate = map[string]template.TemplateSpec{"picked": {Recipe: pickedRecipe()}}

	if _, err := fixture.service.Estimate(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: []string{"max"}, TemplateID: "picked",
	}); err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if len(fixture.assigner.calls) != 0 {
		t.Fatalf("the estimate assigned %+v", fixture.assigner.calls)
	}
	if fixture.engine.assigned["max"] != "picked" {
		t.Fatalf("the estimate priced %v, want the picked template", fixture.engine.assigned)
	}
}

func TestARefusedStartAssignsNothing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		arrange func(*fixture)
		request runs.StartRequest
	}{
		{
			name: "the estimate blocks the run",
			arrange: func(f *fixture) {
				f.engine.estimate.Findings = []run.EstimateFinding{{Severity: "error", Code: "provider_key_missing", Message: "no key"}}
			},
			request: runs.StartRequest{PageIDs: []string{"max"}, TemplateID: "picked"},
		},
		{
			name:    "a kind that owns its recipe",
			request: runs.StartRequest{PageIDs: []string{"max"}, TemplateID: "picked", Kind: string(run.KindRelink)},
		},
		{
			name:    "no template was picked",
			request: runs.StartRequest{PageIDs: []string{"max"}},
		},
		{
			name: "the picked template's recipe cannot run",
			arrange: func(f *fixture) {
				f.specs.byTemplate["picked"] = template.TemplateSpec{Recipe: []template.StepSpec{{Name: "publish", Enabled: true}}}
			},
			request: runs.StartRequest{PageIDs: []string{"max"}, TemplateID: "picked"},
		},
		{
			name:    "a negative budget",
			request: runs.StartRequest{PageIDs: []string{"max"}, TemplateID: "picked", Budget: run.Budget{MaxUSD: -1}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			fixture.chain(chain{id: "max", path: "/max/", mapped: true})
			fixture.specs.byTemplate = map[string]template.TemplateSpec{"picked": {Recipe: pickedRecipe()}}
			if tc.arrange != nil {
				tc.arrange(fixture)
			}
			request := tc.request
			request.SiteID = fixture.siteID

			_, err := fixture.service.Start(t.Context(), request)
			if len(fixture.assigner.calls) != 0 {
				t.Fatalf("assigned %+v, want nothing", fixture.assigner.calls)
			}
			if tc.request.TemplateID != "" && tc.request.Kind == "" && err == nil {
				t.Fatal("Start went ahead, want it refused before anything was assigned")
			}
		})
	}
}

func TestAFailedAssignmentQueuesNothing(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.chain(chain{id: "max", path: "/max/", mapped: true})
	fixture.specs.byTemplate = map[string]template.TemplateSpec{"picked": {Recipe: pickedRecipe()}}
	fixture.assigner.err = errors.New(errors.Invalid, "the page belongs to another site")

	_, err := fixture.service.Start(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: []string{"max"}, TemplateID: "picked",
	})
	if !errors.IsCode(err, errors.Invalid) || fixture.engine.queued.ID != "" {
		t.Fatalf("Start = %v, queued %+v; want the assignment's refusal and no run", err, fixture.engine.queued)
	}
}
