package runtime_test

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

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
