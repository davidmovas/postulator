package runtime_test

import (
	"context"
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestEnqueueDispatchesAParentBeforeItsChild(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 0)
	given := []string{
		"/components/batteries/e-bike-range/",
		"/components/",
		"/electric-bikes/",
		"/components/batteries/",
	}

	pathOf := make(map[string]string, len(given))
	for _, path := range given {
		page := sqlitetest.Page(t, harness.store, harness.siteID, path)
		pathOf[page.ID] = path
		harness.pages = append(harness.pages, page.ID)
	}

	step := producing("only", run.ArtifactFinalReport, nil,
		func(context.Context, *run.StepContext) (run.Result, error) {
			return run.Result{Next: run.TransitionComplete}, nil
		})
	engine := harness.idle(t, mustRegister(t, step))

	if _, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("only"))); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	items, err := harness.items.Runnable(t.Context(), time.Now().UTC().Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("Runnable: %v", err)
	}

	got := make([]string, 0, len(items))
	for i := range items {
		got = append(got, pathOf[items[i].TargetID])
	}
	want := []string{
		"/components/",
		"/electric-bikes/",
		"/components/batteries/",
		"/components/batteries/e-bike-range/",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the run is dispatched as %v, want %v: a parent reaches WordPress before its child", got, want)
	}
}

func TestRetryStepRefusesAnItemWhoseInputsExpired(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		requires []run.ArtifactKind
		purged   []run.ArtifactKind
		refused  bool
	}{
		{
			name:     "the step consumes a purged kind",
			requires: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
			purged:   []run.ArtifactKind{run.ArtifactBodyHTML},
			refused:  true,
		},
		{
			name:     "the purged kind is not consumed by this step",
			requires: []run.ArtifactKind{run.ArtifactPublishResult},
			purged:   []run.ArtifactKind{run.ArtifactBodyHTML},
			refused:  false,
		},
		{
			name:     "nothing was purged",
			requires: []run.ArtifactKind{run.ArtifactBodyHTML},
			purged:   nil,
			refused:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, 1)
			step := producing("second", run.ArtifactFinalReport, tc.requires,
				func(context.Context, *run.StepContext) (run.Result, error) {
					return run.Result{Next: run.TransitionComplete}, nil
				})
			engine := harness.engine(t, mustRegister(t, step))

			record := harness.newRun(recipeOf("second"))
			record.Status = run.StatusFailed
			record.DeadlineAt = time.Now().UTC().Add(time.Hour)
			record.CreatedAt = time.Now().UTC()
			record.Stats.Items = 1
			validated, err := run.NewRun(record)
			if err != nil {
				t.Fatalf("NewRun: %v", err)
			}
			if err = harness.runs.Insert(t.Context(), validated); err != nil {
				t.Fatalf("insert the run: %v", err)
			}

			item, err := run.NewItem(run.Item{
				ID: id.New(), RunID: validated.ID, SiteID: validated.SiteID, TargetID: harness.pages[0],
				Status: run.StatusFailed, CurrentStep: "second", CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			})
			if err != nil {
				t.Fatalf("NewItem: %v", err)
			}
			if err = harness.items.Insert(t.Context(), item); err != nil {
				t.Fatalf("insert the item: %v", err)
			}

			artifacts := make([]run.Artifact, 0, len(tc.purged))
			for _, kind := range tc.purged {
				artifact, newErr := run.NewArtifact(run.Artifact{
					ID: id.New(), RunID: validated.ID, ItemID: item.ID, Step: "first", Kind: kind,
					CreatedAt: time.Now().UTC(),
				})
				if newErr != nil {
					t.Fatalf("NewArtifact: %v", newErr)
				}
				artifact.Purged = true
				artifacts = append(artifacts, artifact)
			}
			if err = harness.blobs.ReplaceStep(t.Context(), item.ID, "first", artifacts); err != nil {
				t.Fatalf("ReplaceStep: %v", err)
			}

			err = engine.RetryStep(t.Context(), item.ID)
			if !tc.refused {
				if err != nil {
					t.Fatalf("RetryStep = %v, want it to be accepted", err)
				}
				return
			}

			if !errors.IsCode(err, errors.NotFound) {
				t.Fatalf("RetryStep = %v, want a NOT_FOUND refusal", err)
			}
			var refusal *errors.Error
			if !stderrors.As(err, &refusal) {
				t.Fatalf("RetryStep = %v, want a kernel error", err)
			}
			if refusal.Details["reason"] != string(run.RetryBlockedInputsExpired) {
				t.Fatalf("details = %+v, want the frozen reason", refusal.Details)
			}
			if refusal.Details["kinds"] == nil || refusal.Details["step"] != "second" {
				t.Fatalf("details = %+v, want the step and the expired kinds", refusal.Details)
			}

			stored, getErr := harness.items.Get(t.Context(), item.ID)
			if getErr != nil || stored.Status != run.StatusFailed || stored.AdvanceSeq != item.AdvanceSeq {
				t.Fatalf("the refused item = %+v, %v", stored, getErr)
			}
		})
	}
}

func TestAcceptLetsAHeldStepGoOnWithItsFindings(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	grader := producing("validate", run.ArtifactValidationReport, nil,
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			result := run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactValidationReport, Blob: []byte("{}")}}}
			if sc.Accepted() {
				return result, nil
			}
			result.Next = run.TransitionPause
			result.Reason = run.PauseNeedsHuman
			result.Message = "2 findings need a decision"
			return result, nil
		})

	engine := harness.engine(t, mustRegister(t, grader))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("validate")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	paused := harness.waitForRun(t, queued.ID, run.StatusPaused)
	if paused.PauseReason != run.PauseNeedsHuman {
		t.Fatalf("the run is paused for %s", paused.PauseReason)
	}
	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	if items[0].Note != "2 findings need a decision" {
		t.Fatalf("the note reads %q", items[0].Note)
	}

	if err = engine.Accept(t.Context(), items[0].ID); err != nil {
		t.Fatalf("Accept: %v", err)
	}
	finished := harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if finished.Stats.Done != 1 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}
	accepted, err := harness.items.Get(t.Context(), items[0].ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	step, found, err := run.Get[string](accepted.Checkpoint, run.CheckpointAccept)
	if err != nil || !found || step != "validate" {
		t.Fatalf("the checkpoint records the acceptance as %q, %t, %v", step, found, err)
	}
	if accepted.Note != "" || accepted.PauseReason != "" {
		t.Fatalf("the accepted item still reads %+v", accepted)
	}
	assertGapless(t, harness, queued.ID)
}
