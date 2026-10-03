package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (e *Engine) RetryStep(ctx context.Context, itemID string) error {
	return e.requeueStep(ctx, itemID, nil)
}

func (e *Engine) Accept(ctx context.Context, itemID string) error {
	return e.requeueStep(ctx, itemID, func(item *run.Item) error {
		item.Checkpoint = item.Checkpoint.Clone()
		return run.Set(item.Checkpoint, run.CheckpointAccept, item.CurrentStep)
	})
}

func (e *Engine) requeueStep(ctx context.Context, itemID string, amend func(*run.Item) error) error {
	err := e.transact(ctx, func(c context.Context, box *outbox) error {
		item, err := e.retryable(c, itemID)
		if err != nil {
			return err
		}
		record, err := e.deps.Runs.Get(c, item.RunID)
		if err != nil {
			return err
		}

		now := e.now()
		next := rearmed(item, now)
		if amend != nil {
			if amendErr := amend(&next); amendErr != nil {
				return amendErr
			}
		}
		if putErr := e.putBack(c, item, next, now, "retried"); putErr != nil {
			return putErr
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

func (e *Engine) retryable(ctx context.Context, itemID string) (run.Item, error) {
	item, err := e.deps.Items.Get(ctx, itemID)
	if err != nil {
		return run.Item{}, err
	}
	if item.Status == run.StatusRunning || item.Status == run.StatusPending {
		return run.Item{}, errors.New(errors.Conflict, "the item has not stopped, so there is nothing to retry").
			WithDetail("itemId", itemID)
	}
	if refusal := e.refuseExpiredInputs(ctx, item); refusal != nil {
		return run.Item{}, refusal
	}
	return item, nil
}

func (e *Engine) refuseExpiredInputs(ctx context.Context, item run.Item) error {
	def, known := e.registry.Lookup(item.CurrentStep)
	if !known || len(def.Requires) == 0 {
		return nil
	}

	artifacts, err := e.deps.Artifacts.ByItem(ctx, item.ID)
	if err != nil {
		return err
	}

	expired := run.ExpiredInputs(def.Requires, run.PurgedKinds(artifacts))
	if len(expired) == 0 {
		return nil
	}

	kinds := make([]string, 0, len(expired))
	for _, kind := range expired {
		kinds = append(kinds, string(kind))
	}
	return errors.New(errors.NotFound, "step "+item.CurrentStep+
		" reads artifacts the retention sweep purged after the page was published; re-run the page instead of retrying the step").
		WithDetail("itemId", item.ID).
		WithDetail("step", item.CurrentStep).
		WithDetail("kinds", kinds).
		WithDetail("reason", string(run.RetryBlockedInputsExpired))
}

func rearmed(item run.Item, now time.Time) run.Item {
	next := item
	next.Status = run.StatusPending
	next.Attempts = 0
	next.Error = ""
	next.PauseReason = ""
	next.Note = ""
	next.LeaseUntil = nil
	next.WakeAt = nil
	next.FinishedAt = nil
	next.UpdatedAt = now
	return next
}

func (e *Engine) putBack(ctx context.Context, item, next run.Item, now time.Time, done string) error {
	requeued, err := e.deps.Items.Requeue(ctx, item.ID, item.AdvanceSeq, item.Status, now)
	if err != nil {
		return err
	}
	if !requeued {
		return errors.New(errors.Conflict, "the item moved on before it could be "+done).WithDetail("itemId", item.ID)
	}
	_, err = e.deps.Items.Persist(ctx, next, item.AdvanceSeq+1)
	return err
}

func (e *Engine) Wake(ctx context.Context, itemID string) error {
	item, err := e.deps.Items.Get(ctx, itemID)
	if err != nil {
		return err
	}
	if item.Status != run.StatusWaiting {
		return errors.New(errors.Conflict, "only a waiting item can be woken").WithDetail("itemId", itemID)
	}

	woken, err := e.deps.Items.Requeue(ctx, itemID, item.AdvanceSeq, run.StatusWaiting, e.now())
	if err != nil {
		return err
	}
	if !woken {
		return errors.New(errors.Conflict, "the item moved on before it could be woken").WithDetail("itemId", itemID)
	}

	e.nudge()
	return nil
}
