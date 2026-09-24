package runtime_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type itemStore interface {
	Insert(ctx context.Context, item run.Item) error
	Get(ctx context.Context, id string) (run.Item, error)
	Claim(ctx context.Context, id string, expectSeq int64, leaseUntil, now time.Time) (run.Item, error)
	Persist(ctx context.Context, item run.Item, expectSeq int64) (bool, error)
	Requeue(ctx context.Context, id string, expectSeq int64, from run.Status, now time.Time) (bool, error)
	ByRun(ctx context.Context, runID string) ([]run.Item, error)
	Due(ctx context.Context, now time.Time, limit int) ([]run.Item, error)
	Stalled(ctx context.Context, now time.Time, limit int) ([]run.Item, error)
	Runnable(ctx context.Context, now time.Time, limit int) ([]run.Item, error)
	AwaitingParent(ctx context.Context, limit int) ([]run.Item, error)
	BehindStoppedBlockers(ctx context.Context, limit int) ([]run.Item, error)
	Counts(ctx context.Context, runID string) (map[run.Status]int, error)
	StopAll(ctx context.Context, runID string, from []run.Status, to run.Status, reason run.PauseReason, now time.Time) (int64, error)
	ResumeAll(ctx context.Context, runID string, now time.Time) (int64, error)
}

type refusingItems struct {
	itemStore
	once   sync.Once
	broken chan struct{}
}

func refuseFirstPersist(store itemStore) *refusingItems {
	return &refusingItems{itemStore: store, broken: make(chan struct{})}
}

func (r *refusingItems) Persist(ctx context.Context, item run.Item, expectSeq int64) (bool, error) {
	refused := false
	r.once.Do(func() { refused = true })
	if refused {
		close(r.broken)
		return false, errors.New(errors.Internal, "the run item could not be written")
	}
	return r.itemStore.Persist(ctx, item, expectSeq)
}

func siteClient(t *testing.T, server *wptest.Server) *wp.Client {
	t.Helper()

	client, err := wp.New(wp.Config{
		BaseURL: server.URL(), Username: wptest.DefaultUser, AppPassword: wptest.DefaultPassword,
	})
	if err != nil {
		t.Fatalf("build the WordPress client: %v", err)
	}
	return client
}

func onlyAStopInterrupts(def run.StepDef) run.StepDef {
	def.Timeout = time.Minute
	def.Retry = run.RetryPolicy{}
	return def
}

func TestAStepThatWroteToTheSiteIsRecordedWhileTheEngineStops(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	server := wptest.New(t)
	client := siteClient(t, server)

	calls := newCounter()
	created := make(chan struct{})
	writing := onlyAStopInterrupts(producing("publish", run.ArtifactPublishResult, nil,
		func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit(sc.Page.ID)
			if _, err := client.CreateItem(ctx, wp.TypePage, wp.CreateItem{
				Title: "Espresso", Content: "<p>hello</p>", Slug: "espresso", Status: "draft",
			}); err != nil {
				return run.Result{}, err
			}
			close(created)
			<-ctx.Done()
			return run.Result{
				Next:      run.TransitionWait,
				WakeAt:    time.Now().UTC().Add(200 * time.Millisecond),
				Artifacts: []run.Artifact{{Kind: run.ArtifactPublishResult, Blob: []byte(`{"created":true}`)}},
			}, nil
		}))

	first := harness.engine(t, mustRegister(t, writing))
	queued, err := first.Enqueue(t.Context(), harness.newRun(recipeOf("publish")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	<-created
	first.Stop()

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	execs, err := harness.execs.ByItem(t.Context(), items[0].ID)
	if err != nil {
		t.Fatalf("ByItem: %v", err)
	}
	if len(execs) != 1 || execs[0].Status != run.ExecDone {
		t.Fatalf("the finished step left %d executions: %+v", len(execs), execs)
	}
	if items[0].Status == run.StatusRunning {
		t.Fatalf("the item is still running after a graceful stop")
	}

	second := harness.engine(t, mustRegister(t, writing))
	_ = second
	harness.waitForRun(t, queued.ID, run.StatusCompleted)

	if got := calls.get(harness.pages[0]); got != 1 {
		t.Fatalf("the step ran %d times, want the recorded execution to be reused", got)
	}
	if pages := server.Items(); len(pages) != 1 {
		t.Fatalf("the site carries %d pages, want the one the step created", len(pages))
	}
}

func TestAStepStoppedByTheEngineIsHandedBackNotFailed(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	calls := newCounter()
	waiting := onlyAStopInterrupts(producing("generate_body", run.ArtifactBodyHTML, nil,
		func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit(sc.Page.ID)
			<-ctx.Done()
			return run.Result{}, ctx.Err()
		}))

	engine := harness.engine(t, mustRegister(t, waiting))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	waitFor(t, "the step to be in flight", func() bool { return calls.total() == 1 })
	engine.Stop()

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	if items[0].Status != run.StatusPending || items[0].CurrentStep != "generate_body" {
		t.Fatalf("the stopped item is %q at %q, want pending at generate_body", items[0].Status, items[0].CurrentStep)
	}
	if items[0].Attempts != 0 || items[0].Error != "" {
		t.Fatalf("the stopped item counted %d attempts and recorded %q", items[0].Attempts, items[0].Error)
	}

	execs, err := harness.execs.ByItem(t.Context(), items[0].ID)
	if err != nil {
		t.Fatalf("ByItem: %v", err)
	}
	if len(execs) != 0 {
		t.Fatalf("a step the engine stopped left %d executions", len(execs))
	}
}

func TestStopHandsBackAnItemWhoseSettleCouldNotBeWritten(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	refusing := refuseFirstPersist(harness.items)
	harness.engineItems = refusing

	release := make(chan struct{})
	calls := newCounter()
	blocking := onlyAStopInterrupts(producing("generate_body", run.ArtifactBodyHTML, nil,
		func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			calls.hit(sc.Page.ID)
			<-release
			return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactBodyHTML, Blob: []byte("<p>ok</p>")}}}, nil
		}))

	engine := harness.engine(t, mustRegister(t, blocking))
	queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	waitFor(t, "the step to be in flight", func() bool { return calls.total() == 1 })
	close(release)
	<-refusing.broken

	engine.Stop()

	items, err := harness.items.ByRun(t.Context(), queued.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("ByRun = %+v, %v", items, err)
	}
	if items[0].Status != run.StatusPending {
		t.Fatalf("the item is %q after a graceful stop, want pending", items[0].Status)
	}
	if items[0].LeaseUntil != nil {
		t.Fatalf("the item still holds a lease until %v", items[0].LeaseUntil)
	}

	runnable, err := harness.items.Runnable(t.Context(), time.Now().UTC(), 10)
	if err != nil {
		t.Fatalf("Runnable: %v", err)
	}
	if len(runnable) != 1 || runnable[0].ID != items[0].ID {
		t.Fatalf("a restart would pick up %d items, want the one the stop handed back", len(runnable))
	}
}
