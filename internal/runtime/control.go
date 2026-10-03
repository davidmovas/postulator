package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var (
	pausable  = []run.Status{run.StatusPending, run.StatusWaiting}
	stoppable = []run.Status{run.StatusPending, run.StatusRunning, run.StatusWaiting, run.StatusPaused}
)

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

		if _, err = e.deps.Items.StopAll(c, runID, pausable, run.StatusPaused, reason, e.now()); err != nil {
			return err
		}
		return e.pauseRun(c, box, record, reason)
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
		return e.reopen(c, box, record, now)
	})
	if err != nil {
		return err
	}

	e.nudge()
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

func atRest(status run.Status) bool {
	return status.Terminal() || status == run.StatusPaused
}

func (e *Engine) pauseRun(ctx context.Context, box *outbox, record run.Run, reason run.PauseReason) error {
	record.Status = run.StatusPaused
	record.PauseReason = reason
	if err := e.deps.Runs.Update(ctx, record); err != nil {
		return err
	}

	box.add(ctx, record.ID, events.RunPaused, events.RunPausedPayload{RunID: record.ID, Reason: string(reason)})
	return nil
}

func (e *Engine) reopen(ctx context.Context, box *outbox, record run.Run, now time.Time) error {
	record.Status = run.StatusRunning
	record.PauseReason = ""
	record.Error = ""
	record.FinishedAt = nil
	record.DeadlineAt = now.Add(e.cfg.RunDeadline)
	if err := e.deps.Runs.Update(ctx, record); err != nil {
		return err
	}

	box.add(ctx, record.ID, events.RunResumed, events.RunResumedPayload{RunID: record.ID})
	return nil
}
