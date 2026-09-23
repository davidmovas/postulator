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
	"github.com/davidmovas/postulator/internal/kernel/clock"
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
