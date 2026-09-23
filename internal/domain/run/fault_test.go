package run_test

import (
	stderrors "errors"
	"io"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		err    error
		class  run.FaultClass
		action run.FaultAction
		reason run.PauseReason
		code   string
	}{
		{
			name: "rate limited is transient", err: errors.New(errors.RateLimited, "slow down"),
			class: run.FaultTransient, action: run.ActionRetry, code: errors.RateLimited.String(),
		},
		{
			name: "external is transient", err: errors.New(errors.External, "wordpress is down"),
			class: run.FaultTransient, action: run.ActionRetry, code: errors.External.String(),
		},
		{
			name: "a cancelled step reports a timeout", err: errors.New(errors.Cancelled, "deadline"),
			class: run.FaultTransient, action: run.ActionRetry, code: run.CodeStepTimeout,
		},
		{
			name: "a budget failure pauses the run", err: errors.New(errors.BudgetExceeded, "out of money"),
			class: run.FaultExhausted, action: run.ActionPause, reason: run.PauseBudgetExceeded,
			code: errors.BudgetExceeded.String(),
		},
		{
			name: "needs human pauses the item", err: errors.New(errors.NeedsHuman, "review me"),
			class: run.FaultExhausted, action: run.ActionPause, reason: run.PauseNeedsHuman,
			code: errors.NeedsHuman.String(),
		},
		{
			name: "invalid input fails", err: errors.New(errors.Invalid, "no page"),
			class: run.FaultInvalid, action: run.ActionFail, code: errors.Invalid.String(),
		},
		{
			name: "unauthorized fails", err: errors.New(errors.Unauthorized, "no key"),
			class: run.FaultInvalid, action: run.ActionFail, code: errors.Unauthorized.String(),
		},
		{
			name: "a foreign error is fatal", err: io.EOF,
			class: run.FaultFatal, action: run.ActionFail, code: errors.Internal.String(),
		},
		{
			name: "a wrapped kernel error keeps its code", err: stderrors.Join(errors.New(errors.Locked, "sealed")),
			class: run.FaultFatal, action: run.ActionFail, code: errors.Locked.String(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := run.Classify(tc.err)
			if got.Class != tc.class {
				t.Errorf("Class = %q, want %q", got.Class, tc.class)
			}
			if got.Action != tc.action {
				t.Errorf("Action = %q, want %q", got.Action, tc.action)
			}
			if got.Reason != tc.reason {
				t.Errorf("Reason = %q, want %q", got.Reason, tc.reason)
			}
			if got.Code != tc.code {
				t.Errorf("Code = %q, want %q", got.Code, tc.code)
			}
			if got.Message == "" {
				t.Error("a fault must carry a message")
			}
			if !got.Class.Valid() {
				t.Errorf("Class %q is not a declared class", got.Class)
			}
		})
	}
}

func TestClassifyOfNothing(t *testing.T) {
	t.Parallel()

	if got := run.Classify(nil); got != (run.Fault{}) {
		t.Fatalf("Classify(nil) = %+v, want the zero fault", got)
	}
	if run.FaultClass("weird").Valid() {
		t.Error("an unknown fault class must not validate")
	}
	if got := run.DefaultAction(run.FaultClass("weird")); got != run.ActionFail {
		t.Errorf("DefaultAction of an unknown class = %q, want %q", got, run.ActionFail)
	}
}
