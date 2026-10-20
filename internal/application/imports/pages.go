package imports

import (
	"maps"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/id"
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
		sameRef(a.EntityID, b.EntityID) && sameRef(a.TemplateID, b.TemplateID) &&
		sameRef(a.PageTypeID, b.PageTypeID) && a.TypeSource == b.TypeSource && a.TypeReason == b.TypeReason
}

func (b *builder) planPages() error {
	for _, path := range b.sheet.sortedPaths() {
		if draft := b.sheet.pages[path]; !draft.entityOnly {
			if err := b.planPage(draft); err != nil {
				return err
			}
		}
	}
	return b.planSiteOwned()
}

func (b *builder) planPage(draft *pageDraft) error {
	if draft.wpType != "" && draft.cellType() == "" {
		b.p.noteAt(draft.at, string(importmap.FieldWPType), CodeUnknownWPType,
			"the wordpress type is not recognized and was read as a page: "+draft.wpType)
	}
	entityID := b.ownerOf(draft)

	current, exists := b.state.held(draft.path)
	switch {
	case exists:
		return b.updatePage(current, draft, entityID)
	case draft.path == pagemap.RootPath:
		b.p.noteAt(draft.at, string(importmap.FieldPath), CodeRootPageSkipped,
			"the root of the site already exists on WordPress, so the import does not plan it; sync the site first to map it")
		return nil
	default:
		return b.createPage(draft, entityID)
	}
}

func (b *builder) typeOf(draft *pageDraft, wpType pagemap.WPType) (pagetype.Type, bool) {
	if draft.pageType == "" {
		return pagetype.Type{}, false
	}
	named, known := b.state.pageType(draft.pageType, wpType)
	if !known {
		b.p.noteAt(draft.at, string(importmap.FieldPageType), CodeUnknownPageType,
			"no page type of the site answers to "+draft.pageType+", so the sheet sets no type on "+draft.path)
	}
	return named, known
}

func typedByImport(page *pagemap.Page, named pagetype.Type) {
	page.PageTypeID, page.TypeSource, page.TypeReason = &named.ID, pagemap.TypeSourceManual, pagemap.TypeByImport
}

func (b *builder) retype(next *pagemap.Page, current pagemap.Page, draft *pageDraft) {
	named, known := b.typeOf(draft, next.WPType)
	switch {
	case !known || sameRef(current.PageTypeID, &named.ID):
	case current.TypeReason == pagemap.TypeByUser || current.TypeReason == pagemap.TypeByAI:
		chooser := "a person"
		if current.TypeReason == pagemap.TypeByAI {
			chooser = "the agent"
		}
		b.p.noteAt(draft.at, string(importmap.FieldPageType), CodePageTypeKept,
			draft.path+" keeps the page type "+b.state.typeName(current.PageTypeID)+", which "+chooser+
				" chose; the sheet names it "+named.Name)
	default:
		typedByImport(next, named)
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

func (b *builder) createPage(draft *pageDraft, entityID *string) error {
	wpType := draft.createdType()
	title := fill(draft.title, titleFrom(draft.path))
	if wpType == pagemap.WPProduct {
		title = draft.title
	}
	technical := pagemap.TechnicalNone
	if draft.technical() {
		technical = pagemap.TechnicalWorkbook
	}
	planned := pagemap.Page{
		ID: id.New(), SiteID: b.state.siteID, Path: draft.path, WPType: wpType,
		Title: title, H1: draft.h1, MetaTitle: draft.metaTitle, MetaDescription: draft.metaDesc,
		Keywords: draft.keywords, Notes: draft.notes, Status: pagemap.StatusPlanned, EntityID: entityID,
		TechnicalReason: technical, CreatedAt: b.now, UpdatedAt: b.now,
	}
	if named, known := b.typeOf(draft, wpType); known {
		typedByImport(&planned, named)
	}
	page, err := pagemap.NewPage(planned)
	if err != nil {
		return err
	}
	b.p.pages = append(b.p.pages, plannedPage{page: page, created: true})
	b.p.report.Pages = append(b.p.report.Pages, b.pageView(page, draft, ActionCreate))
	return nil
}

func (b *builder) updatePage(current pagemap.Page, draft *pageDraft, entityID *string) error {
	next := current
	next.Title = fill(current.Title, draft.title)
	next.H1 = fill(current.H1, draft.h1)
	next.MetaTitle = fill(current.MetaTitle, draft.metaTitle)
	next.MetaDescription = fill(current.MetaDescription, draft.metaDesc)
	next.Keywords = current.Keywords.Merge(draft.keywords)
	next.Notes = note.Merge(current.Notes, draft.notes)
	next.WPType = b.keptType(current, draft)
	b.retype(&next, current, draft)
	if current.Path != draft.path {
		next.PlannedPath = draft.path
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
		retyped := page.PageTypeID != nil && !sameRef(page.PageTypeID, current.PageTypeID)
		b.p.pages = append(b.p.pages, plannedPage{page: page, retyped: retyped})
	}
	b.p.report.Pages = append(b.p.report.Pages, b.pageView(page, draft, action))
	return nil
}

func (b *builder) pageView(page pagemap.Page, draft *pageDraft, action Action) PreviewPage {
	view := pageView(b.p.sheetAt(draft.at), page, draft, action, b.state.typeName(page.PageTypeID))
	if page.WPType == pagemap.WPProduct && page.WPID != nil {
		view.StoreName = b.state.storeName(page.ID)
	}
	return view
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
			if planned.ParentPageID != nil {
				planned.ParentReason = pagemap.ParentWorkbook
			}
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
