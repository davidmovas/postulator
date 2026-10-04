package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func linkParents(ctx context.Context, deps Deps, siteID string) error {
	pages, err := deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	index := pagemap.NewIndex(pages)
	now := deps.now()
	moved := make([]pagemap.Page, 0)
	for i := range pages {
		page := pages[i]
		wanted := index.PathParentID(page)
		if sameRef(page.ParentPageID, wanted) {
			continue
		}
		page.ParentPageID = wanted
		page.UpdatedAt = now
		moved = append(moved, page)
	}
	if len(moved) == 0 {
		return nil
	}
	return updateAll(ctx, deps, moved)
}

func sameRef(current, wanted *string) bool {
	if current == nil || wanted == nil {
		return current == nil && wanted == nil
	}
	return *current == *wanted
}

func resolveLinks(ctx context.Context, deps Deps, siteID string) error {
	pages, err := deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	index := pagemap.NewIndex(pages)
	pending := make(map[string][]pagemap.PageLink)
	for i := range pages {
		links, listErr := deps.Links.ListForPage(ctx, pages[i].ID)
		if listErr != nil {
			return listErr
		}
		if resolveInto(links, index) {
			pending[pages[i].ID] = links
		}
	}
	if len(pending) == 0 {
		return nil
	}

	return deps.inUnit(ctx, func(c context.Context) error {
		for pageID, links := range pending {
			if replaceErr := deps.Links.ReplaceForPage(c, pageID, links); replaceErr != nil {
				return replaceErr
			}
		}
		return nil
	})
}

func resolveInto(links []pagemap.PageLink, index pagemap.Index) bool {
	changed := false
	for i := range links {
		if links[i].ToPageID != nil {
			continue
		}
		target, ok := index.ByPath(links[i].ToURL)
		if !ok || target.ID == links[i].FromPageID {
			continue
		}
		links[i].ToPageID = &target.ID
		changed = true
	}
	return changed
}
