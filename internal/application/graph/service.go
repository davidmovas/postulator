package graph

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type entityStore interface {
	Insert(ctx context.Context, e graphdomain.Entity) error
	Update(ctx context.Context, e graphdomain.Entity) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (graphdomain.Entity, error)
	List(ctx context.Context, q graphdomain.EntityQuery, page paging.Request) (paging.List[graphdomain.Entity], error)
	ListBySite(ctx context.Context, siteID string) ([]graphdomain.Entity, error)
	SetScore(ctx context.Context, id string, score float64) error
}

type edgeStore interface {
	Insert(ctx context.Context, e graphdomain.Edge) error
	Get(ctx context.Context, id string) (graphdomain.Edge, error)
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, status graphdomain.EdgeStatus) error
	List(ctx context.Context, q graphdomain.EdgeQuery, page paging.Request) (paging.List[graphdomain.Edge], error)
	ListBySite(ctx context.Context, siteID string) ([]graphdomain.Edge, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	entities  entityStore
	edges     edgeStore
	sites     siteReader
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
}

func New(entities entityStore, edges edgeStore, sites siteReader, uow unitOfWork, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{entities: entities, edges: edges, sites: sites, uow: uow, publisher: publisher, clock: clk}
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) changed(siteID string) error {
	return s.publisher.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: siteID})
}

func (s *Service) pagesChanged(siteID string) error {
	return s.publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}

func requireSite(siteID string) error {
	if siteID == "" {
		return errors.New(errors.Invalid, "site id must not be empty").WithDetail("field", "siteId")
	}
	return nil
}
