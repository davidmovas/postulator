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
	m := siteModel{
		pages: pages, identities: make(map[string]pagemap.Identity, len(identities)),
		renders: make(map[string]render.Record, len(renders)), bodies: map[string]block.Snapshot{}, index: pagemap.NewIndex(pages),
	}
	for i := range identities {
		m.identities[identities[i].PageID] = identities[i]
	}
	for i := range renders {
		m.renders[renders[i].PageID] = renders[i]
	}
	slices.SortFunc(m.pages, func(a, b pagemap.Page) int { return strings.Compare(a.ID, b.ID) })
	return m, nil
}

func (s *Service) currentBodies(ctx context.Context, siteID string) (map[string]block.Snapshot, error) {
	bodies, err := s.deps.Bodies.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	current := make(map[string]block.Snapshot, len(bodies)/2)
	for i := range bodies {
		if bodies[i].Side == block.Current {
			current[bodies[i].PageID] = bodies[i]
		}
	}
	return current, nil
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
		if !observed || (state.Full && archive(page, identity, state, readers)) {
			continue
		}
		raw, stale, err := s.candidate(ctx, state, page, identity, m)
		if err != nil {
			return false, err
		}
		if !stale {
			continue
		}
		url, ok := s.renderURL(ctx, readers, page, identity)
		if !ok {
			continue
		}
		targets = append(targets, renderTarget{page: page, identity: identity, url: url, raw: raw})
	}
	if len(targets) == 0 {
		return exhausted, nil
	}

	outcomes := s.fetch(ctx, readers, owner, settings, state.Technical, m, targets)
	if err := s.keepRenders(ctx, state, m, outcomes); err != nil {
		return false, err
	}
	return exhausted, nil
}

func (s *Service) candidate(ctx context.Context, state *State, page pagemap.Page, identity pagemap.Identity, m siteModel) (raw string, stale bool, err error) {
	record, rendered := m.renders[page.ID]
	stale = state.Full || state.Rescan || !rendered || (identity.ModifiedGMT != nil && identity.ModifiedGMT.After(record.ExtractedAt))
	current, _, hasBody, err := s.deps.Bodies.Get(ctx, page.ID)
	if err != nil {
		return "", false, err
	}
	if !stale && hasBody && current.ObservedAt.After(record.ExtractedAt) {
		stale = true
	}
	return current.Raw, stale, nil
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

type extraction struct {
	install   pagemap.Site
	builders  []render.Builder
	dynamic   []string
	technical site.TechnicalPages
	lookup    func(string) (pagemap.WPType, bool)
}

func (e extraction) archive(page pagemap.Page) bool {
	if page.WPType.Term() {
		return true
	}
	if page.WPID == nil {
		return false
	}
	id := *page.WPID
	return (e.technical.Shop != 0 && id == e.technical.Shop) || (e.technical.Posts != 0 && id == e.technical.Posts)
}

func (s *Service) fetch(ctx context.Context, readers Readers, owner site.Site, settings site.Settings, technical site.TechnicalPages,
	m siteModel, targets []renderTarget) []renderOutcome {
	using := extraction{
		install:   pagemap.NewSite(owner.BaseURL),
		builders:  make([]render.Builder, 0, len(settings.Builders)),
		dynamic:   append(slices.Clone(settings.TechnicalBlocks), settings.TechnicalShortcodes...),
		technical: technical,
		lookup: func(path string) (pagemap.WPType, bool) {
			page, found := m.index.ByPath(path)
			return page.WPType, found
		},
	}
	for _, rule := range settings.Builders {
		using.builders = append(using.builders, render.Builder{
			Name: rule.Name, DataAttributes: rule.DataAttributes, ClassPrefixes: rule.ClassPrefixes, ShortcodePrefixes: rule.ShortcodePrefixes,
		})
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
			outcomes[i] = s.fetchOne(ctx, readers, using, targets[i])
		}(i)
	}
	wait.Wait()
	return outcomes
}

func (s *Service) fetchOne(ctx context.Context, readers Readers, using extraction, target renderTarget) renderOutcome {
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
	extracted, err := render.Extract(render.Input{
		Public: visit.Body, Filtered: filtered, Body: block.Parse(target.raw), Site: using.install, Document: target.page.Path,
		Builders: using.builders, Lookup: using.lookup, Archive: using.archive(target.page), Dynamic: using.dynamic,
	})
	outcome.extraction, outcome.err = extracted, err
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
