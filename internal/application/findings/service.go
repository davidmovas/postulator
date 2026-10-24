package findings

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

type findingStore interface {
	Get(ctx context.Context, id string) (finding.Finding, error)
	Settle(ctx context.Context, id string, status finding.Status, resolution, resolvedBy string, at time.Time) error
	ListBySite(ctx context.Context, siteID string, status finding.Status) ([]finding.Finding, error)
}

type fieldAcceptor interface {
	AcceptSite(ctx context.Context, pageID string, key field.Key, at time.Time) error
}

type bodyAcceptor interface {
	AcceptSite(ctx context.Context, pageID string, at time.Time) error
}

type parentSources interface {
	SetParentSource(ctx context.Context, pageID string, source pagemap.ParentSource, at time.Time) error
}

type pageEditor interface {
	Update(ctx context.Context, req pages.UpdateRequest) (pages.UpdateResponse, error)
	SetType(ctx context.Context, req pages.SetTypeRequest) (pages.SetTypeResponse, error)
}

type cardAcceptor interface {
	AcceptCard(ctx context.Context, req graph.AcceptCardRequest) (graph.AcceptCardResponse, error)
}

type rowMapper interface {
	MapRow(ctx context.Context, req imports.MapRowRequest) (imports.MapRowResponse, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type siteMaps interface {
	Map(ctx context.Context, siteID string) (contextpack.Map, error)
}

type Deps struct {
	Findings   findingStore
	Fields     fieldAcceptor
	Bodies     bodyAcceptor
	Parents    parentSources
	Pages      pageEditor
	Cards      cardAcceptor
	Rows       rowMapper
	Places     siteMaps
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

func (s *Service) changed(siteID string) error {
	if s.deps.Publisher == nil {
		return nil
	}
	return s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}
