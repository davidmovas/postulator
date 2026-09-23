package runtime

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type pending struct {
	payload   any
	runID     string
	eventType events.Type
	seq       int64
}

type outbox struct {
	engine *Engine
	err    error
	events []pending
}

func (o *outbox) add(ctx context.Context, runID string, eventType events.Type, payload any) {
	if o.err != nil {
		return
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		o.err = errors.Wrap(err, errors.Internal, "encode the run event payload")
		return
	}

	event, err := o.engine.deps.Events.Append(ctx, runID, string(eventType), o.engine.now(), encoded)
	if err != nil {
		o.err = err
		return
	}
	o.events = append(o.events, pending{runID: runID, seq: event.Seq, eventType: eventType, payload: payload})
}

func (o *outbox) flush() {
	for _, event := range o.events {
		if err := o.engine.deps.Publisher.PublishRun(event.runID, event.seq, event.eventType, event.payload); err != nil {
			o.engine.logger.Warn("a run event was recorded but not delivered live",
				zap.String("runId", event.runID),
				zap.String("type", string(event.eventType)),
				zap.Error(err),
			)
		}
	}
}

func (e *Engine) transact(ctx context.Context, fn func(context.Context, *outbox) error) error {
	box := &outbox{engine: e}

	err := e.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		if fnErr := fn(c, box); fnErr != nil {
			return fnErr
		}
		return box.err
	})
	if err != nil {
		return err
	}

	box.flush()
	return nil
}
