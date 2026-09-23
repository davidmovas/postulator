package applicationtest

import (
	"slices"
	"sync"

	"github.com/davidmovas/postulator/internal/application/events"
)

type Event struct {
	Type    events.Type
	Payload any
}

type Recorder struct {
	mu       sync.Mutex
	recorded []Event
}

func (r *Recorder) Publish(eventType events.Type, payload any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recorded = append(r.recorded, Event{Type: eventType, Payload: payload})
	return nil
}

func (r *Recorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.recorded)
}

func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recorded = nil
}
