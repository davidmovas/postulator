package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var (
	pausable  = []run.Status{run.StatusPending, run.StatusWaiting}
	stoppable = []run.Status{run.StatusPending, run.StatusRunning, run.StatusWaiting, run.StatusPaused}
)

func (e *Engine) Enqueue(ctx context.Context, record run.Run) (run.Run, error) {
	if err := run.ValidateRecipe(e.registry, record.Recipe); err != nil {
		return run.Run{}, err
	}

	enabled := run.Enabled(record.Recipe)
	first := enabled[0].Name
	now := e.now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.DeadlineAt.IsZero() {
		record.DeadlineAt = record.CreatedAt.Add(e.cfg.RunDeadline)
	}
	record.Stats.Items = len(record.Targets)

	validated, err := run.NewRun(record)
	if err != nil {
		return run.Run{}, err
	}

	err = e.transact(ctx, func(c context.Context, box *outbox) error {
		if insertErr := e.deps.Runs.Insert(c, validated); insertErr != nil {
			return insertErr
		}
		for _, pageID := range validated.Targets {
			item, itemErr := run.NewItem(run.Item{
				ID: id.New(), RunID: validated.ID, PageID: pageID, CurrentStep: first,
				CreatedAt: now, UpdatedAt: now,
			})
			if itemErr != nil {
				return itemErr
			}
			if insertErr := e.deps.Items.Insert(c, item); insertErr != nil {
				return insertErr
			}
		}
		box.add(c, validated.ID, events.RunQueued, events.RunQueuedPayload{
			RunID: validated.ID, Kind: string(validated.Kind), Items: len(validated.Targets),
		})
		return nil
	})
	if err != nil {
		return run.Run{}, err
	}

	e.Nudge()
	return validated, nil
}

func (e *Engine) Pause(ctx context.Context, runID string, reason run.PauseReason) error {
	if reason == "" {
		reason = run.PauseUser
	}
	if !reason.Valid() {
		return errors.New(errors.Invalid, "pause reason is not recognized").WithDetail("reason", string(reason))
	}

	return e.transact(ctx, func(c context.Context, box *outbox) error {
		record, err := e.deps.Runs.Get(c, runID)
		if err != nil {
			return err
		}
		if record.Status.Terminal() {
			return errors.New(errors.Conflict, "a finished run cannot be paused").WithDetail("runId", runID)
		}
		if record.Status == run.StatusPaused {
			return nil
		}

		now := e.now()
		if _, err = e.deps.Items.StopAll(c, runID, pausable, run.StatusPaused, reason, now); err != nil {
			return err
		}

		record.Status = run.StatusPaused
		record.PauseReason = reason
		if updateErr := e.deps.Runs.Update(c, record); updateErr != nil {
			return updateErr
		}

		box.add(c, runID, events.RunPaused, events.RunPausedPayload{RunID: runID, Reason: string(reason)})
		return nil
	})
}

func (e *Engine) Resume(ctx context.Context, runID string) error {
	err := e.transact(ctx, func(c context.Context, box *outbox) error {
		record, err := e.deps.Runs.Get(c, runID)
		if err != nil {
			return err
		}
		if record.Status != run.StatusPaused {
			return errors.New(errors.Conflict, "only a paused run can be resumed").WithDetail("runId", runID)
		}

		now := e.now()
		if _, err = e.deps.Items.ResumeAll(c, runID, now); err != nil {
			return err
		}

		record.Status = run.StatusRunning
		record.PauseReason = ""
		if updateErr := e.deps.Runs.Update(c, record); updateErr != nil {
			return updateErr
		}

		box.add(c, runID, events.RunResumed, events.RunResumedPayload{RunID: runID})
		return nil
	})
	if err != nil {
		return err
	}

	e.Nudge()
	return nil
}

func (e *Engine) Cancel(ctx context.Context, runID string) error {
	e.cancelRun(runID)

	return e.transact(ctx, func(c context.Context, box *outbox) error {
		record, err := e.deps.Runs.Get(c, runID)
		if err != nil {
			return err
		}
		if record.Status.Terminal() {
			return errors.New(errors.Conflict, "a finished run cannot be cancelled").WithDetail("runId", runID)
		}

		now := e.now()
		if _, err = e.deps.Items.StopAll(c, runID, stoppable, run.StatusCancelled, "", now); err != nil {
			return err
		}

		record.Status = run.StatusCancelled
		record.PauseReason = ""
		record.FinishedAt = &now
		if updateErr := e.deps.Runs.Update(c, record); updateErr != nil {
			return updateErr
		}

		box.add(c, runID, events.RunCancelled, events.RunCancelledPayload{RunID: runID})
		return nil
	})
}

func (e *Engine) RetryStep(ctx context.Context, itemID string) error {
	err := e.transact(ctx, func(c context.Context, box *outbox) error {
		item, err := e.deps.Items.Get(c, itemID)
		if err != nil {
			return err
		}
		if item.Status == run.StatusRunning || item.Status == run.StatusPending {
			return errors.New(errors.Conflict, "the item has not stopped, so there is nothing to retry").
				WithDetail("itemId", itemID)
		}

		record, err := e.deps.Runs.Get(c, item.RunID)
		if err != nil {
			return err
		}

		now := e.now()
		next := item
		next.Status = run.StatusPending
		next.Attempts = 0
		next.Error = ""
		next.PauseReason = ""
		next.LeaseUntil = nil
		next.WakeAt = nil
		next.FinishedAt = nil
		next.UpdatedAt = now

		requeued, err := e.deps.Items.Requeue(c, itemID, item.AdvanceSeq, item.Status, now)
		if err != nil {
			return err
		}
		if !requeued {
			return errors.New(errors.Conflict, "the item moved on before it could be retried").
				WithDetail("itemId", itemID)
		}
		if _, err = e.deps.Items.Persist(c, next, item.AdvanceSeq+1); err != nil {
			return err
		}

		if record.Status.Terminal() || record.Status == run.StatusPaused {
			record.Status = run.StatusRunning
			record.PauseReason = ""
			record.Error = ""
			record.FinishedAt = nil
			if updateErr := e.deps.Runs.Update(c, record); updateErr != nil {
				return updateErr
			}
			box.add(c, record.ID, events.RunResumed, events.RunResumedPayload{RunID: record.ID})
		}
		return nil
	})
	if err != nil {
		return err
	}

	e.Nudge()
	return nil
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

	e.Nudge()
	return nil
}

func (e *Engine) Recover(ctx context.Context) error {
	return e.Sweep(ctx)
}

func (e *Engine) retention() time.Duration {
	return time.Duration(e.cfg.RetentionDays) * 24 * time.Hour
}
