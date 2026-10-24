package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
)

func (s *Service) Map(ctx context.Context, siteID string) (contextpack.Map, error) {
	var tree contextpack.Map
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		snapshot, err := s.base(c, siteID)
		if err != nil {
			return err
		}
		tree = snapshot.Map(nil)
		return nil
	})
	return tree, err
}
