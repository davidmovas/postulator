package steps

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const variationsPerPage = 100

func observeReaders(ctx context.Context, client *wp.Client, owner site.Site) (observe.Readers, error) {
	readers := observe.Readers{Core: coreReader{client}, Render: renderReader{client}}
	if owner.Commerce == site.CommerceReady {
		readers.Store = storeReader{client}
	}
	capabilities, err := client.Capabilities(ctx)
	switch {
	case err == nil:
		readers.Plugin = pluginReader{client}
		readers.Manifest = observe.Manifest{
			Version: capabilities.Version, Required: wp.PluginVersion, Capabilities: capabilities.Names,
			PostTypes: capabilities.PostTypes, Taxonomies: capabilities.Taxonomies, Outdated: capabilities.Outdated(),
		}
	case !wp.IsPluginMissing(err):
		return observe.Readers{}, err
	}
	return readers, nil
}

type coreReader struct{ client *wp.Client }

func (r coreReader) PostTypes(ctx context.Context) ([]observe.PostType, error) {
	types, err := r.client.PostTypes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]observe.PostType, 0, len(types))
	for _, postType := range types {
		out = append(out, observe.PostType{
			Slug: postType.Slug, RestBase: postType.RestBase, Viewable: postType.Viewable, Hierarchical: postType.Hierarchical,
		})
	}
	return out, nil
}

func (r coreReader) ListItems(ctx context.Context, postType string, query observe.ItemQuery) (observe.ItemPage, error) {
	page, err := r.client.ListItems(ctx, wp.ItemType(postType), wp.ListQuery{
		Page: query.Page, PerPage: query.PerPage, Status: query.Statuses, EmbedTerms: true,
	})
	if err != nil {
		return observe.ItemPage{}, err
	}
	out := observe.ItemPage{Items: make([]observe.Item, 0, len(page.Items)), HasMore: page.HasMore}
	for i := range page.Items {
		item := page.Items[i]
		out.Items = append(out.Items, observe.Item{
			ID: item.ID, Parent: item.Parent, Type: string(item.Type), Link: item.Link, Slug: item.Slug, Status: item.Status,
			Title: item.Title, Content: item.Content, Excerpt: item.Excerpt, Modified: item.Modified, Terms: termsOf(item.Terms),
		})
	}
	return out, nil
}

func (r coreReader) SiteSettings(ctx context.Context) (observe.SiteSettings, error) {
	settings, err := r.client.SiteSettings(ctx)
	switch {
	case errors.IsCode(err, errors.Unauthorized):
		return observe.SiteSettings{}, nil
	case err != nil:
		return observe.SiteSettings{}, err
	}
	return observe.SiteSettings{ShowOnFront: settings.ShowOnFront, FrontPageID: settings.FrontPageID, PostsPageID: settings.PostsPageID}, nil
}

type storeReader struct{ client *wp.Client }

type attributePayload struct {
	Name      string   `json:"name"`
	Options   []string `json:"options"`
	Visible   bool     `json:"visible"`
	Variation bool     `json:"variation"`
}

type variationAttributePayload struct {
	Name   string `json:"name"`
	Option string `json:"option"`
}

func (r storeReader) ListProducts(ctx context.Context, query observe.ProductQuery) (observe.ProductPage, error) {
	page, err := r.client.ListProducts(ctx, wp.ProductQuery{ModifiedAfter: query.ModifiedAfter, Page: query.Page, PerPage: query.PerPage})
	if err != nil {
		return observe.ProductPage{}, err
	}
	out := observe.ProductPage{Items: make([]observe.Product, 0, len(page.Items)), HasMore: page.HasMore}
	for i := range page.Items {
		product := page.Items[i]
		attributes := make([]attributePayload, 0, len(product.Attributes))
		for _, attribute := range product.Attributes {
			attributes = append(attributes, attributePayload{
				Name: attribute.Name, Options: attribute.Options, Visible: attribute.Visible, Variation: attribute.Variation,
			})
		}
		encoded, marshalErr := json.Marshal(attributes)
		if marshalErr != nil {
			return observe.ProductPage{}, marshalErr
		}
		categories := make([]pagemap.Term, 0, len(product.Categories))
		for _, category := range product.Categories {
			categories = append(categories, pagemap.Term{
				Taxonomy: string(pagemap.WPProductCategory), TermID: category.ID, Name: category.Name, Slug: category.Slug,
			})
		}
		images := make([]observe.ProductImage, 0, len(product.Images))
		for _, image := range product.Images {
			images = append(images, observe.ProductImage{ID: image.ID, Alt: image.Alt, Title: image.Name})
		}
		out.Items = append(out.Items, observe.Product{
			ID: product.ID, Permalink: product.Permalink, Slug: product.Slug, Status: product.Status, Name: product.Name, Type: product.Type,
			Description: product.Description, ShortDescription: product.ShortDescription, Price: product.Price,
			RegularPrice: product.RegularPrice, SalePrice: product.SalePrice, SKU: product.SKU, StockStatus: product.StockStatus,
			StockQuantity: product.StockQuantity, ManageStock: string(product.ManageStock), Attributes: encoded,
			Categories: categories, VariationIDs: product.VariationIDs, Images: images, Modified: product.Modified,
			UpsellIDs: product.UpsellIDs, CrossSellIDs: product.CrossSellIDs,
		})
	}
	return out, nil
}

func (r storeReader) ListVariations(ctx context.Context, productID int64, page int) (observe.VariationPage, error) {
	listed, err := r.client.ListVariations(ctx, productID, page, variationsPerPage)
	if err != nil {
		return observe.VariationPage{}, err
	}
	out := observe.VariationPage{Items: make([]observe.Variation, 0, len(listed.Items)), HasMore: listed.HasMore}
	for i := range listed.Items {
		variation := listed.Items[i]
		attributes := make([]variationAttributePayload, 0, len(variation.Attributes))
		for _, attribute := range variation.Attributes {
			attributes = append(attributes, variationAttributePayload{Name: attribute.Name, Option: attribute.Option})
		}
		encoded, marshalErr := json.Marshal(attributes)
		if marshalErr != nil {
			return observe.VariationPage{}, marshalErr
		}
		out.Items = append(out.Items, observe.Variation{
			ID: variation.ID, ParentID: variation.ParentID, Description: variation.Description, Status: variation.Status,
			SKU: variation.SKU, Price: variation.Price, RegularPrice: variation.RegularPrice, SalePrice: variation.SalePrice,
			StockStatus: variation.StockStatus, StockQuantity: variation.StockQuantity, ManageStock: string(variation.ManageStock),
			Attributes: encoded, Modified: variation.Modified,
		})
	}
	return out, nil
}

func (r storeReader) StorePages(ctx context.Context) (site.TechnicalPages, error) {
	pages, err := r.client.StoreSettings(ctx)
	switch {
	case errors.IsCode(err, errors.Unauthorized):
		return site.TechnicalPages{}, nil
	case err != nil:
		return site.TechnicalPages{}, err
	}
	return site.TechnicalPages{Shop: pages.Shop, Cart: pages.Cart, Checkout: pages.Checkout, Account: pages.Account, Terms: pages.Terms}, nil
}

type pluginReader struct{ client *wp.Client }

func (r pluginReader) ListContent(ctx context.Context, query observe.ContentQuery) (observe.ContentPage, error) {
	types := make([]wp.ItemType, 0, len(query.Types))
	for _, name := range query.Types {
		types = append(types, wp.ItemType(name))
	}
	page, err := r.client.ListContent(ctx, wp.ContentQuery{Since: query.Since, Cursor: query.Cursor, Types: types, Limit: query.Limit})
	if err != nil {
		return observe.ContentPage{}, err
	}
	out := observe.ContentPage{Items: make([]observe.ContentItem, 0, len(page.Items))}
	if page.NextCursor != nil {
		out.NextCursor = *page.NextCursor
	}
	for i := range page.Items {
		item := page.Items[i]
		links := make([]observe.ContentLink, 0, len(item.Links))
		for _, link := range item.Links {
			links = append(links, observe.ContentLink{Href: link.Href, Anchor: link.Anchor})
		}
		out.Items = append(out.Items, observe.ContentItem{
			ID: item.ID, Parent: item.Parent, Type: string(item.Type), Path: item.Path, Slug: item.Slug, Status: item.Status,
			Title: item.Title, H1: item.H1, ContentHash: item.ContentHash, Excerpt: item.Excerpt, Modified: item.Modified,
			Meta:  observe.SEOMeta{Title: item.Meta.Title, Description: item.Meta.Description, Canonical: item.Meta.Canonical},
			Links: links, Terms: termsOf(item.Terms),
		})
	}
	return out, nil
}

func (r pluginReader) GetRaw(ctx context.Context, postType string, id int64) (observe.Raw, error) {
	raw, err := r.client.GetRaw(ctx, wp.ItemType(postType), id)
	if err != nil {
		return observe.Raw{}, err
	}
	return observe.Raw{Content: raw.Content, Hash: raw.ContentHash}, nil
}

func (r pluginReader) GetRendered(ctx context.Context, postType string, id int64) (observe.Rendered, error) {
	rendered, err := r.client.GetRendered(ctx, wp.ItemType(postType), id)
	if err != nil {
		return observe.Rendered{}, err
	}
	return observe.Rendered{Content: rendered.Content, Hash: rendered.ContentHash}, nil
}

func (r pluginReader) GetSEOMeta(ctx context.Context, postType string, id int64) (observe.SEOMeta, error) {
	meta, err := r.client.GetSEOMeta(ctx, wp.ItemType(postType), id)
	if err != nil {
		return observe.SEOMeta{}, err
	}
	return observe.SEOMeta{Title: meta.Title, Description: meta.Description, Canonical: meta.Canonical}, nil
}

func (r pluginReader) GetTerms(ctx context.Context, postType string, id int64) ([]pagemap.Term, error) {
	terms, err := r.client.GetTerms(ctx, wp.ItemType(postType), id)
	if err != nil {
		return nil, err
	}
	return termsOf(terms), nil
}

func (r pluginReader) GetMeta(ctx context.Context, postType string, id int64, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for asked := range slices.Chunk(keys, wp.MaxMetaKeys) {
		found, err := r.client.GetMeta(ctx, wp.ItemType(postType), id, asked)
		if err != nil {
			return nil, err
		}
		for key, raw := range found {
			values[key] = metaText(raw)
		}
	}
	return values, nil
}

func metaText(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	if trimmed := strings.TrimSpace(string(raw)); trimmed != "null" {
		return trimmed
	}
	return ""
}

func (r pluginReader) Settings(ctx context.Context) (site.TechnicalPages, error) {
	settings, err := r.client.PluginSettings(ctx)
	if err != nil {
		return site.TechnicalPages{}, err
	}
	pages := settings.Pages
	return site.TechnicalPages{
		Front: pages.Front, Posts: pages.Posts, Privacy: pages.Privacy, Shop: pages.Shop,
		Cart: pages.Cart, Checkout: pages.Checkout, Account: pages.Account, Terms: pages.Terms,
	}, nil
}

func (r pluginReader) PreviewLink(ctx context.Context, postType string, id int64) (string, error) {
	link, err := r.client.PreviewLink(ctx, wp.ItemType(postType), id)
	if err != nil {
		return "", err
	}
	return link.URL, nil
}

type renderReader struct{ client *wp.Client }

func (r renderReader) Visit(ctx context.Context, link string) (observe.Visit, error) {
	visit, err := r.client.Visit(ctx, link)
	if err != nil {
		return observe.Visit{}, err
	}
	return observe.Visit{Body: visit.Body, Status: visit.Status, URL: visit.URL}, nil
}

func (r renderReader) Resolve(ctx context.Context, link string) (observe.Resolution, error) {
	resolved, err := r.client.Resolve(ctx, link)
	if err != nil {
		return observe.Resolution{}, err
	}
	return observe.Resolution{URL: resolved.URL, Status: resolved.Status}, nil
}

func termsOf(terms []wp.Term) []pagemap.Term {
	out := make([]pagemap.Term, 0, len(terms))
	for _, term := range terms {
		out = append(out, pagemap.Term{Taxonomy: term.Taxonomy, TermID: term.ID, Name: term.Name, Slug: term.Slug, ParentTermID: term.Parent})
	}
	return out
}
