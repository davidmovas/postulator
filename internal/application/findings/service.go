package findings

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type findingStore interface {
	Get(ctx context.Context, id string) (finding.Finding, error)
	Settle(ctx context.Context, id string, status finding.Status, resolution, resolvedBy string, at time.Time) error
	List(ctx context.Context, q finding.Query, page paging.Request) (paging.List[finding.Finding], error)
	CountByCode(ctx context.Context, q finding.Query) (map[finding.Code]int, error)
}

type fieldAcceptor interface {
	AcceptSite(ctx context.Context, pageID string, key field.Key, at time.Time) error
}

type bodyAcceptor interface {
	AcceptSite(ctx context.Context, pageID string, at time.Time) error
}

type pageStore interface {
	Get(ctx context.Context, id string) (pagemap.Page, error)
	Update(ctx context.Context, p pagemap.Page) error
	SetType(ctx context.Context, pageID, typeID string, reason pagemap.TypeReason, at time.Time) error
	SetParentSource(ctx context.Context, pageID string, source pagemap.ParentSource, at time.Time) error
}

type cardAcceptor interface {
	AcceptCard(ctx context.Context, req graph.AcceptCardRequest) (graph.AcceptCardResponse, error)
}

type rowMapper interface {
	MapRow(ctx context.Context, req imports.MapRowRequest) (imports.MapRowResponse, error)
}

type parentAdopter interface {
	AdoptParents(ctx context.Context, req graph.AdoptParentsRequest) (graph.AdoptParentsResponse, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type sitePlaces interface {
	Refresh(ctx context.Context, siteID string) error
	Scope(ctx context.Context, siteID, pageID string) (string, error)
	Places(ctx context.Context, siteID string, pageIDs []string) (map[string]contextpack.Place, error)
	PageType(ctx context.Context, siteID, typeID string) (pagetype.Type, bool, error)
}

type Deps struct {
	Findings   findingStore
	Fields     fieldAcceptor
	Bodies     bodyAcceptor
	Pages      pageStore
	Cards      cardAcceptor
	Rows       rowMapper
	Parents    parentAdopter
	Places     sitePlaces
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
