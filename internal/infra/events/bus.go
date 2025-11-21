package events

import (
	"context"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type EventType string

const (
	HealthCheckSettingsUpdated EventType = "health_check_settings_updated"
)

type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      any
}

func NewEvent(eventType EventType, data any) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

type HealthCheckSettingsUpdatedEvent struct {
	Settings *entities.HealthCheckSettings
}

type EventHandler func(ctx context.Context, event Event)

type EventBus struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

var (
	globalBus     *EventBus
	globalBusOnce sync.Once
)

func GetGlobalEventBus() *EventBus {
	globalBusOnce.Do(func() {
		globalBus = NewEventBus()
	})
	return globalBus
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(ctx context.Context, event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			h(ctx, event)
		}(handler)
	}
}

func (eb *EventBus) PublishSync(ctx context.Context, event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			h(ctx, event)
		}(handler)
	}
}
