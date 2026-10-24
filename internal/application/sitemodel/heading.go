package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

func (s *Service) OwesH1(ctx context.Context, pageID string) (bool, error) {
	var owed bool
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		page, err := s.deps.Pages.Get(c, pageID)
		if err != nil {
			return err
		}
		if page.WPID != nil {
			own, listErr := s.deps.Renders.ListForPages(c, []string{page.ID})
			if listErr != nil || len(own) == 0 {
				return listErr
			}
			owed = render.OwesH1(&own[0], nil)
			return nil
		}
		pages, err := s.deps.Pages.ListBySite(c, page.SiteID)
		if err != nil {
			return err
		}
		sameType := make(map[string]struct{}, len(pages))
		for i := range pages {
			if pages[i].WPType == page.WPType && pages[i].WPID != nil && pages[i].Status != pagemap.StatusArchived {
				sameType[pages[i].ID] = struct{}{}
			}
		}
		renders, err := s.deps.Renders.ListBySite(c, page.SiteID)
		if err != nil {
			return err
		}
		observed := make([]render.Record, 0, len(sameType))
		for i := range renders {
			if _, kept := sameType[renders[i].PageID]; kept {
				observed = append(observed, renders[i])
			}
		}
		owed = render.OwesH1(nil, observed)
		return nil
	})
	return owed, err
}
