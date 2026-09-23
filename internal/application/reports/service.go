package reports

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	topEntitiesCap = 10
	recentItems    = 20
)

type entityReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type pageReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	Get(ctx context.Context, id string) (pagemap.Page, error)
}

type linkReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.PageLink, error)
}

type runReader interface {
	Get(ctx context.Context, id string) (run.Run, error)
}

type itemReader interface {
	ByRun(ctx context.Context, runID string) ([]run.Item, error)
	ByTarget(ctx context.Context, targetID string, limit int) ([]run.Item, error)
}

type artifactReader interface {
	ByItem(ctx context.Context, itemID string) ([]run.Artifact, error)
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type specResolver interface {
	ResolveForPage(ctx context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error)
}

type policyReader interface {
	GetEffectivePolicy(ctx context.Context, req templates.GetEffectivePolicyRequest) (templates.GetEffectivePolicyResponse, error)
}

type Deps struct {
	Entities  entityReader
	Edges     edgeReader
	Pages     pageReader
	Links     linkReader
	Runs      runReader
	Items     itemReader
	Artifacts artifactReader
	Sites     siteReader
	Specs     specResolver
	Policies  policyReader
}

type Service struct {
	entities  entityReader
	edges     edgeReader
	pages     pageReader
	links     linkReader
	runs      runReader
	items     itemReader
	artifacts artifactReader
	sites     siteReader
	specs     specResolver
	policies  policyReader
}

func New(deps Deps) *Service {
	return &Service{
		entities: deps.Entities, edges: deps.Edges, pages: deps.Pages, links: deps.Links,
		runs: deps.Runs, items: deps.Items, artifacts: deps.Artifacts,
		sites: deps.Sites, specs: deps.Specs, policies: deps.Policies,
	}
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
