package runtime_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestARunSurvivesTheDeathOfItsEngine(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 3)
	calls := newCounter()
	blocked := make(chan struct{})

	panicking := producing("generate_body", run.ArtifactBodyHTML, nil,
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			if attempt := calls.hit(sc.Page.ID); attempt == 1 && sc.Page.ID == harness.pages[1] {
				panic("the writer blew up on the second page")
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>" + sc.Page.Path + "</p>")}},
				Tokens:    120,
			}, nil
		})

	held := producing("validate", run.ArtifactValidationReport, []run.ArtifactKind{run.ArtifactBodyHTML},
		func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit("validate:" + sc.Page.ID)
			select {
			case <-blocked:
			case <-ctx.Done():
				return run.Result{}, ctx.Err()
			}
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactValidationReport, Blob: []byte(`{}`)}}}, nil
		})

	first := harness.engine(t, mustRegister(t, panicking, held))
	queued, err := first.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body", "validate")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	waitFor(t, "every item to reach the blocked step", func() bool {
		items, listErr := harness.items.ByRun(t.Context(), queued.ID)
		if listErr != nil {
			return false
		}
		for _, item := range items {
			if item.CurrentStep != "validate" {
				return false
			}
		}
		return len(items) == 3
	})

	first.Stop()

	generated := map[string]int{}
	for _, pageID := range harness.pages {
		generated[pageID] = calls.get(pageID)
	}
	if generated[harness.pages[1]] != 2 {
		t.Fatalf("the page whose step panicked ran %d times, want one panic and one retry",
			generated[harness.pages[1]])
	}
	for _, pageID := range []string{harness.pages[0], harness.pages[2]} {
		if generated[pageID] != 1 {
			t.Fatalf("page %s ran generate_body %d times before the engine died", pageID, generated[pageID])
		}
	}

	close(blocked)
	calls2 := newCounter()
	second := harness.engine(t, mustRegister(t, bodyStep(calls2), validateStep(calls2)))
	defer second.Stop()

	finished := harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if finished.Stats.Done != 3 || finished.Stats.Failed != 0 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}

	for _, pageID := range harness.pages {
		if got := calls2.get(pageID); got != 0 {
			t.Errorf("the second engine ran generate_body %d times for page %s; the artifact must be reused",
				got, pageID)
		}
	}

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil {
		t.Fatalf("ByRun: %v", err)
	}
	for _, item := range items {
		if item.Status != run.StatusCompleted {
			t.Errorf("item %s = %q", item.ID, item.Status)
		}
		artifacts, artifactErr := harness.blobs.ByItem(t.Context(), item.ID)
		if artifactErr != nil {
			t.Fatalf("ByItem: %v", artifactErr)
		}
		kinds := map[run.ArtifactKind]int{}
		for _, artifact := range artifacts {
			kinds[artifact.Kind]++
		}
		if kinds[run.ArtifactBodyHTML] != 1 || kinds[run.ArtifactValidationReport] != 1 {
			t.Errorf("item %s carries %v", item.ID, kinds)
		}
	}

	assertGapless(t, harness, queued.ID)
}
