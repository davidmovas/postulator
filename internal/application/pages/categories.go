package pages

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const pageCategoriesCapability = "page_categories"

type filing struct {
	index      application.CategoryIndex
	pagesCarry bool
}

func filedUnderAnEntity(p pagemap.Page) bool {
	_, filed := p.WPType.Taxonomy()
	return filed && p.EntityID != nil
}

func (f filing) of(p pagemap.Page) (categories []dto.Category, needPlugin bool) {
	taxonomy, filed := p.WPType.Taxonomy()
	if !filed || p.EntityID == nil {
		return []dto.Category{}, false
	}
	categories = f.index.Of(*p.EntityID, taxonomy)
	return categories, p.WPType == pagemap.WPPage && len(categories) > 0 && !f.pagesCarry
}

func (s *Service) filingOf(ctx context.Context, siteID string, listed ...pagemap.Page) (filing, error) {
	if !slices.ContainsFunc(listed, filedUnderAnEntity) {
		return filing{}, nil
	}
	owner, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return filing{}, err
	}
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return filing{}, err
	}
	var terms []graph.Term
	if s.terms != nil {
		if terms, err = s.terms.ListBySite(ctx, siteID); err != nil {
			return filing{}, err
		}
	}
	return filing{
		index:      application.NewCategoryIndex(entities, terms),
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
