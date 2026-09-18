package sync

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type engine interface {
	Enqueue(ctx context.Context, record run.Run) (run.Run, error)
}

type siteStore interface {
	Get(ctx context.Context, id string) (site.Site, error)
	Update(ctx context.Context, record site.Site) error
}

type pluginProbe interface {
	Probe(ctx context.Context, record site.Site) (site.PluginState, error)
}

type packager interface {
	Package() ([]byte, error)
}

type Service struct {
	engine engine
	sites  siteStore
	probe  pluginProbe
	packer packager
	clock  clock.Clock
}

func New(engine engine, sites siteStore, probe pluginProbe, packed packager, clk clock.Clock) *Service {
	return &Service{engine: engine, sites: sites, probe: probe, packer: packed, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func (s *Service) SyncSite(ctx context.Context, req SyncSiteRequest) (SyncSiteResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return SyncSiteResponse{}, invalid("a sync needs a site", "siteId")
	}

	record, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return SyncSiteResponse{}, err
	}

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}

	queued, err := s.engine.Enqueue(ctx, run.Run{
		ID:          id.New(),
		SiteID:      record.ID,
		Kind:        run.KindSync,
		Targets:     []string{record.ID},
		Recipe:      run.SyncRecipe(),
		PublishMode: run.PublishDraft,
		CreatedBy:   actor,
		CreatedAt:   s.now(),
	})
	if err != nil {
		return SyncSiteResponse{}, err
	}
	return SyncSiteResponse{RunID: queued.ID}, nil
}

func (s *Service) CheckPlugin(ctx context.Context, req CheckPluginRequest) (CheckPluginResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return CheckPluginResponse{}, invalid("a plugin check needs a site", "siteId")
	}

	record, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return CheckPluginResponse{}, err
	}

	state, err := s.probe.Probe(ctx, record)
	if err != nil {
		return CheckPluginResponse{}, err
	}
	if state.Capabilities == nil {
		state.Capabilities = []string{}
	}

	next := record
	next.Plugin = state
	next.UpdatedAt = s.now()
	if updateErr := s.sites.Update(ctx, next); updateErr != nil {
		return CheckPluginResponse{}, updateErr
	}
	return CheckPluginResponse{Plugin: pluginView(state)}, nil
}

func (s *Service) PluginPackage(_ context.Context, _ PluginPackageRequest) (PluginPackageResponse, error) {
	archive, err := s.packer.Package()
	if err != nil {
		return PluginPackageResponse{}, err
	}
	return PluginPackageResponse{Filename: Filename, Bytes: archive}, nil
}
