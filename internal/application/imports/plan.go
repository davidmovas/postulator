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

const homeTitle = "Home"

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
	report    PreviewReport
	entities  []plannedEntity
	edges     []graph.Edge
	pages     []plannedPage
	canonical []canonical
}

func (p *plan) note(row int, field string, code FindingCode, message string) {
	finding := Finding{Row: row, Field: field, Code: string(code), Message: message}
	if code.Blocking() {
		p.report.Errors = append(p.report.Errors, finding)
		return
	}
	p.report.Warnings = append(p.report.Warnings, finding)
}

func (p *plan) broken() bool {
	return len(p.report.Errors) > 0
}

func titleFrom(path string) string {
	slug := pagemap.Slug(path)
	if slug == "" {
		return homeTitle
	}
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
	return a.Kind == b.Kind && a.PrimaryKeyword == b.PrimaryKeyword &&
		slices.Equal(a.SecondaryKeywords, b.SecondaryKeywords) &&
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
		sameRef(a.EntityID, b.EntityID) && sameRef(a.TemplateID, b.TemplateID)
}

type siteState struct {
	siteID   string
	entities []graph.Entity
	edges    []graph.Edge
	pages    []pagemap.Page
	byName   map[string]graph.Entity
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
		byName:   make(map[string]graph.Entity, len(entities)),
		byPath:   make(map[string]pagemap.Page, len(pages)),
		edgeKeys: make(map[string]struct{}, len(edges)),
	}
	for i := range entities {
		state.byName[key(entities[i].Name)] = entities[i]
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

func read(binding importmap.Binding, table importmap.Table, p *plan) *drafts {
	sheet := newDrafts()
	for i := range table.Rows {
		row, number := table.Rows[i], i+2
		if binding.Blank(row) {
			p.report.Skipped++
			continue
		}

		name, path := binding.Text(row, importmap.FieldEntity), binding.Path(row)
		if name == "" && path == "" {
			p.note(number, "", CodeNoTarget, "the row names neither a path nor an entity")
			p.report.Skipped++
			continue
		}

		if name != "" {
			sheet.entity(name, number).merge(entityDraft{
				kind:     binding.Text(row, importmap.FieldEntityKind),
				primary:  binding.Text(row, importmap.FieldPrimaryKeyword),
				keywords: binding.List(row, importmap.FieldKeywords),
				anchors:  binding.List(row, importmap.FieldAnchors),
				parent:   binding.Text(row, importmap.FieldParentEntity),
				related:  binding.List(row, importmap.FieldRelated),
			})
		}
		if path == "" {
			continue
		}

		normalized, err := pagemap.NormalizePath(path)
		if err != nil {
			p.note(number, string(importmap.FieldPath), CodeBadPath, "the path cannot be read: "+path)
			continue
		}

		draft, known := sheet.page(normalized, number)
		if known {
			p.note(number, string(importmap.FieldPath), CodeDuplicatePath, "the path repeats an earlier row and was merged: "+normalized)
		}
		draft.merge(pageDraft{
			title:     binding.Text(row, importmap.FieldTitle),
			h1:        binding.Text(row, importmap.FieldH1),
			metaTitle: binding.Text(row, importmap.FieldMetaTitle),
			metaDesc:  binding.Text(row, importmap.FieldMetaDescription),
			wpType:    binding.Text(row, importmap.FieldWPType),
			pageKind:  binding.Text(row, importmap.FieldPageKind),
			entity:    name,
		})
	}
	return sheet
}

func fillGaps(sheet *drafts, state siteState, p *plan) {
	for _, path := range sheet.sortedPaths() {
		for parent := pagemap.ParentPath(path); parent != ""; parent = pagemap.ParentPath(parent) {
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

func resolveEntities(sheet *drafts, state siteState, now time.Time, p *plan) (map[string]graph.Entity, error) {
	resolved := make(map[string]graph.Entity, len(sheet.order))
	for _, at := range sheet.order {
		draft := sheet.entities[at]

		kind := graph.Kind(strings.ToLower(draft.kind))
		if draft.kind != "" && !kind.Valid() {
			p.note(draft.row, string(importmap.FieldEntityKind), CodeUnknownEntityKind,
				"the entity kind is not recognized and was read as a topic: "+draft.kind)
			kind = ""
		}

		current, found := state.byName[at]
		if !found {
			if kind == "" {
				kind = graph.KindTopic
			}
			entity, err := graph.NewEntity(graph.Entity{
				ID: id.New(), SiteID: state.siteID, Name: draft.name, Kind: kind,
				PrimaryKeyword: draft.primary, SecondaryKeywords: draft.keywords, Anchors: anchorsOf(draft.anchors),
				Source: graph.SourceImport, CreatedAt: now, UpdatedAt: now,
			})
			if err != nil {
				return nil, err
			}
			resolved[at] = entity
			p.entities = append(p.entities, plannedEntity{entity: entity, created: true})
			p.report.Entities = append(p.report.Entities, entityView(entity, ActionCreate))
			continue
		}

		next := current
		if kind != "" {
			next.Kind = kind
		}
		next.PrimaryKeyword = fill(next.PrimaryKeyword, draft.primary)
		next.SecondaryKeywords = union(next.SecondaryKeywords, draft.keywords)
		next.Anchors = anchorsOf(union(anchorTexts(next.Anchors), draft.anchors))
		next.UpdatedAt = now
		entity, err := graph.NewEntity(next)
		if err != nil {
			return nil, err
		}
		resolved[at] = entity

		if sameEntity(entity, current) {
			p.report.Entities = append(p.report.Entities, entityView(entity, ActionSkip))
			continue
		}
		p.entities = append(p.entities, plannedEntity{entity: entity, created: false})
		p.report.Entities = append(p.report.Entities, entityView(entity, ActionUpdate))
	}
	return resolved, nil
}

func resolveEdges(sheet *drafts, state siteState, resolved map[string]graph.Entity, now time.Time, p *plan) {
	lookup := func(name string) (graph.Entity, bool) {
		if entity, ok := resolved[key(name)]; ok {
			return entity, true
		}
		entity, ok := state.byName[key(name)]
		return entity, ok
	}

	seen := maps.Clone(state.edgeKeys)

	add := func(from, to graph.Entity, kind graph.EdgeKind) {
		edge, err := graph.NewEdge(graph.Edge{
			ID: id.New(), SiteID: state.siteID, FromEntityID: from.ID, ToEntityID: to.ID,
			Kind: kind, Weight: 1, Source: graph.SourceImport, Status: graph.StatusApproved, CreatedAt: now,
		})
		if err != nil {
			return
		}
		view := PreviewEdge{From: from.Name, To: to.Name, Kind: string(kind), Action: string(ActionCreate)}
		if _, known := seen[edgeKey(edge)]; known {
			view.Action = string(ActionSkip)
			p.report.Edges = append(p.report.Edges, view)
			return
		}
		seen[edgeKey(edge)] = struct{}{}
		p.edges = append(p.edges, edge)
		p.report.Edges = append(p.report.Edges, view)
	}

	for _, at := range sheet.order {
		draft := sheet.entities[at]
		from := resolved[at]

		if draft.parent != "" {
			switch to, ok := lookup(draft.parent); {
			case !ok:
				p.note(draft.row, string(importmap.FieldParentEntity), CodeUnknownParent,
					"the parent entity is not in the file and not on the site: "+draft.parent)
			case to.ID == from.ID:
				p.note(draft.row, string(importmap.FieldParentEntity), CodeSelfEdge,
					"the entity names itself as its parent: "+draft.name)
			default:
				add(from, to, graph.EdgeParent)
			}
		}

		for _, name := range draft.related {
			switch to, ok := lookup(name); {
			case !ok:
				p.note(draft.row, string(importmap.FieldRelated), CodeUnknownRelated,
					"the related entity is not in the file and not on the site: "+name)
			case to.ID == from.ID:
				p.note(draft.row, string(importmap.FieldRelated), CodeSelfEdge,
					"the entity names itself as related: "+draft.name)
			default:
				add(from, to, graph.EdgeRelated)
			}
		}
	}
}

func checkAcyclic(state siteState, resolved map[string]graph.Entity, p *plan) error {
	entities := make([]graph.Entity, 0, len(state.entities)+len(resolved))
	taken := make(map[string]struct{}, len(resolved))
	for _, at := range slices.Sorted(maps.Keys(resolved)) {
		entities = append(entities, resolved[at])
		taken[resolved[at].ID] = struct{}{}
	}
	for i := range state.entities {
		if _, replaced := taken[state.entities[i].ID]; !replaced {
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

func (s *Service) resolvePages(ctx context.Context, sheet *drafts, state siteState, resolved map[string]graph.Entity, now time.Time, p *plan) error {
	templates := make(map[string]*string)
	for _, path := range sheet.sortedPaths() {
		draft := sheet.pages[path]

		wpType := pagemap.WPType(strings.ToLower(draft.wpType))
		if draft.wpType != "" && !wpType.Valid() {
			p.note(draft.row, string(importmap.FieldWPType), CodeUnknownWPType,
				"the wordpress type is not recognized and was read as a page: "+draft.wpType)
			wpType = ""
		}

		templateID, ok := templates[key(draft.pageKind)]
		if !ok && draft.pageKind != "" {
			found, err := s.templateFor(ctx, state.siteID, draft.pageKind)
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
		if draft.entity != "" {
			entity := resolved[key(draft.entity)]
			entityID = &entity.ID
		}

		current, exists := state.byPath[path]
		if !exists {
			page, err := pagemap.NewPage(pagemap.Page{
				ID: id.New(), SiteID: state.siteID, Path: path, WPType: wpTypeOr(wpType),
				Title: fill(draft.title, titleFrom(path)), H1: draft.h1, MetaTitle: draft.metaTitle,
				MetaDescription: draft.metaDesc, Status: pagemap.StatusPlanned, EntityID: entityID,
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
		if wpType != "" {
			next.WPType = wpType
		}
		if templateID != nil {
			next.TemplateID = templateID
		}
		if entityID != nil {
			next.EntityID = entityID
		}
		next.UpdatedAt = now
		page, err := pagemap.NewPage(next)
		if err != nil {
			return err
		}

		if samePage(page, current) {
			p.report.Pages = append(p.report.Pages, pageView(page, draft, ActionSkip))
			continue
		}
		p.pages = append(p.pages, plannedPage{page: page, created: false})
		p.report.Pages = append(p.report.Pages, pageView(page, draft, ActionUpdate))
	}
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
	byID := make(map[string]graph.Entity, len(resolved))
	for _, at := range slices.Sorted(maps.Keys(resolved)) {
		byID[resolved[at].ID] = resolved[at]
	}

	index := pagemap.NewIndex(state.pages)
	for i := range p.pages {
		planned := &p.pages[i]
		if !planned.created {
			continue
		}
		var entity graph.Entity
		if planned.page.EntityID != nil {
			entity = byID[*planned.page.EntityID]
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

func markCanonical(state siteState, resolved map[string]graph.Entity, p *plan) {
	final := make(map[string]pagemap.Page, len(state.pages)+len(p.pages))
	for i := range state.pages {
		final[state.pages[i].ID] = state.pages[i]
	}
	for i := range p.pages {
		final[p.pages[i].page.ID] = p.pages[i].page
	}

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
	p := plan{}
	sheet := read(binding, table, &p)
	fillGaps(sheet, state, &p)

	resolved, err := resolveEntities(sheet, state, now, &p)
	if err != nil {
		return plan{}, err
	}
	resolveEdges(sheet, state, resolved, now, &p)
	if err := checkAcyclic(state, resolved, &p); err != nil {
		return plan{}, err
	}
	if err := s.resolvePages(ctx, sheet, state, resolved, now, &p); err != nil {
		return plan{}, err
	}
	if err := checkCannibalization(state, resolved, &p); err != nil {
		return plan{}, err
	}
	linkParents(state, &p)
	markCanonical(state, resolved, &p)
	p.report.settle()
	return p, nil
}
