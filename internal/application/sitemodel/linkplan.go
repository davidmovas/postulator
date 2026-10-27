package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Service) LinkContextOf(ctx context.Context, siteID, pageID string) (content.LinkContext, error) {
	var plan content.LinkContext
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		snapshot, page, err := s.around(c, pageID)
		if err != nil {
			return err
		}
		if page.SiteID != siteID {
			return errors.New(errors.NotFound, "the site holds no such page").WithDetail("pageId", pageID)
		}
		key, err := fieldOf("", page, snapshot.Types)
		if err != nil {
			return err
		}
		_, plan = s.claimed(snapshot, page, key, nil)
		return nil
	})
	return plan, err
}
