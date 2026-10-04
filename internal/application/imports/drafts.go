package imports

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type pageDraft struct {
	path       string
	title      string
	h1         string
	metaTitle  string
	metaDesc   string
	wpType     string
	pageKind   string
	entity     string
	keywords   keyword.List
	notes      []pagemap.Note
	categories categoryLevels
	own        ownership
	modeType   pagemap.WPType
	matchedBy  pagemap.MatchedBy
	rows       []int
	unit       int
	generated  bool
	entityOnly bool
	at         importmap.Origin
}

func (p *pageDraft) kind() string {
	return strings.ToLower(strings.TrimSpace(p.pageKind))
}

func (p *pageDraft) cellType() pagemap.WPType {
	wpType := pagemap.WPType(strings.ToLower(strings.TrimSpace(p.wpType)))
	if !wpType.Valid() {
		return ""
	}
	return wpType
}

func (p *pageDraft) createdType() pagemap.WPType {
	return cmp.Or(p.cellType(), p.modeType, pagemap.WPPage)
}

func (p *pageDraft) technical() bool {
	return p.own == ownNo
}

func (p *pageDraft) merge(row *rowDraft, at int) {
	p.title = fill(p.title, row.title)
	p.h1 = fill(p.h1, row.h1)
	p.metaTitle = fill(p.metaTitle, row.metaTitle)
	p.metaDesc = fill(p.metaDesc, row.metaDesc)
	p.wpType = fill(p.wpType, row.wpType)
	p.pageKind = fill(p.pageKind, row.pageKind)
	p.keywords = p.keywords.Merge(row.keywords)
	p.notes = pagemap.MergeNotes(p.notes, row.notes)
	if len(p.categories) == 0 || len(p.categories.chain()) == 0 && len(row.categories.chain()) > 0 {
		p.categories = row.categories
	}
	if p.own == ownUnset {
		p.own = row.own
	}
	p.rows = append(p.rows, at)
}

type drafts struct {
	pages  map[string]*pageDraft
	sorted []string
}

func newDrafts() *drafts {
	return &drafts{pages: make(map[string]*pageDraft)}
}

func (d *drafts) page(path string, at importmap.Origin) (draft *pageDraft, known bool) {
	if current, held := d.pages[path]; held {
		return current, true
	}
	draft = &pageDraft{path: path, at: at, unit: -1}
	d.pages[path] = draft
	d.sorted = nil
	return draft, false
}

func (d *drafts) sortedPaths() []string {
	if d.sorted == nil {
		d.sorted = slices.Sorted(maps.Keys(d.pages))
	}
	return d.sorted
}

func pagesOf(rows []rowDraft, p *plan) *drafts {
	sheet := newDrafts()
	for i := range rows {
		row := &rows[i]
		if row.path == "" {
			continue
		}
		draft, known := sheet.page(row.path, row.at)
		if known {
			p.noteAt(row.at, string(importmap.FieldPath), CodeDuplicatePath, "the path repeats an earlier row and was merged: "+row.path)
		}
		draft.merge(row, i)
	}
	return sheet
}

func fillGaps(sheet *drafts, state *siteState, p *plan) {
	needsPage := make(map[string]bool)
	order := make([]string, 0)
	for _, path := range sheet.sortedPaths() {
		product := state.productDraft(sheet.pages[path])
		for parent := pagemap.ParentPath(path); parent != "" && parent != pagemap.RootPath; parent = pagemap.ParentPath(parent) {
			if _, planned := sheet.pages[parent]; planned {
				continue
			}
			if _, exists := state.held(parent); exists {
				continue
			}
			page, seen := needsPage[parent]
			if !seen {
				order = append(order, parent)
			}
			needsPage[parent] = page || !product
		}
	}

	for _, parent := range order {
		draft, _ := sheet.page(parent, p.whole())
		draft.generated = true
		draft.title = titleFrom(parent)
		if !needsPage[parent] {
			draft.entityOnly = true
			p.noteAt(draft.at, string(importmap.FieldPath), CodeIntermediateLevel,
				"the level above the products was kept as an entity without a page, because the store decides where a product sits: "+parent)
			continue
		}
		p.noteAt(draft.at, string(importmap.FieldPath), CodeIntermediatePath, "the missing intermediate path was created: "+parent)
	}
}
