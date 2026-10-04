package steps

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

var coreTypes = []wp.ItemType{wp.TypePage, wp.TypePost}

type pulledItem struct {
	Modified    time.Time
	Type        pagemap.WPType
	Path        string
	Slug        string
	Status      string
	Title       string
	H1          string
	ContentHash string
	Meta        wp.ContentMeta
	Links       []wp.ContentLink
	WPID        int64
	ParentWPID  int64
}

func (p pulledItem) mappedType() pagemap.WPType {
	if p.Type.Valid() {
		return p.Type
	}
	return pagemap.WPPage
}

func pull(ctx context.Context, client *wp.Client, state SiteSyncResult, limit int,
	install pagemap.Site) ([]pulledItem, string, error) {
	if state.Source == SourcePlugin {
		return pullBulk(ctx, client, state.Cursor, limit)
	}
	return pullCore(ctx, client, state.Cursor, limit, install)
}

func pullBulk(ctx context.Context, client *wp.Client, cursor string, limit int) ([]pulledItem, string, error) {
	page, err := client.ListContent(ctx, wp.ContentQuery{Cursor: cursor, Limit: limit})
	if err != nil {
		return nil, "", err
	}

	out := make([]pulledItem, 0, len(page.Items))
	for i := range page.Items {
		item := &page.Items[i]
		path, normalizeErr := pagemap.NormalizePath(item.Path)
		if normalizeErr != nil {
			continue
		}
		out = append(out, pulledItem{
			WPID: item.ID, Type: pagemap.WPType(item.Type), Path: path, Slug: item.Slug,
			Status: item.Status, Title: item.Title, H1: item.H1, ContentHash: item.ContentHash,
			Modified: item.Modified, Meta: item.Meta, Links: item.Links,
		})
	}

	next := ""
	if page.NextCursor != nil {
		next = *page.NextCursor
	}
	return out, next, nil
}

func pullCore(ctx context.Context, client *wp.Client, cursor string, limit int,
	install pagemap.Site) ([]pulledItem, string, error) {
	position, err := decodeCore(cursor)
	if err != nil {
		return nil, "", err
	}

	for index := typeIndex(position.Type); index < len(coreTypes); index++ {
		itemType := coreTypes[index]
		page, listErr := client.ListItems(ctx, itemType, wp.ListQuery{
			Page: position.Page, PerPage: limit, Status: editableStatuses,
		})
		if listErr != nil {
			return nil, "", listErr
		}
		if len(page.Items) == 0 {
			position.Page = 1
			continue
		}

		out := make([]pulledItem, 0, len(page.Items))
		for i := range page.Items {
			converted, ok := fromCore(page.Items[i], itemType, install)
			if ok {
				out = append(out, converted)
			}
		}

		if page.HasMore {
			return out, encodeCore(coreCursor{Type: string(itemType), Page: position.Page + 1}), nil
		}
		if index+1 < len(coreTypes) {
			return out, encodeCore(coreCursor{Type: string(coreTypes[index+1]), Page: 1}), nil
		}
		return out, "", nil
	}
	return nil, "", nil
}

func fromCore(item wp.Item, itemType wp.ItemType, install pagemap.Site) (pulledItem, bool) {
	path, kind := install.Resolve(item.Link)
	if kind == pagemap.LinkExternal || kind == pagemap.LinkUnresolved {
		return pulledItem{}, false
	}

	if queryPermalink(item.Link) {
		path = ""
	}

	pulled := pulledItem{
		WPID: item.ID, ParentWPID: item.Parent, Type: pagemap.WPType(itemType), Path: path,
		Slug: item.Slug, Status: item.Status, Title: item.Title,
		ContentHash: wp.ContentHash(item.Content), Modified: item.Modified,
	}
	if doc, err := content.Parse(item.Content); err == nil {
		pulled.H1 = headingOne(doc)
		pulled.Links = internalLinks(doc, install)
	}
	return pulled, true
}

func queryPermalink(link string) bool {
	parsed, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return false
	}
	return parsed.RawQuery != ""
}

func headingOne(doc *content.Document) string {
	for _, heading := range doc.Headings() {
		if heading.Data == "h1" {
			return content.TextOf(heading)
		}
	}
	return ""
}

func internalLinks(doc *content.Document, install pagemap.Site) []wp.ContentLink {
	found := doc.Links()

	out := make([]wp.ContentLink, 0, len(found))
	for i := range found {
		path, kind := install.Resolve(found[i].Href)
		if kind != pagemap.LinkPath || path == "" {
			continue
		}
		out = append(out, wp.ContentLink{Href: path, Anchor: found[i].Anchor})
	}
	return out
}
