package runtime_test

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime"
)

type lifecycle struct {
	name    string
	steps   func(calls *counter) []run.StepDef
	recipe  []string
	prepare func(h *harness, record *run.Run)
	before  func(t *testing.T, engine *runtime.Engine, runID string)
	stopsAt run.Status
	then    func(t *testing.T, engine *runtime.Engine, runID, itemID string)
	want    run.Status
	events  []events.Type
	execs   []string
	settled func(t *testing.T, record run.Run)
}

func linkContext(name string) run.StepDef {
	return producing(name, run.ArtifactLinkContext, nil,
		func(context.Context, *run.StepContext) (run.Result, error) {
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactLinkContext, Blob: []byte("context")}}}, nil
		})
}

func finalReport(name string, firstCall func() (run.Result, error), calls *counter) run.StepDef {
	return producing(name, run.ArtifactFinalReport, []run.ArtifactKind{run.ArtifactLinkContext},
		func(context.Context, *run.StepContext) (run.Result, error) {
			if calls.hit(name) == 1 && firstCall != nil {
				return firstCall()
			}
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactFinalReport, Blob: []byte("report")}}}, nil
		})
}

func onlyReport(firstCall func() (run.Result, error), calls *counter) run.StepDef {
	return producing("report", run.ArtifactFinalReport, nil,
		func(context.Context, *run.StepContext) (run.Result, error) {
			if calls.hit("report") == 1 {
				return firstCall()
			}
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactFinalReport, Blob: []byte("report")}}}, nil
		})
}

func lifecycles() []lifecycle {
	return []lifecycle{
		{
			name: "two steps run and the run completes",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{linkContext("first"), finalReport("second", nil, calls)}
			},
			recipe: []string{"first", "second"},
			want:   run.StatusCompleted,
			events: []events.Type{
				events.RunQueued, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepDone, events.StepStarted, events.StepDone,
				events.ItemDone, events.RunCompleted,
			},
			execs: []string{"first:1:done", "second:1:done"},
		},
		{
			name: "a wait that wrote nothing runs its step again",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{onlyReport(func() (run.Result, error) {
					return run.Result{Next: run.TransitionWait, WakeAt: time.Now().Add(-time.Minute)}, nil
				}, calls)}
			},
			recipe: []string{"report"},
			want:   run.StatusCompleted,
			events: []events.Type{
				events.RunQueued, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepDone, events.StepStarted, events.StepDone,
				events.ItemDone, events.RunCompleted,
			},
			execs: []string{"report:1:started", "report:2:done"},
		},
		{
			name: "a transient fault is retried after its backoff",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{onlyReport(func() (run.Result, error) {
					return run.Result{}, errors.New(errors.External, "the provider hiccuped")
				}, calls)}
			},
			recipe: []string{"report"},
			want:   run.StatusCompleted,
			events: []events.Type{
				events.RunQueued, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepRetrying, events.StepStarted, events.StepDone,
				events.ItemDone, events.RunCompleted,
			},
			execs: []string{"report:1:failed", "report:2:done"},
		},
		{
			name: "a held step goes on once a person accepts it",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{onlyReport(func() (run.Result, error) {
					return run.Result{Next: run.TransitionPause, Reason: run.PauseNeedsHuman, Message: "a finding needs a decision"}, nil
				}, calls)}
			},
			recipe:  []string{"report"},
			stopsAt: run.StatusPaused,
			then: func(t *testing.T, engine *runtime.Engine, _, itemID string) {
				if err := engine.Accept(t.Context(), itemID); err != nil {
					t.Fatalf("Accept: %v", err)
				}
			},
			want: run.StatusCompleted,
			events: []events.Type{
				events.RunQueued, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepDone, events.ItemNeedsHuman, events.RunPaused,
				events.RunResumed, events.StepStarted, events.StepDone, events.ItemDone, events.RunCompleted,
			},
			execs: []string{"report:1:failed", "report:2:done"},
		},
		{
			name: "a failed step is retried by hand",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{onlyReport(func() (run.Result, error) {
					return run.Result{Next: run.TransitionFail, Message: "the body lacks a section"}, nil
				}, calls)}
			},
			recipe:  []string{"report"},
			stopsAt: run.StatusFailed,
			then: func(t *testing.T, engine *runtime.Engine, _, itemID string) {
				if err := engine.RetryStep(t.Context(), itemID); err != nil {
					t.Fatalf("RetryStep: %v", err)
				}
			},
			want: run.StatusCompleted,
			events: []events.Type{
				events.RunQueued, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepFailed, events.ItemFailed, events.RunFailed,
				events.RunResumed, events.StepStarted, events.StepDone, events.ItemDone, events.RunCompleted,
			},
			execs: []string{"report:1:failed", "report:2:done"},
		},
		{
			name: "a stopped item is regenerated from its first step",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{linkContext("first"), finalReport("second", func() (run.Result, error) {
					return run.Result{Next: run.TransitionFail, Message: "the body lacks a section"}, nil
				}, calls)}
			},
			recipe:  []string{"first", "second"},
			stopsAt: run.StatusFailed,
			then: func(t *testing.T, engine *runtime.Engine, runID, itemID string) {
				if err := engine.Regenerate(t.Context(), runID, []string{itemID}); err != nil {
					t.Fatalf("Regenerate: %v", err)
				}
			},
			want: run.StatusCompleted,
			events: []events.Type{
				events.RunQueued, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepDone, events.StepStarted, events.StepFailed,
				events.ItemFailed, events.RunFailed, events.ItemRestarted, events.RunResumed,
				events.StepStarted, events.StepDone, events.StepStarted, events.StepDone,
				events.ItemDone, events.RunCompleted,
			},
			execs: []string{"first:1:done", "first:2:done", "second:1:failed", "second:2:done"},
		},
		{
			name: "a run over its budget is paused with the item it holds",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{linkContext("first"), finalReport("second", nil, calls)}
			},
			recipe: []string{"first", "second"},
			prepare: func(h *harness, record *run.Run) {
				h.spend.set(llm.Spend{Usage: llm.Usage{Input: 10, Output: 10, Total: 20}, USD: 5, Calls: 1})
				record.Budget = run.Budget{MaxUSD: 1}
			},
			want: run.StatusPaused,
			events: []events.Type{
				events.RunQueued, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepDone, events.RunBudgetExceeded, events.RunPaused,
			},
			execs: []string{"first:1:done"},
		},
		{
			name: "a page no template answers for fails before it starts",
			steps: func(*counter) []run.StepDef {
				return []run.StepDef{linkContext("first")}
			},
			recipe: []string{"first"},
			prepare: func(h *harness, _ *run.Run) {
				h.specs.err = errors.New(errors.NotFound, "no template answers for the page")
			},
			want:   run.StatusFailed,
			events: []events.Type{events.RunQueued, events.ItemFailed, events.RunFailed},
			execs:  []string{},
		},
		{
			name: "a run past its deadline fails without running a step",
			steps: func(*counter) []run.StepDef {
				return []run.StepDef{linkContext("first")}
			},
			recipe: []string{"first"},
			prepare: func(_ *harness, record *run.Run) {
				record.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
				record.DeadlineAt = time.Now().UTC().Add(-time.Hour)
			},
			want:   run.StatusFailed,
			events: []events.Type{events.RunQueued, events.RunFailed},
			execs:  []string{},
		},
		{
			name: "a run paused before it started resumes where it stood",
			steps: func(calls *counter) []run.StepDef {
				return []run.StepDef{onlyReport(func() (run.Result, error) {
					return run.Result{}, nil
				}, calls)}
			},
			recipe: []string{"report"},
			before: func(t *testing.T, engine *runtime.Engine, runID string) {
				if err := engine.Pause(t.Context(), runID, ""); err != nil {
					t.Fatalf("Pause: %v", err)
				}
				if err := engine.Resume(t.Context(), runID); err != nil {
					t.Fatalf("Resume: %v", err)
				}
			},
			want: run.StatusCompleted,
			events: []events.Type{
				events.RunQueued, events.RunPaused, events.RunResumed, events.RunStarted, events.ItemStarted,
				events.StepStarted, events.StepDone, events.ItemDone, events.RunCompleted,
			},
			execs: []string{"report:1:done"},
			settled: func(t *testing.T, record run.Run) {
				if record.StartedAt == nil || record.FinishedAt == nil || record.StartedAt.After(*record.FinishedAt) {
					t.Fatalf("the run started at %v and finished at %v, want a start before its finish",
						record.StartedAt, record.FinishedAt)
				}
			},
		},
	}
}

func TestEveryPathThroughARunRecordsItsEventsAndExecsInOrder(t *testing.T) {
	t.Parallel()

	for _, tc := range lifecycles() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, 1)
			record := harness.newRun(recipeOf(tc.recipe...))
			if tc.prepare != nil {
				tc.prepare(harness, &record)
			}

			engine := harness.idle(t, mustRegister(t, tc.steps(newCounter())...))
			queued, err := engine.Enqueue(t.Context(), record)
			if err != nil {
				t.Fatalf("Enqueue: %v", err)
			}
			if tc.before != nil {
				tc.before(t, engine, queued.ID)
			}
			if err = engine.Start(t.Context()); err != nil {
				t.Fatalf("start the engine: %v", err)
			}
			t.Cleanup(engine.Stop)

			itemID := onlyItem(t, harness, queued.ID)
			if tc.then != nil {
				harness.waitForRun(t, queued.ID, tc.stopsAt)
				tc.then(t, engine, queued.ID, itemID)
			}
			settled := harness.waitForRun(t, queued.ID, tc.want)
			if tc.settled != nil {
				tc.settled(t, settled)
			}

			if got := loggedTypes(t, harness, queued.ID); !slices.Equal(got, tc.events) {
				t.Fatalf("the run logged\n%v\nwant\n%v", got, tc.events)
			}
			if got := execsOf(t, harness, itemID); !slices.Equal(got, tc.execs) {
				t.Fatalf("the item recorded the execs %v, want %v", got, tc.execs)
			}
		})
	}
}

func onlyItem(t *testing.T, harness *harness, runID string) string {
	t.Helper()

	items, err := harness.items.ByRun(t.Context(), runID)
	if err != nil || len(items) != 1 {
		t.Fatalf("the run holds %d items (%v), want one", len(items), err)
	}
	return items[0].ID
}

func loggedTypes(t *testing.T, harness *harness, runID string) []events.Type {
	t.Helper()

	stored, err := harness.log.List(t.Context(), runID, 0, 1000)
	if err != nil {
		t.Fatalf("list the run events: %v", err)
	}
	out := make([]events.Type, 0, len(stored))
	for i := range stored {
		out = append(out, events.Type(stored[i].Type))
	}
	return out
}

func execsOf(t *testing.T, harness *harness, itemID string) []string {
	t.Helper()

	execs, err := harness.execs.ByItem(t.Context(), itemID)
	if err != nil {
		t.Fatalf("list the step execs: %v", err)
	}
	out := make([]string, 0, len(execs))
	for i := range execs {
		out = append(out, execs[i].Step+":"+strconv.Itoa(execs[i].Attempt)+":"+string(execs[i].Status))
	}
	slices.Sort(out)
	return out
}
