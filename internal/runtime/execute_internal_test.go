package runtime

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestAnExecIsRecordedByWhatItsStepLeftBehind(t *testing.T) {
	t.Parallel()

	retry := &run.Fault{Action: run.ActionRetry, Message: "the provider hiccuped"}
	failure := &run.Fault{Action: run.ActionFail, Message: "the body lacks a section"}
	held := &run.Fault{Action: run.ActionPause, Message: "the budget ran out"}

	cases := []struct {
		name    string
		out     outcome
		status  run.ExecStatus
		message string
	}{
		{name: "a step that hands on is done", out: outcome{status: run.StatusPending}, status: run.ExecDone},
		{name: "a step that completes the item is done", out: outcome{status: run.StatusCompleted}, status: run.ExecDone},
		{
			name: "a wait that wrote nothing is only started", out: outcome{status: run.StatusWaiting},
			status: run.ExecStarted,
		},
		{
			name:   "a wait that wrote an artifact is done",
			out:    outcome{status: run.StatusWaiting, artifacts: []run.Artifact{{Kind: run.ArtifactDraft}}},
			status: run.ExecDone,
		},
		{
			name:   "a wait that kept a checkpoint is done",
			out:    outcome{status: run.StatusWaiting, checkpoint: run.Checkpoint{"cursor": json.RawMessage(`1`)}},
			status: run.ExecDone,
		},
		{
			name:   "a retry is a failed attempt",
			out:    outcome{status: run.StatusWaiting, fault: retry, message: retry.Message},
			status: run.ExecFailed, message: "the provider hiccuped",
		},
		{
			name:   "a failure carries its fault",
			out:    outcome{status: run.StatusFailed, fault: failure, message: failure.Message},
			status: run.ExecFailed, message: "the body lacks a section",
		},
		{
			name:   "a pause the step asked for keeps its note",
			out:    outcome{status: run.StatusPaused, message: "a finding needs a decision"},
			status: run.ExecFailed, message: "a finding needs a decision",
		},
		{
			name:   "a pause a fault caused carries the fault",
			out:    outcome{status: run.StatusPaused, fault: held, message: "the note"},
			status: run.ExecFailed, message: "the budget ran out",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, message := execOf(tc.out)
			if status != tc.status || message != tc.message {
				t.Fatalf("execOf = %s %q, want %s %q", status, message, tc.status, tc.message)
			}
		})
	}
}
