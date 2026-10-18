package reports

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type siteLinks struct {
	site     pagemap.Site
	types    pagetype.Set
	g        graph.Graph
	index    pagemap.Index
	entities map[string]graph.Entity
	outgoing map[string][]pagemap.PageLink
	incoming map[string]int
	typed    map[string]LinkTypeCounts
}

type pageDetail struct {
	summary  PageAudit
	rules    field.LinkRules
	required []RequiredLink
	extra    []ExtraLink
}

func (s *Service) LinkAudit(ctx context.Context, req LinkAuditRequest) (LinkAuditResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return LinkAuditResponse{}, invalid("a link audit needs a site", "siteId")
	}

	state, err := s.loadLinks(ctx, siteID)
	if err != nil {
		return LinkAuditResponse{}, err
	}

	pages := state.index.Pages()
	response := LinkAuditResponse{SiteID: siteID, Pages: make([]PageAudit, 0, len(pages))}
	for i := range pages {
		if pages[i].Status == pagemap.StatusArchived {
			continue
		}
		detail := auditPage(&state, pages[i])
		response.Pages = append(response.Pages, detail.summary)
		tally(&response.Totals, detail.summary)
	}
	return response, nil
}

func (s *Service) LinkAuditPage(ctx context.Context, req LinkAuditPageRequest) (LinkAuditPageResponse, error) {
	pageID := strings.TrimSpace(req.PageID)
	if pageID == "" {
		return LinkAuditPageResponse{}, invalid("a page audit needs a page", "pageId")
	}

	page, err := s.pages.Get(ctx, pageID)
	if err != nil {
		return LinkAuditPageResponse{}, err
	}
	state, err := s.loadLinks(ctx, page.SiteID)
	if err != nil {
		return LinkAuditPageResponse{}, err
	}
	templateID, err := s.templateOf(ctx, page)
	if err != nil {
		return LinkAuditPageResponse{}, err
	}

	detail := auditPage(&state, page)
	return LinkAuditPageResponse{
		Page: detail.summary, TemplateID: templateID, Rules: detail.rules,
		Required: detail.required, Extra: detail.extra,
	}, nil
}

func (s *Service) templateOf(ctx context.Context, page pagemap.Page) (string, error) {
	if page.EntityID == nil {
		return "", nil
	}
	resolved, err := s.specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: page.ID})
	if errors.IsCode(err, errors.NotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return resolved.TemplateID, nil
}

func (s *Service) loadLinks(ctx context.Context, siteID string) (siteLinks, error) {
	owner, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return siteLinks{}, err
	}
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return siteLinks{}, err
	}
	edges, err := s.edges.ListBySite(ctx, siteID)
	if err != nil {
		return siteLinks{}, err
	}
	pages, err := s.pages.ListBySite(ctx, siteID)
	if err != nil {
		return siteLinks{}, err
	}
	links, err := s.links.ListBySite(ctx, siteID)
	if err != nil {
		return siteLinks{}, err
	}
	types, err := s.types.Set(ctx, siteID)
	if err != nil {
		return siteLinks{}, err
	}
	g, err := graph.New(entities, edges)
	if err != nil {
		return siteLinks{}, err
	}

	byID := make(map[string]graph.Entity, len(entities))
	for i := range entities {
		byID[entities[i].ID] = entities[i]
	}
	owed := contextual(links)
	_, incoming := linkSets(owed)

	return siteLinks{
		site:  pagemap.NewSite(owner.BaseURL),
		types: types,
		g:     g, index: pagemap.NewIndex(pages), entities: byID,
		outgoing: linksByPage(owed), incoming: incoming, typed: typesByPage(links),
	}, nil
}

func contextual(links []pagemap.PageLink) []pagemap.PageLink {
	out := make([]pagemap.PageLink, 0, len(links))
	for i := range links {
		if links[i].Type == pagemap.LinkContextual {
			out = append(out, links[i])
		}
	}
	return out
}

func typesByPage(links []pagemap.PageLink) map[string]LinkTypeCounts {
	out := make(map[string]LinkTypeCounts)
	for i := range links {
		counts := out[links[i].FromPageID]
		counts.add(links[i].Type)
		out[links[i].FromPageID] = counts
	}
	return out
}

func (c *LinkTypeCounts) add(kind pagemap.LinkType) {
	switch kind {
	case pagemap.LinkBreadcrumb:
		c.Breadcrumb++
	case pagemap.LinkListing:
		c.Listing++
	case pagemap.LinkNavigation:
		c.Navigation++
	case pagemap.LinkContextual:
		c.Contextual++
	default:
		c.Unknown++
	}
}

func (c *LinkTypeCounts) plus(other LinkTypeCounts) {
	c.Breadcrumb += other.Breadcrumb
	c.Listing += other.Listing
	c.Navigation += other.Navigation
	c.Contextual += other.Contextual
	c.Unknown += other.Unknown
}

func orphan(page pagemap.Page, inbound int) bool {
	return page.WPID != nil && page.Status != pagemap.StatusArchived && page.TechnicalReason == pagemap.TechnicalNone &&
		!page.WPType.Term() && page.Path != pagemap.RootPath && inbound == 0
}

func auditPage(state *siteLinks, page pagemap.Page) pageDetail {
	inbound := state.incoming[page.ID]
	onSite := page.WPID != nil
	detail := pageDetail{
		summary: PageAudit{
			PageID: page.ID, Path: page.Path, Status: string(page.Status), OnSite: onSite,
			Inbound: inbound, Orphan: orphan(page, inbound), ByType: state.typed[page.ID],
		},
		required: make([]RequiredLink, 0),
		extra:    make([]ExtraLink, 0),
	}
	if page.EntityID == nil {
		detail.summary.SkipReason = string(SkipUnmapped)
		return detail
	}
	if entity, ok := state.entities[*page.EntityID]; ok {
		detail.summary.EntityID = entity.ID
		detail.summary.EntityName = state.g.Label(entity.ID)
	}

	policy := state.types.Policy(page)
	detail.rules = policy.Rules
	plan := content.PlanLinks(state.g, state.index, content.Subject{
		Site: state.site, PageID: page.ID, PagePath: page.Path, EntityID: *page.EntityID,
	}, policy)
	lc := plan.Context
	links := state.outgoing[page.ID]

	for _, target := range lc.Targets {
		row := RequiredLink{
			Relation: string(target.Relation), Required: target.Required,
			TargetEntityID: target.EntityID, TargetEntityName: state.g.Label(target.EntityID),
			TargetPageID: target.PageID, TargetPath: target.URL,
			AnchorsAllowed: slices.Clone(target.Anchors), Weight: target.Weight, Depth: target.Depth,
		}
		if targetPage, known := state.index.ByID(target.PageID); known {
			row.TargetOnSite = targetPage.WPID != nil
		}
		link, placed := firstLinkTo(lc, links, target)
		if placed {
			row.Satisfied = true
			row.Anchor = link.AnchorText
			row.AnchorAllowed = content.AnchorAllowed(target, link.AnchorText)
		}
		row.State = string(stateOf(placed, onSite, row.TargetOnSite))
		detail.required = append(detail.required, row)
	}
	for _, blocked := range plan.Blocked {
		detail.required = append(detail.required, RequiredLink{
			Relation: string(blocked.Relation), Required: blocked.Required,
			TargetEntityID: blocked.EntityID, TargetEntityName: state.g.Label(blocked.EntityID),
			AnchorsAllowed: []string{}, Weight: blocked.Weight, Depth: blocked.Depth,
			BlockedReason: string(blocked.Reason), State: string(LinkBlocked),
		})
	}
	for i := range links {
		class := lc.ClassifyLink(links[i])
		if !class.OffGraph() {
			continue
		}
		extra := ExtraLink{ToURL: links[i].ToURL, Anchor: links[i].AnchorText, Kind: string(class), Origin: string(links[i].Origin)}
		if links[i].ToPageID != nil {
			extra.ToPageID = *links[i].ToPageID
		}
		detail.extra = append(detail.extra, extra)
	}

	summary := &detail.summary
	summary.Targets = len(detail.required)
	summary.OffGraph = len(detail.extra)
	for i := range detail.required {
		row := &detail.required[i]
		if row.Required {
			summary.Required++
		}
		switch LinkState(row.State) {
		case LinkBlocked:
			summary.Blocked++
		case LinkPlaced:
			summary.Satisfied++
		case LinkTargetUnpublished:
			summary.Satisfied++
			summary.Unpublished++
		case LinkAwaitingPage, LinkAwaitingTarget:
			summary.Pending++
		case LinkMissing:
			summary.Missing++
			if row.Required {
				summary.MissingRequired++
			}
		}
	}
	return detail
}

func stateOf(placed, pageOnSite, targetOnSite bool) LinkState {
	switch {
	case placed && targetOnSite:
		return LinkPlaced
	case placed:
		return LinkTargetUnpublished
	case !pageOnSite:
		return LinkAwaitingPage
	case !targetOnSite:
		return LinkAwaitingTarget
	default:
		return LinkMissing
	}
}

func firstLinkTo(lc content.LinkContext, links []pagemap.PageLink, target content.LinkTarget) (pagemap.PageLink, bool) {
	for i := range links {
		if links[i].ToPageID != nil {
			if *links[i].ToPageID == target.PageID {
				return links[i], true
			}
			continue
		}
		resolution := lc.Resolve(links[i].ToURL)
		if resolution.Class == content.ClassGraph && resolution.Target.PageID == target.PageID {
			return links[i], true
		}
	}
	return pagemap.PageLink{}, false
}

func tally(totals *LinkTotals, row PageAudit) {
	totals.Pages++
	if row.SkipReason == "" {
		totals.Audited++
	}
	totals.Targets += row.Targets
	totals.Required += row.Required
	totals.Satisfied += row.Satisfied
	totals.Missing += row.Missing
	totals.MissingRequired += row.MissingRequired
	totals.Blocked += row.Blocked
	totals.OffGraph += row.OffGraph
	totals.Pending += row.Pending
	totals.Unpublished += row.Unpublished
	if row.Orphan {
		totals.Orphans++
	}
	totals.ByType.plus(row.ByType)
}

func linksByPage(links []pagemap.PageLink) map[string][]pagemap.PageLink {
	out := make(map[string][]pagemap.PageLink)
	for i := range links {
		out[links[i].FromPageID] = append(out[links[i].FromPageID], links[i])
	}
	for id := range out {
		slices.SortFunc(out[id], func(a, b pagemap.PageLink) int {
			if !a.ObservedAt.Equal(b.ObservedAt) {
				return a.ObservedAt.Compare(b.ObservedAt)
			}
			return cmp.Compare(a.ID, b.ID)
		})
	}
	return out
}
