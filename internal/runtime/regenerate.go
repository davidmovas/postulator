package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const checkpointGeneration = "generation"

func (e *Engine) Regenerate(ctx context.Context, runID string, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return errors.New(errors.Invalid, "a regeneration needs at least one item").WithDetail("runId", runID)
	}

	err := e.transact(ctx, func(c context.Context, box *outbox) error {
		items := make([]run.Item, 0, len(itemIDs))
		for _, itemID := range itemIDs {
			item, err := e.regenerable(c, runID, itemID)
			if err != nil {
				return err
			}
			items = append(items, item)
		}

		record, err := e.deps.Runs.Get(c, runID)
		if err != nil {
			return err
		}
		def, err := run.Plan(e.registry, record.Kind, record.Recipe, e.cfg.RunDeadline)
		if err != nil {
			return err
		}

		now := e.now()
		for i := range items {
			if restartErr := e.restart(c, box, items[i], def.Steps[0].Name, now); restartErr != nil {
				return restartErr
			}
		}
		if record.Status.Terminal() || record.Status == run.StatusPaused {
			return e.reopen(c, box, record, now)
		}
		return nil
	})
	if err != nil {
		return err
	}

	e.nudge()
	return nil
}

func (e *Engine) regenerable(ctx context.Context, runID, itemID string) (run.Item, error) {
	item, err := e.deps.Items.Get(ctx, itemID)
	if err != nil {
		return run.Item{}, err
	}
	if item.RunID != runID {
		return run.Item{}, errors.New(errors.Invalid, "the item belongs to another run").
			WithDetail("itemId", itemID).WithDetail("runId", runID)
	}
	if item.Status.Advanceable() {
		return run.Item{}, errors.New(errors.Conflict, "the item has not stopped, so there is nothing to regenerate").
			WithDetail("itemId", itemID)
	}

	artifacts, err := e.deps.Artifacts.ByItem(ctx, itemID)
	if err != nil {
		return run.Item{}, err
	}
	for i := range artifacts {
		if artifacts[i].Kind == run.ArtifactPublishResult {
			return run.Item{}, errors.New(errors.Conflict,
				"the page was already written to the site by this run; start a new run over it so its revert stays exact").
				WithDetail("itemId", itemID).
				WithDetail("reason", "published")
		}
	}
	return item, nil
}

func (e *Engine) restart(ctx context.Context, box *outbox, item run.Item, first string, now time.Time) error {
	generation, _, err := run.Get[int](item.Checkpoint, checkpointGeneration)
	if err != nil {
		return err
	}
	fresh := run.NewCheckpoint()
	if setErr := run.Set(fresh, checkpointGeneration, generation+1); setErr != nil {
		return setErr
	}

	next := rearmed(item, now)
	next.CurrentStep = first
	next.Checkpoint = fresh
	if putErr := e.putBack(ctx, item, next, now, "regenerated"); putErr != nil {
		return putErr
	}
	if dropErr := e.deps.Artifacts.DeleteByItem(ctx, item.ID); dropErr != nil {
		return dropErr
	}

	box.add(ctx, item.RunID, events.ItemRestarted, events.ItemRestartedPayload{RunID: item.RunID, ItemID: item.ID})
	return nil
}
