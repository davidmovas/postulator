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
	Enqueue(ctx context.Context, record run.Run, directives ...run.Directive) (run.Run, error)
}

type syncRuns interface {
	ActiveSync(ctx context.Context, siteID string) (run.Run, bool, error)
	Get(ctx context.Context, id string) (run.Run, error)
}

type runItems interface {
	ByRun(ctx context.Context, runID string) ([]run.Item, error)
}

type siteStore interface {
	Get(ctx context.Context, id string) (site.Site, error)
	Update(ctx context.Context, record site.Site) error
}

type pluginProbe interface {
	Probe(ctx context.Context, record site.Site) (site.Extensions, error)
}

type packager interface {
	Package() ([]byte, error)
}

type Service struct {
	engine engine
	syncs  syncRuns
	items  runItems
	sites  siteStore
	probe  pluginProbe
	packer packager
	clock  clock.Clock
}

func New(engine engine, syncs syncRuns, items runItems, sites siteStore, probe pluginProbe, packed packager, clk clock.Clock) *Service {
	return &Service{engine: engine, syncs: syncs, items: items, sites: sites, probe: probe, packer: packed, clock: clk}
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
	if busy := s.underway(ctx, record); busy != nil {
		return SyncSiteResponse{}, busy
	}

	actor, ok := kctx.ActorFrom(ctx)
	if !ok {
		actor = kctx.ActorUser
	}

	recipe := run.SyncRecipe()
	params := map[string]any{}
	if req.Full {
		params[run.ParamFullWalk] = true
	}
	if req.RenderRescan {
		params[run.ParamRenderRescan] = true
	}
	if len(params) > 0 {
		recipe[0].Params = params
	}

	queued, err := s.engine.Enqueue(ctx, run.Run{
		ID:          id.New(),
		SiteID:      record.ID,
		Kind:        run.KindSync,
		Targets:     []string{record.ID},
		Recipe:      recipe,
		PublishMode: run.PublishDraft,
		CreatedBy:   actor,
		CreatedAt:   s.now(),
	})
	if errors.IsCode(err, errors.Conflict) {
		if busy := s.underway(ctx, record); busy != nil {
			return SyncSiteResponse{}, busy
		}
	}
	if err != nil {
		return SyncSiteResponse{}, err
	}
	return SyncSiteResponse{RunID: queued.ID}, nil
}

func (s *Service) underway(ctx context.Context, record site.Site) error {
	active, found, err := s.syncs.ActiveSync(ctx, record.ID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	return errors.New(errors.Conflict, "a sync of "+record.Name+" is already under way; it reads every page, so a second one would only repeat it").
		WithDetail("runId", active.ID)
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

	found, err := s.probe.Probe(ctx, record)
	if err != nil {
		return CheckPluginResponse{}, err
	}
	if found.Plugin.Capabilities == nil {
		found.Plugin.Capabilities = []string{}
	}

	next := record
	next.Plugin = found.Plugin
	next.Commerce = found.Commerce
	next.UpdatedAt = s.now()
	if updateErr := s.sites.Update(ctx, next); updateErr != nil {
		return CheckPluginResponse{}, updateErr
	}
	return CheckPluginResponse{Plugin: pluginView(found), Commerce: string(found.Commerce)}, nil
}

func (s *Service) PluginPackage(_ context.Context, _ PluginPackageRequest) (PluginPackageResponse, error) {
	archive, err := s.packer.Package()
	if err != nil {
		return PluginPackageResponse{}, err
	}
	return PluginPackageResponse{Filename: Filename, Bytes: archive}, nil
}
