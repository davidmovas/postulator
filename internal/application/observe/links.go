package observe

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func Resolver(site pagemap.Site, page pagemap.Page) func(href string) (string, bool) {
	return func(href string) (string, bool) {
		path, kind := site.ResolveAt(page.Path, href)
		return path, kind == pagemap.LinkPath && path != ""
	}
}

func ContentLinks(page pagemap.Page, site pagemap.Site, found []block.LinkAt, held map[block.PlacedKey]bool,
	target func(path string) (pagemap.Page, bool), at time.Time) []pagemap.PageLink {
	resolve := Resolver(site, page)
	links := make([]pagemap.PageLink, 0, len(found))
	for i := range found {
		path, resolved := resolve(found[i].Href)
		if !resolved {
			continue
		}
		position := found[i].Position
		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: path, AnchorText: found[i].Anchor,
			Type: pagemap.LinkContextual, Region: string(regionContent),
			Source: pagemap.SourceRaw, Position: &position, Ours: found[i].Ours(path, held),
			Absolute: pagemap.AbsoluteHref(found[i].Href), ObservedAt: at,
		}
		if link.Ours {
			link.BlockHash = found[i].BlockHash
		}
		if linked, ok := target(path); ok {
			link.ToPageID = &linked.ID
		}
		if built, err := pagemap.NewPageLink(link); err == nil {
			links = append(links, built)
		}
	}
	return links
}
