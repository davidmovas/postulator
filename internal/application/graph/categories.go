package graph

import (
	"context"

	"github.com/davidmovas/postulator/internal/application"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type filing struct {
	tree  application.CategoryTree
	pages map[string]pagemap.Page
}

func newFiling(tree application.CategoryTree, entities []graphdomain.Entity, listed []pagemap.Page) filing {
	if tree.Empty() {
		return filing{}
	}
	byID := make(map[string]pagemap.Page, len(listed))
	owned := make(map[string][]pagemap.Page, len(listed))
	for i := range listed {
		byID[listed[i].ID] = listed[i]
		if entityID := listed[i].EntityID; entityID != nil {
			owned[*entityID] = append(owned[*entityID], listed[i])
		}
	}
	pages := make(map[string]pagemap.Page, len(entities))
	for i := range entities {
		if page, held := filedPageOf(entities[i], byID, owned[entities[i].ID]); held {
			pages[entities[i].ID] = page
		}
	}
	return filing{tree: tree, pages: pages}
}

func filedPageOf(e graphdomain.Entity, byID map[string]pagemap.Page, owned []pagemap.Page) (pagemap.Page, bool) {
	if e.CanonicalPageID != nil {
		if page, held := byID[*e.CanonicalPageID]; held {
			return page, true
		}
	}
	if len(owned) == 1 {
		return owned[0], true
	}
	return pagemap.Page{}, false
}

func (f filing) of(e graphdomain.Entity) []dto.Category {
	page, held := f.pages[e.ID]
	if !held {
		return []dto.Category{}
	}
	taxonomy, filed := page.WPType.Taxonomy()
	if !filed {
		return []dto.Category{}
	}
	return f.tree.Chain(page.CategoryID, taxonomy)
}

func (s *Service) categoryTree(ctx context.Context, siteID string) (application.CategoryTree, error) {
	return application.LoadCategoryTree(ctx, siteID, s.categories, s.categoryTerms)
}

func (s *Service) filingOf(ctx context.Context, siteID string, chosen []graphdomain.Entity) (filing, error) {
	tree, err := s.categoryTree(ctx, siteID)
	if err != nil || tree.Empty() {
		return filing{}, err
	}
	listed, err := s.pages.ListBySite(ctx, siteID)
	if err != nil {
		return filing{}, err
	}
	return newFiling(tree, chosen, listed), nil
}

func (s *Service) viewsOf(ctx context.Context, siteID string, chosen []graphdomain.Entity) ([]Entity, error) {
	filed, err := s.filingOf(ctx, siteID, chosen)
	if err != nil {
		return nil, err
	}
	return entityViews(chosen, filed), nil
}

func (s *Service) viewOf(ctx context.Context, entity graphdomain.Entity) (Entity, error) {
	filed, err := s.filingOf(ctx, entity.SiteID, []graphdomain.Entity{entity})
	if err != nil {
		return Entity{}, err
	}
	return entityView(entity, filed), nil
}
