package pagemap

import "slices"

type Index struct {
	byID     map[string]Page
	byPath   map[string]Page
	byEntity map[string][]Page
	pages    []Page
}

func NewIndex(pages []Page) Index {
	sorted := slices.Clone(pages)
	slices.SortFunc(sorted, byPath)

	index := Index{
		byID:     make(map[string]Page, len(sorted)),
		byPath:   make(map[string]Page, len(sorted)),
		byEntity: make(map[string][]Page),
		pages:    sorted,
	}
	for i := range sorted {
		p := &sorted[i]
		index.byID[p.ID] = *p
		index.byPath[p.Path] = *p
		if p.EntityID != nil {
			index.byEntity[*p.EntityID] = append(index.byEntity[*p.EntityID], *p)
		}
	}
	return index
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

func (i Index) ByEntity(entityID string) []Page {
	return slices.Clone(i.byEntity[entityID])
}

func (i Index) Pages() []Page {
	return slices.Clone(i.pages)
}

func (i Index) Len() int {
	return len(i.pages)
}
