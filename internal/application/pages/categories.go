package pages

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const pageCategoriesCapability = "page_categories"

type filing struct {
	tree       application.CategoryTree
	pagesCarry bool
}

func filedUnderACategory(p pagemap.Page) bool {
	_, filed := p.WPType.Taxonomy()
	return filed && p.CategoryID != ""
}

func (f filing) of(p pagemap.Page) (categories []dto.Category, needPlugin bool) {
	taxonomy, filed := p.WPType.Taxonomy()
	if !filed {
		return []dto.Category{}, false
	}
	categories = f.tree.Chain(p.CategoryID, taxonomy)
	return categories, p.WPType == pagemap.WPPage && len(categories) > 0 && !f.pagesCarry
}

func (s *Service) filingOf(ctx context.Context, siteID string, listed ...pagemap.Page) (filing, error) {
	if !slices.ContainsFunc(listed, filedUnderACategory) {
		return filing{}, nil
	}
	tree, err := application.LoadCategoryTree(ctx, siteID, s.categories, s.categoryTerms)
	if err != nil || tree.Empty() {
		return filing{}, err
	}
	owner, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return filing{}, err
	}
	return filing{
		tree:       tree,
		pagesCarry: owner.Plugin.Installed && slices.Contains(owner.Plugin.Capabilities, pageCategoriesCapability),
	}, nil
}

func (s *Service) viewOf(ctx context.Context, p pagemap.Page) (Page, error) {
	filed, err := s.filingOf(ctx, p.SiteID, p)
	if err != nil {
		return Page{}, err
	}
	return view(p, filed), nil
}

func (s *Service) categoryFilter(ctx context.Context, req ListRequest) ([]string, error) {
	if req.CategoryID == "" {
		return nil, nil
	}
	tree, err := application.LoadCategoryTree(ctx, req.SiteID, s.categories, nil)
	if err != nil {
		return nil, err
	}
	return tree.Subtree(req.CategoryID), nil
}

func (s *Service) ListCategories(ctx context.Context, req ListCategoriesRequest) (ListCategoriesResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return ListCategoriesResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return ListCategoriesResponse{}, err
	}
	tree, err := application.LoadCategoryTree(ctx, req.SiteID, s.categories, s.categoryTerms)
	if err != nil {
		return ListCategoriesResponse{}, err
	}
	if tree.Empty() {
		return ListCategoriesResponse{Categories: []CategoryNode{}}, nil
	}
	all, err := s.pages.ListBySite(ctx, req.SiteID)
	if err != nil {
		return ListCategoriesResponse{}, err
	}
	filed := make(map[string]int, len(all))
	for i := range all {
		if all[i].CategoryID != "" {
			filed[all[i].CategoryID]++
		}
	}
	return ListCategoriesResponse{Categories: categoryNodes(tree, filed)}, nil
}

func categoryNodes(tree application.CategoryTree, filed map[string]int) []CategoryNode {
	listed := tree.Categories()
	nodes := make([]CategoryNode, 0, len(listed))
	for i := range listed {
		node := CategoryNode{ID: listed[i].ID, Name: listed[i].Name}
		if parentID := listed[i].ParentID; parentID != "" {
			node.ParentID = &parentID
		}
		for _, below := range tree.Subtree(listed[i].ID) {
			node.Pages += filed[below]
		}
		if termID, held := tree.TermID(listed[i].ID, category.TaxonomyCategory); held {
			node.TermIDs.Category = &termID
		}
		if termID, held := tree.TermID(listed[i].ID, category.TaxonomyProductCategory); held {
			node.TermIDs.ProductCategory = &termID
		}
		nodes = append(nodes, node)
	}
	return nodes
}
