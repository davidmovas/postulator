package app_test

import (
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var _ application.Publisher = (*app.EventRelay)(nil)

type countingEmitter struct {
	mu    sync.Mutex
	names []string
}

func (c *countingEmitter) Emit(name string, _ ...any) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.names = append(c.names, name)
	return true
}

func (c *countingEmitter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.names)
}

func TestEventRelayDropsBeforeConnectAndForwardsAfter(t *testing.T) {
	t.Parallel()

	relay := &app.EventRelay{}
	if err := relay.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: "s1"}); err != nil {
		t.Fatalf("Publish before Connect must be a no-op, got %v", err)
	}

	emitter := &countingEmitter{}
	now := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	if err := relay.Connect(emitter, now); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := relay.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: "s1"}); err != nil {
		t.Fatalf("Publish after Connect: %v", err)
	}
	if emitter.count() != 1 {
		t.Errorf("emitted %d events, want 1", emitter.count())
	}
	if err := relay.Publish(events.Type("nope"), events.GraphChangedPayload{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown type code = %q, want INVALID", errors.CodeOf(err))
	}
	if err := relay.Connect(emitter, now); !errors.IsCode(err, errors.Internal) {
		t.Errorf("second Connect code = %q, want INTERNAL", errors.CodeOf(err))
	}
}
