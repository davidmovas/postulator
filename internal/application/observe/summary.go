package observe

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Summary struct {
	ByType         map[string]int `json:"byType"`
	Roots          map[string]int `json:"roots"`
	LinksByType    map[string]int `json:"linksByType"`
	Technical      map[string]int `json:"technical"`
	Findings       map[string]int `json:"findings"`
	TrailsByMethod map[string]int `json:"trailsByMethod"`
	TrailsByReason map[string]int `json:"trailsByReason"`
	Pages          int            `json:"pages"`
	Observed       int            `json:"observed"`
	Rendered       int            `json:"rendered"`
	TrailsResolved int            `json:"trailsResolved"`
	Listings       int            `json:"listings"`
	ReadOnly       int            `json:"readOnly"`
	LastObservedAt dto.Time       `json:"lastObservedAt"`
	LastFullWalkAt dto.Time       `json:"lastFullWalkAt"`
}

func (s *Service) Summary(ctx context.Context, siteID string) (Summary, error) {
	owner, err := s.deps.Sites.Get(ctx, siteID)
	if err != nil {
		return Summary{}, err
	}
	m, err := s.model(ctx, siteID)
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{
		ByType: map[string]int{}, Roots: map[string]int{}, LinksByType: map[string]int{}, Technical: map[string]int{}, Findings: map[string]int{},
		TrailsByMethod: map[string]int{}, TrailsByReason: map[string]int{},
		LastObservedAt: dto.TimeOf(owner.Settings.Observation.LastFinishedAt), LastFullWalkAt: dto.TimeOf(owner.Settings.Observation.LastFullWalkAt),
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
			summary.TrailsByReason[string(page.ParentReason)]++
			summary.TrailsByMethod[string(page.ParentMethod)]++
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
	identity, err := s.deps.Identity.Get(ctx, pageID)
	switch {
	case err == nil:
		model.Identity = identity
	case !errors.IsCode(err, errors.NotFound):
		return PageModel{}, err
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
	record, err := s.deps.Renders.Get(ctx, pageID)
	switch {
	case err == nil:
		model.Render = &record
	case !errors.IsCode(err, errors.NotFound):
		return PageModel{}, err
	}
	if model.Links, err = s.deps.Links.ListForPage(ctx, pageID); err != nil {
		return PageModel{}, err
	}
	if model.Findings, err = s.deps.Findings.ListForPage(ctx, pageID); err != nil {
		return PageModel{}, err
	}
	return model, nil
}
