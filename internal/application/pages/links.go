package pages

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) ReplaceLinks(ctx context.Context, req ReplaceLinksRequest) (ReplaceLinksResponse, error) {
	var (
		stored []pagemap.PageLink
		siteID string
	)
	err := s.uow.Do(ctx, func(c context.Context) error {
		page, getErr := s.pages.Get(c, req.PageID)
		if getErr != nil {
			return getErr
		}
		siteID = page.SiteID
		now := s.now()

		links := make([]pagemap.PageLink, 0, len(req.Links))
		for i := range req.Links {
			input := &req.Links[i]
			if input.ToPageID != nil {
				target, targetErr := s.pages.Get(c, *input.ToPageID)
				if targetErr != nil {
					return targetErr
				}
				if target.SiteID != page.SiteID {
					return errors.New(errors.Invalid, "link target belongs to another site").WithDetail("toPageId", target.ID)
				}
			}
			origin := pagemap.LinkOrigin(input.Origin)
			if input.Origin == "" {
				origin = pagemap.OriginGenerated
			}
			link, linkErr := pagemap.NewPageLink(pagemap.PageLink{
				ID:         id.New(),
				SiteID:     page.SiteID,
				FromPageID: page.ID,
				ToPageID:   input.ToPageID,
				ToURL:      input.ToURL,
				AnchorText: input.AnchorText,
				Origin:     origin,
				ObservedAt: now,
			})
			if linkErr != nil {
				return linkErr
			}
			links = append(links, link)
		}

		if replaceErr := s.links.ReplaceForPage(c, page.ID, links); replaceErr != nil {
			return replaceErr
		}
		stored = links
		return nil
	})
	if err != nil {
		return ReplaceLinksResponse{}, err
	}
	if publishErr := s.changed(siteID); publishErr != nil {
		return ReplaceLinksResponse{}, publishErr
	}
	return ReplaceLinksResponse{Links: linkViews(stored)}, nil
}
