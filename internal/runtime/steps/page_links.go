package steps

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func persist(ctx context.Context, deps Deps, page pagemap.Page, links []pagemap.PageLink) error {
	return deps.inUnit(ctx, func(c context.Context) error {
		if err := deps.Pages.Update(c, page); err != nil {
			return err
		}
		return deps.Links.ReplaceForPage(c, page.ID, links)
	})
}

func updateAll(ctx context.Context, deps Deps, pages []pagemap.Page) error {
	return deps.inUnit(ctx, func(c context.Context) error {
		for i := range pages {
			if err := deps.Pages.Update(c, pages[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func adopt(ctx context.Context, deps Deps, page pagemap.Page, index pagemap.Index, site pagemap.Site,
	doc *content.Document, hash string) error {
	now := deps.now()
	next := page
	next.ContentHash = hash
	next.Drift = false
	next.LastSyncedAt = &now
	next.UpdatedAt = now

	return persist(ctx, deps, next, observedOn(page, index, site, doc.Links(), now))
}

func observedOn(page pagemap.Page, index pagemap.Index, site pagemap.Site, found []content.Link,
	at time.Time) []pagemap.PageLink {
	out := make([]pagemap.PageLink, 0, len(found))
	for i := range found {
		path, kind := site.Resolve(found[i].Href)
		if kind != pagemap.LinkPath || path == "" {
			continue
		}

		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID,
			ToURL: path, AnchorText: found[i].Anchor, Origin: pagemap.OriginObserved, ObservedAt: at,
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
