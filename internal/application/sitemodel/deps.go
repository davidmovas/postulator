package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type reader interface {
	Read(ctx context.Context, fn func(context.Context) error) error
}

type siteStore interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type pageStore interface {
	Get(ctx context.Context, id string) (pagemap.Page, error)
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	ListByIDs(ctx context.Context, ids []string) ([]pagemap.Page, error)
}

type entityStore interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeStore interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type typeSets interface {
	Set(ctx context.Context, siteID string) (pagetype.Set, error)
}

type termStore interface {
	ListTermsBySite(ctx context.Context, siteID string) (map[string][]pagemap.Term, error)
}

type fieldStore interface {
	ListBySite(ctx context.Context, siteID string) (map[string][]field.Value, error)
	ListNamesBySite(ctx context.Context, siteID string) (map[string][]field.Value, error)
}

type mapStore interface {
	State(ctx context.Context, siteID string) (generation, built int64, err error)
	ListBySite(ctx context.Context, siteID string) ([]contextpack.MapRow, error)
	ListByIDs(ctx context.Context, pageIDs []string) ([]contextpack.MapRow, error)
	IDsUnder(ctx context.Context, siteID, trail, typeID string) ([]string, error)
	List(ctx context.Context, q contextpack.MapRowQuery, page paging.Request) (paging.List[contextpack.MapRow], error)
	Write(ctx context.Context, siteID string, moved []contextpack.MapRow, gone []string, generation int64) error
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type bodyStore interface {
	ListCurrent(ctx context.Context, pageIDs []string) ([]block.Snapshot, error)
	ListBySite(ctx context.Context, siteID string) ([]block.Snapshot, error)
	ListWritten(ctx context.Context, pageID string) ([]block.Written, error)
	ListWrittenFor(ctx context.Context, pageIDs []string) ([]block.Written, error)
}

type renderStore interface {
	ListForPages(ctx context.Context, pageIDs []string) ([]render.Record, error)
	ListBySite(ctx context.Context, siteID string) ([]render.Record, error)
	ListRenderedBySite(ctx context.Context, siteID string) (map[string]pagetype.Rendered, error)
	URLForms(ctx context.Context, siteID string) ([]string, error)
}

type linkStore interface {
	ListFromPages(ctx context.Context, pageIDs []string) ([]pagemap.PageLink, error)
	ListToPage(ctx context.Context, pageID string) ([]pagemap.PageLink, error)
	ListContextual(ctx context.Context, siteID string) ([]pagemap.PageLink, error)
	ListBySite(ctx context.Context, siteID string) ([]pagemap.PageLink, error)
	ListedProductsBySite(ctx context.Context, siteID string) (map[string]int, error)
	CountContextual(ctx context.Context, pageIDs []string) (inbound, outbound map[string]int, err error)
}

type findingStore interface {
	ListForPage(ctx context.Context, pageID string) ([]finding.Finding, error)
	ListBySite(ctx context.Context, siteID string, status finding.Status) ([]finding.Finding, error)
	ListOpenFor(ctx context.Context, pageIDs []string) ([]finding.Finding, error)
	CountOpen(ctx context.Context, pageIDs []string) (map[string]int, error)
}

type budgets interface {
	InputBudget(role llm.Role) int
}

type audits interface {
	LinkAudit(ctx context.Context, req reports.LinkAuditRequest) (reports.LinkAuditResponse, error)
	LinkAuditPage(ctx context.Context, req reports.LinkAuditPageRequest) (reports.LinkAuditPageResponse, error)
	LinkAuditPages(ctx context.Context, req reports.LinkAuditPagesRequest) (reports.LinkAuditPagesResponse, error)
}

type Deps struct {
	Budgets    budgets
	Audits     audits
	Reader     reader
	UnitOfWork unitOfWork
	Maps       mapStore
	Sites      siteStore
	Pages      pageStore
	Entities   entityStore
	Edges      edgeStore
	Types      typeSets
	Terms      termStore
	Fields     fieldStore
	Bodies     bodyStore
	Renders    renderStore
	Links      linkStore
	Findings   findingStore
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	return &Service{deps: deps}
}
