package main

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const seedPrefix = "seed:"

func refused(what string) error {
	return errors.New(errors.Invalid, "the token report reads a captured model and writes nothing, so it cannot "+what)
}

func typedStand(name string) (*sitemodeltest.Memory, error) {
	held, err := sitemodeltest.Read(name)
	if err != nil {
		return nil, err
	}
	seeds := seededTemplates()
	for i := range held.Types {
		named := held.Types[i].Preset.DefaultTemplate()
		if held.Types[i].TemplateID != nil || named == "" {
			continue
		}
		if _, seeded := seeds[seedPrefix+named]; seeded {
			held.Types[i].TemplateID = new(seedPrefix + named)
		}
	}
	return held.Memory(), nil
}

type capturedEntities struct{ m *sitemodeltest.Memory }

func (r capturedEntities) ListBySite(context.Context, string) ([]graph.Entity, error) {
	return slices.Clone(r.m.Entities), nil
}

type capturedEdges struct{ m *sitemodeltest.Memory }

func (r capturedEdges) ListBySite(context.Context, string) ([]graph.Edge, error) {
	return slices.Clone(r.m.Edges), nil
}

type capturedPages struct{ m *sitemodeltest.Memory }

func (r capturedPages) Get(_ context.Context, id string) (pagemap.Page, error) {
	for i := range r.m.Pages {
		if r.m.Pages[i].ID == id {
			return r.m.Pages[i], nil
		}
	}
	return pagemap.Page{}, errors.New(errors.NotFound, "the capture holds no such page").WithDetail("pageId", id)
}

func (r capturedPages) ListBySite(_ context.Context, siteID string) ([]pagemap.Page, error) {
	out := make([]pagemap.Page, 0, len(r.m.Pages))
	for i := range r.m.Pages {
		if r.m.Pages[i].SiteID == siteID {
			out = append(out, r.m.Pages[i])
		}
	}
	return out, nil
}

func (capturedPages) Insert(context.Context, pagemap.Page) error {
	return refused("insert a page")
}

func (capturedPages) Update(context.Context, pagemap.Page) error {
	return refused("update a page")
}

func (capturedPages) UpdatePublished(context.Context, pagemap.Page) error {
	return refused("record a publication")
}

type capturedSite struct{ m *sitemodeltest.Memory }

func (r capturedSite) Get(_ context.Context, id string) (site.Site, error) {
	if id != r.m.Site.ID {
		return site.Site{}, errors.New(errors.NotFound, "the capture holds no such site").WithDetail("siteId", id)
	}
	return r.m.Site, nil
}

type capturedTypes struct{ m *sitemodeltest.Memory }

func (r capturedTypes) Set(context.Context, string) (pagetype.Set, error) {
	return r.m.Types, nil
}

func (r capturedTypes) ListBySite(context.Context, string) ([]pagetype.Type, error) {
	return r.m.Types.Types(), nil
}

type capturedFields struct{ m *sitemodeltest.Memory }

func (r capturedFields) ListForPage(_ context.Context, pageID string) ([]field.Value, error) {
	return slices.Clone(r.m.Fields[pageID]), nil
}

func (r capturedFields) ListBySite(context.Context, string) (map[string][]field.Value, error) {
	return r.m.Fields, nil
}

func (capturedFields) Record(context.Context, string, field.Key, string, string, time.Time) error {
	return refused("record a field")
}

type seedStore map[string]template.Template

func seededTemplates() seedStore {
	seeds := seedStore{}
	seeded := template.Seed()
	for i := range seeded {
		seeded[i].ID = seedPrefix + seeded[i].Name
		seeds[seeded[i].ID] = seeded[i]
	}
	return seeds
}

func (s seedStore) Get(_ context.Context, id string) (template.Template, error) {
	seeded, ok := s[id]
	if !ok {
		return template.Template{}, errors.New(errors.NotFound, "no built-in template has this id").WithDetail("templateId", id)
	}
	return seeded, nil
}

func (seedStore) GetOverride(context.Context, string, template.OverrideScope, string) (template.Override, error) {
	return template.Override{}, errors.New(errors.NotFound, "a built-in template carries no override")
}

func (seedStore) List(context.Context, template.Query, paging.Request) (paging.List[template.Template], error) {
	return paging.List[template.Template]{}, refused("page through templates")
}

func (seedStore) ListOverrides(context.Context, string) ([]template.Override, error) {
	return nil, refused("list overrides")
}

func (seedStore) Insert(context.Context, template.Template) error {
	return refused("insert a template")
}

func (seedStore) Update(context.Context, template.Template) error {
	return refused("update a template")
}

func (seedStore) Delete(context.Context, string) error {
	return refused("delete a template")
}

func (seedStore) UpsertOverride(context.Context, template.Override) (template.Override, error) {
	return template.Override{}, refused("write an override")
}

func (seedStore) DeleteOverride(context.Context, string) error {
	return refused("delete an override")
}

type readOnly struct{}

func (readOnly) Do(context.Context, func(context.Context) error) error {
	return refused("open a unit of work")
}

func (readOnly) Publish(events.Type, any) error {
	return refused("publish an event")
}

type catalogProfiles struct{ models *catalog.Catalog }

func (p catalogProfiles) Resolve(_ context.Context, _ string, role domainllm.Role, pinned map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error) {
	if ref, ok := pinned[role]; ok && ref.Supported() {
		return ref, nil
	}
	return p.models.Default(role)
}

type recordingClient struct {
	mu    sync.Mutex
	asked []llm.Request
}

func (c *recordingClient) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.asked = append(c.asked, req)
	return llm.Response{}, errors.New(errors.External, "the token report records a request and sends none")
}

func (c *recordingClient) Stream(context.Context, llm.Request) (<-chan llm.Delta, error) {
	return nil, errors.New(errors.External, "the token report records a request and streams none")
}

func (c *recordingClient) take() []llm.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	taken := c.asked
	c.asked = nil
	return taken
}

type recordedPictures struct {
	service  *sitemodel.Service
	mu       sync.Mutex
	answered []sitemodel.Picture
}

func (p *recordedPictures) Picture(ctx context.Context, req sitemodel.PackRequest) (sitemodel.Picture, error) {
	seen, err := p.service.Picture(ctx, req)
	if err != nil {
		return sitemodel.Picture{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.answered = append(p.answered, seen)
	return seen, nil
}

func (p *recordedPictures) Absolute(ctx context.Context, siteID string) (bool, error) {
	return p.service.Absolute(ctx, siteID)
}

func (p *recordedPictures) OwesH1(ctx context.Context, pageID string) (bool, error) {
	return p.service.OwesH1(ctx, pageID)
}

func (p *recordedPictures) take() []sitemodel.Picture {
	p.mu.Lock()
	defer p.mu.Unlock()
	taken := p.answered
	p.answered = nil
	return taken
}
