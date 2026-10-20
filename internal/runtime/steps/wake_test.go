package steps_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func driveWaking(t *testing.T, def run.StepDef, sc *run.StepContext, at time.Time) int {
	t.Helper()

	waits := 0
	for range 64 {
		result, err := def.Run(t.Context(), sc)
		if err != nil {
			t.Fatalf("%s: %v", def.Name, err)
		}
		settle(def, sc, result)
		if result.Next != run.TransitionWait {
			return waits
		}
		waits++
		if !result.WakeAt.Equal(at) {
			t.Fatalf("%s waited to %v after %q, want it to wake at once (%v): a zero wake is a whole sweep interval",
				def.Name, result.WakeAt, result.Message, at)
		}
	}
	t.Fatalf("step %s never settled", def.Name)
	return waits
}

func TestEveryWriteUnitWakesAtOnce(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		build func(t *testing.T) (run.StepDef, *run.StepContext)
	}{
		{name: "publish", build: func(t *testing.T) (run.StepDef, *run.StepContext) {
			h := newProductHarness(t, storeProduct())
			h.deps.Clock = clock.NewFake(at)
			return steps.Publish(h.deps), storeContext(t, h.held.ID)
		}},
		{name: "relink_page", build: func(t *testing.T) (run.StepDef, *run.StepContext) {
			deps, _, _, wpID := relinkPageDeps(t, espressoBody)
			deps.Clock = clock.NewFake(at)
			return steps.RelinkPage(deps), relinkPageContext(t, deps, wpID)
		}},
		{name: "relink_neighbors", build: func(t *testing.T) (run.StepDef, *run.StepContext) {
			deps, _, _, _ := relinkDeps(t, parentBody)
			deps.Clock = clock.NewFake(at)
			return steps.RelinkNeighbors(deps), relinkContext(t, deps)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			def, sc := tc.build(t)
			if waits := driveWaking(t, def, sc, at); waits == 0 {
				t.Fatalf("%s settled without a wait, so the test proves nothing", def.Name)
			}
		})
	}
}
