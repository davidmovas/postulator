package steps_test

import (
	"context"
	"sync"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type tuningStub struct {
	budget int
}

func (tuningStub) Effort(domainllm.Role) domainllm.ReasoningEffort {
	return domainllm.EffortLow
}

func (t tuningStub) InputBudget(domainllm.Role) int {
	return t.budget
}

type catalogStub struct {
	context int
}

func (c catalogStub) Lookup(context.Context, domainllm.ModelRef) (domainllm.ModelInfo, error) {
	window := c.context
	if window == 0 {
		window = 400000
	}
	return domainllm.ModelInfo{ContextTokens: window, MaxOutputTokens: 128000}, nil
}

type graphSource interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeSource interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type pageSource interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
}

type typeSource interface {
	Set(ctx context.Context, siteID string) (pagetype.Set, error)
}

type pictureStub struct {
	mu       sync.Mutex
	entities graphSource
	edges    edgeSource
	pages    pageSource
	types    typeSource
	fields   map[string][]field.Value
	bodies   map[string]block.Snapshot
	written  map[string][]string
	links    []pagemap.PageLink
	asked    []sitemodel.PackRequest
	absolute bool
	owesH1   map[string]bool
}

func (p *pictureStub) Picture(ctx context.Context, req sitemodel.PackRequest) (sitemodel.Picture, error) {
	p.mu.Lock()
	p.asked = append(p.asked, req)
	p.mu.Unlock()

	const site = "site"
	entities, err := p.entities.ListBySite(ctx, site)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	edges, err := p.edges.ListBySite(ctx, site)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	pages, err := p.pages.ListBySite(ctx, site)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	types, err := p.types.Set(ctx, site)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	g, err := graph.New(entities, edges)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	snapshot := contextpack.Snapshot{
		Site: pagemap.NewSite("https://shop.example.com"), Index: pagemap.NewIndex(pages), Types: types, Graph: g,
		Fields: p.fields, Bodies: p.bodies, Written: p.written, Links: p.links,
	}
	plan := content.LinkContext{PageID: req.PageID}
	if req.Claim != nil && req.Claim.Links != nil {
		plan = *req.Claim.Links
	}
	facts, err := snapshot.Facts(req.PageID, plan)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	scope := field.Scope{Key: field.Key(req.Field), Enabled: true, Links: field.NoLinks()}
	if req.Claim != nil {
		scope = req.Claim.Scope
		if req.Claim.Type != nil {
			facts.Page.Type, facts.Page.Preset = req.Claim.Type.Name, req.Claim.Type.Preset
		}
	}
	pack := contextpack.Build(facts, scope)
	fitted, err := contextpack.Fit(pack, req.Budget)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	return sitemodel.Picture{Facts: facts, Pack: pack, Fitted: fitted, Links: plan, Scope: scope}, nil
}

func (p *pictureStub) Absolute(context.Context, string) (bool, error) {
	return p.absolute, nil
}

func (p *pictureStub) OwesH1(_ context.Context, pageID string) (bool, error) {
	return p.owesH1[pageID], nil
}

func (p *pictureStub) requests() []sitemodel.PackRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]sitemodel.PackRequest(nil), p.asked...)
}

func pictured(deps steps.Deps) steps.Deps {
	deps.Tuning = tuningStub{}
	deps.Models = catalogStub{}
	deps.Pictures = &pictureStub{entities: deps.Entities, edges: deps.Edges, pages: deps.Pages, types: deps.Types}
	return deps
}
