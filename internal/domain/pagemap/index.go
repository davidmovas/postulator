package pagemap

import (
	"slices"

	"github.com/davidmovas/postulator/internal/domain/field"
)

type Index struct {
	byID     map[string]Page
	byPath   map[string]Page
	byEntity map[string][]Page
	byPostID map[int64]Page
	byTermID map[int64]Page
	names    map[string][]string
	pages    []Page
}

func NewIndexNamed(pages []Page, names map[string][]string) Index {
	index := NewIndex(pages)
	index.names = names
	return index
}

func Names(stored map[string][]field.Value) map[string][]string {
	names := make(map[string][]string, len(stored))
	for pageID, values := range stored {
		for _, key := range []field.Key{field.Title, field.H1} {
			for i := range values {
				if values[i].Key == key && values[i].Current != "" {
					names[pageID] = append(names[pageID], values[i].Current)
				}
			}
		}
	}
	return names
}

func NewIndex(pages []Page) Index {
	sorted := slices.Clone(pages)
	slices.SortFunc(sorted, byPath)

	index := Index{
		byID:     make(map[string]Page, len(sorted)),
		byPath:   make(map[string]Page, len(sorted)),
		byEntity: make(map[string][]Page),
		byPostID: make(map[int64]Page),
		byTermID: make(map[int64]Page),
		pages:    sorted,
	}
	for i := range sorted {
		p := &sorted[i]
		index.byID[p.ID] = *p
		index.byPath[p.Path] = *p
		if p.EntityID != nil {
			index.byEntity[*p.EntityID] = append(index.byEntity[*p.EntityID], *p)
		}
		switch {
		case p.WPID == nil:
		case p.WPType.Term():
			index.byTermID[*p.WPID] = *p
		default:
			index.byPostID[*p.WPID] = *p
		}
	}
	return index
}

func (i Index) ByWordPressID(family WPType, id int64) (page Page, found bool) {
	if family.Term() {
		page, found = i.byTermID[id]
		return page, found
	}
	page, found = i.byPostID[id]
	return page, found
}

func (i Index) ByID(id string) (page Page, found bool) {
	page, found = i.byID[id]
	return page, found
}

func (i Index) ByPath(path string) (page Page, found bool) {
	normalized, err := NormalizePath(path)
	if err != nil {
		return Page{}, false
	}
	page, found = i.byPath[normalized]
	return page, found
}

func (i Index) PathParentID(page Page) *string {
	if page.WPType.StoreAddressed() {
		return nil
	}
	parent, found := i.ByPath(ParentPath(page.Path))
	if !found || parent.ID == page.ID {
		return nil
	}
	return &parent.ID
}

func (i Index) ByName(name string) []Page {
	out := make([]Page, 0)
	for at := range i.pages {
		if page := i.pages[at]; page.WPID != nil && page.Status != StatusArchived && carriesName(page, name, i.names[page.ID]) {
			out = append(out, page)
		}
	}
	return out
}

func (i Index) ByEntity(entityID string) []Page {
	return slices.Clone(i.byEntity[entityID])
}

func (i Index) Pages() []Page {
	return slices.Clone(i.pages)
}
