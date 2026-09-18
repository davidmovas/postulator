package pages

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
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
}

type linkStore interface {
	ReplaceForPage(ctx context.Context, pageID string, links []pagemap.PageLink) error
	ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error)
}

type entityStore interface {
	Get(ctx context.Context, id string) (graph.Entity, error)
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
	SetCanonicalPage(ctx context.Context, id string, pageID *string, updatedAt time.Time) error
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	pages     pageStore
	links     linkStore
	entities  entityStore
	sites     siteReader
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
}

func New(pages pageStore, links linkStore, entities entityStore, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{pages: pages, links: links, entities: entities, sites: sites, uow: uow, publisher: publisher, clock: clk}
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
