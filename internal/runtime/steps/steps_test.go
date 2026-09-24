package steps_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func startRun(t *testing.T, f *factory) (string, run.Item) {
	t.Helper()

	engine := f.engine(t)
	queued, err := engine.Enqueue(t.Context(), run.Run{
		ID: newID(), SiteID: f.siteID, Kind: run.KindGenerate, Targets: []string{f.pageID},
		Recipe: recipe(), TemplateID: "template", TemplateVersion: 1, PublishMode: run.PublishDraft,
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	f.waitForRun(t, queued.ID, run.StatusCompleted)

	items, err := f.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	return queued.ID, items[0]
}

func TestAPlannedPageIsWrittenAndLinkedIntoItsGraph(t *testing.T) {
	t.Parallel()

	f := newFactory(t, goodDraft)
	runID, item := startRun(t, f)

	if item.Status != run.StatusCompleted {
		t.Fatalf("the item = %q: %s", item.Status, item.Error)
	}

	body := string(f.artifact(t, item.ID, steps.NameRepairLinks, run.ArtifactBodyHTML).Blob)
	for _, href := range []string{`href="/drinks/"`, `href="/drinks/coffee/"`} {
		if !strings.Contains(body, href) {
			t.Fatalf("the body does not link to %s:\n%s", href, body)
		}
	}
	if strings.Contains(body, "http://") || strings.Contains(body, "https://") {
		t.Fatalf("the body carries an external link:\n%s", body)
	}
	if strings.Contains(body, `href="/drinks/coffee/espresso/"`) {
		t.Fatalf("the body links to itself:\n%s", body)
	}

	doc, err := content.Parse(body)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Links()) != 2 {
		t.Fatalf("the body carries %d links", len(doc.Links()))
	}
	if anchors := anchorsOf(doc); anchors[0] != "coffee" || anchors[1] != "drinks" {
		t.Fatalf("anchors = %v", anchors)
	}

	report := f.report(t, item.ID)
	if report.Score != 1 {
		t.Fatalf("score = %v, findings %v and %v", report.Score, report.Compliance.Items, report.Structure.Items)
	}
	if report.PageID != f.pageID || len(report.Links.Placed) != 2 || len(report.Links.Missing) != 0 {
		t.Fatalf("report = %+v", report)
	}

	if f.llm.CallsTo(steps.NameGenerateBody) != 1 {
		t.Fatalf("the writer was called %d times", f.llm.CallsTo(steps.NameGenerateBody))
	}
	if f.llm.CallsTo(steps.NameRepairLinks) != 0 {
		t.Fatalf("the linker ran although nothing was missing")
	}

	assertEventOrder(t, f, runID)
}

func anchorsOf(doc *content.Document) []string {
	links := doc.Links()

	out := make([]string, 0, len(links))
	for _, link := range links {
		out = append(out, link.Anchor)
	}
	return out
}

func assertEventOrder(t *testing.T, f *factory, runID string) {
	t.Helper()

	stored, err := f.log.List(t.Context(), runID, 0, 100)
	if err != nil {
		t.Fatalf("list the run events: %v", err)
	}

	want := []events.Type{
		events.RunQueued, events.RunStarted, events.ItemStarted,
		events.StepStarted, events.StepDone,
		events.StepStarted, events.StepDone,
		events.StepStarted, events.StepDone,
		events.StepStarted, events.StepDone,
		events.StepStarted, events.StepDone,
		events.ItemDone, events.RunCompleted,
	}
	if len(stored) != len(want) {
		t.Fatalf("the log holds %d events, want %d: %v", len(stored), len(want), typesOf(stored))
	}
	for i, event := range stored {
		if event.Seq != int64(i+1) {
			t.Fatalf("event %d carries seq %d", i, event.Seq)
		}
		if event.Type != string(want[i]) {
			t.Fatalf("event %d = %q, want %q", i, event.Type, want[i])
		}
	}
}

func typesOf(stored []run.Event) []string {
	out := make([]string, 0, len(stored))
	for i := range stored {
		out = append(out, stored[i].Type)
	}
	return out
}

func TestAMissingAnchorIsRepairedByTheLinker(t *testing.T) {
	t.Parallel()

	f := newFactory(t, anchorlessDraft)
	_, item := startRun(t, f)

	body := string(f.artifact(t, item.ID, steps.NameRepairLinks, run.ArtifactBodyHTML).Blob)
	for _, href := range []string{`href="/drinks/"`, `href="/drinks/coffee/"`} {
		if !strings.Contains(body, href) {
			t.Fatalf("the body does not link to %s:\n%s", href, body)
		}
	}
	doc, err := content.Parse(body)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if text := doc.Text(); strings.Count(text, "It sits in our drinks range") != 1 {
		t.Fatalf("the repaired sentence is not there exactly once:\n%s", body)
	}
	if calls := f.llm.CallsTo(steps.NameRepairLinks); calls != 1 {
		t.Fatalf("the linker ran %d times, want one sentence to settle both anchors", calls)
	}

	if report := f.report(t, item.ID); len(report.Links.Missing) != 0 {
		t.Fatalf("the report still lists %d missing targets", len(report.Links.Missing))
	}
}

func TestAPageWithNoEntityFailsItsItem(t *testing.T) {
	t.Parallel()

	f := newFactory(t, goodDraft)
	unmapped := seedPage(t, sqlitePages(f), f.siteID, "/orphan/", "", pagemap.StatusPlanned)

	engine := f.engine(t)
	queued, err := engine.Enqueue(t.Context(), run.Run{
		ID: newID(), SiteID: f.siteID, Kind: run.KindGenerate, Targets: []string{unmapped.ID},
		Recipe: recipe(), TemplateID: "template", TemplateVersion: 1, PublishMode: run.PublishDraft,
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	finished := f.waitForRun(t, queued.ID, run.StatusFailed)
	if finished.Stats.Failed != 1 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}

	items, err := f.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	if !strings.Contains(items[0].Error, "not mapped to an entity") {
		t.Fatalf("the item failed with %q", items[0].Error)
	}
	if f.llm.CallsTo(steps.NameGenerateBody) != 0 {
		t.Fatal("the writer ran for a page with no entity")
	}
}

func TestTheShippedStepsRegisterInRecipeOrder(t *testing.T) {
	t.Parallel()

	registry := run.NewRegistry()
	if err := steps.Register(registry, steps.Deps{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	names := registry.Names()
	want := make([]string, 0, len(run.StepNames())+len(run.PerKindStepNames()))
	for _, declared := range run.StepNames() {
		want = append(want, string(declared))
	}
	for _, owned := range run.PerKindStepNames() {
		want = append(want, string(owned))
	}
	if len(names) != len(want) {
		t.Fatalf("the registry holds %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("the registry holds %v, want %v", names, want)
		}
	}

	if err := run.ValidateRecipe(registry, recipe()); err != nil {
		t.Fatalf("the shipped recipe must validate: %v", err)
	}
	if err := steps.Register(registry, steps.Deps{}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("registering twice = %v", err)
	}
}

func heldByTheLinkBudget() template.TemplateSpec {
	tight := spec()
	tight.LinkRules.MaxLinks = 1
	return tight
}

func TestValidateHoldsAPageWithErrorsForAHumanUnlessTheyAreAllowed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params map[string]any
		want   run.Status
	}{
		{name: "an error finding holds the page", want: run.StatusPaused},
		{name: "allowErrors lets it through", params: map[string]any{steps.ParamAllowErrors: true}, want: run.StatusCompleted},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFactoryWithSpec(t, goodDraft, heldByTheLinkBudget())
			recipe := recipe()
			recipe[len(recipe)-1].Params = tc.params

			engine := f.engine(t)
			queued, err := engine.Enqueue(t.Context(), run.Run{
				ID: newID(), SiteID: f.siteID, Kind: run.KindGenerate, Targets: []string{f.pageID},
				Recipe: recipe, TemplateID: "template", TemplateVersion: 1, PublishMode: run.PublishDraft,
			})
			if err != nil {
				t.Fatalf("Enqueue: %v", err)
			}

			waitForItem(t, f, queued.ID, tc.want)

			items, listErr := f.items.ByRun(t.Context(), queued.ID)
			if listErr != nil || len(items) != 1 {
				t.Fatalf("ByRun = %+v, %v", items, listErr)
			}
			report := f.report(t, items[0].ID)
			if report.Score >= 1 {
				t.Fatalf("score = %v, want a penalty", report.Score)
			}
			if !report.Compliance.HasErrors() {
				t.Fatalf("compliance findings = %+v", report.Compliance.Items)
			}
			if tc.want != run.StatusPaused {
				return
			}
			if items[0].PauseReason != run.PauseNeedsHuman || !strings.Contains(items[0].Note, "/drinks/") {
				t.Fatalf("the item is held as %s with the note %q", items[0].PauseReason, items[0].Note)
			}
			if items[0].Error != "" {
				t.Fatalf("a held page carries the error %q", items[0].Error)
			}
			paused := f.waitForRun(t, queued.ID, run.StatusPaused)
			if paused.PauseReason != run.PauseNeedsHuman {
				t.Fatalf("the run is paused for %s", paused.PauseReason)
			}
		})
	}
}

func TestAcceptingAHeldValidationLetsThePageGoOn(t *testing.T) {
	t.Parallel()

	f := newFactoryWithSpec(t, goodDraft, heldByTheLinkBudget())
	engine := f.engine(t)
	queued, err := engine.Enqueue(t.Context(), run.Run{
		ID: newID(), SiteID: f.siteID, Kind: run.KindGenerate, Targets: []string{f.pageID},
		Recipe: recipe(), TemplateID: "template", TemplateVersion: 1, PublishMode: run.PublishDraft,
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	waitForItem(t, f, queued.ID, run.StatusPaused)
	f.waitForRun(t, queued.ID, run.StatusPaused)

	items, err := f.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	if err = engine.Accept(t.Context(), items[0].ID); err != nil {
		t.Fatalf("Accept: %v", err)
	}

	f.waitForRun(t, queued.ID, run.StatusCompleted)
	waitForItem(t, f, queued.ID, run.StatusCompleted)
	report := f.report(t, items[0].ID)
	if !report.Compliance.HasErrors() {
		t.Fatalf("the accepted report lost its findings: %+v", report.Compliance.Items)
	}
	if f.llm.CallsTo(steps.NameGenerateBody) != 1 {
		t.Fatalf("the writer ran %d times for an accepted page", f.llm.CallsTo(steps.NameGenerateBody))
	}
}
