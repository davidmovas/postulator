package events

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsBridge forwards EventBus events to the Wails frontend
type WailsBridge struct {
	ctx      context.Context
	eventBus *EventBus
	mu       sync.RWMutex
}

func NewWailsBridge(eventBus *EventBus) *WailsBridge {
	return &WailsBridge{
		eventBus: eventBus,
	}
}

// SetContext sets the Wails runtime context
func (b *WailsBridge) SetContext(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ctx = ctx
}

func (b *WailsBridge) getContext() context.Context {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ctx
}

// SubscribeAndForward subscribes to an event type and forwards it to the Wails frontend
func (b *WailsBridge) SubscribeAndForward(eventType EventType) {
	b.eventBus.Subscribe(eventType, func(_ context.Context, event Event) {
		ctx := b.getContext()
		if ctx == nil {
			return
		}
		// Forward to Wails frontend using the event type as the event name
		runtime.EventsEmit(ctx, string(event.Type), event.Data)
	})
}

// SubscribeToPageGeneration sets up forwarding for all page generation events
func (b *WailsBridge) SubscribeToPageGeneration() {
	pageGenEvents := []EventType{
		"pagegeneration.task.started",
		"pagegeneration.task.progress",
		"pagegeneration.task.paused",
		"pagegeneration.task.resumed",
		"pagegeneration.task.completed",
		"pagegeneration.task.failed",
		"pagegeneration.task.cancelled",
		"pagegeneration.node.queued",
		"pagegeneration.node.generating",
		"pagegeneration.node.generated",
		"pagegeneration.node.publishing",
		"pagegeneration.node.completed",
		"pagegeneration.node.failed",
		"pagegeneration.node.skipped",
	}

	for _, eventType := range pageGenEvents {
		b.SubscribeAndForward(eventType)
	}
}

// SubscribeToLinking sets up forwarding for all linking events
func (b *WailsBridge) SubscribeToLinking() {
	linkingEvents := []EventType{
		// Apply events
		"linking.apply.started",
		"linking.apply.progress",
		"linking.apply.completed",
		"linking.apply.failed",
		"linking.page.processing",
		"linking.page.completed",
		"linking.page.failed",
		// Suggest events
		"linking.suggest.started",
		"linking.suggest.progress",
		"linking.suggest.completed",
		"linking.suggest.failed",
	}

	for _, eventType := range linkingEvents {
		b.SubscribeAndForward(eventType)
	}
}
