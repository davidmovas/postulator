package sync_test

import (
	"context"
	stderrors "errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/registry"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const siteID = "3f4bb0e4-9b4f-4f2a-8a91-6b8f0a4f1c22"

type queue struct {
	queued run.Run
	err    error
}

func (q *queue) Enqueue(_ context.Context, record run.Run, _ ...run.Directive) (run.Run, error) {
	if q.err != nil {
		return run.Run{}, q.err
	}
	q.queued = record
	return record, nil
}

type sites struct {
	record  site.Site
	written []site.Site
	getErr  error
	setErr  error
}

func (s *sites) Get(context.Context, string) (site.Site, error) {
	if s.getErr != nil {
		return site.Site{}, s.getErr
	}
	return s.record, nil
}

func (s *sites) Update(_ context.Context, record site.Site) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.written = append(s.written, record)
	return nil
}

type prober struct {
	state    site.PluginState
	err      error
	commerce site.Commerce
	required string
	outdated bool
}

func (p prober) Probe(context.Context, site.Site) (site.Extensions, error) {
	return site.Extensions{Plugin: p.state, Commerce: p.commerce, PluginRequired: p.required, PluginOutdated: p.outdated}, p.err
}

type packer struct {
	archive []byte
	err     error
}

func (p packer) Package() ([]byte, error) {
	return p.archive, p.err
}

type underway struct {
	active []run.Run
	asked  int
	runs   map[string]run.Run
	items  map[string][]run.Item
}

func (u *underway) Get(_ context.Context, runID string) (run.Run, error) {
	record, held := u.runs[runID]
	if !held {
		return run.Run{}, errors.New(errors.NotFound, "no such run")
	}
	return record, nil
}

func (u *underway) ByRun(_ context.Context, runID string) ([]run.Item, error) {
	return u.items[runID], nil
}

func (u *underway) ActiveSync(context.Context, string) (run.Run, bool, error) {
	u.asked++
	if len(u.active) == 0 {
		return run.Run{}, false, nil
	}
	next := u.active[0]
	if len(u.active) > 1 {
		u.active = u.active[1:]
	}
	if next.ID == "" {
		return run.Run{}, false, nil
	}
	return next, true, nil
}

func newService(store *sites, q *queue, probe prober, pack packer) *sync.Service {
	held := &underway{}
	return sync.New(q, held, held, store, probe, pack, clock.NewFake(sqlitetest.Stamp))
}

func TestTheProgressOfASyncSaysItsStageCountsAndTrouble(t *testing.T) {
	t.Parallel()

	wake := sqlitetest.Stamp.Add(2 * time.Second)
	checkpoint := run.NewCheckpoint()
	if err := run.Set(checkpoint, observe.CheckpointKey, observe.Progress{
		State: observe.State{
			StartedAt: sqlitetest.Stamp, Source: observe.SourcePlugin, Stage: observe.StageRender, Full: true,
			Counts: observe.Counts{
				Pulled: 412, Created: 400, Updated: 12, Rendered: 150, Unreachable: 2,
				Entities: 30, Placed: 27, Unplaced: 4, UnplacedBy: map[string]int{"unknown_link": 3, "disagrees": 1},
			},
		},
		Batches: 11,
	}); err != nil {
		t.Fatalf("set the checkpoint: %v", err)
	}
	active := run.Run{ID: "run-active", SiteID: siteID, Kind: run.KindSync, Status: run.StatusWaiting, CreatedAt: sqlitetest.Stamp}
	done := run.Run{ID: "run-done", SiteID: siteID, Kind: run.KindSync, Status: run.StatusFailed, Error: "the step ran past its time limit"}
	other := run.Run{ID: "run-generate", SiteID: siteID, Kind: run.KindGenerate, Status: run.StatusRunning}
	held := &underway{
		active: []run.Run{active},
		runs:   map[string]run.Run{active.ID: active, done.ID: done, other.ID: other},
		items: map[string][]run.Item{active.ID: {{
			RunID: active.ID, Status: run.StatusWaiting, Attempts: 1, WakeAt: &wake, Checkpoint: checkpoint,
			Error: "reading the body of /about/: the WordPress site failed on its side: WordPress answered 500 on GET /x",
		}}},
	}
	service := sync.New(&queue{}, held, held, newSites(), prober{}, packer{}, clock.NewFake(sqlitetest.Stamp))

	got, err := service.Progress(t.Context(), sync.ProgressRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("Progress: %v", err)
	}
	if !got.Found || got.RunID != active.ID || got.Status != "waiting" || got.Stage != "render" || !got.Full || got.Source != "plugin" ||
		got.Batches != 11 || got.Pulled != 412 || got.Created != 400 || got.Rendered != 150 || got.Unreachable != 2 ||
		got.Attempts != 1 || got.WakeAt == nil || !strings.Contains(got.Error, "WordPress answered 500") {
		t.Fatalf("the active sync's progress = %+v", got)
	}
	if got.Entities != 30 || got.Placed != 27 || got.Unplaced != 4 || got.UnplacedBy["unknown_link"] != 3 || got.UnplacedBy["disagrees"] != 1 {
		t.Errorf("the topical map's counts = %d %d %d %v", got.Entities, got.Placed, got.Unplaced, got.UnplacedBy)
	}
	if named, namedErr := service.Progress(t.Context(), sync.ProgressRequest{SiteID: siteID, RunID: done.ID}); namedErr != nil || named.UnplacedBy == nil {
		t.Errorf("a sync without a checkpoint says its reasons as %v (%v), want an empty map", named.UnplacedBy, namedErr)
	}

	named, err := service.Progress(t.Context(), sync.ProgressRequest{SiteID: siteID, RunID: done.ID})
	if err != nil || !named.Found || named.Status != "failed" || named.Stage != "" || named.Error != done.Error {
		t.Fatalf("a finished sync's progress = %+v %v, want its status and the run's error", named, err)
	}
	if _, err = service.Progress(t.Context(), sync.ProgressRequest{SiteID: siteID, RunID: other.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("the progress of a generate run = %v, want it refused", err)
	}

	idle := sync.New(&queue{}, &underway{}, &underway{}, newSites(), prober{}, packer{}, clock.NewFake(sqlitetest.Stamp))
	if none, idleErr := idle.Progress(t.Context(), sync.ProgressRequest{SiteID: siteID}); idleErr != nil || none.Found {
		t.Fatalf("the progress of a site with no sync = %+v %v, want nothing found", none, idleErr)
	}
}

func TestASecondSyncOfASiteIsRefusedAndNamesTheOneUnderWay(t *testing.T) {
	t.Parallel()

	running := run.Run{ID: "run-under-way", SiteID: siteID, Kind: run.KindSync, Status: run.StatusRunning}
	cases := []struct {
		name   string
		active []run.Run
		queue  *queue
	}{
		{name: "one already under way", active: []run.Run{running}, queue: &queue{}},
		{
			name: "one that won the race to the index", active: []run.Run{{}, running},
			queue: &queue{err: errors.New(errors.Conflict, "insert the run")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			held := &underway{active: tc.active}
			service := sync.New(tc.queue, held, held, newSites(), prober{}, packer{}, clock.NewFake(sqlitetest.Stamp))
			_, err := service.SyncSite(t.Context(), sync.SyncSiteRequest{SiteID: siteID})

			var kernel *errors.Error
			if !stderrors.As(err, &kernel) || kernel.Code != errors.Conflict || kernel.Details["runId"] != running.ID {
				t.Fatalf("err = %v, want a conflict naming %s", err, running.ID)
			}
			if kernel.Message != "a sync of Shop is already under way; it reads every page, so a second one would only repeat it" {
				t.Errorf("message = %q", kernel.Message)
			}
			if tc.queue.queued.ID != "" {
				t.Errorf("a second sync was queued: %+v", tc.queue.queued)
			}
		})
	}
}

func newSites() *sites {
	record := site.Site{
		ID: siteID, Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor",
		Status: site.StatusActive, Plugin: site.PluginState{Capabilities: []string{}},
	}
	record.SecretRef = site.SecretRef(record.ID)
	return &sites{record: record}
}

func TestSyncSiteQueuesASiteScopedRun(t *testing.T) {
	t.Parallel()

	q := &queue{}
	service := newService(newSites(), q, prober{}, packer{})

	resp, err := service.SyncSite(kctx.WithActor(t.Context(), kctx.ActorAgent), sync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("SyncSite: %v", err)
	}
	if resp.RunID != q.queued.ID || resp.RunID == "" {
		t.Fatalf("response = %+v, queued = %+v", resp, q.queued)
	}
	if q.queued.Kind != run.KindSync || len(q.queued.Targets) != 1 || q.queued.Targets[0] != siteID {
		t.Fatalf("queued = %+v", q.queued)
	}
	if len(q.queued.Recipe) != 1 || q.queued.Recipe[0].Name != string(run.StepSyncSite) || !q.queued.Recipe[0].Enabled {
		t.Fatalf("recipe = %+v", q.queued.Recipe)
	}
	if q.queued.CreatedBy != kctx.ActorAgent {
		t.Errorf("createdBy = %q", q.queued.CreatedBy)
	}
}

func TestSyncSiteCarriesTheWalkOptionsToTheStep(t *testing.T) {
	t.Parallel()

	q := &queue{}
	service := newService(newSites(), q, prober{}, packer{})

	if _, err := service.SyncSite(t.Context(), sync.SyncSiteRequest{SiteID: siteID}); err != nil {
		t.Fatalf("SyncSite: %v", err)
	}
	if q.queued.Recipe[0].Params != nil {
		t.Fatalf("a plain sync carries params: %+v", q.queued.Recipe[0].Params)
	}

	if _, err := service.SyncSite(t.Context(), sync.SyncSiteRequest{SiteID: siteID, Full: true, RenderRescan: true}); err != nil {
		t.Fatalf("SyncSite: %v", err)
	}
	params := q.queued.Recipe[0].Params
	if params[run.ParamFullWalk] != true || params[run.ParamRenderRescan] != true {
		t.Fatalf("the step params = %+v, want the full walk and the render rescan asked for", params)
	}
}

func TestSyncSiteReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		id    string
		store func() *sites
		queue *queue
		want  errors.Code
	}{
		{name: "no site", id: "  ", store: newSites, queue: &queue{}, want: errors.Invalid},
		{
			name: "the site is gone", id: siteID, queue: &queue{},
			store: func() *sites { s := newSites(); s.getErr = errors.New(errors.NotFound, "gone"); return s },
			want:  errors.NotFound,
		},
		{
			name: "the engine refuses", id: siteID, store: newSites,
			queue: &queue{err: errors.New(errors.Conflict, "already running")}, want: errors.Conflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := newService(tc.store(), tc.queue, prober{}, packer{})
			if _, err := service.SyncSite(t.Context(), sync.SyncSiteRequest{SiteID: tc.id}); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}

func TestCheckPluginStoresWhatItFound(t *testing.T) {
	t.Parallel()

	store := newSites()
	service := newService(store, &queue{}, prober{state: site.PluginState{
		Installed: true, Version: "1.0.0", Capabilities: []string{"bulk", "seo_meta"}, SEOPlugin: "yoast",
	}, commerce: site.CommerceReady}, packer{})

	resp, err := service.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CheckPlugin: %v", err)
	}
	if !resp.Plugin.Installed || resp.Plugin.Version != "1.0.0" || resp.Plugin.SEOPlugin != "yoast" {
		t.Fatalf("plugin = %+v", resp.Plugin)
	}
	if resp.Commerce != string(site.CommerceReady) {
		t.Errorf("commerce = %q, want ready", resp.Commerce)
	}
	if len(store.written) != 1 || !store.written[0].Plugin.Installed || store.written[0].Commerce != site.CommerceReady {
		t.Fatalf("the site was written as %+v", store.written)
	}
}

func TestCheckPluginSaysThePluginIsOutdated(t *testing.T) {
	t.Parallel()

	store := newSites()
	service := newService(store, &queue{}, prober{state: site.PluginState{
		Installed: true, Version: "1.2.0", Capabilities: []string{"bulk"}, SEOPlugin: "none",
	}, required: "1.4.0", outdated: true}, packer{})

	resp, err := service.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CheckPlugin: %v", err)
	}
	if !resp.Plugin.Outdated || resp.Plugin.RequiredVersion != "1.4.0" || resp.Plugin.Version != "1.2.0" {
		t.Errorf("plugin = %+v, want version 1.2.0 outdated against 1.4.0", resp.Plugin)
	}
}

func TestCheckPluginRecordsAnAbsentPlugin(t *testing.T) {
	t.Parallel()

	store := newSites()
	service := newService(store, &queue{}, prober{}, packer{})

	resp, err := service.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CheckPlugin: %v", err)
	}
	if resp.Plugin.Installed || resp.Plugin.Capabilities == nil || len(resp.Plugin.Capabilities) != 0 {
		t.Fatalf("plugin = %+v", resp.Plugin)
	}
}

func TestCheckPluginReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		id      string
		probe   prober
		setErr  error
		wantErr errors.Code
	}{
		{name: "no site", id: " ", wantErr: errors.Invalid},
		{name: "the probe fails", id: siteID, probe: prober{err: errors.New(errors.External, "down")}, wantErr: errors.External},
		{name: "the write fails", id: siteID, setErr: errors.New(errors.Conflict, "busy"), wantErr: errors.Conflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := newSites()
			store.setErr = tc.setErr
			service := newService(store, &queue{}, tc.probe, packer{})
			if _, err := service.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: tc.id}); !errors.IsCode(err, tc.wantErr) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.wantErr, err)
			}
		})
	}
}

func TestPluginPackageHandsBackTheArchive(t *testing.T) {
	t.Parallel()

	service := newService(newSites(), &queue{}, prober{}, packer{archive: []byte("PK\x03\x04")})

	resp, err := service.PluginPackage(t.Context(), sync.PluginPackageRequest{})
	if err != nil {
		t.Fatalf("PluginPackage: %v", err)
	}
	if resp.Filename != sync.Filename || string(resp.Bytes) != "PK\x03\x04" {
		t.Fatalf("response = %+v", resp)
	}

	broken := newService(newSites(), &queue{}, prober{}, packer{err: errors.New(errors.Internal, "no tree")})
	if _, err = broken.PluginPackage(t.Context(), sync.PluginPackageRequest{}); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Internal)
	}
}

type vault struct {
	password string
}

func (v vault) Get(context.Context, string) (string, error) {
	return v.password, nil
}

func TestCheckPluginReprobesASiteThatGainedThePlugin(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	store := newSites()
	store.record.BaseURL = server.URL()
	store.record.Username = wptest.DefaultUser
	store.record.AllowInsecure = true

	held := registry.New(store, vault{password: wptest.DefaultPassword}, wp.WithRateLimit(0))
	service := sync.New(&queue{}, &underway{}, &underway{}, store, held, packer{}, clock.NewFake(sqlitetest.Stamp))

	absent, err := service.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CheckPlugin without the plugin: %v", err)
	}
	if absent.Plugin.Installed || len(absent.Plugin.Capabilities) != 0 {
		t.Fatalf("plugin = %+v, want an absent plugin and no capabilities", absent.Plugin)
	}

	server.EnablePlugin()

	present, err := service.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CheckPlugin after the plugin was installed: %v", err)
	}
	if !present.Plugin.Installed || present.Plugin.Version != wp.PluginVersion {
		t.Fatalf("plugin = %+v, want the companion reported as installed", present.Plugin)
	}
	if present.Plugin.Outdated || present.Plugin.RequiredVersion != wp.PluginVersion {
		t.Errorf("plugin = %+v, want a current plugin measured against %s", present.Plugin, wp.PluginVersion)
	}
	if !slices.Contains(present.Plugin.Capabilities, "bulk") ||
		!slices.Contains(present.Plugin.Capabilities, "seo_meta") {
		t.Errorf("capabilities = %v, want the companion capabilities", present.Plugin.Capabilities)
	}
}
