package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (s *Service) ClassificationFacts(ctx context.Context, siteID string) ([]pagetype.Facts, error) {
	if err := s.Refresh(ctx, siteID); err != nil {
		return nil, err
	}
	var facts []pagetype.Facts
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		rows, err := s.deps.Maps.ListBySite(c, siteID)
		if err != nil {
			return err
		}
		pages, err := s.deps.Pages.ListBySite(c, siteID)
		if err != nil {
			return err
		}
		rendered, err := s.deps.Renders.ListRenderedBySite(c, siteID)
		if err != nil {
			return err
		}
		products, err := s.deps.Links.ListedProductsBySite(c, siteID)
		if err != nil {
			return err
		}
		facts = contextpack.FactsOf(rows, pagemap.NewIndex(pages), rendered, products)
		return nil
	})
	return facts, err
}
