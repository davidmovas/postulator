package sitemodeltest

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

type Memory struct {
	Site     site.Site
	Pages    []pagemap.Page
	Entities []graph.Entity
	Edges    []graph.Edge
	Types    pagetype.Set
	Terms    map[string][]pagemap.Term
	Fields   map[string][]field.Value
	Bodies   []block.Snapshot
	Written  []block.Written
	Renders  []render.Record
	Links    []pagemap.PageLink
	Findings []finding.Finding
	Budgets  map[domainllm.Role]int
	Audit    reports.LinkAuditResponse

	mu    sync.Mutex
	reads int
	asked [][]string
}

func (m *Memory) Service() *sitemodel.Service {
	return sitemodel.New(sitemodel.Deps{
		Reader: m, Sites: siteReader{m}, Pages: pageReader{m}, Entities: entityReader{m}, Edges: edgeReader{m},
		Types: typeReader{m}, Terms: termReader{m}, Fields: fieldReader{m}, Bodies: bodyReader{m},
		Renders: renderReader{m}, Links: linkReader{m}, Findings: findingReader{m}, Budgets: budgetReader{m},
		Audits: auditReader{m},
	})
}

func (m *Memory) Reads() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reads
}

func (m *Memory) Asked() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.asked)
}

func (m *Memory) Read(ctx context.Context, fn func(context.Context) error) error {
	m.mu.Lock()
	m.reads++
	m.mu.Unlock()
	return fn(ctx)
}

func (m *Memory) ask(ids []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.asked = append(m.asked, slices.Clone(ids))
}

type auditReader struct{ m *Memory }

func (r auditReader) LinkAudit(_ context.Context, req reports.LinkAuditRequest) (reports.LinkAuditResponse, error) {
	if req.SiteID != r.m.Site.ID {
		return reports.LinkAuditResponse{}, errors.New(errors.NotFound, "no such site")
	}
	return r.m.Audit, nil
}

func (r auditReader) LinkAuditPage(_ context.Context, req reports.LinkAuditPageRequest) (reports.LinkAuditPageResponse, error) {
	for i := range r.m.Audit.Pages {
		if r.m.Audit.Pages[i].PageID == req.PageID {
			return reports.LinkAuditPageResponse{Page: r.m.Audit.Pages[i]}, nil
		}
	}
	return reports.LinkAuditPageResponse{Page: reports.PageAudit{PageID: req.PageID}}, nil
}

type budgetReader struct{ m *Memory }

func (r budgetReader) InputBudget(role domainllm.Role) int {
	return r.m.Budgets[role]
}

type siteReader struct{ m *Memory }

func (r siteReader) Get(_ context.Context, id string) (site.Site, error) {
	if id != r.m.Site.ID {
		return site.Site{}, errors.New(errors.NotFound, "no such site")
	}
	return r.m.Site, nil
}

type pageReader struct{ m *Memory }

func (r pageReader) Get(_ context.Context, id string) (pagemap.Page, error) {
	for i := range r.m.Pages {
		if r.m.Pages[i].ID == id {
			return r.m.Pages[i], nil
		}
	}
	return pagemap.Page{}, errors.New(errors.NotFound, "no such page")
}

func (r pageReader) ListBySite(_ context.Context, siteID string) ([]pagemap.Page, error) {
	out := make([]pagemap.Page, 0, len(r.m.Pages))
	for i := range r.m.Pages {
		if r.m.Pages[i].SiteID == siteID {
			out = append(out, r.m.Pages[i])
		}
	}
	return out, nil
}

type entityReader struct{ m *Memory }

func (r entityReader) ListBySite(context.Context, string) ([]graph.Entity, error) {
	return slices.Clone(r.m.Entities), nil
}

type edgeReader struct{ m *Memory }

func (r edgeReader) ListBySite(context.Context, string) ([]graph.Edge, error) {
	return slices.Clone(r.m.Edges), nil
}

type typeReader struct{ m *Memory }

func (r typeReader) Set(context.Context, string) (pagetype.Set, error) {
	return r.m.Types, nil
}

type termReader struct{ m *Memory }

func (r termReader) ListTermsBySite(context.Context, string) (map[string][]pagemap.Term, error) {
	return r.m.Terms, nil
}

type fieldReader struct{ m *Memory }

func (r fieldReader) ListBySite(context.Context, string) (map[string][]field.Value, error) {
	return r.m.Fields, nil
}

type bodyReader struct{ m *Memory }

func (r bodyReader) ListCurrent(_ context.Context, ids []string) ([]block.Snapshot, error) {
	r.m.ask(ids)
	out := make([]block.Snapshot, 0)
	for i := range r.m.Bodies {
		if slices.Contains(ids, r.m.Bodies[i].PageID) {
			out = append(out, r.m.Bodies[i])
		}
	}
	return out, nil
}

func (r bodyReader) ListBySite(context.Context, string) ([]block.Snapshot, error) {
	return slices.Clone(r.m.Bodies), nil
}

func (r bodyReader) ListWritten(_ context.Context, pageID string) ([]block.Written, error) {
	out := make([]block.Written, 0)
	for i := range r.m.Written {
		if r.m.Written[i].PageID == pageID {
			out = append(out, r.m.Written[i])
		}
	}
	return out, nil
}

func (r bodyReader) ListWrittenFor(_ context.Context, ids []string) ([]block.Written, error) {
	out := make([]block.Written, 0)
	for i := range r.m.Written {
		if slices.Contains(ids, r.m.Written[i].PageID) {
			out = append(out, r.m.Written[i])
		}
	}
	return out, nil
}

type renderReader struct{ m *Memory }

func (r renderReader) ListForPages(_ context.Context, ids []string) ([]render.Record, error) {
	out := make([]render.Record, 0)
	for i := range r.m.Renders {
		if slices.Contains(ids, r.m.Renders[i].PageID) {
			out = append(out, r.m.Renders[i])
		}
	}
	return out, nil
}

func (r renderReader) ListBySite(context.Context, string) ([]render.Record, error) {
	return slices.Clone(r.m.Renders), nil
}

func (r renderReader) URLForms(context.Context, string) ([]string, error) {
	out := make([]string, 0)
	for i := range r.m.Renders {
		if r.m.Renders[i].URLForm != "" {
			out = append(out, r.m.Renders[i].URLForm)
		}
	}
	return out, nil
}

type linkReader struct{ m *Memory }

func (r linkReader) ListFromPages(_ context.Context, ids []string) ([]pagemap.PageLink, error) {
	out := make([]pagemap.PageLink, 0)
	for i := range r.m.Links {
		if slices.Contains(ids, r.m.Links[i].FromPageID) {
			out = append(out, r.m.Links[i])
		}
	}
	return out, nil
}

func (r linkReader) ListToPage(_ context.Context, pageID string) ([]pagemap.PageLink, error) {
	out := make([]pagemap.PageLink, 0)
	for i := range r.m.Links {
		if r.m.Links[i].ToPageID != nil && *r.m.Links[i].ToPageID == pageID {
			out = append(out, r.m.Links[i])
		}
	}
	return out, nil
}

func (r linkReader) ListContextual(context.Context, string) ([]pagemap.PageLink, error) {
	out := make([]pagemap.PageLink, 0)
	for i := range r.m.Links {
		if r.m.Links[i].Type == pagemap.LinkContextual && r.m.Links[i].Source == pagemap.SourceRaw {
			out = append(out, r.m.Links[i])
		}
	}
	return out, nil
}

func (r linkReader) ListBySite(context.Context, string) ([]pagemap.PageLink, error) {
	return slices.Clone(r.m.Links), nil
}

type findingReader struct{ m *Memory }

func (r findingReader) ListForPage(_ context.Context, pageID string) ([]finding.Finding, error) {
	out := make([]finding.Finding, 0)
	for i := range r.m.Findings {
		if r.m.Findings[i].PageID != nil && *r.m.Findings[i].PageID == pageID {
			out = append(out, r.m.Findings[i])
		}
	}
	return out, nil
}

func (r findingReader) ListBySite(_ context.Context, _ string, status finding.Status) ([]finding.Finding, error) {
	out := make([]finding.Finding, 0)
	for i := range r.m.Findings {
		if r.m.Findings[i].Status == status {
			out = append(out, r.m.Findings[i])
		}
	}
	return out, nil
}
