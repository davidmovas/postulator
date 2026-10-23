package imports

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type pageDraft struct {
	path        string
	url         string
	title       string
	h1          string
	metaTitle   string
	metaDesc    string
	wpType      string
	pageType    string
	entity      string
	keywords    keyword.List
	notes       []note.Note
	trail       []string
	own         ownership
	modeType    pagemap.WPType
	match       pagemap.Match
	rows        []int
	unit        int
	generated   bool
	entityOnly  bool
	typeUnknown bool
	at          importmap.Origin
}

func (p *pageDraft) cellType() pagemap.WPType {
	wpType := pagemap.WPType(strings.ToLower(strings.TrimSpace(p.wpType)))
	if p.typeUnknown || !wpType.Valid() {
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

func (p *pageDraft) page() (pagemap.Page, bool) {
	return p.match.Page, p.match.Found()
}

func (p *pageDraft) plansAtItsURL() bool {
	if p.url == "" || p.path != p.url {
		return false
	}
	current, exists := p.page()
	return !exists || (current.WPID == nil && current.Status != pagemap.StatusArchived)
}

func (p *pageDraft) sheetPath() string {
	if p.url != "" {
		return p.url
	}
	return p.path
}

func (p *pageDraft) matchedBy() string {
	if !p.match.Found() {
		return ""
	}
	return string(p.match.Rule)
}

func (p *pageDraft) merge(row *rowDraft, at int) {
	p.title = fill(p.title, row.title)
	p.h1 = fill(p.h1, row.h1)
	p.metaTitle = fill(p.metaTitle, row.metaTitle)
	p.metaDesc = fill(p.metaDesc, row.metaDesc)
	p.wpType = fill(p.wpType, row.wpType)
	p.pageType = fill(p.pageType, row.pageType)
	p.keywords = p.keywords.Merge(row.keywords)
	p.notes = note.Merge(p.notes, row.pageNotes())
	if len(p.trail) == 0 {
		p.trail = row.expected
	}
	if p.own == ownUnset {
		p.own = row.own
	}
	p.rows = append(p.rows, at)
}

type drafts struct {
	pages  map[string]*pageDraft
	byURL  map[string]*pageDraft
	sorted []string
}

func newDrafts() *drafts {
	return &drafts{pages: make(map[string]*pageDraft), byURL: make(map[string]*pageDraft)}
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

func (d *drafts) at(path string) (*pageDraft, bool) {
	if draft, held := d.pages[path]; held {
		return draft, true
	}
	draft, held := d.byURL[path]
	return draft, held
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
		} else {
			draft.url, draft.match = row.url, row.match
			if row.modeProduct {
				draft.modeType = pagemap.WPProduct
			}
		}
		if _, claimed := sheet.byURL[row.url]; row.url != "" && !claimed {
			sheet.byURL[row.url] = draft
		}
		draft.merge(row, i)
	}
	return sheet
}

func fillGaps(sheet *drafts, state *siteState, p *plan) {
	needsPage := make(map[string]bool)
	order := make([]string, 0)
	for _, path := range sheet.sortedPaths() {
		draft := sheet.pages[path]
		if !draft.plansAtItsURL() {
			continue
		}
		product := draft.product()
		for parent := pagemap.ParentPath(draft.url); parent != "" && parent != pagemap.RootPath; parent = pagemap.ParentPath(parent) {
			if _, planned := sheet.at(parent); planned {
				continue
			}
			if _, exists := state.pageAt(parent); exists {
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
