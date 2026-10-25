package pages

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
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

type linkReader interface {
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

type fieldReader interface {
	ListForPages(ctx context.Context, pageIDs []string) (map[string][]field.Value, error)
}

type renderReader interface {
	ListForPages(ctx context.Context, pageIDs []string) ([]render.Record, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type findingClearer interface {
	Clear(ctx context.Context, siteID string, code finding.Code, subjectKey, resolvedBy, resolution string, at time.Time) error
}

type Deps struct {
	Pages      pageStore
	Types      typeReader
	Links      linkReader
	Entities   entityStore
	Edges      edgeReader
	Sites      siteReader
	Bodies     bodyReader
	Fields     fieldReader
	Renders    renderReader
	Findings   findingClearer
	UnitOfWork unitOfWork
	Publisher  application.Publisher
	Clock      clock.Clock
	Preview    previewIssuer
}

type Service struct {
	pages     pageStore
	types     typeReader
	links     linkReader
	entities  entityStore
	edges     edgeReader
	sites     siteReader
	bodies    bodyReader
	fields    fieldReader
	renders   renderReader
	findings  findingClearer
	uow       unitOfWork
	publisher application.Publisher
	clock     clock.Clock
	preview   previewIssuer
}

func New(deps Deps) *Service {
	return &Service{
		pages: deps.Pages, types: deps.Types, links: deps.Links, entities: deps.Entities, edges: deps.Edges, sites: deps.Sites,
		bodies: deps.Bodies, fields: deps.Fields, renders: deps.Renders, findings: deps.Findings, uow: deps.UnitOfWork, publisher: deps.Publisher, clock: deps.Clock, preview: deps.Preview,
	}
}

func (s *Service) carried(ctx context.Context, page pagemap.Page) error {
	if s.findings == nil {
		return nil
	}
	return s.findings.Clear(ctx, page.SiteID, finding.PageWithoutEntity, finding.SubjectPage(page.ID),
		finding.ResolvedByModel, finding.ResolutionPageMapped, s.now())
}

type observedH1 struct {
	text   string
	source render.H1Source
}

type annotations struct {
	drifted map[string]bool
	h1s     map[string]observedH1
}

func (a annotations) h1(pageID string) observedH1 {
	if seen, ok := a.h1s[pageID]; ok {
		return seen
	}
	return observedH1{source: render.H1None}
}

func (s *Service) annotate(ctx context.Context, siteID string, pages []pagemap.Page) (annotations, error) {
	drifted, err := s.bodies.Drifted(ctx, siteID)
	if err != nil {
		return annotations{}, err
	}
	out := annotations{drifted: make(map[string]bool, len(drifted)), h1s: make(map[string]observedH1, len(pages))}
	for _, pageID := range drifted {
		out.drifted[pageID] = true
	}
	ids := make([]string, 0, len(pages))
	for i := range pages {
		ids = append(ids, pages[i].ID)
	}
	if len(ids) == 0 {
		return out, nil
	}
	stored, err := s.fields.ListForPages(ctx, ids)
	if err != nil {
		return annotations{}, err
	}
	records, err := s.renders.ListForPages(ctx, ids)
	if err != nil {
		return annotations{}, err
	}
	byPage := make(map[string]*render.Record, len(records))
	for i := range records {
		byPage[records[i].PageID] = &records[i]
	}
	for _, pageID := range ids {
		content := ""
		for _, value := range stored[pageID] {
			if value.Key == field.H1 {
				content = value.Current
			}
		}
		text, source := render.VisibleH1(byPage[pageID], content)
		out.h1s[pageID] = observedH1{text: text, source: source}
	}
	return out, nil
}

func (s *Service) viewOf(ctx context.Context, page pagemap.Page) (Page, error) {
	shown, err := s.annotate(ctx, page.SiteID, []pagemap.Page{page})
	if err != nil {
		return Page{}, err
	}
	return view(page, shown), nil
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
