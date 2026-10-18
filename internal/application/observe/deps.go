package observe

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

type siteStore interface {
	Get(ctx context.Context, id string) (site.Site, error)
	UpdateObservation(ctx context.Context, id string, technical site.TechnicalPages, observation site.ObservationState, at time.Time) error
}

type pageStore interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	Get(ctx context.Context, id string) (pagemap.Page, error)
	Insert(ctx context.Context, page pagemap.Page) error
	Update(ctx context.Context, page pagemap.Page) error
	UpdateDerived(ctx context.Context, page pagemap.Page) error
}

type identityStore interface {
	Put(ctx context.Context, identity pagemap.Identity) error
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Identity, error)
	ReplaceTerms(ctx context.Context, pageID string, terms []pagemap.Term) error
	ListTerms(ctx context.Context, pageID string) ([]pagemap.Term, error)
}

type fieldStore interface {
	Observe(ctx context.Context, pageID string, values []field.Value) error
	Agree(ctx context.Context, pageID string, key field.Key, at time.Time) error
	ListForPage(ctx context.Context, pageID string) ([]field.Value, error)
	ListBySite(ctx context.Context, siteID string) (map[string][]field.Value, error)
}

type bodyStore interface {
	Observe(ctx context.Context, snapshot block.Snapshot) error
	Get(ctx context.Context, pageID string) (current, base block.Snapshot, found bool, err error)
	ListBySite(ctx context.Context, siteID string) ([]block.Snapshot, error)
	ListWritten(ctx context.Context, pageID string) ([]block.Written, error)
}

type renderStore interface {
	Put(ctx context.Context, record render.Record) error
	Get(ctx context.Context, pageID string) (render.Record, error)
	ListBySite(ctx context.Context, siteID string) ([]render.Record, error)
}

type linkStore interface {
	ReplaceForPage(ctx context.Context, pageID string, source pagemap.LinkSource, links []pagemap.PageLink) error
	ListForPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error)
	ListBySite(ctx context.Context, siteID string) ([]pagemap.PageLink, error)
}

type findingStore interface {
	Upsert(ctx context.Context, record finding.Finding) error
	Clear(ctx context.Context, siteID string, code finding.Code, subjectKey, resolvedBy, resolution string, at time.Time) error
	OpenBySite(ctx context.Context, siteID string, code finding.Code) ([]finding.Finding, error)
	ListBySite(ctx context.Context, siteID string, status finding.Status) ([]finding.Finding, error)
	ListForPage(ctx context.Context, pageID string) ([]finding.Finding, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Deps struct {
	Sites      siteStore
	Pages      pageStore
	Identity   identityStore
	Fields     fieldStore
	Bodies     bodyStore
	Renders    renderStore
	Links      linkStore
	Findings   findingStore
	UnitOfWork unitOfWork
	Publisher  application.Publisher
	Clock      clock.Clock
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	return &Service{deps: deps}
}

func (s *Service) now() time.Time {
	return s.deps.Clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) inUnit(ctx context.Context, fn func(context.Context) error) error {
	if s.deps.UnitOfWork == nil {
		return fn(ctx)
	}
	return s.deps.UnitOfWork.Do(ctx, fn)
}
