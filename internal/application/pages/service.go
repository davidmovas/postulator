package pages

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type pageStore interface {
	Insert(ctx context.Context, p pagemap.Page) error
	Update(ctx context.Context, p pagemap.Page) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (pagemap.Page, error)
	List(ctx context.Context, q pagemap.Query, page paging.Request) (paging.List[pagemap.Page], error)
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	SetType(ctx context.Context, pageID, typeID string, reason pagemap.TypeReason, at time.Time) error
}

type typeReader interface {
	Get(ctx context.Context, id string) (pagetype.Type, error)
}

type linkStore interface {
	ReplaceForPage(ctx context.Context, pageID string, source pagemap.LinkSource, links []pagemap.PageLink) error
	ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error)
}

type entityStore interface {
	Get(ctx context.Context, id string) (graph.Entity, error)
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
	SetCanonicalPage(ctx context.Context, id string, pageID *string, updatedAt time.Time) error
}

type edgeReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type bodyReader interface {
	Drifted(ctx context.Context, siteID string) ([]string, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Deps struct {
	Pages      pageStore
	Types      typeReader
	Links      linkStore
	Entities   entityStore
	Edges      edgeReader
	Sites      siteReader
	Bodies     bodyReader
	UnitOfWork unitOfWork
	Publisher  application.Publisher
	Clock      clock.Clock
	Preview    previewIssuer
}

type Service struct {
	pages     pageStore
	types     typeReader
	links     linkStore
	entities  entityStore
	edges     edgeReader
	sites     siteReader
	bodies    bodyReader
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
	preview   previewIssuer
}

func New(deps Deps) *Service {
	return &Service{
		pages: deps.Pages, types: deps.Types, links: deps.Links, entities: deps.Entities, edges: deps.Edges, sites: deps.Sites,
		bodies: deps.Bodies, uow: deps.UnitOfWork, publisher: deps.Publisher, clock: deps.Clock, preview: deps.Preview,
	}
}

func (s *Service) drifted(ctx context.Context, siteID string) (map[string]bool, error) {
	ids, err := s.bodies.Drifted(ctx, siteID)
	if err != nil {
		return nil, err
	}
	drifted := make(map[string]bool, len(ids))
	for _, pageID := range ids {
		drifted[pageID] = true
	}
	return drifted, nil
}

func (s *Service) viewOf(ctx context.Context, page pagemap.Page) (Page, error) {
	drifted, err := s.drifted(ctx, page.SiteID)
	if err != nil {
		return Page{}, err
	}
	return view(page, drifted[page.ID]), nil
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed(siteID string) error {
	return s.publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}

func (s *Service) graphChanged(siteID string) error {
	return s.publisher.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: siteID})
}

func requireSite(siteID string) error {
	if siteID == "" {
		return errors.New(errors.Invalid, "site id must not be empty").WithDetail("field", "siteId")
	}
	return nil
}

func (s *Service) entityFor(ctx context.Context, page *pagemap.Page) (graph.Entity, error) {
	if page.EntityID == nil {
		return graph.Entity{}, nil
	}
	entity, err := s.entities.Get(ctx, *page.EntityID)
	if err != nil {
		return graph.Entity{}, err
	}
	if entity.SiteID != page.SiteID {
		return graph.Entity{}, errors.New(errors.Invalid, "entity belongs to another site").WithDetail("entityId", entity.ID)
	}
	return entity, nil
}

func (s *Service) verdict(ctx context.Context, candidate pagemap.Page, index pagemap.Index, entity graph.Entity) error {
	g := graph.Graph{}
	if entity.ID != "" {
		entities, err := s.entities.ListBySite(ctx, candidate.SiteID)
		if err != nil {
			return err
		}
		built, err := graph.New(entities, nil)
		if err != nil {
			return err
		}
		g = built
	}

	result := pagemap.Cannibalization(candidate, entity, index, g)
	if result.Allowed {
		return nil
	}
	return errors.New(errors.Conflict, "page would cannibalize an existing page").WithDetail("evidence", conflicts(result.Evidence))
}
