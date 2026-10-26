package observe

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
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
	landed     string
}

type siteModel struct {
	pages      []pagemap.Page
	identities map[string]pagemap.Identity
	renders    map[string]render.Record
	bodies     map[string]block.Snapshot
	fields     map[string][]field.Value
	names      map[string][]string
	index      pagemap.Index
	landed     map[string]string
}

func (m *siteModel) reindex() {
	m.index = pagemap.NewIndexNamed(m.pages, m.names)
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
	stored, err := s.deps.Fields.ListBySite(ctx, siteID)
	if err != nil {
		return siteModel{}, err
	}
	m := siteModel{
		pages: pages, identities: make(map[string]pagemap.Identity, len(identities)),
		renders: make(map[string]render.Record, len(renders)), bodies: map[string]block.Snapshot{}, fields: stored,
		names: pagemap.Names(stored),
	}
	m.reindex()
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

func (s *Service) render(ctx context.Context, state *State, readers Readers, batch int) (exhausted bool, err error) {
	if readers.Render == nil {
		return true, nil
	}
	defer func() {
		if exhausted || err != nil {
			s.unstage(state)
		}
	}()
	owner, err := s.deps.Sites.Get(ctx, state.SiteID)
	if err != nil {
		return false, err
	}
	settings := owner.Settings.Normalized()
	held, err := s.staged(ctx, state, func(c context.Context) (*staged, error) {
		m, modelErr := s.model(c, state.SiteID)
		if modelErr != nil {
			return nil, modelErr
		}
		observed, observedErr := s.deps.Bodies.ObservedBySite(c, state.SiteID)
		if observedErr != nil {
			return nil, observedErr
		}
		return &staged{model: m, observed: observed}, nil
	})
	if err != nil {
		return false, err
	}
	m := held.model

	targets := make([]renderTarget, 0, batch)
	exhausted = true
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
		if !stale(state, page, identity, m, held.observed) {
			continue
		}
		url, ok := s.renderURL(ctx, readers, page, identity)
		if !ok {
			continue
		}
		targets = append(targets, renderTarget{page: page, identity: identity, url: url})
	}
	if len(targets) == 0 {
		return exhausted, nil
	}
	if targets, err = s.current(ctx, targets); err != nil || len(targets) == 0 {
		return exhausted, err
	}

	outcomes := s.fetch(ctx, readers, owner, settings, state.Technical, m, targets)
	lookUpCrumbs(ctx, readers, pagemap.NewSite(owner.BaseURL), m.index, outcomes)
	h1s, err := s.observedH1s(ctx, targets)
	if err != nil {
		return false, err
	}
	if keepErr := s.keepRenders(ctx, state, m, outcomes, h1s); keepErr != nil {
		return false, keepErr
	}
	return exhausted, nil
}

func stale(state *State, page pagemap.Page, identity pagemap.Identity, m siteModel, observed map[string]time.Time) bool {
	record, rendered := m.renders[page.ID]
	if state.Full || state.Rescan || !rendered || (identity.ModifiedGMT != nil && identity.ModifiedGMT.After(record.ExtractedAt)) {
		return true
	}
	if record.H1Source == render.H1Unknown && record.FinalPath == "" && record.ThemeH1 != render.ThemeH1Absent {
		return true
	}
	at, hasBody := observed[page.ID]
	return hasBody && at.After(record.ExtractedAt)
}

func (s *Service) current(ctx context.Context, targets []renderTarget) ([]renderTarget, error) {
	ids := make([]string, 0, len(targets))
	for i := range targets {
		ids = append(ids, targets[i].page.ID)
	}
	pages, err := s.deps.Pages.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	bodies, err := s.deps.Bodies.ListCurrent(ctx, ids)
	if err != nil {
		return nil, err
	}
	live := make(map[string]pagemap.Page, len(pages))
	for i := range pages {
		if pages[i].Status != pagemap.StatusArchived && pages[i].WPID != nil {
			live[pages[i].ID] = pages[i]
		}
	}
	raw := make(map[string]string, len(bodies))
	for i := range bodies {
		raw[bodies[i].PageID] = bodies[i].Raw
	}
	out := make([]renderTarget, 0, len(targets))
	for i := range targets {
		page, held := live[targets[i].page.ID]
		if !held {
			continue
		}
		targets[i].page, targets[i].raw = page, raw[page.ID]
		out = append(out, targets[i])
	}
	return out, nil
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
	wpID := *page.WPID
	return (e.technical.Shop != 0 && wpID == e.technical.Shop) || (e.technical.Posts != 0 && wpID == e.technical.Posts)
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
	if landed := landedElsewhere(using.install, target.url, visit.URL); landed != "" {
		outcome.landed = landed
		outcome.extraction = render.Extraction{
			Hash: block.BodyHash(visit.Body),
			Root: render.BodyState(block.Parse(target.raw), using.archive(target.page), using.dynamic),
		}
		return outcome
	}

	filtered := ""
	if readers.plugin(capabilityRendered) && !target.page.WPType.Term() {
		rendered, renderErr := readers.Plugin.GetRendered(ctx, string(target.page.WPType), *target.page.WPID)
		if renderErr != nil && !closedToUser(renderErr) {
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

func landedElsewhere(install pagemap.Site, requested, final string) string {
	if final == "" {
		return ""
	}
	asked, askedKind := install.Resolve(requested)
	landed, landedKind := install.Resolve(final)
	if askedKind != pagemap.LinkPath || landedKind != pagemap.LinkPath || landed == asked {
		return ""
	}
	return landed
}

type crumbLookups struct {
	readers Readers
	install pagemap.Site
	index   pagemap.Index
	found   map[string]Resolution
}

func lookUpCrumbs(ctx context.Context, readers Readers, install pagemap.Site, index pagemap.Index, outcomes []renderOutcome) {
	lookups := crumbLookups{readers: readers, install: install, index: index, found: make(map[string]Resolution)}
	for i := range outcomes {
		if outcomes[i].err != nil || outcomes[i].landed != "" {
			continue
		}
		outcomes[i].err = lookups.trails(ctx, outcomes[i].target.page, outcomes[i].extraction.Trails)
	}
}

func (l crumbLookups) trails(ctx context.Context, page pagemap.Page, trails []pagemap.Trail) error {
	for t := range trails {
		for c := range trails[t].Crumbs {
			crumb := &trails[t].Crumbs[c]
			if crumb.Path == "" || crumb.Path == pagemap.RootPath || crumb.Path == page.Path {
				continue
			}
			if _, modeled := l.index.ByPath(crumb.Path); modeled {
				continue
			}
			resolved, known := l.found[crumb.Path]
			if !known {
				looked, err := l.readers.Render.Resolve(ctx, l.install.URL(crumb.Path))
				if err != nil {
					return err
				}
				resolved, l.found[crumb.Path] = looked, looked
			}
			crumb.Status = resolved.Status
			if path, kind := l.install.Resolve(resolved.URL); kind == pagemap.LinkPath {
				crumb.Final = path
			}
		}
	}
	return nil
}

func (s *Service) observedH1s(ctx context.Context, targets []renderTarget) (map[string]field.Value, error) {
	ids := make([]string, 0, len(targets))
	for i := range targets {
		ids = append(ids, targets[i].page.ID)
	}
	stored, err := s.deps.Fields.ListForPages(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]field.Value, len(ids))
	for pageID, values := range stored {
		for i := range values {
			if values[i].Key == field.H1 {
				out[pageID] = values[i]
			}
		}
	}
	return out, nil
}

func (s *Service) keepH1(ctx context.Context, page pagemap.Page, extraction render.Extraction, was field.Value, held bool, now time.Time) error {
	if extraction.H1Source == render.H1Unknown {
		return nil
	}
	value := field.NewValue(page.ID, field.H1, extraction.H1, now)
	if held && was.CurrentHash == value.CurrentHash {
		return nil
	}
	if err := s.deps.Fields.Observe(ctx, page.ID, []field.Value{value}); err != nil {
		return err
	}
	if err := s.deps.Fields.Agree(ctx, page.ID, field.H1, now); err != nil {
		return err
	}
	return s.deps.Pages.FollowObservedH1(ctx, page.ID, was.Current, extraction.H1, now)
}

func (s *Service) keepRenders(ctx context.Context, state *State, m siteModel, outcomes []renderOutcome, h1s map[string]field.Value) error {
	tracked, err := s.findings(ctx, state.SiteID, finding.RenderUnreachable, finding.RenderRedirected)
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
			record := render.Record{PageID: page.ID, Extraction: outcomes[i].extraction, FinalPath: outcomes[i].landed, ExtractedAt: now}
			if err := s.deps.Renders.Put(c, record); err != nil {
				return err
			}
			was, held := h1s[page.ID]
			if err := s.keepH1(c, page, outcomes[i].extraction, was, held, now); err != nil {
				return err
			}
			if err := s.deps.Links.ReplaceForPage(c, page.ID, pagemap.SourceRender, renderLinks(page, outcomes[i].extraction, m.index, now)); err != nil {
				return err
			}
			if err := tracked.clear(c, finding.RenderUnreachable, subject); err != nil {
				return err
			}
			if outcomes[i].landed != "" {
				if err := tracked.raise(c, finding.RenderRedirected, &page.ID, subject, map[string]string{"target": outcomes[i].landed}); err != nil {
					return err
				}
			} else if err := tracked.clear(c, finding.RenderRedirected, subject); err != nil {
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
			Type: found.Type, Region: string(found.Region), Source: pagemap.SourceRender, ObservedAt: now,
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
