package runtime

import (
	"context"
	"maps"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Deps struct {
	Runs       runStore
	Items      itemStore
	Artifacts  artifactStore
	Execs      execStore
	Events     eventStore
	Pages      pageReader
	Specs      specResolver
	Spend      spendReader
	Catalog    modelCatalog
	Profiles   profileResolver
	UnitOfWork unitOfWork
	Publisher  publisher
}

type Engine struct {
	deps     Deps
	registry *run.Registry
	cfg      Config
	clock    clock.Clock
	logger   *zap.Logger

	queue  chan string
	wakeup chan struct{}
	stop   chan struct{}
	wg     sync.WaitGroup

	running atomic.Bool

	mu       sync.Mutex
	inflight map[string]string
	load     map[string]int
	leased   map[string]int64
	cancels  map[string]map[string]context.CancelFunc
	release  context.CancelFunc
}

func New(deps Deps, registry *run.Registry, cfg Config, clk clock.Clock, logger *zap.Logger) *Engine {
	return &Engine{
		deps:     deps,
		registry: registry,
		cfg:      cfg.normalized(),
		clock:    clk,
		logger:   logger,
		queue:    make(chan string, queueCapacity),
		wakeup:   make(chan struct{}, 1),
		stop:     make(chan struct{}),
		inflight: make(map[string]string),
		load:     make(map[string]int),
		leased:   make(map[string]int64),
		cancels:  make(map[string]map[string]context.CancelFunc),
	}
}

func (e *Engine) now() time.Time {
	return e.clock.Now().UTC().Truncate(time.Second)
}

func (e *Engine) Start(ctx context.Context) error {
	if !e.running.CompareAndSwap(false, true) {
		return errors.New(errors.Conflict, "the run engine is already started")
	}

	base, release := context.WithCancel(context.WithoutCancel(ctx))
	e.mu.Lock()
	e.release = release
	e.mu.Unlock()

	for range e.cfg.Workers {
		e.wg.Add(1)
		go e.worker(base)
	}
	e.wg.Add(1)
	go e.keeper(base)

	if err := e.Recover(ctx); err != nil {
		return err
	}
	e.nudge()
	return nil
}

func (e *Engine) Stop() {
	if !e.running.CompareAndSwap(true, false) {
		return
	}

	close(e.stop)

	e.mu.Lock()
	release := e.release
	cancels := make([]context.CancelFunc, 0, len(e.cancels))
	for _, byItem := range e.cancels {
		for _, cancel := range byItem {
			cancels = append(cancels, cancel)
		}
	}
	e.cancels = make(map[string]map[string]context.CancelFunc)
	e.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	if release != nil {
		release()
	}
	e.wg.Wait()
	e.handBack(context.Background())
}

func (e *Engine) stopping() bool {
	select {
	case <-e.stop:
		return true
	default:
		return false
	}
}

func (e *Engine) hold(itemID string, expectSeq int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.leased[itemID] = expectSeq
}

func (e *Engine) forget(itemID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.leased, itemID)
}

func (e *Engine) handBack(ctx context.Context) {
	e.mu.Lock()
	leased := maps.Clone(e.leased)
	clear(e.leased)
	e.mu.Unlock()

	if len(leased) == 0 {
		return
	}

	now := e.now()
	err := e.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		for _, itemID := range slices.Sorted(maps.Keys(leased)) {
			if _, requeueErr := e.deps.Items.Requeue(c, itemID, leased[itemID], run.StatusRunning, now); requeueErr != nil {
				return requeueErr
			}
		}
		return nil
	})
	if err != nil {
		e.logger.Warn("handing the items still running back to the queue failed; the lease sweep will re-arm them",
			zap.Int("items", len(leased)),
			zap.Error(err),
		)
	}
}

func (e *Engine) nudge() {
	select {
	case e.wakeup <- struct{}{}:
	default:
	}
}

func (e *Engine) worker(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case <-e.stop:
			return
		case <-ctx.Done():
			return
		case itemID := <-e.queue:
			again := e.run(ctx, itemID)
			e.finish(itemID)
			if again {
				e.dispatch(ctx, itemID)
			}
		}
	}
}

func (e *Engine) run(ctx context.Context, itemID string) bool {
	again, err := e.advance(ctx, itemID)
	if err != nil {
		e.logger.Warn("advancing a run item failed",
			zap.String("itemId", itemID),
			zap.String("code", errors.CodeOf(err).String()),
			zap.Error(err),
		)
	}
	return again
}

func (e *Engine) keeper(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.cfg.SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.sweepAndFill(ctx)
		case <-e.wakeup:
			e.sweepAndFill(ctx)
		}
	}
}

func (e *Engine) sweepAndFill(ctx context.Context) {
	if err := e.sweep(ctx); err != nil {
		e.logger.Warn("the run engine sweep failed", zap.Error(err))
	}
	if err := e.fill(ctx); err != nil {
		e.logger.Warn("the run engine could not dispatch runnable items", zap.Error(err))
	}
}

func (e *Engine) fill(ctx context.Context) error {
	items, err := e.deps.Items.Runnable(ctx, e.now(), dispatchBatch)
	if err != nil {
		return err
	}

	for i := range items {
		e.offer(items[i].ID, items[i].SiteID)
	}
	return nil
}

func (e *Engine) dispatch(ctx context.Context, itemID string) {
	siteID, ok := e.siteOf(ctx, itemID)
	if !ok {
		return
	}
	e.offer(itemID, siteID)
}

func (e *Engine) offer(itemID, siteID string) {
	if !e.reserve(itemID, siteID) {
		return
	}

	select {
	case e.queue <- itemID:
	default:
		e.finish(itemID)
	}
}

func (e *Engine) siteOf(ctx context.Context, itemID string) (string, bool) {
	item, err := e.deps.Items.Get(ctx, itemID)
	if err != nil {
		return "", false
	}
	return item.SiteID, true
}

func (e *Engine) reserve(itemID, siteID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, busy := e.inflight[itemID]; busy {
		return false
	}
	if e.load[siteID] >= e.cfg.PerSite {
		return false
	}
	e.inflight[itemID] = siteID
	e.load[siteID]++
	return true
}

func (e *Engine) finish(itemID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	siteID, busy := e.inflight[itemID]
	if !busy {
		return
	}
	delete(e.inflight, itemID)
	if e.load[siteID] <= 1 {
		delete(e.load, siteID)
		return
	}
	e.load[siteID]--
}

func (e *Engine) runContext(parent context.Context, runID, itemID string) (ctx context.Context, done func()) {
	stepCtx, cancel := context.WithCancel(parent)

	e.mu.Lock()
	byItem, ok := e.cancels[runID]
	if !ok {
		byItem = make(map[string]context.CancelFunc)
		e.cancels[runID] = byItem
	}
	byItem[itemID] = cancel
	e.mu.Unlock()

	return stepCtx, func() {
		cancel()
		e.mu.Lock()
		if current, present := e.cancels[runID]; present {
			delete(current, itemID)
			if len(current) == 0 {
				delete(e.cancels, runID)
			}
		}
		e.mu.Unlock()
	}
}

func (e *Engine) cancelRun(runID string) {
	e.mu.Lock()
	byItem := e.cancels[runID]
	cancels := make([]context.CancelFunc, 0, len(byItem))
	for _, cancel := range byItem {
		cancels = append(cancels, cancel)
	}
	delete(e.cancels, runID)
	e.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}
