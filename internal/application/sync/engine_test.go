package sync_test

import (
	"context"
	stdsync "sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	pollInterval   = 5 * time.Millisecond
	pollFallback   = 10 * time.Minute
	deadlineMargin = 30 * time.Second
)

func waitDeadline(t *testing.T) time.Time {
	t.Helper()

	held, bounded := t.Deadline()
	if !bounded {
		return time.Now().Add(pollFallback)
	}
	return held.Add(-deadlineMargin)
}

type oneClient struct {
	client *wp.Client
}

func (o oneClient) Client(context.Context, string) (*wp.Client, error) {
	return o.client, nil
}

type silentBus struct{}

func (silentBus) Publish(events.Type, any) error { return nil }

func (silentBus) PublishRun(string, int64, events.Type, any) error { return nil }

type noSpend struct{}

func (noSpend) SumByRun(context.Context, string) (llm.Spend, error) { return llm.Spend{}, nil }

type noCatalog struct{}

func (noCatalog) Lookup(context.Context, llm.ModelRef) (llm.ModelInfo, error) {
	return llm.ModelInfo{}, nil
}

type noProfiles struct{}

func (noProfiles) Resolve(context.Context, string, llm.Role, map[llm.Role]llm.ModelRef) (llm.ModelRef, error) {
	return llm.ModelRef{Provider: "openai", Model: "test"}, nil
}

type noSpecs struct{}

func (noSpecs) ResolveForPage(context.Context, templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
	return templates.ResolveForPageResponse{}, nil
}

func engineOver(t *testing.T, store *sqlite.Store, registry *run.Registry) *runtime.Engine {
	t.Helper()

	return runtime.New(runtime.Deps{
		Runs: sqlite.NewRunRepo(store), Items: sqlite.NewRunItemRepo(store), Artifacts: sqlite.NewArtifactRepo(store),
		Execs: sqlite.NewStepExecRepo(store), Events: sqlite.NewRunEventRepo(store), Pages: sqlite.NewPageRepo(store),
		Specs: noSpecs{}, Spend: noSpend{}, Catalog: noCatalog{}, Profiles: noProfiles{},
		UnitOfWork: store, Publisher: silentBus{},
	}, registry, runtime.Config{
		Workers: 2, PerSite: 1, SweepInterval: 20 * time.Millisecond,
		StepTimeout: 10 * time.Second, LeaseDuration: time.Minute, RunDeadline: time.Hour,
	}, clock.System{}, zaptest.NewLogger(t))
}

func TestManySyncsOfOneSiteAtOnceQueueOne(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	siteRepo := sqlite.NewSiteRepo(store)
	owner := site.Site{
		ID: id.New(), Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor",
		Status: site.StatusActive, Plugin: site.PluginState{Capabilities: []string{}},
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	owner.SecretRef = site.SecretRef(owner.ID)
	if err := siteRepo.Insert(t.Context(), owner); err != nil {
		t.Fatalf("insert the site: %v", err)
	}
	registry := run.NewRegistry()
	if err := registry.Register(steps.SyncSite(steps.Deps{Clock: clock.System{}})); err != nil {
		t.Fatalf("register the sync step: %v", err)
	}
	runRepo := sqlite.NewRunRepo(store)
	service := sync.New(engineOver(t, store, registry), runRepo, siteRepo, prober{}, packer{}, clock.System{})

	const callers = 8
	answers := make(chan error, callers)
	var wait stdsync.WaitGroup
	for range callers {
		wait.Go(func() {
			_, err := service.SyncSite(t.Context(), sync.SyncSiteRequest{SiteID: owner.ID})
			answers <- err
		})
	}
	wait.Wait()
	close(answers)

	queued, refused := 0, 0
	for err := range answers {
		switch {
		case err == nil:
			queued++
		case errors.IsCode(err, errors.Conflict):
			refused++
		default:
			t.Errorf("a sync answered %v, want it queued or refused as a conflict", err)
		}
	}
	if queued != 1 || refused != callers-1 {
		t.Fatalf("queued %d and refused %d of %d syncs started at once, want one queued", queued, refused, callers)
	}
	if active, found, err := runRepo.ActiveSync(t.Context(), owner.ID); err != nil || !found || active.Kind != run.KindSync {
		t.Fatalf("the active sync = %+v %t %v", active, found, err)
	}
}

func TestSyncSiteCompletesThroughTheEngine(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Shoes", Slug: "shoes", Status: "publish", Content: "<h1>Shoes</h1>"},
		wptest.Item{Type: wptest.TypePage, Title: "Boots", Slug: "boots", Status: "publish", Content: "<h1>Boots</h1>"},
	)

	store := sqlitetest.Open(t)
	siteRepo := sqlite.NewSiteRepo(store)
	pageRepo := sqlite.NewPageRepo(store)

	owner := site.Site{
		ID: id.New(), Name: "Shop", BaseURL: server.URL(), Username: wptest.DefaultUser,
		Status: site.StatusActive, AllowInsecure: true, Plugin: site.PluginState{Capabilities: []string{}},
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	owner.SecretRef = site.SecretRef(owner.ID)
	if err := siteRepo.Insert(t.Context(), owner); err != nil {
		t.Fatalf("insert the site: %v", err)
	}

	client, err := wp.New(wp.Config{
		BaseURL: server.URL(), Username: wptest.DefaultUser,
		AppPassword: wptest.DefaultPassword, AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("build the WordPress client: %v", err)
	}

	linkRepo := sqlite.NewPageLinkRepo(store)
	observer := observe.New(observe.Deps{
		Sites: siteRepo, Pages: pageRepo, Identity: sqlite.NewPageIdentityRepo(store), Fields: sqlite.NewPageFieldRepo(store),
		Bodies: sqlite.NewPageBodyRepo(store), Renders: sqlite.NewPageRenderRepo(store), Links: linkRepo,
		Findings: sqlite.NewFindingRepo(store), UnitOfWork: store, Publisher: silentBus{}, Clock: clock.System{},
		Entities: sqlite.NewEntityRepo(store), Edges: sqlite.NewEdgeRepo(store),
		Types: pagetypes.New(pagetypes.Deps{
			Types: sqlite.NewPageTypeRepo(store), Scopes: sqlite.NewFieldScopeRepo(store), Rules: sqlite.NewPageTypeRuleRepo(store), Pages: pageRepo,
			Templates: sqlite.NewTemplateRepo(store), Sites: siteRepo, UnitOfWork: store, Publisher: silentBus{}, Clock: clock.System{},
		}),
	})
	registry := run.NewRegistry()
	if regErr := registry.Register(steps.SyncSite(steps.Deps{
		Pages: pageRepo, Links: linkRepo, Sites: siteRepo, SiteWriter: siteRepo,
		WordPress: oneClient{client: client}, Observe: observer, UnitOfWork: store, Publisher: silentBus{},
		Clock: clock.System{}, BatchSize: 1,
	})); regErr != nil {
		t.Fatalf("register the sync step: %v", regErr)
	}

	runRepo := sqlite.NewRunRepo(store)
	engine := engineOver(t, store, registry)
	if startErr := engine.Start(t.Context()); startErr != nil {
		t.Fatalf("start the engine: %v", startErr)
	}
	t.Cleanup(engine.Stop)

	service := sync.New(engine, runRepo, siteRepo, prober{}, packer{}, clock.System{})
	queued, err := service.SyncSite(t.Context(), sync.SyncSiteRequest{SiteID: owner.ID})
	if err != nil {
		t.Fatalf("SyncSite: %v", err)
	}

	deadline := waitDeadline(t)
	var final run.Run
	for time.Now().Before(deadline) {
		record, getErr := runRepo.Get(t.Context(), queued.RunID)
		if getErr == nil && record.Status.Terminal() {
			final = record
			break
		}
		time.Sleep(pollInterval)
	}
	if final.Status != run.StatusCompleted {
		t.Fatalf("the sync run ended as %q, want completed: %s", final.Status, final.Error)
	}

	pages, err := pageRepo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("the sync stored %d pages, want 2", len(pages))
	}
}
