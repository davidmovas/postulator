package runtime_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime"
)

const (
	pollInterval = 5 * time.Millisecond
	pollTimeout  = 20 * time.Second
)

type stubSpecs struct {
	spec    template.TemplateSpec
	version int
	err     error
}

func (s *stubSpecs) ResolveForPage(context.Context, templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
	if s.err != nil {
		return templates.ResolveForPageResponse{}, s.err
	}
	return templates.ResolveForPageResponse{TemplateID: "template", Version: s.version, Spec: s.spec}, nil
}

type stubSpend struct {
	mu    sync.Mutex
	spend llm.Spend
}

func (s *stubSpend) SumByRun(context.Context, string) (llm.Spend, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.spend, nil
}

func (s *stubSpend) set(spend llm.Spend) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spend = spend
}

type stubCatalog struct {
	info llm.ModelInfo
	err  error
}

func (c stubCatalog) Lookup(context.Context, llm.ModelRef) (llm.ModelInfo, error) {
	return c.info, c.err
}

type stubProfiles struct {
	ref llm.ModelRef
	err error
}

func (p stubProfiles) Resolve(context.Context, string, llm.Role, map[llm.Role]llm.ModelRef) (llm.ModelRef, error) {
	return p.ref, p.err
}

type record struct {
	payload   any
	runID     string
	eventType events.Type
	seq       int64
}

type recorder struct {
	mu   sync.Mutex
	rows []record
}

func (r *recorder) PublishRun(runID string, seq int64, eventType events.Type, payload any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, record{runID: runID, seq: seq, eventType: eventType, payload: payload})
	return nil
}

func (r *recorder) types() []events.Type {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]events.Type, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row.eventType)
	}
	return out
}

func (r *recorder) count(eventType events.Type) int {
	total := 0
	for _, seen := range r.types() {
		if seen == eventType {
			total++
		}
	}
	return total
}

type harness struct {
	store       *sqlite.Store
	runs        *sqlite.RunRepo
	items       *sqlite.RunItemRepo
	engineItems itemStore
	execs       *sqlite.StepExecRepo
	blobs       *sqlite.ArtifactRepo
	log         *sqlite.RunEventRepo
	specs       *stubSpecs
	spend       *stubSpend
	bus         *recorder
	pages       []string
	siteID      string
	logger      *zap.Logger
	deadline    time.Duration
}

func newHarness(t *testing.T, targets int) *harness {
	t.Helper()

	store := sqlitetest.Open(t)
	site := sqlitetest.Site(t, store, "shop")

	pages := make([]string, 0, targets)
	for i := range targets {
		page := sqlitetest.Page(t, store, site.ID, "/page-"+string(rune('a'+i))+"/")
		pages = append(pages, page.ID)
	}

	itemRepo := sqlite.NewRunItemRepo(store)
	return &harness{
		store:       store,
		runs:        sqlite.NewRunRepo(store),
		items:       itemRepo,
		engineItems: itemRepo,
		execs:       sqlite.NewStepExecRepo(store),
		blobs:       sqlite.NewArtifactRepo(store),
		log:         sqlite.NewRunEventRepo(store),
		specs: &stubSpecs{
			version: 1,
			spec: template.TemplateSpec{
				Sections: []template.Section{{Heading: "Intro", TargetWords: 300}},
				Length:   template.Length{Min: 200, Max: 900},
			},
		},
		spend:    &stubSpend{},
		bus:      &recorder{},
		pages:    pages,
		siteID:   site.ID,
		logger:   zaptest.NewLogger(t),
		deadline: time.Hour,
	}
}

func (h *harness) engine(t *testing.T, registry *run.Registry) *runtime.Engine {
	t.Helper()

	engine := h.idle(t, registry)
	if err := engine.Start(t.Context()); err != nil {
		t.Fatalf("start the engine: %v", err)
	}
	t.Cleanup(engine.Stop)
	return engine
}

func (h *harness) idle(t *testing.T, registry *run.Registry) *runtime.Engine {
	t.Helper()

	return runtime.New(runtime.Deps{
		Runs:       h.runs,
		Items:      h.engineItems,
		Artifacts:  h.blobs,
		Execs:      h.execs,
		Events:     h.log,
		Pages:      sqlite.NewPageRepo(h.store),
		Specs:      h.specs,
		Spend:      h.spend,
		Catalog:    stubCatalog{info: llm.ModelInfo{InputUSDPerM: 1, OutputUSDPerM: 2}},
		Profiles:   stubProfiles{ref: llm.ModelRef{Provider: "openai", Model: "test"}},
		UnitOfWork: h.store,
		Publisher:  h.bus,
	}, registry, runtime.Config{
		Workers:       3,
		PerSite:       3,
		SweepInterval: 20 * time.Millisecond,
		StepTimeout:   2 * time.Second,
		LeaseDuration: 200 * time.Millisecond,
		RunDeadline:   h.deadline,
	}, clock.System{}, h.logger)
}

func (h *harness) newRun(recipe []template.StepSpec) run.Run {
	return run.Run{
		ID: id.New(), SiteID: h.siteID, Kind: run.KindGenerate, Targets: h.pages, Recipe: recipe,
		TemplateID: "template", TemplateVersion: 1, PublishMode: run.PublishDraft,
	}
}

func (h *harness) waitForRun(t *testing.T, runID string, want run.Status) run.Run {
	t.Helper()

	var last run.Run
	waitFor(t, "run "+runID+" to reach "+string(want), func() bool {
		record, err := h.runs.Get(t.Context(), runID)
		if err != nil {
			return false
		}
		last = record
		return record.Status == want
	})
	return last
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func recipeOf(names ...string) []template.StepSpec {
	specs := make([]template.StepSpec, 0, len(names))
	for _, name := range names {
		specs = append(specs, template.StepSpec{Name: name, Enabled: true})
	}
	return specs
}

type counter struct {
	mu    sync.Mutex
	calls map[string]int
}

func newCounter() *counter {
	return &counter{calls: make(map[string]int)}
}

func (c *counter) hit(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[key]++
	return c.calls[key]
}

func (c *counter) get(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[key]
}

func (c *counter) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	sum := 0
	for _, count := range c.calls {
		sum += count
	}
	return sum
}

func mustRegister(t *testing.T, defs ...run.StepDef) *run.Registry {
	t.Helper()

	registry := run.NewRegistry()
	for i := range defs {
		if err := registry.Register(defs[i]); err != nil {
			t.Fatalf("register %s: %v", defs[i].Name, err)
		}
	}
	return registry
}

func producing(name string, kind run.ArtifactKind, requires []run.ArtifactKind, body func(context.Context, *run.StepContext) (run.Result, error)) run.StepDef {
	return run.StepDef{
		Name:     name,
		Requires: requires,
		Produces: []run.ArtifactKind{kind},
		Retry:    run.RetryPolicy{Max: 3, Backoff: func(int) time.Duration { return 10 * time.Millisecond }},
		Run:      body,
	}
}

func faulty(code errors.Code, message string) error {
	return errors.New(code, message)
}

func pageRepoOf(h *harness) *sqlite.PageRepo {
	return sqlite.NewPageRepo(h.store)
}

func systemClock() clock.Clock {
	return clock.System{}
}
