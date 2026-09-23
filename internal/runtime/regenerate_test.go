package runtime_test

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestRegenerateRunsAStoppedItemFromItsFirstStepAgain(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	frozen := clock.NewFake(time.Now().UTC())
	harness.clock = frozen

	calls := newCounter()
	first := producing("first", run.ArtifactLinkContext, nil,
		func(context.Context, *run.StepContext) (run.Result, error) {
			calls.hit("first")
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactLinkContext, Blob: []byte("context")}}}, nil
		})
	second := producing("second", run.ArtifactFinalReport, []run.ArtifactKind{run.ArtifactLinkContext},
		func(context.Context, *run.StepContext) (run.Result, error) {
			if calls.hit("second") == 1 {
				return run.Result{Next: run.TransitionFail, Message: "the template asked for a section the body lacks"}, nil
			}
			return run.Result{}, nil
		})
	engine := harness.engine(t, mustRegister(t, first, second))

	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("first", "second")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusFailed)
	frozen.Advance(2 * harness.deadline)

	failed := itemOf(t, harness, queued.ID, harness.pages[0])
	if err = engine.Regenerate(t.Context(), queued.ID, []string{failed.ID}); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)

	if calls.get("first") != 2 || calls.get("second") != 2 {
		t.Fatalf("the steps ran %d and %d times, want every step run again from the start",
			calls.get("first"), calls.get("second"))
	}
	done := itemOf(t, harness, queued.ID, harness.pages[0])
	if done.Status != run.StatusCompleted || done.Error != "" {
		t.Fatalf("the regenerated item = %+v", done)
	}
	generation, found, err := run.Get[int](done.Checkpoint, "generation")
	if err != nil || !found || generation != 1 {
		t.Fatalf("the checkpoint carries generation %d (%v, %v), want 1", generation, found, err)
	}

	restarted := harness.bus.payloads(events.ItemRestarted)
	if len(restarted) != 1 {
		t.Fatalf("item.restarted was announced %d times, want once", len(restarted))
	}
	if payload, ok := restarted[0].(events.ItemRestartedPayload); !ok || payload.ItemID != failed.ID {
		t.Fatalf("item.restarted = %+v", restarted[0])
	}
}

func TestRegeneratingAFailedParentLetsItsChildGoOn(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 0)
	pages := pageRepoOf(harness)
	parent, child := nestedPages(t, harness)
	harness.pages = []string{child.ID, parent.ID}

	attempts := newCounter()
	placed := placing(pages)
	gated := placed
	gated.Run = func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
		if sc.Page.ID == parent.ID && attempts.hit(parent.ID) == 1 {
			return run.Result{Next: run.TransitionFail, Message: "the required section Overview is missing"}, nil
		}
		return placed.Run(ctx, sc)
	}
	engine := harness.engine(t, mustRegister(t, gated))

	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("place")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	paused := harness.waitForRun(t, queued.ID, run.StatusPaused)
	if paused.PauseReason != run.PauseAwaitingParent {
		t.Fatalf("the run paused for %q, want it waiting on the failed parent", paused.PauseReason)
	}

	failed := itemOf(t, harness, queued.ID, parent.ID)
	if failed.Status != run.StatusFailed {
		t.Fatalf("the parent item is %s, want failed", failed.Status)
	}
	if err = engine.Regenerate(t.Context(), queued.ID, []string{failed.ID}); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}

	harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if done := itemOf(t, harness, queued.ID, child.ID); done.Status != run.StatusCompleted {
		t.Fatalf("the child is %s, want it published once its parent was", done.Status)
	}
}

func TestRegenerateRefusesWhatItCannotRedo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status run.Status
		kinds  []run.ArtifactKind
		other  bool
		want   errors.Code
		reason string
	}{
		{name: "an item still in flight", status: run.StatusPending, want: errors.Conflict},
		{
			name: "an item that already wrote to the site", status: run.StatusFailed,
			kinds: []run.ArtifactKind{run.ArtifactPublishResult}, want: errors.Conflict, reason: "published",
		},
		{name: "an item of another run", status: run.StatusFailed, other: true, want: errors.Invalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, 1)
			step := producing("only", run.ArtifactFinalReport, nil,
				func(context.Context, *run.StepContext) (run.Result, error) {
					return run.Result{Next: run.TransitionComplete}, nil
				})
			engine := harness.idle(t, mustRegister(t, step))

			queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("only")))
			if err != nil {
				t.Fatalf("Enqueue: %v", err)
			}
			item := itemOf(t, harness, queued.ID, harness.pages[0])
			item.Status = tc.status
			if ok, persistErr := harness.items.Persist(t.Context(), item, item.AdvanceSeq); persistErr != nil || !ok {
				t.Fatalf("set the item status: %v, %v", ok, persistErr)
			}

			artifacts := make([]run.Artifact, 0, len(tc.kinds))
			for _, kind := range tc.kinds {
				artifact, newErr := run.NewArtifact(run.Artifact{
					ID: id.New(), RunID: queued.ID, ItemID: item.ID, Step: "only", Kind: kind,
					Blob: []byte("{}"), CreatedAt: time.Now().UTC(),
				})
				if newErr != nil {
					t.Fatalf("NewArtifact: %v", newErr)
				}
				artifacts = append(artifacts, artifact)
			}
			if err = harness.blobs.ReplaceStep(t.Context(), item.ID, "only", artifacts); err != nil {
				t.Fatalf("ReplaceStep: %v", err)
			}

			runID := queued.ID
			if tc.other {
				runID = id.New()
			}
			err = engine.Regenerate(t.Context(), runID, []string{item.ID})
			if !errors.IsCode(err, tc.want) {
				t.Fatalf("Regenerate = %v, want %s", err, tc.want)
			}
			if tc.reason != "" {
				var refusal *errors.Error
				if !stderrors.As(err, &refusal) || refusal.Details["reason"] != tc.reason {
					t.Fatalf("the refusal = %v, want the reason %q", err, tc.reason)
				}
			}

			kept := itemOf(t, harness, queued.ID, harness.pages[0])
			if kept.Status != tc.status || kept.CurrentStep != item.CurrentStep {
				t.Fatalf("a refused regeneration moved the item to %+v", kept)
			}
		})
	}
}
