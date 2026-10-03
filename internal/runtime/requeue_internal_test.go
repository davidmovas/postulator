package runtime

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestAnItemPutBackInTheQueueStartsItsStepClean(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, time.October, 3, 9, 0, 0, 0, time.UTC)
	now := created.Add(time.Hour)
	later := now.Add(time.Minute)

	stopped := func(mutate func(*run.Item)) run.Item {
		item := run.Item{
			ID: "item-1", RunID: "run-1", SiteID: "site-1", TargetID: "page-1", CurrentStep: "validate",
			Seq: 2, AdvanceSeq: 7, BlockedBy: "item-0", Checkpoint: run.Checkpoint{"cursor": json.RawMessage(`3`)},
			CreatedAt: created, UpdatedAt: created,
		}
		mutate(&item)
		return item
	}
	clean := stopped(func(item *run.Item) {
		item.Status = run.StatusPending
		item.UpdatedAt = now
	})

	cases := []struct {
		name string
		item run.Item
	}{
		{
			name: "a failed item drops its error, its attempts and its finish",
			item: stopped(func(item *run.Item) {
				item.Status = run.StatusFailed
				item.Attempts = 3
				item.Error = "step validate failed 3 times and has no attempts left"
				item.FinishedAt = &created
			}),
		},
		{
			name: "a paused item drops its reason and its note",
			item: stopped(func(item *run.Item) {
				item.Status = run.StatusPaused
				item.PauseReason = run.PauseNeedsHuman
				item.Note = "2 findings need a decision"
			}),
		},
		{
			name: "a waiting item drops its wake and its lease",
			item: stopped(func(item *run.Item) {
				item.Status = run.StatusWaiting
				item.Attempts = 1
				item.WakeAt = &later
				item.LeaseUntil = &later
			}),
		},
		{
			name: "a cancelled item is pending again",
			item: stopped(func(item *run.Item) {
				item.Status = run.StatusCancelled
				item.FinishedAt = &created
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := rearmed(tc.item, now); !reflect.DeepEqual(got, clean) {
				t.Fatalf("rearmed =\n%+v\nwant\n%+v", got, clean)
			}
		})
	}
}
