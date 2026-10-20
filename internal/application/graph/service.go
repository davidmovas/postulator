package graph

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
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
	SetScope(ctx context.Context, id string, scopeID *string, updatedAt time.Time) error
	SetCanonicalPage(ctx context.Context, id string, pageID *string, updatedAt time.Time) error
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

type pageStore interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	Update(ctx context.Context, p pagemap.Page) error
}

type workReader interface {
	ActiveBySite(ctx context.Context, siteID string) ([]run.Item, error)
}

type identityReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Identity, error)
}

type fieldReader interface {
	ListBySite(ctx context.Context, siteID string) (map[string][]field.Value, error)
}

type profileResolver interface {
	Resolve(ctx context.Context, siteID string, role domainllm.Role, templateProfiles map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type findingStore interface {
	Upsert(ctx context.Context, f finding.Finding) error
	Get(ctx context.Context, id string) (finding.Finding, error)
	Settle(ctx context.Context, id string, status finding.Status, resolution, resolvedBy string, at time.Time) error
	OpenBySite(ctx context.Context, siteID string, code finding.Code) ([]finding.Finding, error)
}

type Deps struct {
	Entities   entityStore
	Edges      edgeStore
	Sites      siteReader
	Pages      pageStore
	Identities identityReader
	Fields     fieldReader
	Findings   findingStore
	Work       workReader
	Profiles   profileResolver
	LLM        llm.Client
	UnitOfWork unitOfWork
	Publisher  application.Publisher
	Clock      clock.Clock
}

type Service struct {
	entities   entityStore
	edges      edgeStore
	sites      siteReader
	pages      pageStore
	identities identityReader
	fields     fieldReader
	work       workReader
	findings   findingStore
	profiles   profileResolver
	llm        llm.Client
	uow        unitOfWork
	publisher  application.Publisher
	clock      clock.Clock
}

func New(deps Deps) *Service {
	return &Service{
		entities: deps.Entities, edges: deps.Edges, sites: deps.Sites, pages: deps.Pages, identities: deps.Identities, fields: deps.Fields,
		work: deps.Work, findings: deps.Findings,
		profiles: deps.Profiles, llm: deps.LLM, uow: deps.UnitOfWork, publisher: deps.Publisher,
		clock: deps.Clock,
	}
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
