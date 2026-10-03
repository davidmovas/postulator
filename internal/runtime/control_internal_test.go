package runtime

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestARunIsAtRestOnlyWhenPausedOrFinished(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status run.Status
		want   bool
	}{
		{name: "pending", status: run.StatusPending},
		{name: "running", status: run.StatusRunning},
		{name: "waiting", status: run.StatusWaiting},
		{name: "paused", status: run.StatusPaused, want: true},
		{name: "completed", status: run.StatusCompleted, want: true},
		{name: "failed", status: run.StatusFailed, want: true},
		{name: "cancelled", status: run.StatusCancelled, want: true},
		{name: "no status", status: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := atRest(tc.status); got != tc.want {
				t.Fatalf("atRest(%q) = %t, want %t", tc.status, got, tc.want)
			}
		})
	}
}
