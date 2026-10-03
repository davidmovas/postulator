package graph

import (
	"context"

	"github.com/davidmovas/postulator/internal/application"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
)

func (s *Service) categoryIndex(ctx context.Context, siteID string, entities []graphdomain.Entity) (application.CategoryIndex, error) {
	if s.terms == nil {
		return application.NewCategoryIndex(entities, nil), nil
	}
	terms, err := s.terms.ListBySite(ctx, siteID)
	if err != nil {
		return application.CategoryIndex{}, err
	}
	return application.NewCategoryIndex(entities, terms), nil
}

func (s *Service) siteCategoryIndex(ctx context.Context, siteID string) (application.CategoryIndex, error) {
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return application.CategoryIndex{}, err
	}
	return s.categoryIndex(ctx, siteID, entities)
}

func (s *Service) viewsOf(ctx context.Context, siteID string, chosen []graphdomain.Entity) ([]Entity, error) {
	filed, err := s.siteCategoryIndex(ctx, siteID)
	if err != nil {
		return nil, err
	}
	return entityViews(chosen, filed), nil
}

func (s *Service) viewOf(ctx context.Context, entity graphdomain.Entity) (Entity, error) {
	filed, err := s.siteCategoryIndex(ctx, entity.SiteID)
	if err != nil {
		return Entity{}, err
	}
	return entityView(entity, filed), nil
}
