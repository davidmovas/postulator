package observe

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

var editableStatuses = []string{"publish", "future", "draft", "pending", "private"}

type item struct {
	Modified    time.Time
	Type        pagemap.WPType
	Path        string
	URL         string
	Slug        string
	Status      string
	Title       string
	H1          string
	Excerpt     string
	ContentHash string
	Content     string
	Meta        SEOMeta
	Links       []ContentLink
	Terms       []pagemap.Term
	WPID        int64
	ParentWPID  int64
	HasContent  bool
}

func (it item) mappedType() pagemap.WPType {
	if it.Type.Valid() {
		return it.Type
	}
	return pagemap.WPPage
}

func (s *Service) pull(ctx context.Context, state *State, readers Readers, install pagemap.Site, batch int) ([]item, bool, error) {
	if state.Source == SourcePlugin {
		return pullBulk(ctx, state, readers, install, batch)
	}
	return pullCore(ctx, state, readers, install, batch)
}

func pullBulk(ctx context.Context, state *State, readers Readers, install pagemap.Site, batch int) ([]item, bool, error) {
	page, err := readers.Plugin.ListContent(ctx, ContentQuery{
		Types: state.Types, Since: state.Since, Cursor: state.Cursor, Limit: batch,
	})
	if err != nil {
		return nil, false, err
	}
	out := make([]item, 0, len(page.Items))
	for i := range page.Items {
		pulled := &page.Items[i]
		path, normalizeErr := pagemap.NormalizePath(pulled.Path)
		if normalizeErr != nil {
			continue
		}
		out = append(out, item{
			WPID: pulled.ID, ParentWPID: pulled.Parent, Type: pagemap.WPType(pulled.Type), Path: path, URL: install.URL(path),
			Slug: pulled.Slug, Status: pulled.Status, Title: pulled.Title, H1: pulled.H1, Excerpt: pulled.Excerpt,
			ContentHash: pulled.ContentHash, Meta: pulled.Meta, Links: pulled.Links, Terms: pulled.Terms, Modified: pulled.Modified,
		})
	}
	state.Cursor = page.NextCursor
	return out, page.NextCursor == "", nil
}

func pullCore(ctx context.Context, state *State, readers Readers, install pagemap.Site, batch int) ([]item, bool, error) {
	index := 0
	if state.CoreType != "" {
		index = max(slices.Index(state.Types, state.CoreType), 0)
	}
	page := max(state.CorePage, 1)

	for ; index < len(state.Types); index++ {
		listed, err := readers.Core.ListItems(ctx, state.Types[index], ItemQuery{Page: page, PerPage: batch, Statuses: editableStatuses})
		if err != nil {
			return nil, false, err
		}
		out := make([]item, 0, len(listed.Items))
		for i := range listed.Items {
			if converted, ok := fromCore(listed.Items[i], state.Types[index], install); ok {
				out = append(out, converted)
			}
		}
		switch {
		case listed.HasMore:
			state.CoreType, state.CorePage = state.Types[index], page+1
			return out, false, nil
		case index+1 < len(state.Types):
			state.CoreType, state.CorePage = state.Types[index+1], 1
			return out, false, nil
		default:
			state.CoreType, state.CorePage = "", 0
			return out, true, nil
		}
	}
	return nil, true, nil
}

func fromCore(pulled Item, postType string, install pagemap.Site) (item, bool) {
	path, kind := install.Resolve(pulled.Link)
	if kind == pagemap.LinkExternal || kind == pagemap.LinkUnresolved {
		return item{}, false
	}
	if queryPermalink(pulled.Link) {
		path = ""
	}
	converted := item{
		WPID: pulled.ID, ParentWPID: pulled.Parent, Type: pagemap.WPType(postType), Path: path, URL: pulled.Link,
		Slug: pulled.Slug, Status: pulled.Status, Title: pulled.Title, Excerpt: pulled.Excerpt,
		ContentHash: block.BodyHash(pulled.Content), Content: pulled.Content, HasContent: true, Terms: pulled.Terms, Modified: pulled.Modified,
	}
	if doc, err := content.Parse(pulled.Content); err == nil {
		converted.H1 = headingOne(doc)
	}
	return converted, true
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

func fromProduct(product Product, install pagemap.Site) (item, bool) {
	path, kind := install.Resolve(product.Permalink)
	if kind != pagemap.LinkPath || path == "" {
		return item{}, false
	}
	return item{
		WPID: product.ID, Type: pagemap.WPProduct, Path: path, URL: product.Permalink, Slug: product.Slug, Status: product.Status,
		Title: product.Name, Excerpt: product.ShortDescription, ContentHash: block.BodyHash(product.Description),
		Content: product.Description, HasContent: true, Terms: product.Categories, Modified: product.Modified,
	}, true
}
