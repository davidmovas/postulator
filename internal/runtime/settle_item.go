package runtime

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/run"
)

func (e *Engine) settle(parent context.Context, held *claim, out outcome) (bool, error) {
	var again bool

	err := e.transact(context.WithoutCancel(parent), func(ctx context.Context, box *outbox) error {
		now := e.now()
		if !out.reuse {
			if err := e.record(ctx, held, out, now); err != nil {
				return err
			}
		}

		next := settledItem(held.item, out, now)
		persisted, err := e.deps.Items.Persist(ctx, next, held.expectSeq)
		if err != nil || !persisted {
			return err
		}

		e.announce(ctx, box, held, out, next)
		again = out.again
		return e.settleRun(ctx, box, held.record, now)
	})
	if err != nil {
		return false, err
	}
	return again, nil
}

func settledItem(item run.Item, out outcome, now time.Time) run.Item {
	next := item
	next.Checkpoint = item.Checkpoint.MergedWith(out.checkpoint)
	next.Status = out.status
	next.CurrentStep = out.nextStep
	next.Attempts = out.attempts
	next.PauseReason = out.reason
	next.Error = ""
	next.Note = ""
	next.LeaseUntil = nil
	next.WakeAt = out.wakeAt
	next.UpdatedAt = now
	if out.fault != nil {
		next.Error = out.fault.Message
	}
	if out.fault == nil && (out.status == run.StatusPaused || out.status == run.StatusWaiting) {
		next.Note = out.message
	}
	if out.status == run.StatusCompleted {
		next.Note = out.notice
	}
	if next.Status.Terminal() {
		next.FinishedAt = &now
	}
	return next
}

func (e *Engine) announce(ctx context.Context, box *outbox, held *claim, out outcome, next run.Item) {
	runID, itemID, step := held.record.ID, held.item.ID, held.step.Name

	switch {
	case out.status == run.StatusWaiting && out.attempts > 0:
		box.add(ctx, runID, events.StepRetrying, events.StepRetryingPayload{
			RunID: runID, ItemID: itemID, Step: step, Attempt: out.attempts,
			AfterMs: e.delayMs(out.wakeAt), Code: faultCode(out.fault), Message: out.message,
		})
	case out.fault != nil:
		box.add(ctx, runID, events.StepFailed, events.StepFailedPayload{
			RunID: runID, ItemID: itemID, Step: step, Code: out.fault.Code, Message: out.fault.Message,
		})
	default:
		box.add(ctx, runID, events.StepDone, events.StepDonePayload{
			RunID: runID, ItemID: itemID, Step: step, DurationMs: out.duration.Milliseconds(), Message: out.message,
		})
	}

	switch next.Status {
	case run.StatusCompleted:
		box.add(ctx, runID, events.ItemDone, events.ItemDonePayload{RunID: runID, ItemID: itemID, Note: next.Note})
	case run.StatusFailed:
		box.add(ctx, runID, events.ItemFailed, events.ItemFailedPayload{
			RunID: runID, ItemID: itemID, Code: faultCode(out.fault), Message: out.message,
		})
	case run.StatusPaused:
		message := next.Note
		if message == "" {
			message = out.message
		}
		box.add(ctx, runID, events.ItemNeedsHuman, events.ItemNeedsHumanPayload{
			RunID: runID, ItemID: itemID, Reason: string(next.PauseReason), Message: message,
		})
	}
}

func (e *Engine) delayMs(wakeAt *time.Time) int64 {
	if wakeAt == nil {
		return 0
	}
	return max(wakeAt.Sub(e.now()).Milliseconds(), 0)
}

func faultCode(fault *run.Fault) string {
	if fault == nil {
		return run.CodeUnclassified
	}
	return fault.Code
}

func (e *Engine) abandon(ctx context.Context, box *outbox, record run.Run, item run.Item, fault run.Fault, now time.Time) error {
	next := item
	next.Status = run.StatusFailed
	next.Error = fault.Message
	next.LeaseUntil = nil
	next.WakeAt = nil
	next.UpdatedAt = now
	next.FinishedAt = &now

	persisted, err := e.deps.Items.Persist(ctx, next, item.AdvanceSeq)
	if err != nil || !persisted {
		return err
	}

	box.add(ctx, record.ID, events.ItemFailed, events.ItemFailedPayload{
		RunID: record.ID, ItemID: item.ID, Code: fault.Code, Message: fault.Message,
	})
	return e.settleRun(ctx, box, record, now)
}
