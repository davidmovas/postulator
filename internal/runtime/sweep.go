package runtime

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/domain/run"
)

func (e *Engine) sweep(ctx context.Context) error {
	if err := e.rearm(ctx); err != nil {
		return err
	}
	if err := e.reap(ctx); err != nil {
		return err
	}
	return e.purge(ctx)
}

func (e *Engine) rearm(ctx context.Context) error {
	now := e.now()

	due, err := e.deps.Items.Due(ctx, now, sweepBatch)
	if err != nil {
		return err
	}
	stalled, err := e.deps.Items.Stalled(ctx, now, sweepBatch)
	if err != nil {
		return err
	}

	for i := range due {
		e.requeue(ctx, due[i], run.StatusWaiting)
	}
	for i := range stalled {
		e.requeue(ctx, stalled[i], run.StatusRunning)
	}
	return nil
}

func (e *Engine) requeue(ctx context.Context, item run.Item, from run.Status) {
	requeued, err := e.deps.Items.Requeue(ctx, item.ID, item.AdvanceSeq, from, e.now())
	if err != nil {
		e.logger.Warn("re-arming a run item failed; the next sweep will try again",
			zap.String("itemId", item.ID),
			zap.String("from", string(from)),
			zap.Error(err),
		)
		return
	}
	if requeued {
		e.logger.Info("the sweep re-armed a run item",
			zap.String("itemId", item.ID),
			zap.String("from", string(from)),
		)
	}
}

func (e *Engine) reap(ctx context.Context) error {
	stale, err := e.deps.Runs.PastDeadline(ctx, e.now(), sweepBatch)
	if err != nil {
		return err
	}

	for i := range stale {
		record := stale[i]
		reapErr := e.transact(ctx, func(c context.Context, box *outbox) error {
			return e.expire(c, box, record, e.now())
		})
		if reapErr != nil {
			e.logger.Warn("reaping a run past its deadline failed",
				zap.String("runId", record.ID),
				zap.Error(reapErr),
			)
		}
	}
	return nil
}

func (e *Engine) purge(ctx context.Context) error {
	cutoff := e.now().Add(-e.retention())

	purged, err := e.deps.Artifacts.PurgePublishedBefore(ctx, cutoff)
	if err != nil {
		return err
	}
	if purged > 0 {
		e.logger.Info("purged the bodies of published artifacts past their retention window",
			zap.Int64("count", purged),
		)
	}
	return nil
}
