package steps_test

import (
	"encoding/json"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type busRecorder struct {
	types []events.Type
	err   error
	mu    sync.Mutex
}

func (b *busRecorder) Publish(eventType events.Type, _ any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.types = append(b.types, eventType)
	return b.err
}

func (b *busRecorder) seen() []events.Type {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.types)
}

type syncHarness struct {
	store  *sqlite.Store
	server *wptest.Server
	deps   steps.Deps
	bus    *busRecorder
	clock  *clock.Fake
	pages  *sqlite.PageRepo
	links  *sqlite.PageLinkRepo
	siteID string
	check  run.Checkpoint
}

func newSyncHarness(t *testing.T, batch int, opts ...wptest.Option) *syncHarness {
	t.Helper()

	server := wptest.New(t, opts...)
	store := sqlitetest.Open(t)

	owner := site.Site{
		ID: id.New(), Name: "Shop", BaseURL: server.URL(), Username: wptest.DefaultUser,
		Status: site.StatusActive, AllowInsecure: true, Plugin: site.PluginState{Capabilities: []string{}},
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	owner.SecretRef = site.SecretRef(owner.ID)
	if err := sqlite.NewSiteRepo(store).Insert(t.Context(), owner); err != nil {
		t.Fatalf("insert the site: %v", err)
	}

	bus := &busRecorder{}
	fake := clock.NewFake(sqlitetest.Stamp)
	siteRepo := sqlite.NewSiteRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	linkRepo := sqlite.NewPageLinkRepo(store)

	return &syncHarness{
		store: store, server: server, bus: bus, clock: fake,
		pages: pageRepo, links: linkRepo, siteID: owner.ID, check: run.NewCheckpoint(),
		deps: steps.Deps{
			Pages: pageRepo, Links: linkRepo, Sites: siteRepo, SiteWriter: siteRepo,
			WordPress: oneClient{client: syncClient(t, server)}, UnitOfWork: store, Publisher: bus,
			Clock: fake, BatchSize: batch,
		},
	}
}

func (h *syncHarness) once(t *testing.T) (steps.SiteSyncResult, run.Result) {
	t.Helper()

	sc := &run.StepContext{
		Run:       run.Run{ID: "run", SiteID: h.siteID, Kind: run.KindSync},
		Item:      run.Item{ID: "item", RunID: "run", SiteID: h.siteID, TargetID: h.siteID},
		Params:    map[string]any{},
		Artifacts: map[run.ArtifactKind]run.Artifact{},
		Check:     h.check.Clone(),
	}

	result, err := steps.SyncSite(h.deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("SyncSite: %v", err)
	}
	h.check = h.check.MergedWith(result.Checkpoint)

	var state steps.SiteSyncResult
	if err = json.Unmarshal(result.Artifacts[0].Blob, &state); err != nil {
		t.Fatalf("decode the site sync result: %v", err)
	}
	return state, result
}

func (h *syncHarness) all(t *testing.T) steps.SiteSyncResult {
	t.Helper()

	for range 20 {
		state, _ := h.once(t)
		if state.Done {
			return state
		}
	}
	t.Fatal("the sync never finished")
	return steps.SiteSyncResult{}
}

func syncClient(t *testing.T, server *wptest.Server) *wp.Client {
	t.Helper()

	client, err := wp.New(wp.Config{
		BaseURL: server.URL(), Username: wptest.DefaultUser,
		AppPassword: wptest.DefaultPassword, AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func (h *syncHarness) restart(t *testing.T) {
	t.Helper()
	h.check = run.NewCheckpoint()
	h.deps.WordPress = oneClient{client: syncClient(t, h.server)}
}

func (h *syncHarness) reconnect(t *testing.T) {
	t.Helper()
	h.deps.WordPress = oneClient{client: syncClient(t, h.server)}
}

func (h *syncHarness) byPath(t *testing.T, path string) pagemap.Page {
	t.Helper()

	stored, err := h.pages.ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	page, ok := pagemap.NewIndex(stored).ByPath(path)
	if !ok {
		t.Fatalf("no page at %s; the site map holds %d pages", path, len(stored))
	}
	return page
}

func seedSite(t *testing.T, h *syncHarness) {
	t.Helper()

	seeded := h.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Coffee", Slug: "coffee",
		Content: `<h1>Coffee</h1><p>All about coffee.</p>`,
	})
	h.server.Seed(
		wptest.Item{
			Type: wptest.TypePage, Title: "Espresso", Slug: "espresso", Parent: seeded[0].ID,
			Content: `<h1>Espresso</h1><p>Part of <a href="/coffee/">coffee</a>.</p>`,
		},
		wptest.Item{
			Type: wptest.TypePage, Title: "Filter", Slug: "filter", Parent: seeded[0].ID,
			Content: `<h1>Filter</h1><p>Also <a href="/coffee/">coffee</a>.</p>`, Status: "draft",
		},
	)
}

func TestSyncSitePullsThroughThePluginAndThroughCore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		opts   []wptest.Option
		source string
	}{
		{name: "with the plugin", source: steps.SourcePlugin},
		{name: "without the plugin", opts: []wptest.Option{wptest.WithoutPlugin()}, source: steps.SourceCore},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newSyncHarness(t, 0, tc.opts...)
			seedSite(t, h)

			state := h.all(t)
			if state.Source != tc.source || state.Pulled != 3 || state.Created != 3 {
				t.Fatalf("state = %+v", state)
			}

			child := h.byPath(t, "/coffee/espresso/")
			if child.WPID == nil || child.Title != "Espresso" || child.H1 != "Espresso" {
				t.Fatalf("the child page is %+v", child)
			}
			if child.Status != pagemap.StatusPublished || child.ContentHash == "" {
				t.Fatalf("the child page is %+v", child)
			}
			parent := h.byPath(t, "/coffee/")
			if child.ParentPageID == nil || *child.ParentPageID != parent.ID {
				t.Fatalf("the child page is %+v, want the parent %s", child, parent.ID)
			}
			if draft := h.byPath(t, "/coffee/filter/"); draft.Status != pagemap.StatusExists {
				t.Fatalf("the draft page is %+v", draft)
			}

			links, err := h.links.ListForPage(t.Context(), child.ID)
			if err != nil {
				t.Fatalf("list the links: %v", err)
			}
			if len(links) != 1 || links[0].ToPageID == nil || *links[0].ToPageID != parent.ID {
				t.Fatalf("the links of the child are %+v", links)
			}

			if seen := h.bus.seen(); len(seen) != 1 || seen[0] != events.PagesChanged {
				t.Fatalf("the bus saw %v", h.bus.seen())
			}
		})
	}
}

func TestSyncSiteAdoptsTheManifest(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	seedSite(t, h)
	h.all(t)

	owner, err := sqlite.NewSiteRepo(h.store).Get(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("read the site: %v", err)
	}
	if !owner.Plugin.Installed || owner.Plugin.SEOPlugin != "yoast" || len(owner.Plugin.Capabilities) == 0 {
		t.Fatalf("the site plugin state is %+v", owner.Plugin)
	}
}

func TestSyncSiteFlagsDriftAndArchivesWhatIsGone(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	seedSite(t, h)
	h.all(t)

	child := h.byPath(t, "/coffee/espresso/")
	filter := h.byPath(t, "/coffee/filter/")
	if child.Drift {
		t.Fatalf("the first sync must not report drift: %+v", child)
	}

	h.server.Rewrite(*child.WPID, `<h1>Espresso</h1><p>Edited by a human.</p>`)
	if !h.server.Delete(*filter.WPID) {
		t.Fatal("the filter page could not be removed from the site")
	}

	h.clock.Advance(time.Minute)
	h.restart(t)
	state := h.all(t)

	if state.Drifted != 1 || state.Archived != 1 {
		t.Fatalf("state = %+v", state)
	}
	if drifted := h.byPath(t, "/coffee/espresso/"); !drifted.Drift {
		t.Fatalf("the edited page is %+v", drifted)
	}
	if archived := h.byPath(t, "/coffee/filter/"); archived.Status != pagemap.StatusArchived {
		t.Fatalf("the deleted page is %+v", archived)
	}
}

func TestSyncSiteResumesFromItsCursor(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 2)
	seedSite(t, h)

	first, result := h.once(t)
	if first.Done || first.Cursor == "" || first.Pulled != 2 {
		t.Fatalf("the first batch = %+v", first)
	}
	if result.Next != run.TransitionWait {
		t.Fatalf("the first batch asked for %q, want a wait", result.Next)
	}

	h.reconnect(t)
	state := h.all(t)
	if !state.Done || state.Pulled != 3 || state.Batches < 2 {
		t.Fatalf("state = %+v", state)
	}
	if seen := h.bus.seen(); len(seen) != 1 {
		t.Fatalf("the bus saw %v, want one pages.changed at the end", seen)
	}
}

func TestSyncSiteReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0)
	h.deps.WordPress = oneClient{err: errors.New(errors.External, "the site is down")}

	sc := &run.StepContext{
		Run:       run.Run{ID: "run", SiteID: h.siteID, Kind: run.KindSync},
		Item:      run.Item{ID: "item", RunID: "run", SiteID: h.siteID, TargetID: h.siteID},
		Artifacts: map[run.ArtifactKind]run.Artifact{},
		Check:     run.NewCheckpoint(),
	}
	if _, err := steps.SyncSite(h.deps).Run(t.Context(), sc); !errors.IsCode(err, errors.External) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.External, err)
	}
}

func TestSyncSiteResolvesALinkWhoseTargetArrivesLater(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 1)
	h.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Espresso", Slug: "espresso",
		Content: `<h1>Espresso</h1><p>Part of <a href="/coffee/">coffee</a>.</p>`,
	})
	h.server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Coffee", Slug: "coffee", Content: `<h1>Coffee</h1><p>All of it.</p>`,
	})

	state := h.all(t)
	if state.Pulled != 2 || state.Batches < 2 {
		t.Fatalf("state = %+v", state)
	}

	child := h.byPath(t, "/espresso/")
	parent := h.byPath(t, "/coffee/")
	links, err := h.links.ListForPage(t.Context(), child.ID)
	if err != nil {
		t.Fatalf("list the links: %v", err)
	}
	if len(links) != 1 || links[0].ToPageID == nil || *links[0].ToPageID != parent.ID {
		t.Fatalf("the links of the child are %+v, want the target resolved after the second batch", links)
	}
}

func TestSyncSiteRefusesAnUnreadableCursor(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0, wptest.WithoutPlugin())
	seedSite(t, h)

	broken := run.NewCheckpoint()
	if err := run.Set(broken, "sync", steps.SiteSyncResult{Source: steps.SourceCore, Cursor: "{{{"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	h.check = broken

	sc := &run.StepContext{
		Run:       run.Run{ID: "run", SiteID: h.siteID, Kind: run.KindSync},
		Item:      run.Item{ID: "item", RunID: "run", SiteID: h.siteID, TargetID: h.siteID},
		Artifacts: map[run.ArtifactKind]run.Artifact{},
		Check:     broken,
	}
	if _, err := steps.SyncSite(h.deps).Run(t.Context(), sc); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Invalid, err)
	}
}

func TestBatchSizeCarriesItsDefaultAndItsSetting(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if got := steps.BatchSize(values); got != steps.DefaultBatchSize {
		t.Fatalf("BatchSize = %d, want %d", got, steps.DefaultBatchSize)
	}

	if _, err := settings.Default().Apply(values, map[string]json.RawMessage{
		"sync.batchSize": json.RawMessage(`25`),
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := steps.BatchSize(values); got != 25 {
		t.Fatalf("BatchSize = %d, want 25", got)
	}
}

func TestSyncSiteLeavesTypesTheCorePullNeverSaw(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 0, wptest.WithoutPlugin())
	seedSite(t, h)

	product := int64(4242)
	stray := pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: "/shop/grinder/", Slug: "grinder",
		WPType: pagemap.WPProduct, WPID: &product, Status: pagemap.StatusPublished,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := h.pages.Insert(t.Context(), stray); err != nil {
		t.Fatalf("insert the product page: %v", err)
	}

	h.clock.Advance(time.Minute)
	state := h.all(t)
	if state.Archived != 0 {
		t.Fatalf("state = %+v, want a core pull to leave a product alone", state)
	}
	if kept := h.byPath(t, "/shop/grinder/"); kept.Status != pagemap.StatusPublished {
		t.Fatalf("the product page is %+v", kept)
	}
}

func TestSyncSiteArchivesNothingUntilThePullIsComplete(t *testing.T) {
	t.Parallel()

	h := newSyncHarness(t, 1)
	seedSite(t, h)
	h.all(t)

	gone := h.byPath(t, "/coffee/filter/")
	if !h.server.Delete(*gone.WPID) {
		t.Fatal("the filter page could not be removed from the site")
	}

	h.clock.Advance(time.Minute)
	h.restart(t)

	for range 20 {
		state, _ := h.once(t)
		if !state.Done {
			if state.Archived != 0 {
				t.Fatalf("a partial pull archived %d pages: %+v", state.Archived, state)
			}
			if page := h.byPath(t, "/coffee/filter/"); page.Status == pagemap.StatusArchived {
				t.Fatal("a partial pull archived the page that had not been paged through yet")
			}
			continue
		}
		if state.Archived != 1 {
			t.Fatalf("the completed pull archived %d pages, want 1", state.Archived)
		}
		return
	}
	t.Fatal("the sync never finished")
}
