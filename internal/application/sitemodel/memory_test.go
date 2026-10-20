package sitemodel_test

import (
	"context"
	"slices"
	"sync"

	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type memory struct {
	mu       sync.Mutex
	site     site.Site
	pages    []pagemap.Page
	entities []graph.Entity
	edges    []graph.Edge
	types    pagetype.Set
	terms    map[string][]pagemap.Term
	fields   map[string][]field.Value
	bodies   []block.Snapshot
	written  []block.Written
	renders  []render.Record
	links    []pagemap.PageLink
	findings []finding.Finding
	reads    int
	asked    [][]string
	budgets  map[domainllm.Role]int
	audit    reports.LinkAuditResponse
}

func (m *memory) service() *sitemodel.Service {
	return sitemodel.New(sitemodel.Deps{
		Reader: m, Sites: siteReader{m}, Pages: pageReader{m}, Entities: entityReader{m}, Edges: edgeReader{m},
		Types: typeReader{m}, Terms: termReader{m}, Fields: fieldReader{m}, Bodies: bodyReader{m},
		Renders: renderReader{m}, Links: linkReader{m}, Findings: findingReader{m}, Budgets: budgetReader{m},
		Audits: auditReader{m},
	})
}

type auditReader struct{ m *memory }

func (r auditReader) LinkAudit(_ context.Context, req reports.LinkAuditRequest) (reports.LinkAuditResponse, error) {
	if req.SiteID != r.m.site.ID {
		return reports.LinkAuditResponse{}, errors.New(errors.NotFound, "no such site")
	}
	return r.m.audit, nil
}

func (r auditReader) LinkAuditPage(_ context.Context, req reports.LinkAuditPageRequest) (reports.LinkAuditPageResponse, error) {
	for i := range r.m.audit.Pages {
		if r.m.audit.Pages[i].PageID == req.PageID {
			return reports.LinkAuditPageResponse{Page: r.m.audit.Pages[i]}, nil
		}
	}
	return reports.LinkAuditPageResponse{Page: reports.PageAudit{PageID: req.PageID}}, nil
}

type budgetReader struct{ m *memory }

func (r budgetReader) InputBudget(role domainllm.Role) int {
	return r.m.budgets[role]
}

func (m *memory) Read(ctx context.Context, fn func(context.Context) error) error {
	m.mu.Lock()
	m.reads++
	m.mu.Unlock()
	return fn(ctx)
}

func (m *memory) ask(ids []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.asked = append(m.asked, slices.Clone(ids))
}

type siteReader struct{ m *memory }

func (r siteReader) Get(_ context.Context, id string) (site.Site, error) {
	if id != r.m.site.ID {
		return site.Site{}, errors.New(errors.NotFound, "no such site")
	}
	return r.m.site, nil
}

type pageReader struct{ m *memory }

func (r pageReader) Get(_ context.Context, id string) (pagemap.Page, error) {
	for i := range r.m.pages {
		if r.m.pages[i].ID == id {
			return r.m.pages[i], nil
		}
	}
	return pagemap.Page{}, errors.New(errors.NotFound, "no such page")
}

func (r pageReader) ListBySite(_ context.Context, siteID string) ([]pagemap.Page, error) {
	out := make([]pagemap.Page, 0, len(r.m.pages))
	for i := range r.m.pages {
		if r.m.pages[i].SiteID == siteID {
			out = append(out, r.m.pages[i])
		}
	}
	return out, nil
}

type entityReader struct{ m *memory }

func (r entityReader) ListBySite(context.Context, string) ([]graph.Entity, error) {
	return slices.Clone(r.m.entities), nil
}

type edgeReader struct{ m *memory }

func (r edgeReader) ListBySite(context.Context, string) ([]graph.Edge, error) {
	return slices.Clone(r.m.edges), nil
}

type typeReader struct{ m *memory }

func (r typeReader) Set(context.Context, string) (pagetype.Set, error) {
	return r.m.types, nil
}

type termReader struct{ m *memory }

func (r termReader) ListTermsBySite(context.Context, string) (map[string][]pagemap.Term, error) {
	return r.m.terms, nil
}

type fieldReader struct{ m *memory }

func (r fieldReader) ListBySite(context.Context, string) (map[string][]field.Value, error) {
	return r.m.fields, nil
}

type bodyReader struct{ m *memory }

func (r bodyReader) ListCurrent(_ context.Context, ids []string) ([]block.Snapshot, error) {
	r.m.ask(ids)
	out := make([]block.Snapshot, 0)
	for i := range r.m.bodies {
		if slices.Contains(ids, r.m.bodies[i].PageID) {
			out = append(out, r.m.bodies[i])
		}
	}
	return out, nil
}

func (r bodyReader) ListBySite(context.Context, string) ([]block.Snapshot, error) {
	return slices.Clone(r.m.bodies), nil
}

func (r bodyReader) ListWritten(_ context.Context, pageID string) ([]block.Written, error) {
	out := make([]block.Written, 0)
	for i := range r.m.written {
		if r.m.written[i].PageID == pageID {
			out = append(out, r.m.written[i])
		}
	}
	return out, nil
}

type renderReader struct{ m *memory }

func (r renderReader) ListForPages(_ context.Context, ids []string) ([]render.Record, error) {
	out := make([]render.Record, 0)
	for i := range r.m.renders {
		if slices.Contains(ids, r.m.renders[i].PageID) {
			out = append(out, r.m.renders[i])
		}
	}
	return out, nil
}

func (r renderReader) ListBySite(context.Context, string) ([]render.Record, error) {
	return slices.Clone(r.m.renders), nil
}

func (r renderReader) URLForms(context.Context, string) ([]string, error) {
	out := make([]string, 0)
	for i := range r.m.renders {
		if r.m.renders[i].URLForm != "" {
			out = append(out, r.m.renders[i].URLForm)
		}
	}
	return out, nil
}

type linkReader struct{ m *memory }

func (r linkReader) ListFromPages(_ context.Context, ids []string) ([]pagemap.PageLink, error) {
	out := make([]pagemap.PageLink, 0)
	for i := range r.m.links {
		if slices.Contains(ids, r.m.links[i].FromPageID) {
			out = append(out, r.m.links[i])
		}
	}
	return out, nil
}

func (r linkReader) ListToPage(_ context.Context, pageID string) ([]pagemap.PageLink, error) {
	out := make([]pagemap.PageLink, 0)
	for i := range r.m.links {
		if r.m.links[i].ToPageID != nil && *r.m.links[i].ToPageID == pageID {
			out = append(out, r.m.links[i])
		}
	}
	return out, nil
}

func (r linkReader) ListContextual(context.Context, string) ([]pagemap.PageLink, error) {
	out := make([]pagemap.PageLink, 0)
	for i := range r.m.links {
		if r.m.links[i].Type == pagemap.LinkContextual && r.m.links[i].Source == pagemap.SourceRaw {
			out = append(out, r.m.links[i])
		}
	}
	return out, nil
}

func (r linkReader) ListBySite(context.Context, string) ([]pagemap.PageLink, error) {
	return slices.Clone(r.m.links), nil
}

type findingReader struct{ m *memory }

func (r findingReader) ListForPage(_ context.Context, pageID string) ([]finding.Finding, error) {
	out := make([]finding.Finding, 0)
	for i := range r.m.findings {
		if r.m.findings[i].PageID != nil && *r.m.findings[i].PageID == pageID {
			out = append(out, r.m.findings[i])
		}
	}
	return out, nil
}

func (r findingReader) ListBySite(_ context.Context, _ string, status finding.Status) ([]finding.Finding, error) {
	out := make([]finding.Finding, 0)
	for i := range r.m.findings {
		if r.m.findings[i].Status == status {
			out = append(out, r.m.findings[i])
		}
	}
	return out, nil
}
