package runtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func placing(pages *sqlite.PageRepo) run.StepDef {
	return producing("place", run.ArtifactFinalReport, nil,
		func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			if sc.Page.ParentPageID == nil {
				placed := sc.Page
				wpID := int64(len(placed.Path))
				placed.WPID = &wpID
				placed.Status = pagemap.StatusPublished
				if err := pages.Update(ctx, placed); err != nil {
					return run.Result{}, err
				}
				return run.Result{Next: run.TransitionComplete}, nil
			}
			parent, err := pages.Get(ctx, *sc.Page.ParentPageID)
			if err != nil {
				return run.Result{}, err
			}
			if parent.WPID == nil {
				return run.Result{
					Next: run.TransitionPause, Reason: run.PauseAwaitingParent,
					Message: "waits for its parent " + parent.Path + ", which is not on the site yet",
				}, nil
			}
			return run.Result{Next: run.TransitionComplete}, nil
		})
}

func nestedPages(t *testing.T, h *harness) (parent, child pagemap.Page) {
	t.Helper()

	pages := pageRepoOf(h)
	parent = sqlitetest.Page(t, h.store, h.siteID, "/coffee/")
	child = sqlitetest.Page(t, h.store, h.siteID, "/coffee/espresso/")
	child.ParentPageID = &parent.ID
	if err := pages.Update(t.Context(), child); err != nil {
		t.Fatalf("link the child to its parent: %v", err)
	}
	return parent, child
}

func TestAChildWaitsForItsParentAndGoesOnOnceTheParentIsOnTheSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		parentInRun  bool
		publishAside bool
	}{
		{name: "the parent is published outside the run", publishAside: true},
		{name: "the parent is an item of the same run", parentInRun: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, 0)
			pages := pageRepoOf(harness)
			parent, child := nestedPages(t, harness)
			harness.pages = []string{child.ID}
			if tc.parentInRun {
				harness.pages = []string{child.ID, parent.ID}
			}

			engine := harness.engine(t, mustRegister(t, placing(pages)))
			queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("place")))
			if err != nil {
				t.Fatalf("Enqueue: %v", err)
			}

			if tc.publishAside {
				paused := harness.waitForRun(t, queued.ID, run.StatusPaused)
				if paused.PauseReason != run.PauseAwaitingParent {
					t.Fatalf("the run paused for %q, want %q", paused.PauseReason, run.PauseAwaitingParent)
				}

				held := itemOf(t, harness, queued.ID, child.ID)
				if held.Status != run.StatusPaused || held.PauseReason != run.PauseAwaitingParent {
					t.Fatalf("the child is %s/%s, want paused awaiting its parent", held.Status, held.PauseReason)
				}
				if !strings.Contains(held.Note, "/coffee/") {
					t.Fatalf("the child's note %q does not name the parent it waits for", held.Note)
				}
				announced := harness.bus.payloads(events.ItemNeedsHuman)
				if len(announced) == 0 {
					t.Fatal("the pause of the child was not announced")
				}
				payload, ok := announced[0].(events.ItemNeedsHumanPayload)
				if !ok || payload.Reason != string(run.PauseAwaitingParent) || !strings.Contains(payload.Message, "/coffee/") {
					t.Fatalf("the announcement = %+v, want the reason and the step's own sentence", announced[0])
				}

				wpID := int64(77)
				parent.WPID = &wpID
				parent.Status = pagemap.StatusPublished
				if err = pages.Update(t.Context(), parent); err != nil {
					t.Fatalf("publish the parent aside: %v", err)
				}
			}

			harness.waitForRun(t, queued.ID, run.StatusCompleted)
			done := itemOf(t, harness, queued.ID, child.ID)
			if done.Status != run.StatusCompleted || done.PauseReason != "" || done.Note != "" {
				t.Fatalf("the child ended as %+v, want it completed with its hold cleared", done)
			}
		})
	}
}

func TestARunPausedForAHumanOutlivesItsDeadline(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	frozen := clock.NewFake(time.Now().UTC())
	harness.clock = frozen

	calls := newCounter()
	step := producing("ask", run.ArtifactFinalReport, nil,
		func(context.Context, *run.StepContext) (run.Result, error) {
			if calls.hit("ask") == 1 {
				return run.Result{Next: run.TransitionPause, Reason: run.PauseNeedsHuman, Message: "a human decides"}, nil
			}
			return run.Result{Next: run.TransitionComplete}, nil
		})
	engine := harness.engine(t, mustRegister(t, step))

	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("ask")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusPaused)

	frozen.Advance(3 * harness.deadline)
	time.Sleep(20 * pollInterval)
	still, err := harness.runs.Get(t.Context(), queued.ID)
	if err != nil {
		t.Fatalf("read the run: %v", err)
	}
	if still.Status != run.StatusPaused {
		t.Fatalf("a run waiting for a human became %s past its deadline, want it still paused", still.Status)
	}

	if err = engine.Resume(t.Context(), queued.ID); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)
}

func TestRetryStepGivesAStoppedRunAFreshDeadline(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	frozen := clock.NewFake(time.Now().UTC())
	harness.clock = frozen

	calls := newCounter()
	step := producing("try", run.ArtifactFinalReport, nil,
		func(context.Context, *run.StepContext) (run.Result, error) {
			if calls.hit("try") == 1 {
				return run.Result{Next: run.TransitionFail, Message: "the first try fails"}, nil
			}
			return run.Result{Next: run.TransitionComplete}, nil
		})
	engine := harness.engine(t, mustRegister(t, step))

	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("try")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusFailed)
	frozen.Advance(2 * harness.deadline)

	failed := itemOf(t, harness, queued.ID, harness.pages[0])
	if err = engine.RetryStep(t.Context(), failed.ID); err != nil {
		t.Fatalf("RetryStep: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)
}

func itemOf(t *testing.T, h *harness, runID, targetID string) run.Item {
	t.Helper()

	items, err := h.items.ByRun(t.Context(), runID)
	if err != nil {
		t.Fatalf("list the items: %v", err)
	}
	for i := range items {
		if items[i].TargetID == targetID {
			return items[i]
		}
	}
	t.Fatalf("the run %s holds no item for %s", runID, targetID)
	return run.Item{}
}

func publishing(pages *sqlite.PageRepo, calls *counter, failFirst string) run.StepDef {
	return producing(string(run.StepPublish), run.ArtifactPublishResult, nil,
		func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			if sc.Page.ParentPageID != nil {
				parent, err := pages.Get(ctx, *sc.Page.ParentPageID)
				if err != nil {
					return run.Result{}, err
				}
				if parent.WPID == nil {
					return run.Result{}, faulty(errors.Internal, sc.Page.Path+" ran before its parent reached the site")
				}
			}
			if calls.hit(sc.Page.Path) == 1 && sc.Page.Path == failFirst {
				return run.Result{}, faulty(errors.Invalid, "the writer gave up on "+sc.Page.Path)
			}
			placed := sc.Page
			wpID := int64(len(placed.Path))
			placed.WPID = &wpID
			placed.Status = pagemap.StatusPublished
			if err := pages.Update(ctx, placed); err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactPublishResult, Blob: []byte("{}")}},
				Next:      run.TransitionComplete,
			}, nil
		})
}

func lineage(t *testing.T, h *harness, paths ...string) []pagemap.Page {
	t.Helper()

	pages := pageRepoOf(h)
	out := make([]pagemap.Page, 0, len(paths))
	for i, path := range paths {
		page := sqlitetest.Page(t, h.store, h.siteID, path)
		if i > 0 {
			page.ParentPageID = &out[i-1].ID
			if err := pages.Update(t.Context(), page); err != nil {
				t.Fatalf("link %s to its parent: %v", path, err)
			}
		}
		out = append(out, page)
	}
	return out
}

func TestAChildQueuedBehindItsParentWaitsWhenTheParentStopsAndGoesOnOnceItIsRegenerated(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 0)
	family := lineage(t, harness, "/coffee/", "/coffee/espresso/", "/coffee/espresso/ristretto/")
	parent, child, grandchild := family[0], family[1], family[2]
	harness.pages = []string{grandchild.ID, child.ID, parent.ID}

	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, publishing(pageRepoOf(harness), calls, parent.Path)))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf(string(run.StepPublish))))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if got := itemOf(t, harness, queued.ID, child.ID); got.BlockedBy != itemOf(t, harness, queued.ID, parent.ID).ID {
		t.Fatalf("the child is queued behind %q, want the parent's item", got.BlockedBy)
	}
	if got := itemOf(t, harness, queued.ID, grandchild.ID); got.BlockedBy != itemOf(t, harness, queued.ID, child.ID).ID {
		t.Fatalf("the grandchild is queued behind %q, want the child's item", got.BlockedBy)
	}

	paused := harness.waitForRun(t, queued.ID, run.StatusPaused)
	if paused.PauseReason != run.PauseAwaitingParent {
		t.Fatalf("the run paused for %q, want %q", paused.PauseReason, run.PauseAwaitingParent)
	}
	failed := itemOf(t, harness, queued.ID, parent.ID)
	if failed.Status != run.StatusFailed {
		t.Fatalf("the parent is %s, want it failed", failed.Status)
	}
	held := itemOf(t, harness, queued.ID, child.ID)
	if held.Status != run.StatusPaused || held.PauseReason != run.PauseAwaitingParent {
		t.Fatalf("the child is %s/%s, want it paused awaiting its parent", held.Status, held.PauseReason)
	}
	if !strings.Contains(held.Note, "/coffee/") || !strings.Contains(held.Note, "failed") {
		t.Fatalf("the child's note %q neither names the parent nor says it failed", held.Note)
	}
	if held.CurrentStep != string(run.StepPublish) || held.Attempts != 0 {
		t.Fatalf("the child moved to %s after %d attempts, want it untouched", held.CurrentStep, held.Attempts)
	}
	further := itemOf(t, harness, queued.ID, grandchild.ID)
	if further.Status != run.StatusPaused || further.PauseReason != run.PauseAwaitingParent || !strings.Contains(further.Note, child.Path) {
		t.Fatalf("the grandchild is %s/%s with the note %q, want it held behind the child", further.Status, further.PauseReason, further.Note)
	}
	announced := harness.bus.payloads(events.ItemNeedsHuman)
	if len(announced) != 2 {
		t.Fatalf("%d holds were announced, want the child and the grandchild", len(announced))
	}
	if calls.get(child.Path) != 0 || calls.get(grandchild.Path) != 0 {
		t.Fatalf("the children ran %d and %d times before their parent", calls.get(child.Path), calls.get(grandchild.Path))
	}

	if err = engine.Regenerate(t.Context(), queued.ID, []string{failed.ID}); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}
	finished := harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if finished.Stats.Done != 3 || finished.Stats.Failed != 0 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}
	for _, page := range family {
		done := itemOf(t, harness, queued.ID, page.ID)
		if done.Status != run.StatusCompleted || done.PauseReason != "" || done.Note != "" {
			t.Fatalf("%s ended as %+v, want it completed with its hold cleared", page.Path, done)
		}
	}
	if calls.get(parent.Path) != 2 || calls.get(child.Path) != 1 || calls.get(grandchild.Path) != 1 {
		t.Fatalf("the steps ran %d, %d and %d times", calls.get(parent.Path), calls.get(child.Path), calls.get(grandchild.Path))
	}
	assertGapless(t, harness, queued.ID)
}

func TestEnqueueQueuesAPageBehindItsParentOnlyWhenTheRunWillPublishIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    run.Kind
		recipe  []template.StepSpec
		post    bool
		onSite  bool
		alone   bool
		blocked bool
	}{
		{name: "a page under a planned parent of the same run", kind: run.KindGenerate, recipe: recipeOf(string(run.StepPublish)), blocked: true},
		{name: "a recipe that does not publish", kind: run.KindGenerate, recipe: recipeOf("place")},
		{name: "a revert", kind: run.KindRevert, recipe: recipeOf(string(run.StepPublish))},
		{name: "a post, which WordPress does not nest", kind: run.KindGenerate, recipe: recipeOf(string(run.StepPublish)), post: true},
		{name: "a parent already on the site", kind: run.KindGenerate, recipe: recipeOf(string(run.StepPublish)), onSite: true},
		{name: "a parent outside the run", kind: run.KindGenerate, recipe: recipeOf(string(run.StepPublish)), alone: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, 0)
			pages := pageRepoOf(harness)
			family := lineage(t, harness, "/coffee/", "/coffee/espresso/")
			parent, child := family[0], family[1]
			if tc.post {
				child.WPType = pagemap.WPPost
				if err := pages.Update(t.Context(), child); err != nil {
					t.Fatalf("make the child a post: %v", err)
				}
			}
			if tc.onSite {
				wpID := int64(5)
				parent.WPID = &wpID
				if err := pages.Update(t.Context(), parent); err != nil {
					t.Fatalf("put the parent on the site: %v", err)
				}
			}
			harness.pages = []string{child.ID, parent.ID}
			if tc.alone {
				harness.pages = []string{child.ID}
			}

			engine := harness.idle(t, mustRegister(t,
				publishing(pages, newCounter(), ""), placing(pages)))
			record := harness.newRun(tc.recipe)
			record.Kind = tc.kind
			queued, err := engine.Enqueue(t.Context(), record)
			if err != nil {
				t.Fatalf("Enqueue: %v", err)
			}

			got := itemOf(t, harness, queued.ID, child.ID)
			if tc.blocked {
				if want := itemOf(t, harness, queued.ID, parent.ID).ID; got.BlockedBy != want {
					t.Fatalf("the child is queued behind %q, want %s", got.BlockedBy, want)
				}
				return
			}
			if got.BlockedBy != "" {
				t.Fatalf("the child is queued behind %q, want nothing", got.BlockedBy)
			}
		})
	}
}
