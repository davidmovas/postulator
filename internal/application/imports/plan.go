package imports

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type plannedEntity struct {
	entity  graph.Entity
	created bool
}

type plannedPage struct {
	page    pagemap.Page
	created bool
}

type canonical struct {
	entityID string
	pageID   string
}

type plan struct {
	siteID    string
	report    PreviewReport
	entities  []plannedEntity
	edges     []graph.Edge
	pages     []plannedPage
	canonical []canonical
	rows      int
}

func (p *plan) note(row int, field string, code FindingCode, message string) {
	p.noteAt(importmap.Origin{Row: row}, field, code, message)
}

func (p *plan) noteAt(at importmap.Origin, field string, code FindingCode, message string) {
	finding := Finding{Row: at.Row, Sheet: at.Sheet, Field: field, Code: string(code), Message: message}
	if code.Blocking() {
		p.report.Errors = append(p.report.Errors, finding)
		return
	}
	p.report.Warnings = append(p.report.Warnings, finding)
}

func (p *plan) counts() Counts {
	tally := Counts{Skipped: p.report.Skipped}
	for i := range p.entities {
		if p.entities[i].created {
			tally.EntitiesCreated++
			continue
		}
		tally.EntitiesUpdated++
	}
	tally.EdgesCreated = len(p.edges)
	for i := range p.pages {
		if p.pages[i].created {
			tally.PagesCreated++
			continue
		}
		tally.PagesUpdated++
	}
	return tally
}

func (p *plan) broken() bool {
	return len(p.report.Errors) > 0
}

func titleFrom(path string) string {
	slug := pagemap.Slug(path)
	words := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func anchorsOf(texts []string) []graph.Anchor {
	out := make([]graph.Anchor, 0, len(texts))
	for _, text := range texts {
		out = append(out, graph.Anchor{Text: text, Source: graph.AnchorUser, Weight: 1})
	}
	return out
}

func anchorTexts(anchors []graph.Anchor) []string {
	out := make([]string, 0, len(anchors))
	for i := range anchors {
		out = append(out, anchors[i].Text)
	}
	return out
}

func edgeKey(e graph.Edge) string {
	return string(e.Kind) + "|" + e.FromEntityID + "|" + e.ToEntityID
}

func sameEntity(a, b graph.Entity) bool {
	return a.Kind == b.Kind && a.Keywords.Equal(b.Keywords) &&
		slices.Equal(anchorTexts(a.Anchors), anchorTexts(b.Anchors))
}

func sameRef(a, b *string) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

func samePage(a, b pagemap.Page) bool {
	return a.Title == b.Title && a.H1 == b.H1 && a.MetaTitle == b.MetaTitle &&
		a.MetaDescription == b.MetaDescription && a.WPType == b.WPType &&
		a.Keywords.Equal(b.Keywords) &&
		sameRef(a.EntityID, b.EntityID) && sameRef(a.TemplateID, b.TemplateID)
}

type siteState struct {
	siteID   string
	entities []graph.Entity
	edges    []graph.Edge
	pages    []pagemap.Page
	byName   map[string][]graph.Entity
	byPath   map[string]pagemap.Page
	edgeKeys map[string]struct{}
}

func (s *Service) state(ctx context.Context, siteID string) (siteState, error) {
	entities, err := s.deps.Entities.ListBySite(ctx, siteID)
	if err != nil {
		return siteState{}, err
	}
	edges, err := s.deps.Edges.ListBySite(ctx, siteID)
	if err != nil {
		return siteState{}, err
	}
	pages, err := s.deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return siteState{}, err
	}
	state := siteState{
		siteID: siteID, entities: entities, edges: edges, pages: pages,
		byName:   make(map[string][]graph.Entity, len(entities)),
		byPath:   make(map[string]pagemap.Page, len(pages)),
		edgeKeys: make(map[string]struct{}, len(edges)),
	}
	for i := range entities {
		state.byName[key(entities[i].Name)] = append(state.byName[key(entities[i].Name)], entities[i])
	}
	for i := range pages {
		state.byPath[pages[i].Path] = pages[i]
	}
	for i := range edges {
		state.edgeKeys[edgeKey(edges[i])] = struct{}{}
	}
	return state, nil
}

func (s *Service) templateFor(ctx context.Context, siteID, pageKind string) (*string, error) {
	list, err := s.deps.Templates.List(ctx, template.Query{PageKind: strings.ToLower(pageKind)}, paging.Request{Limit: paging.MaxLimit})
	if err != nil {
		return nil, err
	}

	var global *string
	for i := range list.Items {
		found := &list.Items[i]
		if found.SiteID != nil && *found.SiteID == siteID {
			return &found.ID, nil
		}
		if found.Scope == template.ScopeGlobal && global == nil {
			global = &found.ID
		}
	}
	return global, nil
}

func fillGaps(sheet *drafts, state siteState, p *plan) {
	for _, path := range sheet.sortedPaths() {
		for parent := pagemap.ParentPath(path); parent != "" && parent != pagemap.RootPath; parent = pagemap.ParentPath(parent) {
			if _, planned := sheet.pages[parent]; planned {
				continue
			}
			if _, exists := state.byPath[parent]; exists {
				continue
			}
			draft, _ := sheet.page(parent, 0)
			draft.generated = true
			draft.title = titleFrom(parent)
			p.note(0, string(importmap.FieldPath), CodeIntermediatePath, "the missing intermediate path was created: "+parent)
		}
	}
}

func (b *builder) planEntities(now time.Time) (map[string]graph.Entity, error) {
	resolved := make(map[string]graph.Entity, len(b.units))
	for _, at := range b.roots() {
		u := &b.units[at]
		kind, known := kindOf(u.kind)
		if !known {
			b.p.noteAt(u.at, string(importmap.FieldEntityKind), CodeUnknownEntityKind,
				"the entity kind is not recognized and was read as a topic: "+u.kind)
		}

		parent := b.parentName(u)
		if u.matched == "" {
			if kind == "" {
				kind = graph.KindTopic
				if u.group >= 0 {
					kind = graph.KindCategory
				}
			}
			var scope *string
			if parentID := b.parentID(u); parentID != "" && parentID != u.id {
				scope = &parentID
			}
			entity, err := graph.NewEntity(graph.Entity{
				ID: u.id, SiteID: b.state.siteID, Name: u.name, Kind: kind, ScopeID: scope,
				Keywords: u.keywords, Anchors: anchorsOf(u.anchors),
				Source: graph.SourceImport, CreatedAt: now, UpdatedAt: now,
			})
			if err != nil {
				return nil, err
			}
			resolved[entity.ID] = entity
			b.p.entities = append(b.p.entities, plannedEntity{entity: entity, created: true})
			b.p.report.Entities = append(b.p.report.Entities, entityView(entity, parent, ActionCreate))
			continue
		}

		current := b.byID[u.matched]
		next := current
		if kind != "" {
			next.Kind = kind
		}
		next.Keywords = next.Keywords.Merge(u.keywords)
		next.Anchors = anchorsOf(union(anchorTexts(next.Anchors), u.anchors))
		next.UpdatedAt = now
		entity, err := graph.NewEntity(next)
		if err != nil {
			return nil, err
		}
		resolved[entity.ID] = entity
		if sameEntity(entity, current) {
			b.p.report.Entities = append(b.p.report.Entities, entityView(entity, parent, ActionSkip))
			continue
		}
		b.p.entities = append(b.p.entities, plannedEntity{entity: entity, created: false})
		b.p.report.Entities = append(b.p.report.Entities, entityView(entity, parent, ActionUpdate))
	}
	return resolved, nil
}

func (b *builder) nameOf(entityID string) string {
	for at := range b.units {
		if b.units[at].alias < 0 && b.units[at].id == entityID {
			return b.units[at].name
		}
	}
	return b.byID[entityID].Name
}

func (b *builder) parentName(u *unit) string {
	if parentID := b.parentID(u); parentID != "" {
		return b.nameOf(parentID)
	}
	return ""
}

func (b *builder) planEdges(now time.Time) {
	seen := maps.Clone(b.state.edgeKeys)

	add := func(fromID, toID string, kind graph.EdgeKind) {
		edge, err := graph.NewEdge(graph.Edge{
			ID: id.New(), SiteID: b.state.siteID, FromEntityID: fromID, ToEntityID: toID,
			Kind: kind, Weight: 1, Source: graph.SourceImport, Status: graph.StatusApproved, CreatedAt: now,
		})
		if err != nil {
			return
		}
		view := PreviewEdge{From: b.nameOf(fromID), To: b.nameOf(toID), Kind: string(kind), Action: string(ActionCreate)}
		if _, known := seen[edgeKey(edge)]; known {
			view.Action = string(ActionSkip)
			b.p.report.Edges = append(b.p.report.Edges, view)
			return
		}
		seen[edgeKey(edge)] = struct{}{}
		b.p.edges = append(b.p.edges, edge)
		b.p.report.Edges = append(b.p.report.Edges, view)
	}

	for _, at := range b.roots() {
		u := &b.units[at]
		parentID := b.parentID(u)
		switch {
		case parentID == "":
		case parentID == u.id:
			b.p.noteAt(u.parent.at, string(importmap.FieldParentEntity), CodeSelfEdge,
				"the entity names itself as its parent: "+u.name)
		case u.parent.weak && u.matched != "" && b.parents[u.matched] > 0:
		default:
			add(u.id, parentID, graph.EdgeParent)
		}

		for _, name := range u.related {
			ref, code := b.resolve(name, u.context)
			switch code {
			case CodeUnknownParent:
				b.p.noteAt(u.at, string(importmap.FieldRelated), CodeUnknownRelated,
					"the related entity is not in the file and not on the site: "+name)
				continue
			case CodeAmbiguousParent:
				b.p.noteAt(u.at, string(importmap.FieldRelated), CodeAmbiguousEntity,
					"more than one entity is named "+name+"; name a related entity that only one carries")
				continue
			}
			relatedID := ref.site
			if ref.kind == refUnit {
				relatedID = b.entityID(ref.unit)
			}
			if relatedID == u.id {
				b.p.noteAt(u.at, string(importmap.FieldRelated), CodeSelfEdge, "the entity names itself as related: "+u.name)
				continue
			}
			add(u.id, relatedID, graph.EdgeRelated)
		}
	}
}

func checkAcyclic(state siteState, resolved map[string]graph.Entity, p *plan) error {
	entities := make([]graph.Entity, 0, len(state.entities)+len(resolved))
	for _, at := range slices.Sorted(maps.Keys(resolved)) {
		entities = append(entities, resolved[at])
	}
	for i := range state.entities {
		if _, replaced := resolved[state.entities[i].ID]; !replaced {
			entities = append(entities, state.entities[i])
		}
	}

	edges := make([]graph.Edge, 0, len(state.edges)+len(p.edges))
	edges = append(edges, state.edges...)
	edges = append(edges, p.edges...)

	g, err := graph.New(entities, edges)
	if err != nil {
		return err
	}
	if cycleErr := g.ValidateAcyclic(); cycleErr != nil {
		p.note(0, string(importmap.FieldParentEntity), CodeCycle, cycleErr.Error())
	}
	return nil
}

func (s *Service) resolvePages(ctx context.Context, b *builder, now time.Time) error {
	p := b.p
	templates := make(map[string]*string)
	for _, path := range b.sheet.sortedPaths() {
		draft := b.sheet.pages[path]

		wpType := pagemap.WPType(strings.ToLower(draft.wpType))
		if draft.wpType != "" && !wpType.Valid() {
			p.note(draft.row, string(importmap.FieldWPType), CodeUnknownWPType,
				"the wordpress type is not recognized and was read as a page: "+draft.wpType)
			wpType = ""
		}

		templateID, ok := templates[key(draft.pageKind)]
		if !ok && draft.pageKind != "" {
			found, err := s.templateFor(ctx, b.state.siteID, draft.pageKind)
			if err != nil {
				return err
			}
			if found == nil {
				p.note(draft.row, string(importmap.FieldPageKind), CodeUnknownPageKind,
					"no template carries this page kind: "+draft.pageKind)
			}
			templates[key(draft.pageKind)] = found
			templateID = found
		}

		var entityID *string
		if owner, owned := b.pageUnit(path); owned {
			entityID = &b.units[owner].id
			draft.entity = b.units[owner].name
		}

		current, exists := b.state.byPath[path]
		if !exists && path == pagemap.RootPath {
			p.note(draft.row, string(importmap.FieldPath), CodeRootPageSkipped,
				"the root of the site already exists on WordPress, so the import does not plan it; sync the site first to map it")
			continue
		}
		if !exists {
			page, err := pagemap.NewPage(pagemap.Page{
				ID: id.New(), SiteID: b.state.siteID, Path: path, WPType: wpTypeOr(wpType),
				Title: fill(draft.title, titleFrom(path)), H1: draft.h1, MetaTitle: draft.metaTitle,
				MetaDescription: draft.metaDesc, Keywords: draft.keywords,
				Status: pagemap.StatusPlanned, EntityID: entityID,
				TemplateID: templateID, CreatedAt: now, UpdatedAt: now,
			})
			if err != nil {
				return err
			}
			p.pages = append(p.pages, plannedPage{page: page, created: true})
			p.report.Pages = append(p.report.Pages, pageView(page, draft, ActionCreate))
			continue
		}

		next := current
		next.Title = fill(next.Title, draft.title)
		next.H1 = fill(next.H1, draft.h1)
		next.MetaTitle = fill(next.MetaTitle, draft.metaTitle)
		next.MetaDescription = fill(next.MetaDescription, draft.metaDesc)
		next.Keywords = next.Keywords.Merge(draft.keywords)
		if wpType != "" {
			next.WPType = wpType
		}
		if templateID != nil {
			next.TemplateID = templateID
		}
		if entityID != nil {
			next.EntityID = entityID
		}
		if err := planUpdate(p, current, next, draft, now); err != nil {
			return err
		}
	}

	for _, path := range slices.Sorted(maps.Keys(b.siteOwned())) {
		owner := b.siteOwned()[path]
		current := b.state.byPath[path]
		next := current
		next.EntityID = &b.units[owner].id
		if err := planUpdate(p, current, next, &pageDraft{path: path, entity: b.units[owner].name}, now); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) siteOwned() map[string]int {
	out := make(map[string]int)
	for path, at := range b.assigned {
		out[path] = b.root(at)
	}
	for path, node := range b.adopted {
		if _, planned := b.sheet.pages[path]; !planned {
			out[path] = b.root(b.groups.nodes[node].unit)
		}
	}
	return out
}

func planUpdate(p *plan, current, next pagemap.Page, draft *pageDraft, now time.Time) error {
	next.UpdatedAt = now
	page, err := pagemap.NewPage(next)
	if err != nil {
		return err
	}
	if samePage(page, current) {
		p.report.Pages = append(p.report.Pages, pageView(page, draft, ActionSkip))
		return nil
	}
	p.pages = append(p.pages, plannedPage{page: page, created: false})
	p.report.Pages = append(p.report.Pages, pageView(page, draft, ActionUpdate))
	return nil
}

func wpTypeOr(wpType pagemap.WPType) pagemap.WPType {
	if wpType == "" {
		return pagemap.WPPage
	}
	return wpType
}

func checkCannibalization(state siteState, resolved map[string]graph.Entity, p *plan) error {
	g, err := graph.New(state.entities, state.edges)
	if err != nil {
		return err
	}

	index := pagemap.NewIndex(state.pages)
	for i := range p.pages {
		planned := &p.pages[i]
		if !planned.created {
			continue
		}
		var entity graph.Entity
		if planned.page.EntityID != nil {
			entity = resolved[*planned.page.EntityID]
		}
		for _, evidence := range pagemap.Cannibalization(planned.page, entity, index, g).Evidence {
			p.report.Cannibalization = append(p.report.Cannibalization, conflictView(evidence))
			p.note(0, string(importmap.FieldPath), CodeCannibalization,
				"the page "+planned.page.Path+" overlaps "+evidence.Path+" ("+string(evidence.Reason)+")")
		}
	}
	return nil
}

func linkParents(state siteState, p *plan) {
	byPath := make(map[string]string, len(state.pages)+len(p.pages))
	for i := range state.pages {
		byPath[state.pages[i].Path] = state.pages[i].ID
	}
	for i := range p.pages {
		byPath[p.pages[i].page.Path] = p.pages[i].page.ID
	}

	for i := range p.pages {
		planned := &p.pages[i]
		if planned.page.ParentPageID != nil {
			continue
		}
		parentID, found := byPath[pagemap.ParentPath(planned.page.Path)]
		if !found || parentID == planned.page.ID {
			continue
		}
		planned.page.ParentPageID = &parentID
	}
}

func finalPages(state siteState, p *plan) map[string]pagemap.Page {
	final := make(map[string]pagemap.Page, len(state.pages)+len(p.pages))
	for i := range state.pages {
		final[state.pages[i].ID] = state.pages[i]
	}
	for i := range p.pages {
		final[p.pages[i].page.ID] = p.pages[i].page
	}
	return final
}

func markCanonical(state siteState, resolved map[string]graph.Entity, p *plan) {
	final := finalPages(state, p)
	owners := make(map[string][]string, len(resolved))
	for _, at := range slices.Sorted(maps.Keys(final)) {
		if page := final[at]; page.EntityID != nil {
			owners[*page.EntityID] = append(owners[*page.EntityID], page.ID)
		}
	}

	for _, at := range slices.Sorted(maps.Keys(resolved)) {
		entity := resolved[at]
		if entity.CanonicalPageID != nil || len(owners[entity.ID]) != 1 {
			continue
		}
		p.canonical = append(p.canonical, canonical{entityID: entity.ID, pageID: owners[entity.ID][0]})
	}
}

func (b *builder) reportGroups() {
	owned := make(map[string]string)
	final := finalPages(b.state, b.p)
	for pageID := range final {
		entityID, path := final[pageID].EntityID, final[pageID].Path
		if entityID == nil {
			continue
		}
		if held, seen := owned[*entityID]; !seen || path < held {
			owned[*entityID] = path
		}
	}

	for at := range b.groups.nodes {
		node := &b.groups.nodes[at]
		entityID := b.entityID(node.unit)
		view := PreviewGroup{Path: b.groups.chain(at), Page: node.page, Rows: len(node.under)}
		if view.Page == "" {
			view.Page = owned[entityID]
		}
		b.p.report.Groups = append(b.p.report.Groups, view)
		if view.Page == "" {
			b.p.noteAt(b.units[node.unit].at, "", CodeGroupWithoutPage,
				"the group "+strings.Join(view.Path, " › ")+" has no page of its own in the sheet or on the site; "+
					"it is kept as an entity, and the links of the pages under it pass over it")
		}
	}
}

func (s *Service) plan(ctx context.Context, siteID string, table importmap.Table, mapping importmap.Mapping) (plan, error) {
	binding, err := mapping.Bind(table.Headers)
	if err != nil {
		return plan{}, err
	}

	state, err := s.state(ctx, siteID)
	if err != nil {
		return plan{}, err
	}

	now := s.now()
	p := plan{siteID: siteID, rows: len(table.Rows)}
	rows := readRows(binding, table, &p)
	sheet := pagesOf(rows, &p)
	fillGaps(sheet, state, &p)

	b := newBuilder(state, &p, rows, sheet)
	b.build()
	resolved, err := b.planEntities(now)
	if err != nil {
		return plan{}, err
	}
	b.planEdges(now)
	if err := checkAcyclic(state, resolved, &p); err != nil {
		return plan{}, err
	}
	if err := s.resolvePages(ctx, b, now); err != nil {
		return plan{}, err
	}
	if err := checkCannibalization(state, resolved, &p); err != nil {
		return plan{}, err
	}
	linkParents(state, &p)
	markCanonical(state, resolved, &p)
	b.reportGroups()
	p.report.settle()
	return p, nil
}
