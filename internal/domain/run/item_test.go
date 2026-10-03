package run_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestNewItem(t *testing.T) {
	t.Parallel()

	item, err := run.NewItem(run.Item{ID: "item-1", RunID: "run-1", SiteID: "site-1", TargetID: "page-1", CurrentStep: "resolve_context"})
	if err != nil {
		t.Fatalf("NewItem: %v", err)
	}
	if item.Status != run.StatusPending || item.Checkpoint == nil {
		t.Fatalf("NewItem = %+v", item)
	}

	cases := []struct {
		name string
		item run.Item
	}{
		{name: "no id", item: run.Item{RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s"}},
		{name: "no run", item: run.Item{ID: "i", SiteID: "s1", TargetID: "p", CurrentStep: "s"}},
		{name: "no site", item: run.Item{ID: "i", RunID: "r", TargetID: "p", CurrentStep: "s"}},
		{name: "no target", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", CurrentStep: "s"}},
		{name: "no step", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p"}},
		{name: "unknown status", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s", Status: "hmm"}},
		{name: "negative attempts", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s", Attempts: -1}},
		{name: "unknown pause", item: run.Item{ID: "i", RunID: "r", SiteID: "s1", TargetID: "p", CurrentStep: "s", PauseReason: "hmm"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := run.NewItem(tc.item); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewItem = %v, want an invalid error", err)
			}
		})
	}
}

func TestExecStatusIsAClosedSet(t *testing.T) {
	t.Parallel()

	for _, status := range []run.ExecStatus{run.ExecStarted, run.ExecDone, run.ExecFailed} {
		if !status.Valid() {
			t.Errorf("ExecStatus(%q).Valid() = false", status)
		}
	}
	if run.ExecStatus("pending").Valid() {
		t.Error("an unknown exec status must not validate")
	}
}
