package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (s *Service) ClassificationFacts(ctx context.Context, siteID string) ([]pagetype.Facts, error) {
	var facts []pagetype.Facts
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		tree, err := s.folded(c, siteID)
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
		facts = tree.Facts(rendered, products)
		return nil
	})
	return facts, err
}
