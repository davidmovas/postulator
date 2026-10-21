package steps

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
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

func adopt(ctx context.Context, deps Deps, page pagemap.Page, index pagemap.Index, site pagemap.Site,
	raw string, placed []block.LinkKey) error {
	written, err := deps.ourBlocks(ctx, page.ID)
	if err != nil {
		return err
	}
	stored, err := deps.Links.ListForPage(ctx, page.ID)
	if err != nil {
		return err
	}
	held := block.HeldOurs(stored)
	for _, key := range placed {
		held[key] = true
	}

	now := deps.now()
	next := page
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	return persist(ctx, deps, next, observedOn(page, index, site, block.LinksOf(raw, written), held, now))
}

func observedOn(page pagemap.Page, index pagemap.Index, site pagemap.Site, found []block.LinkAt,
	held map[block.LinkKey]bool, at time.Time) []pagemap.PageLink {
	out := make([]pagemap.PageLink, 0, len(found))
	for i := range found {
		path, kind := site.Resolve(found[i].Href)
		if kind != pagemap.LinkPath || path == "" {
			continue
		}

		position := found[i].Position
		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID,
			ToURL: path, AnchorText: found[i].Anchor, Position: &position,
			Type: pagemap.LinkContextual, Source: pagemap.SourceRaw, Ours: found[i].Ours(path, held),
			Absolute: pagemap.AbsoluteHref(found[i].Href), ObservedAt: at,
		}
		if target, ok := index.ByPath(path); ok {
			link.ToPageID = &target.ID
		}
		if built, ok := observedLink(link); ok {
			out = append(out, built)
		}
	}
	return out
}

func observedLink(link pagemap.PageLink) (pagemap.PageLink, bool) {
	built, err := pagemap.NewPageLink(link)
	if err != nil {
		return pagemap.PageLink{}, false
	}
	return built, true
}
