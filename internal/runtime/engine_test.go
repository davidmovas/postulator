package runtime_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func bodyStep(calls *counter) run.StepDef {
	return producing("generate_body", run.ArtifactBodyHTML, nil,
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit(sc.Page.ID)
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>" + sc.Page.Path + "</p>")}},
				Tokens:    120,
				USD:       0.01,
			}, nil
		})
}

func validateStep(calls *counter) run.StepDef {
	return producing("validate", run.ArtifactValidationReport, []run.ArtifactKind{run.ArtifactBodyHTML},
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit("validate:" + sc.Page.ID)
			body, err := sc.Artifact(run.ArtifactBodyHTML)
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactValidationReport, Blob: []byte(`{"size":` +
					itoa(len(body.Blob)) + `}`)}},
			}, nil
		})
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

func TestRunCompletesEveryItemInOrder(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 3)
	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls), validateStep(calls)))

	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body", "validate")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	finished := harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if finished.Stats.Done != 3 || finished.Stats.Failed != 0 || finished.Stats.Items != 3 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}
	if finished.FinishedAt == nil || finished.StartedAt == nil {
		t.Fatalf("times = %v, %v", finished.StartedAt, finished.FinishedAt)
	}

	for _, pageID := range harness.pages {
		if got := calls.get(pageID); got != 1 {
			t.Errorf("generate_body ran %d times for page %s, want once", got, pageID)
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
		if artifactErr != nil || len(artifacts) != 2 {
			t.Fatalf("artifacts of %s = %d, %v", item.ID, len(artifacts), artifactErr)
		}
	}

	assertGapless(t, harness, queued.ID)
	types := harness.bus.types()
	if types[0] != events.RunQueued || types[1] != events.RunStarted {
		t.Fatalf("the first events were %v", types[:2])
	}
	if types[len(types)-1] != events.RunCompleted {
		t.Fatalf("the last event was %q", types[len(types)-1])
	}
	if harness.bus.count(events.ItemDone) != 3 || harness.bus.count(events.StepDone) != 6 {
		t.Fatalf("item.done = %d, step.done = %d", harness.bus.count(events.ItemDone), harness.bus.count(events.StepDone))
	}
}

func assertGapless(t *testing.T, harness *harness, runID string) {
	t.Helper()

	stored, err := harness.log.List(t.Context(), runID, 0, 1000)
	if err != nil {
		t.Fatalf("list the run events: %v", err)
	}
	if len(stored) == 0 {
		t.Fatal("the run recorded no events")
	}
	for i, event := range stored {
		if event.Seq != int64(i+1) {
			t.Fatalf("event %d carries seq %d", i, event.Seq)
		}
		if event.RunID != runID {
			t.Fatalf("event %d belongs to run %s", i, event.RunID)
		}
	}
}

func TestATransientFaultIsRetriedAndThenSucceeds(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	flaky := producing("generate_body", run.ArtifactBodyHTML, nil,
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			if calls.hit(sc.Page.ID) == 1 {
				return run.Result{}, faulty(errors.External, "the provider hung up")
			}
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>ok</p>")}}}, nil
		})

	engine := harness.engine(t, mustRegister(t, flaky))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if got := calls.get(harness.pages[0]); got != 2 {
		t.Fatalf("the step ran %d times, want one failure and one success", got)
	}
	if harness.bus.count(events.StepRetrying) != 1 {
		t.Fatalf("step.retrying was published %d times", harness.bus.count(events.StepRetrying))
	}
	retrying, ok := harness.bus.payloads(events.StepRetrying)[0].(events.StepRetryingPayload)
	if !ok || retrying.Code != "EXTERNAL" || !strings.Contains(retrying.Message, "hung up") {
		t.Fatalf("step.retrying = %+v, want the fault's code and sentence", harness.bus.payloads(events.StepRetrying)[0])
	}
	assertGapless(t, harness, queued.ID)
}

func TestStepDoneCarriesTheSentenceOfTheStep(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	writer := producing("generate_body", run.ArtifactBodyHTML, nil,
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>ok</p>")}},
				Message:   "wrote the body of " + sc.Page.Path,
			}, nil
		})

	engine := harness.engine(t, mustRegister(t, writer))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)

	done := harness.bus.payloads(events.StepDone)
	if len(done) != 1 {
		t.Fatalf("step.done was published %d times", len(done))
	}
	payload, ok := done[0].(events.StepDonePayload)
	if !ok || payload.Message != "wrote the body of /page-a/" || payload.Step != "generate_body" {
		t.Fatalf("step.done = %+v, want the step's own sentence", done[0])
	}
}

func TestAnExhaustedStepFailsTheItemButNotTheRun(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	calls := newCounter()
	selective := run.StepDef{
		Name:     "generate_body",
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 2, Backoff: func(int) time.Duration { return 5 * time.Millisecond }},
		Run: func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit(sc.Page.ID)
			if sc.Page.ID == harness.pages[1] {
				return run.Result{}, faulty(errors.External, "this page always fails")
			}
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>ok</p>")}}}, nil
		},
	}

	engine := harness.engine(t, mustRegister(t, selective))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	finished := harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if finished.Stats.Done != 1 || finished.Stats.Failed != 1 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}
	if got := calls.get(harness.pages[1]); got != 2 {
		t.Fatalf("the failing page ran %d times, want the retry ceiling", got)
	}
	if harness.bus.count(events.ItemFailed) != 1 {
		t.Fatalf("item.failed was published %d times", harness.bus.count(events.ItemFailed))
	}
	assertGapless(t, harness, queued.ID)
}

func TestARunFailsOnlyWhenEveryItemFails(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	doomed := run.StepDef{
		Name:     "generate_body",
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 1},
		Run: func(context.Context, *run.StepContext) (run.Result, error) {
			return run.Result{}, faulty(errors.Invalid, "the page has no entity")
		},
	}

	engine := harness.engine(t, mustRegister(t, doomed))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	finished := harness.waitForRun(t, queued.ID, run.StatusFailed)
	if finished.Stats.Done != 0 || finished.Stats.Failed != 2 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}
	if harness.bus.count(events.RunFailed) != 1 {
		t.Fatalf("run.failed was published %d times", harness.bus.count(events.RunFailed))
	}
	assertGapless(t, harness, queued.ID)
}

func TestAPausingStepStopsTheItemForAHuman(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	needy := producing("generate_body", run.ArtifactBodyHTML, nil,
		func(context.Context, *run.StepContext) (run.Result, error) {
			return run.Result{}, faulty(errors.NeedsHuman, "the anchor list needs review")
		})

	engine := harness.engine(t, mustRegister(t, needy))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	waitFor(t, "the item to pause", func() bool {
		items, listErr := harness.items.ByRun(t.Context(), queued.ID)
		return listErr == nil && len(items) == 1 && items[0].Status == run.StatusPaused
	})

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil {
		t.Fatalf("ByRun: %v", err)
	}
	if items[0].PauseReason != run.PauseNeedsHuman {
		t.Fatalf("PauseReason = %q", items[0].PauseReason)
	}
	if harness.bus.count(events.ItemNeedsHuman) != 1 {
		t.Fatalf("item.needs_human was published %d times", harness.bus.count(events.ItemNeedsHuman))
	}

	if err = engine.RetryStep(t.Context(), items[0].ID); err != nil {
		t.Fatalf("RetryStep: %v", err)
	}
	waitFor(t, "the item to be retried", func() bool {
		stored, getErr := harness.items.Get(t.Context(), items[0].ID)
		return getErr == nil && stored.AdvanceSeq > items[0].AdvanceSeq
	})
}

func TestBudgetExhaustionPausesTheRun(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	harness.spend.set(llm.Spend{Usage: llm.Usage{Input: 10, Output: 10, Total: 20}, USD: 5, Calls: 1})

	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls), validateStep(calls)))

	record := harness.newRun(recipeOf("generate_body", "validate"))
	record.Budget = run.Budget{MaxUSD: 1}
	queued, err := engine.Enqueue(t.Context(), record)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	paused := harness.waitForRun(t, queued.ID, run.StatusPaused)
	if paused.PauseReason != run.PauseBudgetExceeded {
		t.Fatalf("PauseReason = %q", paused.PauseReason)
	}
	if harness.bus.count(events.RunBudgetExceeded) != 1 {
		t.Fatalf("run.budget_exceeded was published %d times", harness.bus.count(events.RunBudgetExceeded))
	}
	said := budgetExceeded(t, harness)
	if said.SpentUSD != 5 || said.BudgetUSD != 1 {
		t.Fatalf("the event says %+v, want the 5 spent against the cap of 1", said)
	}
	assertGapless(t, harness, queued.ID)
}

func budgetExceeded(t *testing.T, harness *harness) events.RunBudgetExceededPayload {
	t.Helper()

	published := harness.bus.payloads(events.RunBudgetExceeded)
	if len(published) != 1 {
		t.Fatalf("run.budget_exceeded was published %d times", len(published))
	}
	said, ok := published[0].(events.RunBudgetExceededPayload)
	if !ok {
		t.Fatalf("run.budget_exceeded carried %T", published[0])
	}
	return said
}

func TestPauseResumeAndCancel(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	release := make(chan struct{})
	calls := newCounter()
	blocking := producing("generate_body", run.ArtifactBodyHTML, nil,
		func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit(sc.Page.ID)
			select {
			case <-release:
				return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>ok</p>")}}}, nil
			case <-ctx.Done():
				return run.Result{}, faulty(errors.Cancelled, "the run was cancelled")
			}
		})

	engine := harness.engine(t, mustRegister(t, blocking))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	waitFor(t, "both steps to be in flight", func() bool { return calls.total() >= 2 })

	if err = engine.Pause(t.Context(), queued.ID, run.PauseUser); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	paused, err := harness.runs.Get(t.Context(), queued.ID)
	if err != nil || paused.Status != run.StatusPaused || paused.PauseReason != run.PauseUser {
		t.Fatalf("Pause left the run as %+v, %v", paused, err)
	}
	if err = engine.Pause(t.Context(), queued.ID, run.PauseUser); err != nil {
		t.Fatalf("Pause of an already paused run: %v", err)
	}

	if err = engine.Resume(t.Context(), queued.ID); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if err = engine.Resume(t.Context(), queued.ID); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Resume of a running run = %v", err)
	}

	if err = engine.Cancel(t.Context(), queued.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	cancelled, err := harness.runs.Get(t.Context(), queued.ID)
	if err != nil || cancelled.Status != run.StatusCancelled {
		t.Fatalf("Cancel left the run as %+v, %v", cancelled, err)
	}
	if err = engine.Cancel(t.Context(), queued.ID); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Cancel of a finished run = %v", err)
	}
	if err = engine.Pause(t.Context(), queued.ID, run.PauseUser); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Pause of a finished run = %v", err)
	}

	close(release)

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil {
		t.Fatalf("ByRun: %v", err)
	}
	for _, item := range items {
		if item.Status != run.StatusCancelled {
			t.Errorf("item %s = %q, want cancelled", item.ID, item.Status)
		}
	}
	assertGapless(t, harness, queued.ID)
}

func TestARepeatedStepReusesItsRecordedOutcome(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls), validateStep(calls)))

	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body", "validate")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	before := calls.get("validate:" + harness.pages[0])

	if err = engine.RetryStep(t.Context(), items[0].ID); err != nil {
		t.Fatalf("RetryStep: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)

	waitFor(t, "the retried item to settle", func() bool {
		stored, getErr := harness.items.Get(t.Context(), items[0].ID)
		return getErr == nil && stored.Status == run.StatusCompleted
	})
	if after := calls.get("validate:" + harness.pages[0]); after != before {
		t.Fatalf("validate ran %d times after the retry, want the recorded outcome to be reused", after)
	}

	execs, err := harness.execs.ByItem(t.Context(), items[0].ID)
	if err != nil || len(execs) != 2 {
		t.Fatalf("step executions = %d, %v", len(execs), err)
	}
}

func TestEstimateRunPricesEveryModelBackedStep(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	writer := run.StepDef{
		Name: "generate_body", Role: llm.RoleWriter, Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Run: func(context.Context, *run.StepContext) (run.Result, error) { return run.Result{}, nil },
	}
	pure := run.StepDef{
		Name: "insert_links", Requires: []run.ArtifactKind{run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Run:      func(context.Context, *run.StepContext) (run.Result, error) { return run.Result{}, nil },
	}

	engine := harness.engine(t, mustRegister(t, writer, pure))
	record := harness.newRun(recipeOf("generate_body", "insert_links"))

	estimate, err := engine.EstimateRun(t.Context(), record)
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if estimate.Tokens <= 0 || estimate.USD <= 0 {
		t.Fatalf("EstimateRun = %+v", estimate)
	}

	single := record
	single.Targets = harness.pages[:1]
	half, err := engine.EstimateRun(t.Context(), single)
	if err != nil {
		t.Fatalf("EstimateRun for one target: %v", err)
	}
	if half.Tokens*2 != estimate.Tokens {
		t.Fatalf("the estimate must scale with the targets: %d and %d", half.Tokens, estimate.Tokens)
	}

	unknown := record
	unknown.Recipe = append(slices.Clone(record.Recipe), template.StepSpec{Name: "summon", Enabled: true})
	if _, err = engine.EstimateRun(t.Context(), unknown); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("EstimateRun of an unknown step = %v", err)
	}
}

func TestEnqueueRejectsABrokenRecipe(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls), validateStep(calls)))

	if _, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("validate"))); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Enqueue of a recipe with an unmet dependency = %v", err)
	}
	if _, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("summon"))); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Enqueue of an unknown step = %v", err)
	}

	empty := harness.newRun(recipeOf("generate_body"))
	empty.Targets = nil
	if _, err := engine.Enqueue(t.Context(), empty); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Enqueue without targets = %v", err)
	}
}

func TestStartingTwiceIsRefused(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls)))

	if err := engine.Start(t.Context()); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Start of a started engine = %v", err)
	}
	engine.Stop()
	engine.Stop()
}
