package imports

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type siteState struct {
	siteID     string
	entities   []graph.Entity
	edges      []graph.Edge
	pages      []pagemap.Page
	categories []category.Category
	byName     map[string][]graph.Entity
	byPath     map[string]pagemap.Page
	byPlanned  map[string]pagemap.Page
	edgeKeys   map[string]struct{}
}

func newSiteState(
	siteID string, entities []graph.Entity, edges []graph.Edge, pages []pagemap.Page, categories []category.Category,
) siteState {
	state := siteState{
		siteID: siteID, entities: entities, edges: edges, pages: pages, categories: categories,
		byName:    make(map[string][]graph.Entity, len(entities)),
		byPath:    make(map[string]pagemap.Page, len(pages)),
		byPlanned: make(map[string]pagemap.Page),
		edgeKeys:  make(map[string]struct{}, len(edges)),
	}
	for i := range entities {
		state.byName[graph.Key(entities[i].Name)] = append(state.byName[graph.Key(entities[i].Name)], entities[i])
	}
	for i := range pages {
		state.byPath[pages[i].Path] = pages[i]
		if pages[i].PlannedPath != "" {
			state.byPlanned[pages[i].PlannedPath] = pages[i]
		}
	}
	for i := range edges {
		state.edgeKeys[edgeKey(edges[i])] = struct{}{}
	}
	return state
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
	categories, err := s.deps.Categories.ListBySite(ctx, siteID)
	if err != nil {
		return siteState{}, err
	}
	return newSiteState(siteID, entities, edges, pages, categories), nil
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

func (s *siteState) held(path string) (pagemap.Page, bool) {
	if page, ok := s.byPath[path]; ok && page.WPID != nil {
		return page, true
	}
	if page, ok := s.byPlanned[path]; ok {
		return page, true
	}
	page, ok := s.byPath[path]
	return page, ok
}

func inTheStore(page pagemap.Page) bool {
	return page.WPType == pagemap.WPProduct && page.WPID != nil
}

func (s *siteState) sells() bool {
	return slices.ContainsFunc(s.pages, inTheStore)
}

func (s *siteState) storeHolds(row pagemap.ProductRow) bool {
	if page, held := s.held(row.Path); held && inTheStore(page) {
		return true
	}
	return pagemap.MatchProduct(row, s.pages).Found
}

func (s *siteState) rowTypeOf(binding importmap.Binding, rows [][]string) importmap.RowType {
	found := 0
	for _, row := range rows {
		raw := binding.Path(row)
		if raw == "" {
			continue
		}
		path, pathErr := pagemap.NormalizePath(raw)
		product := pagemap.ProductRow{Path: path, H1: binding.Text(row, importmap.FieldH1), Title: binding.Text(row, importmap.FieldTitle)}
		if pathErr != nil || !s.storeHolds(product) {
			return importmap.RowPages
		}
		found++
	}
	if found == 0 {
		return importmap.RowPages
	}
	return importmap.RowProducts
}
