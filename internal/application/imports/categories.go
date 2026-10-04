package imports

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type shelf struct {
	all    []category.Category
	byPath map[string]string
	trails map[string][]string
}

func newShelf(held []category.Category) *shelf {
	s := &shelf{all: slices.Clone(held), byPath: make(map[string]string, len(held)), trails: make(map[string][]string)}
	for i := range held {
		s.byPath[held[i].ParentID+"\x00"+held[i].Key] = held[i].ID
	}
	return s
}

func (s *shelf) find(parentID, name string) (string, bool) {
	at, held := s.byPath[parentID+"\x00"+category.Key(name)]
	return at, held
}

func (s *shelf) add(c category.Category) {
	s.all = append(s.all, c)
	s.byPath[c.ParentID+"\x00"+c.Key] = c.ID
}

func (s *shelf) trail(leafID string) []string {
	if known, held := s.trails[leafID]; held {
		return known
	}
	chain := category.Chain(s.all, leafID)
	names := make([]string, 0, len(chain))
	for i := range chain {
		names = append(names, chain[i].Name)
	}
	s.trails[leafID] = names
	return names
}

type filing struct {
	at   importmap.Origin
	rows int
}

func (b *builder) file(draft *pageDraft) (string, error) {
	parentID := ""
	for _, name := range draft.chain {
		at, held := b.shelf.find(parentID, name)
		if !held {
			made, err := category.New(category.Category{
				ID: id.New(), SiteID: b.state.siteID, Name: name, ParentID: parentID, CreatedAt: b.now, UpdatedAt: b.now,
			})
			if err != nil {
				return "", err
			}
			b.shelf.add(made)
			b.p.categories = append(b.p.categories, made)
			at = made.ID
		}
		b.tally(at, draft)
		parentID = at
	}
	return parentID, nil
}

func (b *builder) tally(categoryID string, draft *pageDraft) {
	held, seen := b.filings[categoryID]
	if !seen {
		held = &filing{at: draft.at}
		b.filings[categoryID] = held
	}
	held.rows += len(draft.rows)
}

func (b *builder) reportCategories() {
	made := make(map[string]struct{}, len(b.p.categories))
	for i := range b.p.categories {
		made[b.p.categories[i].ID] = struct{}{}
	}
	filed := slices.SortedFunc(maps.Keys(b.filings), func(x, y string) int {
		return cmp.Or(slices.Compare(b.shelf.trail(x), b.shelf.trail(y)), strings.Compare(x, y))
	})
	for _, at := range filed {
		action := CategoryMatch
		if _, created := made[at]; created {
			action = CategoryCreate
		}
		b.p.report.Categories = append(b.p.report.Categories, PreviewCategory{
			Sheet: b.p.sheetAt(b.filings[at].at), Path: b.shelf.trail(at), Action: string(action), Rows: b.filings[at].rows,
		})
	}
}

func unused(categories []category.Category, pages []pagemap.Page, terms []category.Term) []category.Category {
	parentOf := make(map[string]string, len(categories))
	for i := range categories {
		parentOf[categories[i].ID] = categories[i].ParentID
	}

	live := make(map[string]struct{}, len(categories))
	keep := func(leafID string) {
		for at := leafID; at != ""; at = parentOf[at] {
			if _, known := parentOf[at]; !known {
				return
			}
			if _, done := live[at]; done {
				return
			}
			live[at] = struct{}{}
		}
	}
	for i := range pages {
		keep(pages[i].CategoryID)
	}
	for i := range terms {
		keep(terms[i].CategoryID)
	}

	gone := slices.DeleteFunc(slices.Clone(categories), func(c category.Category) bool {
		_, held := live[c.ID]
		return held
	})
	slices.SortFunc(gone, func(a, b category.Category) int {
		return cmp.Or(cmp.Compare(depthOf(parentOf, b.ID), depthOf(parentOf, a.ID)), strings.Compare(a.Key, b.Key),
			strings.Compare(a.ID, b.ID))
	})
	return gone
}

func depthOf(parentOf map[string]string, at string) int {
	depth := 0
	for parentOf[at] != "" && depth < len(parentOf) {
		at = parentOf[at]
		depth++
	}
	return depth
}
