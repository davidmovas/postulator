package app

import (
	"sync/atomic"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type EventRelay struct {
	bridge atomic.Pointer[wails.EventBridge]
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
	bridge := r.bridge.Load()
	if bridge == nil {
		return nil
	}
	return bridge.PublishRun(runID, seq, eventType, payload)
}
