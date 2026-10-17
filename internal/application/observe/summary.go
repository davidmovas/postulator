package observe

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

type Summary struct {
	ByType         map[string]int `json:"byType"`
	Roots          map[string]int `json:"roots"`
	LinksByType    map[string]int `json:"linksByType"`
	Technical      map[string]int `json:"technical"`
	Findings       map[string]int `json:"findings"`
	Pages          int            `json:"pages"`
	Observed       int            `json:"observed"`
	Rendered       int            `json:"rendered"`
	TrailsResolved int            `json:"trailsResolved"`
	Listings       int            `json:"listings"`
	ReadOnly       int            `json:"readOnly"`
}

func (s *Service) Summary(ctx context.Context, siteID string) (Summary, error) {
	m, err := s.model(ctx, siteID)
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{
		ByType: map[string]int{}, Roots: map[string]int{}, LinksByType: map[string]int{}, Technical: map[string]int{}, Findings: map[string]int{},
	}
	for i := range m.pages {
		page := m.pages[i]
		summary.Pages++
		summary.ByType[string(page.WPType)]++
		if _, observed := m.identities[page.ID]; observed {
			summary.Observed++
		}
		if page.TechnicalReason != pagemap.TechnicalNone {
			summary.Technical[string(page.TechnicalReason)]++
		}
		if page.ReadOnlyReason != pagemap.ReadOnlyNone {
			summary.ReadOnly++
		}
		if page.ParentReason == pagemap.ParentJSONLD || page.ParentReason == pagemap.ParentARIA {
			summary.TrailsResolved++
		}
	}
	for id := range m.renders {
		record := m.renders[id]
		summary.Rendered++
		summary.Roots[string(record.Root)]++
		for _, region := range record.Regions {
			if region.Kind == render.RegionListing {
				summary.Listings++
				break
			}
		}
	}
	links, err := s.deps.Links.ListBySite(ctx, siteID)
	if err != nil {
		return Summary{}, err
	}
	for i := range links {
		summary.LinksByType[string(links[i].Type)]++
	}
	open, err := s.deps.Findings.ListBySite(ctx, siteID, finding.Open)
	if err != nil {
		return Summary{}, err
	}
	for i := range open {
		summary.Findings[string(open[i].Code)]++
	}
	return summary, nil
}

type PageModel struct {
	Page     pagemap.Page
	Identity pagemap.Identity
	Terms    []pagemap.Term
	Fields   []field.Value
	Body     *block.Snapshot
	Base     *block.Snapshot
	Render   *render.Record
	Links    []pagemap.PageLink
	Findings []finding.Finding
}

func (s *Service) Page(ctx context.Context, pageID string) (PageModel, error) {
	page, err := s.deps.Pages.Get(ctx, pageID)
	if err != nil {
		return PageModel{}, err
	}
	model := PageModel{Page: page}
	identities, err := s.deps.Identity.ListBySite(ctx, page.SiteID)
	if err != nil {
		return PageModel{}, err
	}
	for i := range identities {
		if identities[i].PageID == pageID {
			model.Identity = identities[i]
		}
	}
	if model.Terms, err = s.deps.Identity.ListTerms(ctx, pageID); err != nil {
		return PageModel{}, err
	}
	if model.Fields, err = s.deps.Fields.ListForPage(ctx, pageID); err != nil {
		return PageModel{}, err
	}
	current, base, found, err := s.deps.Bodies.Get(ctx, pageID)
	if err != nil {
		return PageModel{}, err
	}
	if found {
		model.Body, model.Base = &current, &base
	}
	if record, renderErr := s.deps.Renders.Get(ctx, pageID); renderErr == nil {
		model.Render = &record
	}
	if model.Links, err = s.deps.Links.ListForPage(ctx, pageID); err != nil {
		return PageModel{}, err
	}
	if model.Findings, err = s.deps.Findings.ListForPage(ctx, pageID); err != nil {
		return PageModel{}, err
	}
	return model, nil
}
