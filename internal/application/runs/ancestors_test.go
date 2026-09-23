package runs_test

import (
	stderrors "errors"
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
