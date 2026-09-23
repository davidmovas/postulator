package wails

import (
	"reflect"
	"sync/atomic"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Emitter interface {
	Emit(name string, data ...any) bool
}

type EventBridge struct {
	emitter  Emitter
	clock    clock.Clock
	registry events.Registry
	seq      atomic.Int64
}

func NewEventBridge(emitter Emitter, now clock.Clock) *EventBridge {
	return &EventBridge{emitter: emitter, clock: now, registry: events.NewRegistry()}
}

func (b *EventBridge) Publish(eventType events.Type, payload any) error {
	entry, err := b.entry(eventType, payload)
	if err != nil {
		return err
	}
	if entry.Run {
		return errors.New(errors.Invalid, "a run event must be published with its run sequence").
			WithDetail("type", string(eventType))
	}

	b.emit(events.Envelope{
		Type:    eventType,
		Seq:     b.seq.Add(1),
		At:      dto.NewTime(b.clock.Now()),
		Payload: payload,
	})
	return nil
}

func (b *EventBridge) PublishRun(runID string, seq int64, eventType events.Type, payload any) error {
	entry, err := b.entry(eventType, payload)
	if err != nil {
		return err
	}
	if !entry.Run {
		return errors.New(errors.Invalid, "only a run event may be published with a run sequence").
			WithDetail("type", string(eventType))
	}
	if runID == "" {
		return errors.New(errors.Invalid, "a run event needs a run id").
			WithDetail("type", string(eventType))
	}

	b.emit(events.Envelope{
		Type:    eventType,
		Seq:     seq,
		RunID:   &runID,
		At:      dto.NewTime(b.clock.Now()),
		Payload: payload,
	})
	return nil
}

func (b *EventBridge) entry(eventType events.Type, payload any) (events.Entry, error) {
	entry, ok := b.registry.Lookup(eventType)
	if !ok {
		return events.Entry{}, errors.New(errors.Invalid, "unknown event type").
			WithDetail("type", string(eventType))
	}
	if reflect.TypeOf(payload) != reflect.TypeOf(entry.Payload) {
		return events.Entry{}, errors.New(errors.Invalid, "payload does not match the registered type").
			WithDetail("type", string(eventType))
	}
	return entry, nil
}

func (b *EventBridge) emit(envelope events.Envelope) {
	b.emitter.Emit(string(envelope.Type), envelope)
}
