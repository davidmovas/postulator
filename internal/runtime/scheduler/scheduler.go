package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const DefaultTickInterval = time.Minute

var tickSetting = settings.Duration("schedules.tickInterval", DefaultTickInterval,
	settings.DurationRange(time.Second, time.Hour))

func TickInterval(values *settings.Values) time.Duration {
	return tickSetting.Get(values)
}

type ticker interface {
	Tick(ctx context.Context) (schedules.TickResponse, error)
}

type Scheduler struct {
	schedules ticker
	interval  time.Duration
	logger    *zap.Logger

	running atomic.Bool
	stop    chan struct{}
	wg      sync.WaitGroup

	mu      sync.Mutex
	release context.CancelFunc
}

func New(due ticker, interval time.Duration, logger *zap.Logger) *Scheduler {
	if interval <= 0 {
		interval = DefaultTickInterval
	}
	return &Scheduler{schedules: due, interval: interval, logger: logger, stop: make(chan struct{})}
}

func (s *Scheduler) Start(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return errors.New(errors.Conflict, "the scheduler is already started")
	}

	base, release := context.WithCancel(context.WithoutCancel(ctx))
	s.mu.Lock()
	s.release = release
	s.mu.Unlock()

	s.wg.Add(1)
	go s.loop(base)
	return nil
}

func (s *Scheduler) Stop() {
	if !s.running.CompareAndSwap(true, false) {
		return
	}

	close(s.stop)

	s.mu.Lock()
	release := s.release
	s.mu.Unlock()

	if release != nil {
		release()
	}
	s.wg.Wait()
}

func (s *Scheduler) loop(ctx context.Context) {
	defer s.wg.Done()

	ticks := time.NewTicker(s.interval)
	defer ticks.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ctx.Done():
			return
		case <-ticks.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	started, err := s.schedules.Tick(ctx)
	if err != nil {
		s.logger.Warn("the schedule tick failed", zap.Error(err))
		return
	}
	if len(started.Started) == 0 {
		return
	}
	s.logger.Info("the scheduler started runs",
		zap.Strings("runIds", started.Started), zap.Int("skipped", started.Skipped))
}
