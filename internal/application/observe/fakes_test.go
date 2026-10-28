package observe_test

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type unitGuard struct {
	mu          sync.Mutex
	open        bool
	readsInUnit int
}

func (g *unitGuard) enter() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.open = true
}

func (g *unitGuard) leave() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.open = false
}

func (g *unitGuard) read() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open {
		g.readsInUnit++
	}
}

func (g *unitGuard) inUnit() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.readsInUnit
}

type stores struct {
	mu            sync.Mutex
	guard         *unitGuard
	sites         map[string]site.Site
	pages         map[string]pagemap.Page
	identities    map[string]pagemap.Identity
	terms         map[string][]pagemap.Term
	fields        map[string]map[field.Key]field.Value
	bodies        map[string]map[block.Side]block.Snapshot
	written       map[string][]block.Written
	renders       map[string]render.Record
	links         map[string]map[pagemap.LinkSource][]pagemap.PageLink
	findings      map[string]finding.Finding
	entities      []graph.Entity
	edges         []graph.Edge
	published     []events.Type
	bodyLists     int
	bodyGets      int
	renderLists   int
	identityLists int
	interleave    func()
	types         *typesFake
	maps          mapRefresher
}

func (s *stores) interleaved() {
	s.mu.Lock()
	hook := s.interleave
	s.interleave = nil
	s.mu.Unlock()
	if hook != nil {
		hook()
	}
}

func (s *stores) bodyListings() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bodyLists
}

func (s *stores) modelReads() (renders, identities, bodies int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.renderLists, s.identityLists, s.bodyGets
}

func newStores(owner site.Site) *stores {
	guard := &unitGuard{}
	return &stores{
		maps:       mapRefresher{guard: guard},
		guard:      guard,
		sites:      map[string]site.Site{owner.ID: owner},
		pages:      map[string]pagemap.Page{},
		identities: map[string]pagemap.Identity{},
		terms:      map[string][]pagemap.Term{},
		fields:     map[string]map[field.Key]field.Value{},
		bodies:     map[string]map[block.Side]block.Snapshot{},
		written:    map[string][]block.Written{},
		renders:    map[string]render.Record{},
		links:      map[string]map[pagemap.LinkSource][]pagemap.PageLink{},
		findings:   map[string]finding.Finding{},
		types:      &typesFake{types: map[string][]pagetype.Type{}, rules: map[string][]pagetype.Rule{}},
	}
}

func (s *stores) Get(_ context.Context, id string) (site.Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner, ok := s.sites[id]
	if !ok {
		return site.Site{}, errors.New(errors.NotFound, "no such site")
	}
	return owner, nil
}

func (s *stores) UpdateObservation(_ context.Context, id string, technical site.TechnicalPages, observation site.ObservationState, at time.Time) error {
	s.interleaved()
	s.mu.Lock()
	defer s.mu.Unlock()
	owner, ok := s.sites[id]
	if !ok {
		return errors.New(errors.NotFound, "no such site")
	}
	owner.Settings.Technical = technical
	owner.Settings.Observation = observation
	owner.UpdatedAt = at
	s.sites[id] = owner
	return nil
}

type pageStore struct{ s *stores }

func (p pageStore) ListBySite(_ context.Context, siteID string) ([]pagemap.Page, error) {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	out := make([]pagemap.Page, 0, len(p.s.pages))
	for id := range p.s.pages {
		if p.s.pages[id].SiteID == siteID {
			out = append(out, p.s.pages[id])
		}
	}
	return out, nil
}

func (p pageStore) ListByIDs(_ context.Context, ids []string) ([]pagemap.Page, error) {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	out := make([]pagemap.Page, 0, len(ids))
	for _, id := range ids {
		if page, ok := p.s.pages[id]; ok {
			out = append(out, page)
		}
	}
	return out, nil
}

func (p pageStore) Get(_ context.Context, id string) (pagemap.Page, error) {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	page, ok := p.s.pages[id]
	if !ok {
		return pagemap.Page{}, errors.New(errors.NotFound, "no such page")
	}
	return page, nil
}

func (p pageStore) Insert(_ context.Context, page pagemap.Page) error {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	for id := range p.s.pages {
		if p.s.pages[id].SiteID == page.SiteID && p.s.pages[id].Path == page.Path {
			return errors.New(errors.Conflict, "path taken")
		}
	}
	p.s.pages[page.ID] = page
	return nil
}

func (p pageStore) Update(_ context.Context, page pagemap.Page) error {
	p.s.interleaved()
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	stored, ok := p.s.pages[page.ID]
	if !ok {
		return errors.New(errors.NotFound, "no such page")
	}
	page.PageTypeID, page.TypeSource, page.TypeReason = stored.PageTypeID, stored.TypeSource, stored.TypeReason
	p.s.pages[page.ID] = page
	return nil
}

func (p pageStore) Classify(_ context.Context, pageID string, was, now pagetype.Assignment, at time.Time) (bool, error) {
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	stored, ok := p.s.pages[pageID]
	if !ok || pagetype.AssignmentOf(stored) != was {
		return false, nil
	}
	typeID := now.TypeID
	stored.PageTypeID, stored.TypeSource, stored.TypeReason, stored.TypeRuleID, stored.UpdatedAt = &typeID, now.Source, now.Reason, now.RuleID, at
	p.s.pages[pageID] = stored
	return true, nil
}

func (s *stores) setType(pageID, typeID string, reason pagemap.TypeReason) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.pages[pageID]
	stored.PageTypeID, stored.TypeSource, stored.TypeReason, stored.TypeRuleID = &typeID, pagemap.TypeSourceManual, reason, ""
	s.pages[pageID] = stored
}

func (s *stores) setLegacyType(pageID, typeID string, reason pagemap.TypeReason) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.pages[pageID]
	stored.PageTypeID, stored.TypeSource, stored.TypeReason, stored.TypeRuleID = &typeID, pagemap.TypeSourceAuto, reason, ""
	s.pages[pageID] = stored
}

type typesFake struct {
	mu     sync.Mutex
	types  map[string][]pagetype.Type
	scopes []field.Scope
	rules  map[string][]pagetype.Rule
}

func (f *typesFake) Seed(_ context.Context, siteID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.types[siteID]) > 0 {
		return nil
	}
	for i, preset := range pagetype.Seeded() {
		kind := pagetype.Type{
			ID: siteID + "-" + string(preset), SiteID: siteID, Name: preset.DefaultName(), Preset: preset,
			CreatedAt: earlier.Add(time.Duration(i) * time.Second), UpdatedAt: earlier,
		}
		f.types[siteID] = append(f.types[siteID], kind)
		f.scopes = append(f.scopes, pagetype.DefaultScopes(preset, kind.ID)...)
	}
	next := 0
	f.rules[siteID] = pagetype.DefaultRules(pagetype.NewSet(f.types[siteID], nil), siteID, earlier, func() string {
		next++
		return siteID + "-rule-" + strconv.Itoa(next)
	})
	return nil
}

func (f *typesFake) Rules(_ context.Context, siteID string) ([]pagetype.Rule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.rules[siteID]), nil
}

func (f *typesFake) changeRule(siteID string, position int, change func(*pagetype.Rule)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.rules[siteID] {
		if f.rules[siteID][i].Position == position {
			change(&f.rules[siteID][i])
		}
	}
}

func (f *typesFake) Set(_ context.Context, siteID string) (pagetype.Set, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return pagetype.NewSet(slices.Clone(f.types[siteID]), slices.Clone(f.scopes)), nil
}

func (f *typesFake) add(scope field.Scope) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scopes = append(f.scopes, scope)
}

func (p pageStore) UpdateDerived(_ context.Context, page pagemap.Page) error {
	p.s.interleaved()
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	stored, ok := p.s.pages[page.ID]
	if !ok {
		return errors.New(errors.NotFound, "no such page")
	}
	stored.ParentPageID, stored.ParentReason, stored.ParentMethod = page.ParentPageID, page.ParentReason, page.ParentMethod
	stored.TechnicalReason, stored.ReadOnlyReason = page.TechnicalReason, page.ReadOnlyReason
	stored.Status, stored.UpdatedAt = page.Status, page.UpdatedAt
	p.s.pages[page.ID] = stored
	return nil
}

func (p pageStore) FollowObservedH1(_ context.Context, pageID, was, now string, at time.Time) error {
	p.s.interleaved()
	p.s.mu.Lock()
	defer p.s.mu.Unlock()
	stored, ok := p.s.pages[pageID]
	if !ok || stored.H1 != was || stored.H1 == now {
		return nil
	}
	stored.H1, stored.UpdatedAt = now, at
	p.s.pages[pageID] = stored
	return nil
}

type identityStore struct{ s *stores }

func (i identityStore) Put(_ context.Context, identity pagemap.Identity) error {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	i.s.identities[identity.PageID] = identity
	return nil
}

func (i identityStore) Get(_ context.Context, pageID string) (pagemap.Identity, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	identity, ok := i.s.identities[pageID]
	if !ok {
		return pagemap.Identity{}, errors.New(errors.NotFound, "no identity")
	}
	return identity, nil
}

func (i identityStore) ListBySite(_ context.Context, siteID string) ([]pagemap.Identity, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	i.s.identityLists++
	out := make([]pagemap.Identity, 0)
	for pageID := range i.s.identities {
		if i.s.pages[pageID].SiteID == siteID {
			out = append(out, i.s.identities[pageID])
		}
	}
	return out, nil
}

func (i identityStore) ListForPages(_ context.Context, pageIDs []string) ([]pagemap.Identity, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	out := make([]pagemap.Identity, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		if identity, ok := i.s.identities[pageID]; ok {
			out = append(out, identity)
		}
	}
	return out, nil
}

func (i identityStore) ReplaceTerms(_ context.Context, pageID string, terms []pagemap.Term) error {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	i.s.terms[pageID] = terms
	return nil
}

func (i identityStore) ListTerms(_ context.Context, pageID string) ([]pagemap.Term, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	return i.s.terms[pageID], nil
}

func (i identityStore) ListTermsBySite(_ context.Context, siteID string) (map[string][]pagemap.Term, error) {
	i.s.mu.Lock()
	defer i.s.mu.Unlock()
	out := make(map[string][]pagemap.Term)
	for pageID, terms := range i.s.terms {
		if i.s.pages[pageID].SiteID == siteID && len(terms) > 0 {
			out[pageID] = slices.Clone(terms)
		}
	}
	return out, nil
}

type fieldStore struct{ s *stores }

func (f fieldStore) Observe(_ context.Context, pageID string, values []field.Value) error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if f.s.fields[pageID] == nil {
		f.s.fields[pageID] = map[field.Key]field.Value{}
	}
	for i := range values {
		value := values[i]
		kept, exists := f.s.fields[pageID][value.Key]
		if exists {
			value.Base, value.BaseHash = kept.Base, kept.BaseHash
			value.WrittenAt, value.WrittenRunID = kept.WrittenAt, kept.WrittenRunID
		}
		f.s.fields[pageID][value.Key] = value
	}
	return nil
}

func (f fieldStore) Agree(_ context.Context, pageID string, key field.Key, at time.Time) error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	value, ok := f.s.fields[pageID][key]
	if !ok {
		return errors.New(errors.NotFound, "no such field")
	}
	value.Base, value.BaseHash, value.ObservedAt = value.Current, value.CurrentHash, at
	f.s.fields[pageID][key] = value
	return nil
}

func (f fieldStore) ListForPage(_ context.Context, pageID string) ([]field.Value, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	out := make([]field.Value, 0, len(f.s.fields[pageID]))
	for key := range f.s.fields[pageID] {
		out = append(out, f.s.fields[pageID][key])
	}
	return out, nil
}

func (f fieldStore) ListForPages(_ context.Context, pageIDs []string) (map[string][]field.Value, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	out := make(map[string][]field.Value, len(pageIDs))
	for _, pageID := range pageIDs {
		for key := range f.s.fields[pageID] {
			out[pageID] = append(out[pageID], f.s.fields[pageID][key])
		}
	}
	return out, nil
}

func (f fieldStore) ListBySite(_ context.Context, siteID string) (map[string][]field.Value, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	out := make(map[string][]field.Value)
	for pageID, values := range f.s.fields {
		if f.s.pages[pageID].SiteID != siteID {
			continue
		}
		for key := range values {
			out[pageID] = append(out[pageID], values[key])
		}
	}
	return out, nil
}

type bodyStore struct{ s *stores }

func (b bodyStore) Observe(_ context.Context, snapshot block.Snapshot) error {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	if b.s.bodies[snapshot.PageID] == nil {
		b.s.bodies[snapshot.PageID] = map[block.Side]block.Snapshot{}
	}
	snapshot.Side = block.Current
	b.s.bodies[snapshot.PageID][block.Current] = snapshot
	if _, hasBase := b.s.bodies[snapshot.PageID][block.Base]; !hasBase {
		base := snapshot
		base.Side = block.Base
		b.s.bodies[snapshot.PageID][block.Base] = base
	}
	return nil
}

func (b bodyStore) Get(_ context.Context, pageID string) (current, base block.Snapshot, found bool, err error) {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	b.s.bodyGets++
	sides, ok := b.s.bodies[pageID]
	if !ok {
		return block.Snapshot{}, block.Snapshot{}, false, nil
	}
	return sides[block.Current], sides[block.Base], true, nil
}

func (b bodyStore) ListCurrent(_ context.Context, pageIDs []string) ([]block.Snapshot, error) {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	out := make([]block.Snapshot, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		if current, ok := b.s.bodies[pageID][block.Current]; ok {
			out = append(out, current)
		}
	}
	return out, nil
}

func (b bodyStore) ObservedBySite(_ context.Context, siteID string) (map[string]time.Time, error) {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	out := make(map[string]time.Time)
	for pageID, sides := range b.s.bodies {
		if current, ok := sides[block.Current]; ok && b.s.pages[pageID].SiteID == siteID {
			out[pageID] = current.ObservedAt
		}
	}
	return out, nil
}

func (b bodyStore) ListBySite(_ context.Context, siteID string) ([]block.Snapshot, error) {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	b.s.bodyLists++
	out := make([]block.Snapshot, 0)
	for pageID, sides := range b.s.bodies {
		if b.s.pages[pageID].SiteID != siteID {
			continue
		}
		for side := range sides {
			out = append(out, sides[side])
		}
	}
	return out, nil
}

func (b bodyStore) ListWritten(_ context.Context, pageID string) ([]block.Written, error) {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	return b.s.written[pageID], nil
}

func (b bodyStore) ListWrittenFor(_ context.Context, pageIDs []string) ([]block.Written, error) {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	out := make([]block.Written, 0)
	for _, pageID := range pageIDs {
		out = append(out, b.s.written[pageID]...)
	}
	return out, nil
}

type renderStore struct{ s *stores }

func (r renderStore) Put(_ context.Context, record render.Record) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.renders[record.PageID] = record
	return nil
}

func (r renderStore) Get(_ context.Context, pageID string) (render.Record, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	record, ok := r.s.renders[pageID]
	if !ok {
		return render.Record{}, errors.New(errors.NotFound, "no render")
	}
	return record, nil
}

func (r renderStore) ListBySite(_ context.Context, siteID string) ([]render.Record, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	r.s.renderLists++
	out := make([]render.Record, 0)
	for pageID := range r.s.renders {
		if r.s.pages[pageID].SiteID == siteID {
			out = append(out, r.s.renders[pageID])
		}
	}
	return out, nil
}

func (r renderStore) ListForPages(_ context.Context, pageIDs []string) ([]render.Record, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	out := make([]render.Record, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		if record, ok := r.s.renders[pageID]; ok {
			out = append(out, record)
		}
	}
	return out, nil
}

type linkStore struct{ s *stores }

func (l linkStore) ReplaceForPage(_ context.Context, pageID string, source pagemap.LinkSource, links []pagemap.PageLink) error {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	if l.s.links[pageID] == nil {
		l.s.links[pageID] = map[pagemap.LinkSource][]pagemap.PageLink{}
	}
	l.s.links[pageID][source] = links
	return nil
}

func (l linkStore) ListForPage(_ context.Context, pageID string) ([]pagemap.PageLink, error) {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	out := make([]pagemap.PageLink, 0)
	for _, source := range []pagemap.LinkSource{pagemap.SourceRaw, pagemap.SourceRender} {
		out = append(out, l.s.links[pageID][source]...)
	}
	return out, nil
}

func (l linkStore) ListedProductsBySite(_ context.Context, siteID string) (map[string]int, error) {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	out := map[string]int{}
	for pageID, sources := range l.s.links {
		if l.s.pages[pageID].SiteID != siteID {
			continue
		}
		seen := map[string]bool{}
		rendered := sources[pagemap.SourceRender]
		for i := range rendered {
			link := &rendered[i]
			if link.Type != pagemap.LinkListing || link.ToPageID == nil || seen[*link.ToPageID] {
				continue
			}
			if target, held := l.s.pages[*link.ToPageID]; held && target.WPType == pagemap.WPProduct {
				seen[*link.ToPageID] = true
				out[pageID]++
			}
		}
	}
	return out, nil
}

func (l linkStore) ListBySite(_ context.Context, siteID string) ([]pagemap.PageLink, error) {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	out := make([]pagemap.PageLink, 0)
	for pageID, sources := range l.s.links {
		if l.s.pages[pageID].SiteID != siteID {
			continue
		}
		for _, source := range []pagemap.LinkSource{pagemap.SourceRaw, pagemap.SourceRender} {
			out = append(out, sources[source]...)
		}
	}
	return out, nil
}

type findingStore struct{ s *stores }

func findingKey(siteID string, code finding.Code, subject string) string {
	return siteID + "|" + string(code) + "|" + subject
}

func (f findingStore) Upsert(_ context.Context, record finding.Finding) error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	key := findingKey(record.SiteID, record.Code, record.SubjectKey)
	kept, exists := f.s.findings[key]
	switch {
	case !exists:
		record.Status = finding.Open
	case kept.Status == finding.Dismissed && string(kept.Subject) == string(record.Subject):
		return nil
	default:
		record.ID = kept.ID
		record.CreatedAt = kept.CreatedAt
		record.Status = finding.Open
		record.Resolution, record.ResolvedBy = "", ""
	}
	f.s.findings[key] = record
	return nil
}

func (f findingStore) Clear(_ context.Context, siteID string, code finding.Code, subjectKey, resolvedBy, resolution string, at time.Time) error {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	key := findingKey(siteID, code, subjectKey)
	kept, exists := f.s.findings[key]
	if !exists || kept.Status != finding.Open {
		return nil
	}
	kept.Status = finding.Resolved
	kept.ResolvedBy, kept.Resolution, kept.UpdatedAt = resolvedBy, resolution, at
	f.s.findings[key] = kept
	return nil
}

func (f findingStore) OpenBySite(_ context.Context, siteID string, code finding.Code) ([]finding.Finding, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	out := make([]finding.Finding, 0)
	for key := range f.s.findings {
		if record := f.s.findings[key]; record.SiteID == siteID && record.Code == code && record.Status == finding.Open {
			out = append(out, record)
		}
	}
	return out, nil
}

func (f findingStore) ListBySite(_ context.Context, siteID string, status finding.Status) ([]finding.Finding, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	out := make([]finding.Finding, 0)
	for key := range f.s.findings {
		if record := f.s.findings[key]; record.SiteID == siteID && (status == "" || record.Status == status) {
			out = append(out, record)
		}
	}
	return out, nil
}

func (f findingStore) ListForPage(_ context.Context, pageID string) ([]finding.Finding, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	out := make([]finding.Finding, 0)
	for key := range f.s.findings {
		if record := f.s.findings[key]; record.PageID != nil && *record.PageID == pageID {
			out = append(out, record)
		}
	}
	return out, nil
}

type unit struct{ guard *unitGuard }

func (u unit) Do(ctx context.Context, fn func(context.Context) error) error {
	u.guard.enter()
	defer u.guard.leave()
	return fn(ctx)
}

type bus struct{ s *stores }

func (b bus) Publish(eventType events.Type, _ any) error {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	b.s.published = append(b.s.published, eventType)
	return nil
}

func (s *stores) deps(clk clock.Clock) observe.Deps {
	return observe.Deps{
		Sites: s, Pages: pageStore{s}, Identity: identityStore{s}, Fields: fieldStore{s}, Bodies: bodyStore{s},
		Renders: renderStore{s}, Links: linkStore{s}, Findings: findingStore{s}, Types: s.types, UnitOfWork: unit{s.guard}, Publisher: bus{s}, Clock: clk,
		Maps: &s.maps, Entities: graphStore{s}, Edges: edgeList{s},
	}
}

type graphStore struct{ s *stores }

func (g graphStore) ListBySite(context.Context, string) ([]graph.Entity, error) {
	g.s.mu.Lock()
	defer g.s.mu.Unlock()
	return slices.Clone(g.s.entities), nil
}

type edgeList struct{ s *stores }

func (e edgeList) ListBySite(context.Context, string) ([]graph.Edge, error) {
	e.s.mu.Lock()
	defer e.s.mu.Unlock()
	return slices.Clone(e.s.edges), nil
}

type mapRefresher struct {
	mu     sync.Mutex
	guard  *unitGuard
	sites  []string
	inUnit int
}

func (r *mapRefresher) Refresh(_ context.Context, siteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sites = append(r.sites, siteID)
	r.guard.mu.Lock()
	defer r.guard.mu.Unlock()
	if r.guard.open {
		r.inUnit++
	}
	return nil
}

func (r *mapRefresher) refreshed() (sites []string, inUnit int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.sites), r.inUnit
}

type pluginFake struct {
	mu       sync.Mutex
	guard    *unitGuard
	items    []observe.ContentItem
	raws     map[int64]observe.Raw
	rendered map[int64]observe.Rendered
	meta     map[int64]observe.SEOMeta
	terms    map[int64][]pagemap.Term
	pages    site.TechnicalPages
	previews map[int64]string
	queries  []observe.ContentQuery
	custom   map[int64]map[string]string
	asked    map[int64][]string
	rawsRead []int64
	termRead []int64
	failRaw  map[int64]error
	slow     time.Duration
	active   int
	peak     int
}

func (p *pluginFake) enter() {
	p.mu.Lock()
	p.active++
	p.peak = max(p.peak, p.active)
	slow := p.slow
	p.mu.Unlock()
	time.Sleep(slow)
}

func (p *pluginFake) leave() {
	p.mu.Lock()
	p.active--
	p.mu.Unlock()
}

func (p *pluginFake) reads() (raws, terms []int64, peak int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.rawsRead), slices.Clone(p.termRead), p.peak
}

func (p *pluginFake) GetMeta(_ context.Context, _ string, id int64, keys []string) (map[string]string, error) {
	p.guard.read()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.asked == nil {
		p.asked = map[int64][]string{}
	}
	p.asked[id] = append(p.asked[id], keys...)
	out := map[string]string{}
	for _, key := range keys {
		if value, held := p.custom[id][key]; held {
			out[key] = value
		}
	}
	return out, nil
}

func (p *pluginFake) askedFor(id int64) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.asked[id])
}

func (p *pluginFake) ListContent(_ context.Context, q observe.ContentQuery) (observe.ContentPage, error) {
	p.guard.read()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queries = append(p.queries, q)
	matching := make([]observe.ContentItem, 0)
	for i := range p.items {
		if !contains(q.Types, p.items[i].Type) {
			continue
		}
		if q.Since != nil && !p.items[i].Modified.After(*q.Since) {
			continue
		}
		matching = append(matching, p.items[i])
	}
	start := 0
	if q.Cursor != "" {
		parsed, err := strconv.Atoi(q.Cursor)
		if err != nil {
			return observe.ContentPage{}, err
		}
		start = parsed
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	end := min(start+limit, len(matching))
	page := observe.ContentPage{Items: matching[start:end]}
	if end < len(matching) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

func (p *pluginFake) GetRaw(_ context.Context, _ string, id int64) (observe.Raw, error) {
	p.guard.read()
	p.enter()
	defer p.leave()
	p.mu.Lock()
	p.rawsRead = append(p.rawsRead, id)
	failure := p.failRaw[id]
	p.mu.Unlock()
	if failure != nil {
		return observe.Raw{}, failure
	}
	raw, ok := p.raws[id]
	if !ok {
		return observe.Raw{}, errors.New(errors.NotFound, "no raw")
	}
	return raw, nil
}

func (p *pluginFake) GetRendered(_ context.Context, _ string, id int64) (observe.Rendered, error) {
	p.guard.read()
	rendered, ok := p.rendered[id]
	if !ok {
		return observe.Rendered{}, errors.New(errors.NotFound, "no rendered")
	}
	return rendered, nil
}

func (p *pluginFake) GetSEOMeta(_ context.Context, _ string, id int64) (observe.SEOMeta, error) {
	p.guard.read()
	return p.meta[id], nil
}

func (p *pluginFake) GetTerms(_ context.Context, _ string, id int64) ([]pagemap.Term, error) {
	p.guard.read()
	p.enter()
	defer p.leave()
	p.mu.Lock()
	p.termRead = append(p.termRead, id)
	p.mu.Unlock()
	return p.terms[id], nil
}

func (p *pluginFake) Settings(_ context.Context) (site.TechnicalPages, error) {
	p.guard.read()
	return p.pages, nil
}

func (p *pluginFake) PreviewLink(_ context.Context, _ string, id int64) (string, error) {
	p.guard.read()
	link, ok := p.previews[id]
	if !ok {
		return "", errors.New(errors.NotFound, "no preview")
	}
	return link, nil
}

type coreFake struct {
	guard    *unitGuard
	types    []observe.PostType
	items    map[string][]observe.Item
	settings observe.SiteSettings
}

func (c *coreFake) PostTypes(_ context.Context) ([]observe.PostType, error) {
	c.guard.read()
	return c.types, nil
}

func (c *coreFake) ListItems(_ context.Context, postType string, q observe.ItemQuery) (observe.ItemPage, error) {
	c.guard.read()
	items := c.items[postType]
	perPage := q.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	start := (q.Page - 1) * perPage
	if start >= len(items) {
		return observe.ItemPage{}, nil
	}
	end := min(start+perPage, len(items))
	return observe.ItemPage{Items: items[start:end], HasMore: end < len(items)}, nil
}

func (c *coreFake) SiteSettings(_ context.Context) (observe.SiteSettings, error) {
	c.guard.read()
	return c.settings, nil
}

type storeFake struct {
	mu         sync.Mutex
	guard      *unitGuard
	products   []observe.Product
	variations map[int64][]observe.Variation
	pages      site.TechnicalPages
	queries    []observe.ProductQuery
	asked      []int64
}

func (s *storeFake) askedVariations() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.asked)
}

func (s *storeFake) ListProducts(_ context.Context, q observe.ProductQuery) (observe.ProductPage, error) {
	s.guard.read()
	s.queries = append(s.queries, q)
	matching := make([]observe.Product, 0)
	for i := range s.products {
		if q.ModifiedAfter != nil && !s.products[i].Modified.After(*q.ModifiedAfter) {
			continue
		}
		matching = append(matching, s.products[i])
	}
	perPage := q.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	start := (q.Page - 1) * perPage
	if start >= len(matching) {
		return observe.ProductPage{}, nil
	}
	end := min(start+perPage, len(matching))
	return observe.ProductPage{Items: matching[start:end], HasMore: end < len(matching)}, nil
}

func (s *storeFake) ListVariations(_ context.Context, productID int64, page int) (observe.VariationPage, error) {
	s.guard.read()
	s.mu.Lock()
	s.asked = append(s.asked, productID)
	s.mu.Unlock()
	if page > 1 {
		return observe.VariationPage{}, nil
	}
	return observe.VariationPage{Items: s.variations[productID]}, nil
}

func (s *storeFake) StorePages(_ context.Context) (site.TechnicalPages, error) {
	s.guard.read()
	return s.pages, nil
}

type visitorFake struct {
	mu        sync.Mutex
	guard     *unitGuard
	pages     map[string]observe.Visit
	failed    map[string]bool
	redirects map[string]string
	visits    []string
	lookups   []string
	active    int
	peak      int
	failures  int
}

func (v *visitorFake) Resolve(_ context.Context, url string) (observe.Resolution, error) {
	v.guard.read()
	v.mu.Lock()
	defer v.mu.Unlock()
	v.lookups = append(v.lookups, url)
	if v.failed[url] {
		return observe.Resolution{}, errors.New(errors.External, "the site did not answer "+url)
	}
	if target, moved := v.redirects[url]; moved {
		url = target
	}
	if _, ok := v.pages[url]; ok {
		return observe.Resolution{URL: url, Status: 200}, nil
	}
	return observe.Resolution{URL: url, Status: 404}, nil
}

func (v *visitorFake) lookedUp() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.lookups)
}

func (v *visitorFake) Visit(_ context.Context, url string) (observe.Visit, error) {
	v.guard.read()
	v.mu.Lock()
	v.visits = append(v.visits, url)
	v.active++
	if v.active > v.peak {
		v.peak = v.active
	}
	v.mu.Unlock()
	time.Sleep(2 * time.Millisecond)
	v.mu.Lock()
	defer v.mu.Unlock()
	v.active--
	if v.failed[url] {
		v.failures++
		return observe.Visit{}, errors.New(errors.External, "the site did not answer "+url+"?_="+strconv.Itoa(v.failures))
	}
	landed := url
	if target, moved := v.redirects[url]; moved {
		landed = target
	}
	visit, ok := v.pages[landed]
	if !ok {
		return observe.Visit{Status: 404, Body: "<html><body><h1>Not found</h1></body></html>", URL: landed}, nil
	}
	visit.URL = landed
	return visit, nil
}

func contains(list []string, value string) bool {
	return slices.Contains(list, value)
}
