package imports

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type tableStore interface {
	Read(ctx context.Context, path string) (importmap.Table, error)
	Write(path string, table importmap.Table) error
}

type entityStore interface {
	Insert(ctx context.Context, e graph.Entity) error
	Update(ctx context.Context, e graph.Entity) error
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
	SetCanonicalPage(ctx context.Context, id string, pageID *string, updatedAt time.Time) error
}

type edgeStore interface {
	Insert(ctx context.Context, e graph.Edge) error
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type pageStore interface {
	Insert(ctx context.Context, p pagemap.Page) error
	Update(ctx context.Context, p pagemap.Page) error
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
}

type templateStore interface {
	Get(ctx context.Context, id string) (template.Template, error)
	List(ctx context.Context, q template.Query, page paging.Request) (paging.List[template.Template], error)
}

type mappingStore interface {
	Upsert(ctx context.Context, m importmap.Mapping) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (importmap.Mapping, error)
	ListBySite(ctx context.Context, siteID string) ([]importmap.Mapping, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Deps struct {
	Tables     tableStore
	Entities   entityStore
	Edges      edgeStore
	Pages      pageStore
	Templates  templateStore
	Mappings   mappingStore
	Sites      siteReader
	UnitOfWork unitOfWork
	Publisher  application.Publisher
	Clock      clock.Clock
	MaxRows    int
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	if deps.MaxRows <= 0 {
		deps.MaxRows = DefaultMaxRows
	}
	return &Service{deps: deps}
}

func (s *Service) now() time.Time {
	return s.deps.Clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) requireSite(ctx context.Context, siteID string) error {
	if siteID == "" {
		return errors.New(errors.Invalid, "site id must not be empty").WithDetail("field", "siteId")
	}
	_, err := s.deps.Sites.Get(ctx, siteID)
	return err
}

func (s *Service) table(ctx context.Context, path string) (importmap.Table, error) {
	table, err := s.deps.Tables.Read(ctx, path)
	if err != nil {
		return importmap.Table{}, err
	}
	if len(table.Rows) > s.deps.MaxRows {
		return importmap.Table{}, errors.New(errors.Invalid, "the import file carries more rows than the import.maxRows setting allows").
			WithDetail("rows", len(table.Rows)).WithDetail("maxRows", s.deps.MaxRows)
	}
	return table, nil
}

func (s *Service) announce(siteID string) error {
	if err := s.deps.Publisher.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: siteID}); err != nil {
		return err
	}
	return s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}
