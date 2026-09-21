package wails_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

var _ wails.Emitter = (*application.EventManager)(nil)

type recordedEvent struct {
	name string
	data any
}

type fakeEmitter struct {
	mu       sync.Mutex
	recorded []recordedEvent
}

func (f *fakeEmitter) Emit(name string, data ...any) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	event := recordedEvent{name: name}
	if len(data) == 1 {
		event.data = data[0]
	}
	f.recorded = append(f.recorded, event)
	return false
}

func (f *fakeEmitter) events() []recordedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedEvent(nil), f.recorded...)
}

func newBridge() (*wails.EventBridge, *fakeEmitter) {
	emitter := &fakeEmitter{}
	return wails.NewEventBridge(emitter, clock.NewFake(time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC))), emitter
}

func TestPublishWrapsThePayloadInAnEnvelope(t *testing.T) {
	t.Parallel()

	bridge, emitter := newBridge()
	if err := bridge.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: "s1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	recorded := emitter.events()
	if len(recorded) != 1 {
		t.Fatalf("emitter saw %d events, want 1", len(recorded))
	}
	if recorded[0].name != "graph.changed" {
		t.Fatalf("event name = %q, want %q", recorded[0].name, "graph.changed")
	}

	encoded, err := json.Marshal(recorded[0].data)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `{"type":"graph.changed","seq":1,"at":"2026-09-17T10:30:00Z","payload":{"siteId":"s1"}}`
	if string(encoded) != want {
		t.Fatalf("envelope = %s, want %s", encoded, want)
	}
}

func TestPublishNumbersApplicationEventsMonotonically(t *testing.T) {
	t.Parallel()

	bridge, emitter := newBridge()
	for range 3 {
		if err := bridge.Publish(events.TemplatesChanged, events.TemplatesChangedPayload{}); err != nil {
			t.Fatalf("Publish: %v", err)
		}
	}

	for index, recorded := range emitter.events() {
		envelope, ok := recorded.data.(events.Envelope)
		if !ok {
			t.Fatalf("event %d carries %T, want events.Envelope", index, recorded.data)
		}
		if envelope.Seq != int64(index+1) {
			t.Fatalf("event %d has seq %d, want %d", index, envelope.Seq, index+1)
		}
	}
}

func TestPublishRunCarriesTheRunSequence(t *testing.T) {
	t.Parallel()

	bridge, emitter := newBridge()
	const runID = "6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90"

	if err := bridge.PublishRun(runID, 42, events.StepDone, events.StepDonePayload{RunID: runID, ItemID: "i1", Step: "validate", DurationMs: 120}); err != nil {
		t.Fatalf("PublishRun: %v", err)
	}

	encoded, err := json.Marshal(emitter.events()[0].data)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `{"type":"step.done","seq":42,"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90","at":"2026-09-17T10:30:00Z","payload":{"runId":"6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90","itemId":"i1","step":"validate","durationMs":120}}`
	if string(encoded) != want {
		t.Fatalf("envelope = %s, want %s", encoded, want)
	}
}

func TestPublishCarriesAModelCallWhateverItBelongsTo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload events.LLMUsagePayload
	}{
		{
			name: "inside a run",
			payload: events.LLMUsagePayload{
				RunID: "6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90", ItemID: "i1",
				Provider: "openai", Model: "gpt-5.6-luna", PromptTokens: 7, CompletionTokens: 50, USD: 0.001,
			},
		},
		{
			name: "outside a run",
			payload: events.LLMUsagePayload{
				Provider: "openai", Model: "gpt-5.6-sol", PromptTokens: 3, CompletionTokens: 9, USD: 0.0002,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bridge, emitter := newBridge()
			if err := bridge.Publish(events.LLMUsage, tc.payload); err != nil {
				t.Fatalf("Publish: %v", err)
			}
			if len(emitter.events()) != 1 {
				t.Fatalf("emitted %d events, want 1", len(emitter.events()))
			}
		})
	}
}

func TestPublishRejectsMisuse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func(*wails.EventBridge) error
	}{
		{
			name: "unknown type",
			call: func(b *wails.EventBridge) error {
				return b.Publish(events.Type("nope"), events.GraphChangedPayload{})
			},
		},
		{
			name: "payload of the wrong type",
			call: func(b *wails.EventBridge) error {
				return b.Publish(events.GraphChanged, events.PagesChangedPayload{})
			},
		},
		{
			name: "run event without a sequence",
			call: func(b *wails.EventBridge) error {
				return b.Publish(events.RunStarted, events.RunStartedPayload{})
			},
		},
		{
			name: "application event with a sequence",
			call: func(b *wails.EventBridge) error {
				return b.PublishRun("r1", 1, events.GraphChanged, events.GraphChangedPayload{})
			},
		},
		{
			name: "run event without a run id",
			call: func(b *wails.EventBridge) error {
				return b.PublishRun("", 1, events.RunStarted, events.RunStartedPayload{})
			},
		},
		{
			name: "unknown type with a sequence",
			call: func(b *wails.EventBridge) error {
				return b.PublishRun("r1", 1, events.Type("nope"), events.RunStartedPayload{})
			},
		},
		{
			name: "run payload of the wrong type",
			call: func(b *wails.EventBridge) error {
				return b.PublishRun("r1", 1, events.RunStarted, events.RunCancelledPayload{})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bridge, emitter := newBridge()
			err := tc.call(bridge)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("error = %v, want an INVALID kernel error", err)
			}
			if len(emitter.events()) != 0 {
				t.Fatalf("emitter saw %d events, want none", len(emitter.events()))
			}
		})
	}
}
