package imports

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

type siteState struct {
	siteID    string
	entities  []graph.Entity
	edges     []graph.Edge
	pages     []pagemap.Page
	site      pagemap.RowSite
	byName    map[string][]graph.Entity
	byPath    map[string]pagemap.Page
	edgeKeys  map[string]struct{}
	types     map[pagemap.WPType]struct{}
	pageTypes []pagetype.Type
	fields    map[string][]field.Value
}

func newSiteState(siteID string, entities []graph.Entity, edges []graph.Edge, pages []pagemap.Page, pageTypes []pagetype.Type,
	fields map[string][]field.Value) siteState {
	state := siteState{
		siteID: siteID, entities: entities, edges: edges, pages: pages, pageTypes: pageTypes, fields: fields,
		site:     pagemap.NewRowSite(pages, fields, entities),
		byName:   make(map[string][]graph.Entity, len(entities)),
		byPath:   make(map[string]pagemap.Page, len(pages)),
		edgeKeys: make(map[string]struct{}, len(edges)),
		types: map[pagemap.WPType]struct{}{
			pagemap.WPPage: {}, pagemap.WPPost: {}, pagemap.WPProduct: {}, pagemap.WPProductCategory: {},
		},
	}
	for i := range entities {
		state.byName[graph.Key(entities[i].Name)] = append(state.byName[graph.Key(entities[i].Name)], entities[i])
	}
	for i := range pages {
		state.byPath[pages[i].Path] = pages[i]
		state.types[pages[i].WPType] = struct{}{}
	}
	for i := range edges {
		state.edgeKeys[edgeKey(edges[i])] = struct{}{}
	}
	return state
}

func (s *siteState) knows(wpType pagemap.WPType) bool {
	_, known := s.types[wpType]
	return known
}

func resolveTypes(sheet *drafts, state *siteState) {
	for _, draft := range sheet.pages {
		if cell := draft.cellType(); cell != "" && !state.knows(cell) {
			draft.typeUnknown = true
		}
	}
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
	pageTypes, err := s.deps.Types.ListBySite(ctx, siteID)
	if err != nil {
		return siteState{}, err
	}
	fields, err := s.deps.Fields.ListBySite(ctx, siteID)
	if err != nil {
		return siteState{}, err
	}
	return newSiteState(siteID, entities, edges, pages, pageTypes, fields), nil
}

func (s *siteState) storeName(pageID string) string {
	values := s.fields[pageID]
	for i := range values {
		if values[i].Key == field.Title {
			return values[i].Current
		}
	}
	return ""
}

func (s *siteState) typeName(typeID *string) string {
	if typeID == nil {
		return ""
	}
	for i := range s.pageTypes {
		if s.pageTypes[i].ID == *typeID {
			return s.pageTypes[i].Name
		}
	}
	return ""
}

func (s *siteState) pageType(cell string, wpType pagemap.WPType) (pagetype.Type, bool) {
	return pagetype.Match(cell, s.pageTypes, wpType)
}

func sortStored(entities []graph.Entity, edges []graph.Edge, pages []pagemap.Page) {
	slices.SortFunc(entities, func(a, b graph.Entity) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), strings.Compare(a.ID, b.ID))
	})
	slices.SortFunc(edges, func(a, b graph.Edge) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), strings.Compare(a.ID, b.ID))
	})
	slices.SortFunc(pages, func(a, b pagemap.Page) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), strings.Compare(a.ID, b.ID))
	})
}

func inTheStore(page pagemap.Page) bool {
	return page.WPType == pagemap.WPProduct && page.WPID != nil
}

func (s *siteState) sells() bool {
	return slices.ContainsFunc(s.pages, inTheStore)
}

func (s *siteState) storeHolds(row *rowDraft) bool {
	found := s.site.Match(row.asked(true))
	return found.Found() && inTheStore(found.Page)
}

func (s *siteState) rowTypeOf(binding importmap.Binding, rows [][]string) importmap.RowType {
	found := 0
	for _, cells := range rows {
		row := cellsOf(binding, cells, importmap.Origin{})
		if raw := binding.Path(cells); raw != "" {
			url, err := pagemap.NormalizePath(raw)
			if err != nil {
				return importmap.RowPages
			}
			row.url = url
		}
		if row.url == "" && row.name == "" && row.title == "" && row.h1 == "" && row.sku == "" {
			continue
		}
		if !s.storeHolds(&row) {
			return importmap.RowPages
		}
		found++
	}
	if found == 0 {
		return importmap.RowPages
	}
	return importmap.RowProducts
}
