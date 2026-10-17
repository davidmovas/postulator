package observe

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var regionContent = render.RegionContent

const statusPublish = "publish"

type renderTarget struct {
	page     pagemap.Page
	identity pagemap.Identity
	url      string
	raw      string
}

type renderOutcome struct {
	target     renderTarget
	extraction render.Extraction
	err        error
	status     int
}

type siteModel struct {
	pages      []pagemap.Page
	identities map[string]pagemap.Identity
	renders    map[string]render.Record
	bodies     map[string]block.Snapshot
	index      pagemap.Index
}

func (s *Service) model(ctx context.Context, siteID string) (siteModel, error) {
	pages, err := s.deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return siteModel{}, err
	}
	identities, err := s.deps.Identity.ListBySite(ctx, siteID)
	if err != nil {
		return siteModel{}, err
	}
	renders, err := s.deps.Renders.ListBySite(ctx, siteID)
	if err != nil {
		return siteModel{}, err
	}
	bodies, err := s.deps.Bodies.ListBySite(ctx, siteID)
	if err != nil {
		return siteModel{}, err
	}
	m := siteModel{
		pages: pages, identities: make(map[string]pagemap.Identity, len(identities)),
		renders: make(map[string]render.Record, len(renders)), bodies: make(map[string]block.Snapshot), index: pagemap.NewIndex(pages),
	}
	for i := range identities {
		m.identities[identities[i].PageID] = identities[i]
	}
	for i := range renders {
		m.renders[renders[i].PageID] = renders[i]
	}
	for i := range bodies {
		if bodies[i].Side == block.Current {
			m.bodies[bodies[i].PageID] = bodies[i]
		}
	}
	slices.SortFunc(m.pages, func(a, b pagemap.Page) int { return strings.Compare(a.ID, b.ID) })
	return m, nil
}

func (s *Service) render(ctx context.Context, state *State, readers Readers, batch int) (bool, error) {
	if readers.Render == nil {
		return true, nil
	}
	owner, err := s.deps.Sites.Get(ctx, state.SiteID)
	if err != nil {
		return false, err
	}
	settings := owner.Settings.Normalized()
	m, err := s.model(ctx, state.SiteID)
	if err != nil {
		return false, err
	}

	targets := make([]renderTarget, 0, batch)
	exhausted := true
	for i := range m.pages {
		page := m.pages[i]
		if page.ID <= state.RenderAfter || page.WPID == nil || page.Status == pagemap.StatusArchived {
			continue
		}
		if len(targets) == batch {
			exhausted = false
			break
		}
		state.RenderAfter = page.ID
		identity, observed := m.identities[page.ID]
		if !observed || (state.Full && archive(page, identity, state, readers)) || !s.needsRender(state, page, identity, m) {
			continue
		}
		url, ok := s.renderURL(ctx, readers, page, identity)
		if !ok {
			continue
		}
		targets = append(targets, renderTarget{page: page, identity: identity, url: url, raw: m.bodies[page.ID].Raw})
	}
	if len(targets) == 0 {
		return exhausted, nil
	}

	outcomes := s.fetch(ctx, readers, owner, settings, m, targets)
	if err := s.keepRenders(ctx, state, m, outcomes); err != nil {
		return false, err
	}
	return exhausted, nil
}

func (s *Service) needsRender(state *State, page pagemap.Page, identity pagemap.Identity, m siteModel) bool {
	if state.Full || state.Rescan {
		return true
	}
	record, rendered := m.renders[page.ID]
	if !rendered {
		return true
	}
	if identity.ModifiedGMT != nil && identity.ModifiedGMT.After(record.ExtractedAt) {
		return true
	}
	body, hasBody := m.bodies[page.ID]
	return hasBody && body.ObservedAt.After(record.ExtractedAt)
}

func (s *Service) renderURL(ctx context.Context, readers Readers, page pagemap.Page, identity pagemap.Identity) (address string, ok bool) {
	switch {
	case identity.Status == statusPublish:
		return identity.URL, identity.URL != ""
	case page.WPType.Term(), !readers.plugin(capabilityPreview):
		return "", false
	}
	link, err := readers.Plugin.PreviewLink(ctx, string(page.WPType), *page.WPID)
	if err != nil {
		return "", false
	}
	return link, link != ""
}

func (s *Service) fetch(ctx context.Context, readers Readers, owner site.Site, settings site.Settings, m siteModel, targets []renderTarget) []renderOutcome {
	install := pagemap.NewSite(owner.BaseURL)
	builders := make([]render.Builder, 0, len(settings.Builders))
	for _, rule := range settings.Builders {
		builders = append(builders, render.Builder{
			Name: rule.Name, DataAttributes: rule.DataAttributes, ClassPrefixes: rule.ClassPrefixes, ShortcodePrefixes: rule.ShortcodePrefixes,
		})
	}
	lookup := func(path string) (pagemap.WPType, bool) {
		page, found := m.index.ByPath(path)
		return page.WPType, found
	}

	outcomes := make([]renderOutcome, len(targets))
	var wait sync.WaitGroup
	slots := make(chan struct{}, max(settings.Observe.RenderConcurrency, 1))
	for i := range targets {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			outcomes[i] = s.fetchOne(ctx, readers, install, builders, lookup, targets[i])
		}(i)
	}
	wait.Wait()
	return outcomes
}

func (s *Service) fetchOne(ctx context.Context, readers Readers, install pagemap.Site, builders []render.Builder,
	lookup func(string) (pagemap.WPType, bool), target renderTarget) renderOutcome {
	outcome := renderOutcome{target: target}
	visit, err := readers.Render.Visit(ctx, target.url)
	if err != nil {
		outcome.err = err
		return outcome
	}
	outcome.status = visit.Status
	if visit.Status >= 400 {
		outcome.err = invalid("the site answered the render with an error status", "status")
		return outcome
	}

	filtered := ""
	if readers.plugin(capabilityRendered) && !target.page.WPType.Term() {
		rendered, renderErr := readers.Plugin.GetRendered(ctx, string(target.page.WPType), *target.page.WPID)
		if renderErr != nil {
			outcome.err = renderErr
			return outcome
		}
		filtered = rendered.Content
	}
	extraction, err := render.Extract(render.Input{
		Public: visit.Body, Filtered: filtered, Body: block.Parse(target.raw), Site: install, Document: target.page.Path,
		Builders: builders, Lookup: lookup,
	})
	outcome.extraction, outcome.err = extraction, err
	return outcome
}

func (s *Service) keepRenders(ctx context.Context, state *State, m siteModel, outcomes []renderOutcome) error {
	tracked, err := s.findings(ctx, state.SiteID, finding.RenderUnreachable)
	if err != nil {
		return err
	}
	now := s.now()
	return s.inUnit(ctx, func(c context.Context) error {
		for i := range outcomes {
			page := outcomes[i].target.page
			subject := finding.SubjectPage(page.ID)
			if outcomes[i].err != nil {
				state.Counts.Unreachable++
				if raiseErr := tracked.raise(c, finding.RenderUnreachable, &page.ID, subject, map[string]any{
					"path": page.Path, "status": outcomes[i].status,
				}); raiseErr != nil {
					return raiseErr
				}
				continue
			}
			if err := s.deps.Renders.Put(c, render.Record{PageID: page.ID, Extraction: outcomes[i].extraction, ExtractedAt: now}); err != nil {
				return err
			}
			if err := s.deps.Links.ReplaceForPage(c, page.ID, pagemap.SourceRender, renderLinks(page, outcomes[i].extraction, m.index, now)); err != nil {
				return err
			}
			if err := tracked.clear(c, finding.RenderUnreachable, subject); err != nil {
				return err
			}
			state.Counts.Rendered++
		}
		return nil
	})
}

func renderLinks(page pagemap.Page, extraction render.Extraction, index pagemap.Index, now timeStamp) []pagemap.PageLink {
	links := make([]pagemap.PageLink, 0, len(extraction.Links))
	for _, found := range extraction.Links {
		if found.Kind != pagemap.LinkPath || found.Path == "" || found.Region == render.RegionContent {
			continue
		}
		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: found.Path, AnchorText: found.Anchor,
			Origin: pagemap.OriginObserved, Type: found.Type, Region: string(found.Region), Source: pagemap.SourceRender, ObservedAt: now,
		}
		if target, ok := index.ByPath(found.Path); ok {
			link.ToPageID = &target.ID
		}
		if built, err := pagemap.NewPageLink(link); err == nil {
			links = append(links, built)
		}
	}
	return links
}
