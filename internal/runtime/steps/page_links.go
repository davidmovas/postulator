package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func persist(ctx context.Context, deps Deps, page pagemap.Page, links []pagemap.PageLink) error {
	return deps.inUnit(ctx, func(c context.Context) error {
		if err := deps.Pages.UpdatePublished(c, page); err != nil {
			return err
		}
		return deps.Links.ReplaceForPage(c, page.ID, pagemap.SourceRaw, links)
	})
}

func onTheSite(pages []pagemap.Page) map[string]bool {
	live := make(map[string]bool, len(pages))
	for i := range pages {
		if pages[i].Live() {
			live[pages[i].ID] = true
		}
	}
	return live
}

type adoption struct {
	raw    string
	last   string
	placed []block.PlacedKey
	moved  []block.Moved
}

func adopt(ctx context.Context, deps Deps, page pagemap.Page, index pagemap.Index, site pagemap.Site, held adoption) error {
	written, err := deps.ourBlocks(ctx, page.ID)
	if err != nil {
		return err
	}
	stored, err := deps.Links.ListForPage(ctx, page.ID)
	if err != nil {
		return err
	}
	marks := block.HeldOurs(stored, block.LinksOf(held.last, written), observe.Resolver(site, page), held.moved)
	for _, key := range held.placed {
		marks[key] = true
	}

	now := deps.now()
	next := page
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	return persist(ctx, deps, next, observe.ContentLinks(page, site, block.LinksOf(held.raw, written), marks, index.ByPath, now))
}
