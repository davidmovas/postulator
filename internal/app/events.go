package app

import (
	"strconv"
	"sync/atomic"

	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type EventRelay struct {
	bridge atomic.Pointer[wails.EventBridge]
	ended  atomic.Pointer[func(agent.RunOutcome)]
}

func (r *EventRelay) OnRunEnded(listen func(agent.RunOutcome)) {
	if listen == nil {
		r.ended.Store(nil)
		return
	}
	r.ended.Store(&listen)
}

func (r *EventRelay) runEnded(runID string, eventType events.Type, payload any) {
	listen := r.ended.Load()
	if listen == nil {
		return
	}
	outcome := agent.RunOutcome{RunID: runID}
	switch held := payload.(type) {
	case events.RunCompletedPayload:
		if held.Failed > 0 {
			outcome.Failure = strconv.Itoa(held.Failed) + " of its pages failed"
		}
	case events.RunFailedPayload:
		outcome.Failure = held.Message
	case events.RunCancelledPayload:
		outcome.Failure = "the run was cancelled"
	default:
		return
	}
	if eventType == events.RunCompleted || eventType == events.RunFailed || eventType == events.RunCancelled {
		(*listen)(outcome)
	}
}

func (r *EventRelay) Connect(emitter wails.Emitter, now clock.Clock) error {
	if !r.bridge.CompareAndSwap(nil, wails.NewEventBridge(emitter, now)) {
		return errors.New(errors.Internal, "the event bridge is already connected")
	}
	return nil
}

func (r *EventRelay) Publish(eventType events.Type, payload any) error {
	bridge := r.bridge.Load()
	if bridge == nil {
		return nil
	}
	return bridge.Publish(eventType, payload)
}

func (r *EventRelay) PublishRun(runID string, seq int64, eventType events.Type, payload any) error {
	r.runEnded(runID, eventType, payload)
	bridge := r.bridge.Load()
	if bridge == nil {
		return nil
	}
	return bridge.PublishRun(runID, seq, eventType, payload)
}
