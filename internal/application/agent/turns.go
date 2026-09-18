package agent

import (
	"context"
	"slices"
	"sync"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Turns struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	running map[string]context.CancelFunc
	dropped []error
	closed  bool
}

func NewTurns() *Turns {
	return &Turns{running: make(map[string]context.CancelFunc)}
}

func (t *Turns) Start(parent context.Context, conversationID string, turn func(context.Context)) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))

	t.mu.Lock()
	switch {
	case t.closed:
		t.mu.Unlock()
		cancel()
		return errors.New(errors.Conflict, "the agent is shutting down")
	case t.running[conversationID] != nil:
		t.mu.Unlock()
		cancel()
		return errors.New(errors.Conflict, "this conversation is already answering").
			WithDetail("conversationId", conversationID)
	}
	t.running[conversationID] = cancel
	t.wg.Add(1)
	t.mu.Unlock()

	go func() {
		defer t.wg.Done()
		defer t.finish(conversationID, cancel)
		turn(ctx)
	}()
	return nil
}

func (t *Turns) finish(conversationID string, cancel context.CancelFunc) {
	cancel()

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.running, conversationID)
}

func (t *Turns) Cancel(conversationID string) bool {
	t.mu.Lock()
	cancel := t.running[conversationID]
	t.mu.Unlock()

	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (t *Turns) Running(conversationID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running[conversationID] != nil
}

func (t *Turns) Close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		t.wg.Wait()
		return
	}
	t.closed = true
	cancels := make([]context.CancelFunc, 0, len(t.running))
	for _, cancel := range t.running {
		cancels = append(cancels, cancel)
	}
	t.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	t.wg.Wait()
}

func (t *Turns) Note(err error) {
	if err == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.dropped = append(t.dropped, err)
}

func (t *Turns) Dropped() []error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.dropped)
}
