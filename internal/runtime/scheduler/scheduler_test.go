package scheduler_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime/scheduler"
)

type ticker struct {
	mu    sync.Mutex
	ticks int
	err   error
}

func (t *ticker) Tick(context.Context) (schedules.TickResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ticks++
	if t.err != nil {
		return schedules.TickResponse{}, t.err
	}
	return schedules.TickResponse{Started: []string{"run-1"}, Skipped: 1}, nil
}

func (t *ticker) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ticks
}

func TestTheSchedulerTicksUntilItIsStopped(t *testing.T) {
	t.Parallel()

	counter := &ticker{}
	running := scheduler.New(counter, 5*time.Millisecond, zaptest.NewLogger(t))

	if err := running.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := running.Start(t.Context()); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("a second Start = %v, want a conflict", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for counter.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if counter.count() == 0 {
		t.Fatal("the scheduler never ticked")
	}

	running.Stop()
	settled := counter.count()
	time.Sleep(20 * time.Millisecond)
	if counter.count() != settled {
		t.Fatal("the scheduler ticked after it was stopped")
	}

	running.Stop()
}

func TestAFailingTickDoesNotStopTheScheduler(t *testing.T) {
	t.Parallel()

	counter := &ticker{err: errors.New(errors.External, "the database is busy")}
	running := scheduler.New(counter, time.Millisecond, zaptest.NewLogger(t))

	if err := running.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(running.Stop)

	deadline := time.Now().Add(5 * time.Second)
	for counter.count() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if counter.count() < 2 {
		t.Fatalf("the scheduler stopped after a failure: %d ticks", counter.count())
	}
}

func TestTheTickIntervalIsASetting(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if got := scheduler.TickInterval(values); got != scheduler.DefaultTickInterval {
		t.Fatalf("TickInterval = %s, want %s", got, scheduler.DefaultTickInterval)
	}

	unknown, err := settings.Default().Apply(values, map[string]json.RawMessage{
		"schedules.tickInterval": json.RawMessage(`"30s"`),
	})
	if err != nil || len(unknown) != 0 {
		t.Fatalf("Apply = %v, %v", unknown, err)
	}
	if got := scheduler.TickInterval(values); got != 30*time.Second {
		t.Fatalf("TickInterval = %s", got)
	}

	if scheduler.New(&ticker{}, 0, zaptest.NewLogger(t)) == nil {
		t.Fatal("a scheduler without an interval still starts on its default")
	}
}
