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
	batch, err := a.prepare(ctx, resolveDraftPaths(items, a.rows.byWPID))
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
			current, known = pagemap.ClaimProduct(storeCandidate(items[i]), slices.Collect(maps.Values(a.rows.byPath)))
			if known {
				waiting--
			}
		}
		next, _ := merge(current, known, items[i], a.state.SiteID, a.now)
		if known && current.Path != next.Path {
			delete(a.rows.byPath, current.Path)
		}
		if !known {
			a.pending[next.ID] = struct{}{}
		}
		a.rows.remember(next)

		terms, err := a.readTerms(ctx, items[i])
		if err != nil {
			return nil, err
		}
		body, err := a.readBody(ctx, next, items[i])
		if err != nil {
			return nil, err
		}
		batch = append(batch, prepared{it: items[i], current: current, next: next, terms: terms, body: body, known: known})
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
	return a.plan(ctx, p.current, p.known, p.next, p.it)
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
	if it.mappedType().Term() {
		return nil
	}
	return a.service.deps.Identity.ReplaceTerms(ctx, page.ID, terms)
}

func (a *absorber) fields(ctx context.Context, page pagemap.Page, it item) error {
	values := []field.Value{field.NewValue(page.ID, field.Title, it.Title, a.now)}
	if !it.mappedType().Term() {
		values = append(values, field.NewValue(page.ID, field.Excerpt, it.Excerpt, a.now))
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
		return bodyRead{raw: it.Content, hash: it.ContentHash, fetched: true}, nil
	case a.readers.plugin(capabilityRaw):
		read, readErr := a.readers.Plugin.GetRaw(ctx, string(it.mappedType()), it.WPID)
		if readErr != nil {
			return bodyRead{}, readErr
		}
		return bodyRead{raw: read.Content, hash: read.Hash, fetched: true}, nil
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
			if err := a.rawLinks(ctx, page, block.Parse(read.current.Raw)); err != nil {
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
	if err := a.rawLinks(ctx, page, body); err != nil {
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

func (a *absorber) rawLinks(ctx context.Context, page pagemap.Page, body block.Body) error {
	written, err := a.service.deps.Bodies.ListWritten(ctx, page.ID)
	if err != nil {
		return err
	}
	ours := make(map[string]struct{}, len(written))
	for i := range written {
		ours[written[i].BlockHash] = struct{}{}
	}
	generated, err := a.generatedTargets(ctx, page.ID)
	if err != nil {
		return err
	}

	links := make([]pagemap.PageLink, 0)
	for i := range body.Blocks {
		if body.Blocks[i].Dynamic {
			continue
		}
		doc, parseErr := content.Parse(body.Blocks[i].Raw)
		if parseErr != nil {
			continue
		}
		_, blockOurs := ours[body.Blocks[i].Hash]
		for _, found := range doc.Links() {
			path, kind := a.install.ResolveAt(page.Path, found.Href)
			if kind != pagemap.LinkPath || path == "" {
				continue
			}
			position := i
			_, wasGenerated := generated[path]
			link := pagemap.PageLink{
				ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: path, AnchorText: found.Anchor,
				Origin: pagemap.OriginObserved, Type: pagemap.LinkContextual, Region: string(regionContent),
				Source: pagemap.SourceRaw, Position: &position, Ours: blockOurs || wasGenerated,
				Absolute: pagemap.AbsoluteHref(found.Href), ObservedAt: a.now,
			}
			if wasGenerated {
				link.Origin = pagemap.OriginGenerated
			}
			if target, ok := a.stored(path); ok {
				link.ToPageID = &target.ID
			}
			if built, buildErr := pagemap.NewPageLink(link); buildErr == nil {
				links = append(links, built)
			}
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
	generated, err := a.generatedTargets(ctx, page.ID)
	if err != nil {
		return err
	}
	links := make([]pagemap.PageLink, 0, len(it.Links))
	for i := range it.Links {
		path, normalizeErr := pagemap.NormalizePath(it.Links[i].Href)
		if normalizeErr != nil {
			continue
		}
		_, wasGenerated := generated[path]
		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: path, AnchorText: it.Links[i].Anchor,
			Origin: pagemap.OriginObserved, Type: pagemap.LinkContextual, Region: string(regionContent),
			Source: pagemap.SourceRaw, Ours: wasGenerated, Absolute: pagemap.AbsoluteHref(it.Links[i].Href), ObservedAt: a.now,
		}
		if wasGenerated {
			link.Origin = pagemap.OriginGenerated
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

func (a *absorber) generatedTargets(ctx context.Context, pageID string) (map[string]struct{}, error) {
	stored, err := a.service.deps.Links.ListForPage(ctx, pageID)
	if err != nil {
		return nil, err
	}
	ours := make(map[string]struct{}, len(stored))
	for i := range stored {
		if stored[i].Source == pagemap.SourceRaw && stored[i].Origin == pagemap.OriginGenerated {
			ours[stored[i].ToURL] = struct{}{}
		}
	}
	return ours, nil
}

func (a *absorber) plan(ctx context.Context, current pagemap.Page, known bool, next pagemap.Page, it item) error {
	planned := hasPlan(current, known)
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
