package pages

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/application"
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
