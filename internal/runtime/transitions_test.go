package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime"
)

func TestWaitingIsWokenOnDemand(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	sleeper := producing("generate_body", run.ArtifactBodyHTML, nil,
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			if calls.hit(sc.Page.ID) == 1 {
				return run.Result{Next: run.TransitionWait, WakeAt: time.Now().Add(time.Hour)}, nil
			}
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>ok</p>")}}}, nil
		})

	engine := harness.engine(t, mustRegister(t, sleeper))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	var itemID string
	waitFor(t, "the item to start waiting", func() bool {
		items, listErr := harness.items.ByRun(t.Context(), queued.ID)
		if listErr != nil || len(items) != 1 || items[0].Status != run.StatusWaiting {
			return false
		}
		itemID = items[0].ID
		return items[0].WakeAt != nil
	})

	if err = engine.Wake(t.Context(), itemID); err != nil {
		t.Fatalf("Wake: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)

	if err = engine.Wake(t.Context(), itemID); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Wake of a finished item = %v", err)
	}
	if err = engine.Wake(t.Context(), "3f1a0a0c-0000-4000-8000-000000000000"); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Wake of an absent item = %v", err)
	}
}

func TestAStepCanCompleteOrFailTheItemItself(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		transition run.Transition
		reason     run.PauseReason
		want       run.Status
		runStatus  run.Status
		event      events.Type
	}{
		{
			name: "complete stops the item early", transition: run.TransitionComplete,
			want: run.StatusCompleted, runStatus: run.StatusCompleted, event: events.ItemDone,
		},
		{
			name: "fail stops the item", transition: run.TransitionFail,
			want: run.StatusFailed, runStatus: run.StatusFailed, event: events.ItemFailed,
		},
		{
			name: "pause holds the item for a human", transition: run.TransitionPause,
			reason: run.PauseAwaitingConfirmation, want: run.StatusPaused, event: events.ItemNeedsHuman,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, 1)
			decisive := producing("generate_body", run.ArtifactBodyHTML, nil,
				func(context.Context, *run.StepContext) (run.Result, error) {
					return run.Result{Next: tc.transition, Reason: tc.reason}, nil
				})
			skipped := producing("validate", run.ArtifactValidationReport, nil,
				func(context.Context, *run.StepContext) (run.Result, error) {
					t.Error("the second step must not run")
					return run.Result{}, nil
				})

			engine := harness.engine(t, mustRegister(t, decisive, skipped))
			queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body", "validate")))
			if err != nil {
				t.Fatalf("Enqueue: %v", err)
			}

			waitFor(t, "the item to settle", func() bool {
				items, listErr := harness.items.ByRun(t.Context(), queued.ID)
				return listErr == nil && len(items) == 1 && items[0].Status == tc.want
			})
			if tc.runStatus != "" {
				harness.waitForRun(t, queued.ID, tc.runStatus)
			}

			waitFor(t, "the event to be published", func() bool { return harness.bus.count(tc.event) == 1 })
			assertGapless(t, harness, queued.ID)
		})
	}
}

func TestAnItemThatCannotBeSetUpIsAbandoned(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	harness.specs.err = errors.New(errors.NotFound, "the page has no template and its site has no default")

	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls)))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	finished := harness.waitForRun(t, queued.ID, run.StatusFailed)
	if finished.Stats.Failed != 1 {
		t.Fatalf("Stats = %+v", finished.Stats)
	}
	if calls.total() != 0 {
		t.Fatalf("the step ran %d times, want none", calls.total())
	}
	if harness.bus.count(events.ItemFailed) != 1 {
		t.Fatalf("item.failed was published %d times", harness.bus.count(events.ItemFailed))
	}
	assertGapless(t, harness, queued.ID)
}

func TestARunPastItsDeadlineIsReaped(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls)))

	record := harness.newRun(recipeOf("generate_body"))
	record.CreatedAt = time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	record.DeadlineAt = record.CreatedAt.Add(time.Minute)

	queued, err := engine.Enqueue(t.Context(), record)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	finished := harness.waitForRun(t, queued.ID, run.StatusFailed)
	if finished.FinishedAt == nil || finished.Error == "" {
		t.Fatalf("the reaped run = %+v", finished)
	}

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil {
		t.Fatalf("ByRun: %v", err)
	}
	for _, item := range items {
		if item.Status != run.StatusCancelled {
			t.Errorf("item %s = %q, want cancelled", item.ID, item.Status)
		}
	}
	if harness.bus.count(events.RunFailed) != 1 {
		t.Fatalf("run.failed was published %d times", harness.bus.count(events.RunFailed))
	}
	assertGapless(t, harness, queued.ID)
}

func TestARetryBacksOffWithoutAPolicy(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	flaky := run.StepDef{
		Name:     "generate_body",
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Run: func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			if calls.hit(sc.Page.ID) == 1 {
				return run.Result{}, errors.New(errors.RateLimited, "too many requests")
			}
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>ok</p>")}}}, nil
		},
	}

	engine := harness.engine(t, mustRegister(t, flaky))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	harness.waitForRun(t, queued.ID, run.StatusCompleted)
	if got := calls.get(harness.pages[0]); got != 2 {
		t.Fatalf("the step ran %d times", got)
	}
}

func TestSettingsCarryTheDeclaredDefaults(t *testing.T) {
	t.Parallel()

	cfg := runtime.Settings(settings.Default().NewValues())
	want := runtime.Config{
		Workers:            runtime.DefaultWorkers,
		PerSite:            runtime.DefaultPerSite,
		SweepInterval:      runtime.DefaultSweepInterval,
		RetentionDays:      runtime.DefaultRetentionDays,
		EventRetentionDays: runtime.DefaultEventRetentionDays,
		StepTimeout:        runtime.DefaultStepTimeout,
		LeaseDuration:      runtime.DefaultLeaseDuration,
		RunDeadline:        runtime.DefaultRunDeadline,
	}
	if cfg != want {
		t.Fatalf("Settings = %+v, want %+v", cfg, want)
	}

	for _, key := range []string{
		"runs.workers", "runs.perSite", "runs.sweepInterval", "runs.artifactRetentionDays",
		"runs.eventRetentionDays", "runs.stepTimeout", "runs.leaseDuration", "runs.deadline",
	} {
		if !settings.Default().Has(key) {
			t.Errorf("the setting %q is not declared", key)
		}
	}
}

func TestAnEngineWithNoConfigurationStillRuns(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	engine := runtime.New(runtime.Deps{
		Runs: harness.runs, Items: harness.items, Artifacts: harness.blobs, Execs: harness.execs,
		Events: harness.log, Pages: pageRepoOf(harness), Specs: harness.specs, Spend: harness.spend,
		Catalog: stubCatalog{}, Profiles: stubProfiles{}, UnitOfWork: harness.store, Publisher: harness.bus,
	}, mustRegister(t, bodyStep(calls)), runtime.Config{}, systemClock(), harness.logger)

	if err := engine.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer engine.Stop()

	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	harness.waitForRun(t, queued.ID, run.StatusCompleted)
}
