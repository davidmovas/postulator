package agent

import (
	"context"
	stderrors "errors"
	"slices"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	DefaultTurnTimeout = 10 * time.Minute

	deadlineMessage = "the agent turn ran out of time"
)

var errTurnDeadline = stderrors.New(deadlineMessage)

type TurnStatus struct {
	StartedAt time.Time
	MessageID string
	LastSeq   int64
}

type turn struct {
	cancel    context.CancelFunc
	startedAt time.Time
	messageID string
	lastSeq   int64
}

type Turns struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	running map[string]*turn
	dropped []error
	timeout func() time.Duration
	closed  bool
}

func NewTurns(timeout func() time.Duration) *Turns {
	return &Turns{running: make(map[string]*turn), timeout: timeout}
}

func (t *Turns) deadline() time.Duration {
	if t.timeout == nil {
		return DefaultTurnTimeout
	}
	if chosen := t.timeout(); chosen > 0 {
		return chosen
	}
	return DefaultTurnTimeout
}

func (t *Turns) Start(parent context.Context, conversationID, messageID string, startedAt time.Time,
	body func(context.Context)) error {
	ctx, cancel := context.WithDeadlineCause(context.WithoutCancel(parent),
		time.Now().Add(t.deadline()), errTurnDeadline)

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
	t.running[conversationID] = &turn{cancel: cancel, messageID: messageID, startedAt: startedAt}
	t.wg.Add(1)
	t.mu.Unlock()

	go func() {
		defer t.wg.Done()
		defer t.finish(conversationID, cancel)
		body(ctx)
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
	held := t.running[conversationID]
	t.mu.Unlock()

	if held == nil {
		return false
	}
	held.cancel()
	return true
}

func (t *Turns) Running(conversationID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running[conversationID] != nil
}

func (t *Turns) Status(conversationID string) (TurnStatus, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	held := t.running[conversationID]
	if held == nil {
		return TurnStatus{}, false
	}
	return TurnStatus{MessageID: held.messageID, StartedAt: held.startedAt, LastSeq: held.lastSeq}, true
}

func (t *Turns) Observe(conversationID string, seq int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if held := t.running[conversationID]; held != nil && seq > held.lastSeq {
		held.lastSeq = seq
	}
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
	for _, held := range t.running {
		cancels = append(cancels, held.cancel)
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

func expired(ctx context.Context) bool {
	return stderrors.Is(context.Cause(ctx), errTurnDeadline)
}
