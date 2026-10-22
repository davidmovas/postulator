package observe

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var modelCodes = []finding.Code{
	finding.FieldChangedOnSite, finding.BodyChangedOnSite, finding.PageTitleDiffers, finding.PageH1Differs,
}

func (s *Service) content(ctx context.Context, state *State, readers Readers, batch int) (bool, error) {
	owner, err := s.deps.Sites.Get(ctx, state.SiteID)
	if err != nil {
		return false, err
	}
	install := pagemap.NewSite(owner.BaseURL)
	items, done, err := s.pull(ctx, state, readers, install, batch)
	if err != nil {
		return false, err
	}
	if err := s.absorb(ctx, state, readers, owner, install, items); err != nil {
		return false, err
	}
	state.Counts.Pulled += len(items)
	return done, nil
}

type absorber struct {
	service  *Service
	state    *State
	readers  Readers
	owner    site.Site
	install  pagemap.Site
	rows     siteRows
	tracked  *findings
	pending  map[string]struct{}
	seen     map[string]lastSeen
	names    map[string][]string
	now      time.Time
	findings []content.Finding
}

func (s *Service) absorb(ctx context.Context, state *State, readers Readers, owner site.Site, install pagemap.Site, items []item) error {
	if len(items) == 0 {
		return nil
	}
	pages, err := s.deps.Pages.ListBySite(ctx, state.SiteID)
	if err != nil {
		return err
	}
	tracked, err := s.findings(ctx, state.SiteID, modelCodes...)
	if err != nil {
		return err
	}
	a := &absorber{
		service: s, state: state, readers: readers, owner: owner, install: install, rows: rowsOf(pages), tracked: tracked,
		pending: map[string]struct{}{}, now: s.now(),
	}
	resolved := resolveDraftPaths(items, a.rows.byWPID)
	if a.seen, err = a.lastSeen(ctx, resolved); err != nil {
		return err
	}
	batch, err := a.prepare(ctx, resolved)
	if err != nil {
		return err
	}

	err = s.inUnit(ctx, func(c context.Context) error {
		for i := range batch {
			if keepErr := a.keep(c, batch[i]); keepErr != nil {
				return keepErr
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	state.Findings = append(state.Findings, a.findings...)
	return nil
}

type prepared struct {
	it      item
	current pagemap.Page
	next    pagemap.Page
	terms   []pagemap.Term
	body    bodyRead
	known   bool
	planned bool
}

func (a *absorber) lastSeen(ctx context.Context, items []item) (map[string]lastSeen, error) {
	ids := make([]string, 0, len(items))
	for i := range items {
		if current, known, _ := match(items[i], a.rows); known {
			ids = append(ids, current.ID)
		}
	}
	seen := make(map[string]lastSeen, len(ids))
	if len(ids) == 0 {
		return seen, nil
	}
	identities, err := a.service.deps.Identity.ListBySite(ctx, a.state.SiteID)
	if err != nil {
		return nil, err
	}
	byPage := make(map[string]pagemap.Identity, len(identities))
	for i := range identities {
		byPage[identities[i].PageID] = identities[i]
	}
	fields, err := a.service.deps.Fields.ListForPages(ctx, ids)
	if err != nil {
		return nil, err
	}
	written, err := a.service.deps.Bodies.ListWrittenFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	blocks := make(map[string]bool, len(written))
	for i := range written {
		blocks[written[i].PageID] = true
	}
	for _, pageID := range ids {
		identity, held := byPage[pageID]
		seen[pageID] = seenOf(identity, held, fields[pageID], blocks[pageID])
	}
	return seen, nil
}

func (a *absorber) siteNames(ctx context.Context) (map[string][]string, error) {
	if a.names != nil {
		return a.names, nil
	}
	stored, err := a.service.deps.Fields.ListBySite(ctx, a.state.SiteID)
	if err != nil {
		return nil, err
	}
	a.names = pagemap.Names(stored)
	return a.names, nil
}

type bodyRead struct {
	current block.Snapshot
	base    block.Snapshot
	raw     string
	hash    string
	stored  bool
	fetched bool
}

func (a *absorber) prepare(ctx context.Context, items []item) ([]prepared, error) {
	batch := make([]prepared, 0, len(items))
	waiting := waitingProducts(a.rows.byPath)
	for i := range items {
		current, known, foreign := match(items[i], a.rows)
		if foreign {
			a.findings = append(a.findings, pathTaken(items[i], current))
			continue
		}
		if !known && items[i].Type == pagemap.WPProduct && waiting > 0 {
			names, err := a.siteNames(ctx)
			if err != nil {
				return nil, err
			}
			current, known = pagemap.ClaimProduct(storeCandidate(items[i]), slices.Collect(maps.Values(a.rows.byPath)), names)
			if known {
				waiting--
			}
		}
		planned := hasPlan(current, known, a.seen[current.ID])
		next := merge(current, known, planned, items[i], a.state.SiteID, a.now)
		if known && current.Path != next.Path {
			delete(a.rows.byPath, current.Path)
		}
		if !known {
			a.pending[next.ID] = struct{}{}
		}
		a.rows.remember(next)

		terms, err := a.readTerms(ctx, items[i])
		closed := closedToUser(err)
		if err != nil && !closed {
			return nil, err
		}
		body, err := a.readBody(ctx, next, items[i])
		if closedToUser(err) {
			closed, err = true, nil
		}
		if err != nil {
			return nil, err
		}
		if closed {
			a.state.Counts.Closed++
		}
		batch = append(batch, prepared{it: items[i], current: current, next: next, terms: terms, body: body, known: known, planned: planned})
	}
	return batch, nil
}

func (a *absorber) keep(ctx context.Context, p prepared) error {
	switch {
	case p.known && sameButStamps(p.current, p.next):
	case p.known:
		if err := a.service.deps.Pages.Update(ctx, p.next); err != nil {
			return err
		}
		a.state.Counts.Updated++
	default:
		if err := a.service.deps.Pages.Insert(ctx, p.next); err != nil {
			return err
		}
		delete(a.pending, p.next.ID)
		a.state.Counts.Created++
	}

	if err := a.identity(ctx, p.next, p.it); err != nil {
		return err
	}
	if err := a.terms(ctx, p.next, p.it, p.terms); err != nil {
		return err
	}
	if err := a.fields(ctx, p.next, p.it); err != nil {
		return err
	}
	if err := a.body(ctx, p.next, p.it, p.body); err != nil {
		return err
	}
	return a.plan(ctx, p.current, p.planned, p.next, p.it)
}

func sameButStamps(current, next pagemap.Page) bool {
	current.LastSyncedAt, current.UpdatedAt = next.LastSyncedAt, next.UpdatedAt
	return reflect.DeepEqual(current, next)
}

func (a *absorber) identity(ctx context.Context, page pagemap.Page, it item) error {
	identity := pagemap.Identity{
		PageID: page.ID, URL: it.URL, Status: it.Status, WPType: it.mappedType(), ObservedAt: a.now,
	}
	if it.ParentWPID != 0 {
		parent := it.ParentWPID
		identity.WPParentID = &parent
	}
	if !it.Modified.IsZero() {
		modified := it.Modified.UTC()
		identity.ModifiedGMT = &modified
	}
	return a.service.deps.Identity.Put(ctx, identity)
}

func (a *absorber) readTerms(ctx context.Context, it item) ([]pagemap.Term, error) {
	terms := it.Terms
	if !it.mappedType().Term() && a.readers.plugin(capabilityTerms) {
		listed, err := a.readers.Plugin.GetTerms(ctx, string(it.mappedType()), it.WPID)
		if err != nil {
			return nil, err
		}
		terms = listed
	}
	if terms == nil {
		terms = []pagemap.Term{}
	}
	return terms, nil
}

func (a *absorber) terms(ctx context.Context, page pagemap.Page, it item, terms []pagemap.Term) error {
	if it.mappedType().Term() || terms == nil {
		return nil
	}
	return a.service.deps.Identity.ReplaceTerms(ctx, page.ID, terms)
}

func (a *absorber) fields(ctx context.Context, page pagemap.Page, it item) error {
	values := []field.Value{field.NewValue(page.ID, field.Title, it.Title, a.now)}
	if !it.mappedType().Term() {
		values = append(values,
			field.NewValue(page.ID, field.Excerpt, it.Excerpt, a.now),
			field.NewValue(page.ID, field.H1, it.H1, a.now),
		)
	}
	if !it.Meta.Empty() {
		values = append(values,
			field.NewValue(page.ID, field.SEOTitle, it.Meta.Title, a.now),
			field.NewValue(page.ID, field.SEODescription, it.Meta.Description, a.now),
			field.NewValue(page.ID, field.SEOCanonical, it.Meta.Canonical, a.now),
		)
	}
	return a.observeFields(ctx, page, values)
}

func (a *absorber) observeFields(ctx context.Context, page pagemap.Page, values []field.Value) error {
	stored, err := a.service.deps.Fields.ListForPage(ctx, page.ID)
	if err != nil {
		return err
	}
	byKey := make(map[field.Key]field.Value, len(stored))
	for i := range stored {
		byKey[stored[i].Key] = stored[i]
	}
	moved := make([]field.Value, 0, len(values))
	for i := range values {
		if kept, exists := byKey[values[i].Key]; !exists || kept.CurrentHash != values[i].CurrentHash {
			moved = append(moved, values[i])
		}
	}
	if len(moved) > 0 {
		if err := a.service.deps.Fields.Observe(ctx, page.ID, moved); err != nil {
			return err
		}
		if stored, err = a.service.deps.Fields.ListForPage(ctx, page.ID); err != nil {
			return err
		}
		for i := range stored {
			byKey[stored[i].Key] = stored[i]
		}
	}
	for i := range values {
		value := byKey[values[i].Key]
		spec, known := field.Lookup(value.Key)
		subject := finding.SubjectPageField(page.ID, string(value.Key))
		switch {
		case !value.Changed():
		case known && spec.Denied:
			if err := a.service.deps.Fields.Agree(ctx, page.ID, value.Key, a.now); err != nil {
				return err
			}
		default:
			if err := a.tracked.raise(ctx, finding.FieldChangedOnSite, &page.ID, subject, map[string]string{
				"key": string(value.Key), "baseHash": value.BaseHash, "currentHash": value.CurrentHash,
			}); err != nil {
				return err
			}
			continue
		}
		if err := a.tracked.clear(ctx, finding.FieldChangedOnSite, subject); err != nil {
			return err
		}
	}
	return nil
}

func (a *absorber) readBody(ctx context.Context, page pagemap.Page, it item) (bodyRead, error) {
	if it.mappedType().Term() {
		return bodyRead{}, nil
	}
	current, base, found, err := a.service.deps.Bodies.Get(ctx, page.ID)
	if err != nil {
		return bodyRead{}, err
	}
	if found && current.Hash == it.ContentHash {
		return bodyRead{current: current, base: base, stored: true}, nil
	}
	switch {
	case it.HasContent:
		return bodyRead{raw: it.Content, hash: it.ContentHash, fetched: true, current: current}, nil
	case a.readers.plugin(capabilityRaw):
		read, readErr := a.readers.Plugin.GetRaw(ctx, string(it.mappedType()), it.WPID)
		if readErr != nil {
			return bodyRead{}, readErr
		}
		return bodyRead{raw: read.Content, hash: read.Hash, fetched: true, current: current}, nil
	default:
		return bodyRead{}, nil
	}
}

func (a *absorber) body(ctx context.Context, page pagemap.Page, it item, read bodyRead) error {
	switch {
	case it.mappedType().Term():
		return nil
	case read.stored:
		if a.state.Full {
			if err := a.rawLinks(ctx, page, read.current.Raw, read.current.Raw); err != nil {
				return err
			}
		}
		return a.bodyFinding(ctx, page, read.current, read.base)
	case !read.fetched:
		return a.linksFromItem(ctx, page, it)
	}

	body := block.Parse(read.raw)
	if err := a.service.deps.Bodies.Observe(ctx, block.Snapshot{
		PageID: page.ID, Side: block.Current, Raw: read.raw, Hash: read.hash, Kind: body.Kind, ObservedAt: a.now,
	}); err != nil {
		return err
	}
	if err := a.rawLinks(ctx, page, read.raw, read.current.Raw); err != nil {
		return err
	}
	current, base, _, err := a.service.deps.Bodies.Get(ctx, page.ID)
	if err != nil {
		return err
	}
	return a.bodyFinding(ctx, page, current, base)
}

func (a *absorber) bodyFinding(ctx context.Context, page pagemap.Page, current, base block.Snapshot) error {
	subject := finding.SubjectPage(page.ID)
	if current.Hash != base.Hash {
		return a.tracked.raise(ctx, finding.BodyChangedOnSite, &page.ID, subject, map[string]string{
			"baseHash": base.Hash, "currentHash": current.Hash,
		})
	}
	return a.tracked.clear(ctx, finding.BodyChangedOnSite, subject)
}

func (a *absorber) rawLinks(ctx context.Context, page pagemap.Page, raw, last string) error {
	stored, err := a.service.deps.Bodies.ListWritten(ctx, page.ID)
	if err != nil {
		return err
	}
	written := make([]string, 0, len(stored))
	for i := range stored {
		written = append(written, stored[i].BlockHash)
	}
	held, err := a.heldOurs(ctx, page, last, written)
	if err != nil {
		return err
	}

	links := make([]pagemap.PageLink, 0)
	for _, found := range block.LinksOf(raw, written) {
		path, kind := a.install.ResolveAt(page.Path, found.Href)
		if kind != pagemap.LinkPath || path == "" {
			continue
		}
		position := found.Position
		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: path, AnchorText: found.Anchor,
			Type: pagemap.LinkContextual, Region: string(regionContent),
			Source: pagemap.SourceRaw, Position: &position, Ours: found.Ours(path, held),
			Absolute: pagemap.AbsoluteHref(found.Href), ObservedAt: a.now,
		}
		if link.Ours {
			link.BlockHash = found.BlockHash
		}
		if target, ok := a.stored(path); ok {
			link.ToPageID = &target.ID
		}
		if built, buildErr := pagemap.NewPageLink(link); buildErr == nil {
			links = append(links, built)
		}
	}
	return a.service.deps.Links.ReplaceForPage(ctx, page.ID, pagemap.SourceRaw, links)
}

func (a *absorber) stored(path string) (pagemap.Page, bool) {
	target, ok := a.rows.byPath[path]
	if !ok {
		return pagemap.Page{}, false
	}
	_, waiting := a.pending[target.ID]
	return target, !waiting
}

func (a *absorber) linksFromItem(ctx context.Context, page pagemap.Page, it item) error {
	if len(it.Links) == 0 {
		return nil
	}
	links := make([]pagemap.PageLink, 0, len(it.Links))
	for i := range it.Links {
		path, normalizeErr := pagemap.NormalizePath(it.Links[i].Href)
		if normalizeErr != nil {
			continue
		}
		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: path, AnchorText: it.Links[i].Anchor,
			Type: pagemap.LinkContextual, Region: string(regionContent),
			Source: pagemap.SourceRaw, Absolute: pagemap.AbsoluteHref(it.Links[i].Href), ObservedAt: a.now,
		}
		if target, ok := a.stored(path); ok {
			link.ToPageID = &target.ID
		}
		if built, buildErr := pagemap.NewPageLink(link); buildErr == nil {
			links = append(links, built)
		}
	}
	return a.service.deps.Links.ReplaceForPage(ctx, page.ID, pagemap.SourceRaw, links)
}

func (a *absorber) heldOurs(ctx context.Context, page pagemap.Page, last string, written []string) (map[block.PlacedKey]bool, error) {
	stored, err := a.service.deps.Links.ListForPage(ctx, page.ID)
	if err != nil {
		return nil, err
	}
	resolve := func(href string) (string, bool) {
		path, kind := a.install.ResolveAt(page.Path, href)
		return path, kind == pagemap.LinkPath && path != ""
	}
	return block.HeldOurs(stored, block.LinksOf(last, written), resolve, nil), nil
}

func (a *absorber) plan(ctx context.Context, current pagemap.Page, planned bool, next pagemap.Page, it item) error {
	if err := a.planFinding(ctx, next, finding.PageTitleDiffers, planned && it.Title != "" && current.Title != "" && current.Title != it.Title, current.Title, it.Title); err != nil {
		return err
	}
	return a.planFinding(ctx, next, finding.PageH1Differs, planned && it.H1 != "" && current.H1 != "" && current.H1 != it.H1, current.H1, it.H1)
}

func (a *absorber) planFinding(ctx context.Context, page pagemap.Page, code finding.Code, differs bool, planned, found string) error {
	subject := finding.SubjectPage(page.ID)
	if differs {
		return a.tracked.raise(ctx, code, &page.ID, subject, map[string]string{"planned": planned, "site": found})
	}
	return a.tracked.clear(ctx, code, subject)
}
