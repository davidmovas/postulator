package reports

import (
	"cmp"
	"context"
	"net/url"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type siteLinks struct {
	host     string
	policy   template.LinkPolicy
	view     LinkPolicySummary
	g        graph.Graph
	index    pagemap.Index
	entities map[string]graph.Entity
	outgoing map[string][]pagemap.PageLink
	incoming map[string]int
}

type pageDetail struct {
	summary    PageAudit
	templateID string
	rules      template.LinkRules
	required   []RequiredLink
	extra      []ExtraLink
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

	response := LinkAuditResponse{SiteID: siteID, Policy: state.view, Pages: make([]PageAudit, 0, state.index.Len())}
	for _, page := range state.index.Pages() {
		if page.Status == pagemap.StatusArchived {
			continue
		}
		templateID, rules, skip, rulesErr := s.rulesFor(ctx, page)
		if rulesErr != nil {
			return LinkAuditResponse{}, rulesErr
		}
		detail := auditPage(&state, page, templateID, rules, skip)
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
	templateID, rules, skip, err := s.rulesFor(ctx, page)
	if err != nil {
		return LinkAuditPageResponse{}, err
	}

	detail := auditPage(&state, page, templateID, rules, skip)
	return LinkAuditPageResponse{
		Page: detail.summary, TemplateID: detail.templateID, Rules: detail.rules,
		Required: detail.required, Extra: detail.extra,
	}, nil
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
	effective, err := s.policies.GetEffectivePolicy(ctx, templates.GetEffectivePolicyRequest{SiteID: siteID})
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
	_, incoming := linkSets(links)

	return siteLinks{
		host: hostOf(owner.BaseURL),
		policy: template.LinkPolicy{
			ForbidExternal: effective.Policy.ForbidExternal,
			ForbidSelf:     effective.Policy.ForbidSelf,
			AnchorStrategy: template.AnchorStrategy(effective.Policy.AnchorStrategy),
		},
		view: LinkPolicySummary{
			ID: effective.Policy.ID, Name: effective.Policy.Name,
			ForbidExternal: effective.Policy.ForbidExternal, ForbidSelf: effective.Policy.ForbidSelf,
			AnchorStrategy: effective.Policy.AnchorStrategy,
		},
		g: g, index: pagemap.NewIndex(pages), entities: byID,
		outgoing: linksByPage(links), incoming: incoming,
	}, nil
}

func (s *Service) rulesFor(ctx context.Context, page pagemap.Page) (string, template.LinkRules, SkipReason, error) {
	if page.EntityID == nil {
		return "", template.LinkRules{}, SkipUnmapped, nil
	}
	resolved, err := s.specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: page.ID})
	if errors.IsCode(err, errors.NotFound) {
		return "", template.LinkRules{}, SkipNoTemplate, nil
	}
	if err != nil {
		return "", template.LinkRules{}, "", err
	}
	return resolved.TemplateID, resolved.Spec.LinkRules, "", nil
}

func auditPage(state *siteLinks, page pagemap.Page, templateID string, rules template.LinkRules, skip SkipReason) pageDetail {
	inbound := state.incoming[page.ID]
	detail := pageDetail{
		summary: PageAudit{
			PageID: page.ID, Path: page.Path, Status: string(page.Status), SkipReason: string(skip),
			Inbound: inbound, Orphan: inbound == 0 && page.Status != pagemap.StatusArchived,
		},
		required: make([]RequiredLink, 0),
		extra:    make([]ExtraLink, 0),
	}
	if page.EntityID != nil {
		if entity, ok := state.entities[*page.EntityID]; ok {
			detail.summary.EntityID = entity.ID
			detail.summary.EntityName = entity.Name
		}
	}
	if skip != "" {
		return detail
	}

	detail.templateID = templateID
	detail.rules = rules
	policy := state.policy
	policy.Rules = rules
	plan := content.PlanLinks(state.g, state.index, *page.EntityID, policy)
	lc := plan.Context
	lc.PageID = page.ID
	lc.PageURL = page.Path
	links := state.outgoing[page.ID]

	for _, target := range lc.Targets {
		row := RequiredLink{
			Relation: string(target.Relation), Required: target.Required,
			TargetEntityID: target.EntityID, TargetEntityName: state.entities[target.EntityID].Name,
			TargetPageID: target.PageID, TargetPath: target.URL,
			AnchorsAllowed: slices.Clone(target.Anchors), Weight: target.Weight, Depth: target.Depth,
		}
		if link, ok := firstLinkTo(links, target, state.host); ok {
			row.Satisfied = true
			row.Anchor = link.AnchorText
			row.AnchorAllowed = content.AnchorAllowed(target, link.AnchorText)
		}
		detail.required = append(detail.required, row)
	}
	for _, blocked := range plan.Blocked {
		detail.required = append(detail.required, RequiredLink{
			Relation: string(blocked.Relation), Required: blocked.Required,
			TargetEntityID: blocked.EntityID, TargetEntityName: state.entities[blocked.EntityID].Name,
			AnchorsAllowed: []string{}, Weight: blocked.Weight, Depth: blocked.Depth,
			BlockedReason: string(blocked.Reason),
		})
	}
	for i := range links {
		class := lc.ClassifyLink(links[i], state.host)
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
	for _, row := range detail.required {
		if row.Required {
			summary.Required++
		}
		switch {
		case row.BlockedReason != "":
			summary.Blocked++
		case row.Satisfied:
			summary.Satisfied++
		default:
			summary.Missing++
			if row.Required {
				summary.MissingRequired++
			}
		}
	}
	return detail
}

func firstLinkTo(links []pagemap.PageLink, target content.LinkTarget, host string) (pagemap.PageLink, bool) {
	for i := range links {
		if links[i].ToPageID != nil {
			if *links[i].ToPageID == target.PageID {
				return links[i], true
			}
			continue
		}
		if path, internal := pagemap.InternalPath(links[i].ToURL, host); internal && path == target.URL {
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
	if row.Orphan {
		totals.Orphans++
	}
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

func hostOf(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Host)
}
