package sync_test

import (
	"context"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/registry"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
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

func (q *queue) Enqueue(_ context.Context, record run.Run) (run.Run, error) {
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
	state site.PluginState
	err   error
}

func (p prober) Probe(context.Context, site.Site) (site.PluginState, error) {
	return p.state, p.err
}

type packer struct {
	archive []byte
	err     error
}

func (p packer) Package() ([]byte, error) {
	return p.archive, p.err
}

func newService(store *sites, q *queue, probe prober, pack packer) *sync.Service {
	return sync.New(q, store, probe, pack, clock.NewFake(sqlitetest.Stamp))
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
	}}, packer{})

	resp, err := service.CheckPlugin(t.Context(), sync.CheckPluginRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("CheckPlugin: %v", err)
	}
	if !resp.Plugin.Installed || resp.Plugin.Version != "1.0.0" || resp.Plugin.SEOPlugin != "yoast" {
		t.Fatalf("plugin = %+v", resp.Plugin)
	}
	if len(store.written) != 1 || !store.written[0].Plugin.Installed {
		t.Fatalf("the site was written as %+v", store.written)
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
	service := sync.New(&queue{}, store, held, packer{}, clock.NewFake(sqlitetest.Stamp))

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
	if !present.Plugin.Installed || present.Plugin.Version != "1.1.0" {
		t.Fatalf("plugin = %+v, want the companion reported as installed", present.Plugin)
	}
	if !slices.Contains(present.Plugin.Capabilities, "bulk") ||
		!slices.Contains(present.Plugin.Capabilities, "seo_meta") {
		t.Errorf("capabilities = %v, want the companion capabilities", present.Plugin.Capabilities)
	}
}
