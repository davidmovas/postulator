package imports

import (
	"context"
	"maps"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

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
	return a.Title == b.Title && a.H1 == b.H1 && a.MetaTitle == b.MetaTitle && a.PlannedPath == b.PlannedPath &&
		a.MetaDescription == b.MetaDescription && a.WPType == b.WPType &&
		a.Keywords.Equal(b.Keywords) && slices.Equal(a.Notes, b.Notes) &&
		sameRef(a.EntityID, b.EntityID) && sameRef(a.TemplateID, b.TemplateID)
}

type kindTemplates struct {
	byKind map[string]*string
	noted  map[string]struct{}
}

func (s *Service) kindTemplates(ctx context.Context, siteID string, sheet *drafts) (*kindTemplates, error) {
	found := &kindTemplates{byKind: make(map[string]*string), noted: make(map[string]struct{})}
	for _, path := range sheet.sortedPaths() {
		draft := sheet.pages[path]
		kind := draft.kind()
		if _, looked := found.byKind[kind]; draft.entityOnly || kind == "" || looked {
			continue
		}
		templateID, err := s.templateFor(ctx, siteID, kind)
		if err != nil {
			return nil, err
		}
		found.byKind[kind] = templateID
	}
	return found, nil
}

func (s *Service) templateFor(ctx context.Context, siteID, pageKind string) (*string, error) {
	list, err := s.deps.Templates.List(ctx, template.Query{PageKind: pageKind}, paging.Request{Limit: paging.MaxLimit})
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

func (k *kindTemplates) of(draft *pageDraft, p *plan) *string {
	kind := draft.kind()
	found := k.byKind[kind]
	if _, noted := k.noted[kind]; kind == "" || found != nil || noted {
		return found
	}
	k.noted[kind] = struct{}{}
	p.noteAt(draft.at, string(importmap.FieldPageKind), CodeUnknownPageKind, "no template carries this page kind: "+draft.pageKind)
	return nil
}

func (b *builder) planPages(templates *kindTemplates) error {
	for _, path := range b.sheet.sortedPaths() {
		if draft := b.sheet.pages[path]; !draft.entityOnly {
			if err := b.planPage(draft, templates); err != nil {
				return err
			}
		}
	}
	return b.planSiteOwned()
}

func (b *builder) planPage(draft *pageDraft, templates *kindTemplates) error {
	if draft.wpType != "" && draft.cellType() == "" {
		b.p.noteAt(draft.at, string(importmap.FieldWPType), CodeUnknownWPType,
			"the wordpress type is not recognized and was read as a page: "+draft.wpType)
	}
	templateID := templates.of(draft, b.p)
	entityID := b.ownerOf(draft)

	current, exists := b.state.held(draft.path)
	switch {
	case exists:
		return b.updatePage(current, draft, templateID, entityID)
	case draft.path == pagemap.RootPath:
		b.p.noteAt(draft.at, string(importmap.FieldPath), CodeRootPageSkipped,
			"the root of the site already exists on WordPress, so the import does not plan it; sync the site first to map it")
		return nil
	default:
		return b.createPage(draft, templateID, entityID)
	}
}

func (b *builder) ownerOf(draft *pageDraft) *string {
	owner, owned := b.pageUnit(draft.path)
	if !owned {
		return nil
	}
	draft.entity = b.units[owner].name
	return &b.units[owner].id
}

func (b *builder) createPage(draft *pageDraft, templateID, entityID *string) error {
	wpType := draft.createdType()
	title := fill(draft.title, titleFrom(draft.path))
	if wpType == pagemap.WPProduct {
		title = draft.title
	}
	page, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: b.state.siteID, Path: draft.path, WPType: wpType,
		Title: title, H1: draft.h1, MetaTitle: draft.metaTitle, MetaDescription: draft.metaDesc,
		Keywords: draft.keywords, Notes: draft.notes, Status: pagemap.StatusPlanned, EntityID: entityID,
		TemplateID: templateID, CreatedAt: b.now, UpdatedAt: b.now,
	})
	if err != nil {
		return err
	}
	b.p.pages = append(b.p.pages, plannedPage{page: page, created: true})
	b.p.report.Pages = append(b.p.report.Pages, pageView(b.p.sheetAt(draft.at), page, draft, ActionCreate))
	return nil
}

func (b *builder) updatePage(current pagemap.Page, draft *pageDraft, templateID, entityID *string) error {
	next := current
	next.Title = fill(current.Title, draft.title)
	next.H1 = fill(current.H1, draft.h1)
	next.MetaTitle = fill(current.MetaTitle, draft.metaTitle)
	next.MetaDescription = fill(current.MetaDescription, draft.metaDesc)
	next.Keywords = current.Keywords.Merge(draft.keywords)
	next.Notes = pagemap.MergeNotes(current.Notes, draft.notes)
	next.WPType = b.keptType(current, draft)
	if current.Path != draft.path {
		next.PlannedPath = draft.path
	}
	if templateID != nil {
		next.TemplateID = templateID
	}
	if entityID != nil {
		next.EntityID = entityID
	}
	return b.planUpdate(current, next, draft)
}

func (b *builder) keptType(current pagemap.Page, draft *pageDraft) pagemap.WPType {
	wanted := draft.cellType()
	switch {
	case wanted == "" || wanted == current.WPType:
		return current.WPType
	case current.WPID != nil:
		b.p.noteAt(draft.at, string(importmap.FieldWPType), CodeWPTypeKept,
			draft.path+" is a "+string(current.WPType)+" on the site, so it stays one; the sheet names it a "+string(wanted))
		return current.WPType
	default:
		return wanted
	}
}

func (b *builder) planUpdate(current, next pagemap.Page, draft *pageDraft) error {
	next.UpdatedAt = b.now
	page, err := pagemap.NewPage(next)
	if err != nil {
		return err
	}
	action := ActionSkip
	if !samePage(page, current) {
		action = ActionUpdate
		b.p.pages = append(b.p.pages, plannedPage{page: page, created: false})
	}
	b.p.report.Pages = append(b.p.report.Pages, pageView(b.p.sheetAt(draft.at), page, draft, action))
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

func (b *builder) planSiteOwned() error {
	owned := b.siteOwned()
	for _, path := range slices.Sorted(maps.Keys(owned)) {
		owner := &b.units[owned[path]]
		current := b.state.byPath[path]
		next := current
		next.EntityID = &owner.id
		if err := b.planUpdate(current, next, &pageDraft{path: path, entity: owner.name}); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) checkCannibalization() error {
	g, err := graph.New(b.state.entities, b.state.edges)
	if err != nil {
		return err
	}

	index := pagemap.NewIndex(b.state.pages)
	for i := range b.p.pages {
		planned := &b.p.pages[i]
		if !planned.created {
			continue
		}
		var entity graph.Entity
		if planned.page.EntityID != nil {
			entity = b.resolved[*planned.page.EntityID]
		}
		for _, evidence := range pagemap.Cannibalization(planned.page, entity, index, g).Evidence {
			b.p.report.Cannibalization = append(b.p.report.Cannibalization, conflictView(evidence))
			b.p.noteAt(b.p.whole(), string(importmap.FieldPath), CodeCannibalization,
				"the page "+planned.page.Path+" overlaps "+evidence.Path+" ("+string(evidence.Reason)+")")
		}
	}
	return nil
}

func finalPages(state siteState, p *plan) []pagemap.Page {
	planned := make(map[string]pagemap.Page, len(p.pages))
	final := make([]pagemap.Page, 0, len(state.pages)+len(p.pages))
	for i := range p.pages {
		if p.pages[i].created {
			final = append(final, p.pages[i].page)
			continue
		}
		planned[p.pages[i].page.ID] = p.pages[i].page
	}
	for i := range state.pages {
		page, replaced := planned[state.pages[i].ID]
		if !replaced {
			page = state.pages[i]
		}
		final = append(final, page)
	}
	return final
}

func (b *builder) linkParents() {
	addressed := slices.DeleteFunc(slices.Clone(b.final), func(page pagemap.Page) bool { return page.WPType.StoreAddressed() })
	index := pagemap.NewIndex(addressed)
	for i := range b.p.pages {
		if planned := &b.p.pages[i].page; planned.ParentPageID == nil {
			planned.ParentPageID = index.PathParentID(*planned)
		}
	}
}

func (b *builder) markCanonical() {
	owners := make(map[string][]string, len(b.resolved))
	for i := range b.final {
		if owner := b.final[i].EntityID; owner != nil {
			owners[*owner] = append(owners[*owner], b.final[i].ID)
		}
	}
	for _, at := range slices.Sorted(maps.Keys(b.resolved)) {
		entity := b.resolved[at]
		if entity.CanonicalPageID == nil && len(owners[entity.ID]) == 1 {
			b.p.canonical = append(b.p.canonical, canonical{entityID: entity.ID, pageID: owners[entity.ID][0]})
		}
	}
}
