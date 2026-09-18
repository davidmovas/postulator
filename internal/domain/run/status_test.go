package run_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestStatusPredicates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       run.Status
		valid       bool
		active      bool
		terminal    bool
		advanceable bool
	}{
		{name: "pending", input: run.StatusPending, valid: true, active: true, advanceable: true},
		{name: "running", input: run.StatusRunning, valid: true, active: true, advanceable: true},
		{name: "waiting", input: run.StatusWaiting, valid: true, active: true, advanceable: true},
		{name: "paused", input: run.StatusPaused, valid: true, active: true},
		{name: "completed", input: run.StatusCompleted, valid: true, terminal: true},
		{name: "failed", input: run.StatusFailed, valid: true, terminal: true},
		{name: "cancelled", input: run.StatusCancelled, valid: true, terminal: true},
		{name: "unknown", input: run.Status("sleeping")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.input.Valid(); got != tc.valid {
				t.Errorf("Valid() = %v, want %v", got, tc.valid)
			}
			if got := tc.input.Active(); got != tc.active {
				t.Errorf("Active() = %v, want %v", got, tc.active)
			}
			if got := tc.input.Terminal(); got != tc.terminal {
				t.Errorf("Terminal() = %v, want %v", got, tc.terminal)
			}
			if got := tc.input.Advanceable(); got != tc.advanceable {
				t.Errorf("Advanceable() = %v, want %v", got, tc.advanceable)
			}
		})
	}
}

func TestTransitionAndPauseReasonAreClosedSets(t *testing.T) {
	t.Parallel()

	transitions := []run.Transition{
		run.TransitionContinue, run.TransitionWait, run.TransitionPause, run.TransitionComplete, run.TransitionFail,
	}
	for _, transition := range transitions {
		if !transition.Valid() {
			t.Errorf("Transition(%q).Valid() = false", transition)
		}
	}
	if run.Transition("teleport").Valid() {
		t.Error("an unknown transition must not validate")
	}

	reasons := []run.PauseReason{
		run.PauseBudgetExceeded, run.PauseAwaitingConfirmation, run.PauseNeedsHuman, run.PauseUser,
	}
	for _, reason := range reasons {
		if !reason.Valid() {
			t.Errorf("PauseReason(%q).Valid() = false", reason)
		}
	}
	if run.PauseReason("bored").Valid() {
		t.Error("an unknown pause reason must not validate")
	}
}

func TestSortIsAClosedSet(t *testing.T) {
	t.Parallel()

	if !run.SortCreatedAt.Valid() || !run.SortStatus.Valid() {
		t.Error("the declared sorts must validate")
	}
	if run.Sort("brightness").Valid() {
		t.Error("an unknown sort must not validate")
	}
}
