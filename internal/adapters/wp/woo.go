package wp

import (
	"context"
	"net/http"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type ProductCategoryRef struct {
	Name string
	Slug string
	ID   int64
}

type Product struct {
	Modified         time.Time
	Name             string
	Slug             string
	Permalink        string
	Status           string
	Description      string
	ShortDescription string
	Categories       []ProductCategoryRef
	ID               int64
	MenuOrder        int
}

type ProductCategory struct {
	Name        string
	Slug        string
	Description string
	ID          int64
	Parent      int64
	Count       int
}

type (
	ProductPage         = Page[Product]
	ProductCategoryPage = Page[ProductCategory]
)

type UpdateProduct struct {
	Description      *string
	ShortDescription *string
	Slug             *string
	Status           *string
	Categories       []int64
}

func (in UpdateProduct) payload() map[string]any {
	attributes := make(map[string]any)
	if in.Description != nil {
		attributes["description"] = *in.Description
	}
	if in.ShortDescription != nil {
		attributes["short_description"] = *in.ShortDescription
	}
	if in.Slug != nil {
		attributes["slug"] = *in.Slug
	}
	if in.Status != nil {
		attributes["status"] = *in.Status
	}
	if in.Categories != nil {
		refs := make([]map[string]any, 0, len(in.Categories))
		for _, id := range in.Categories {
			refs = append(refs, map[string]any{"id": id})
		}
		attributes["categories"] = refs
	}
	return attributes
}

func (p productPayload) product() Product {
	categories := make([]ProductCategoryRef, 0, len(p.Categories))
	for _, ref := range p.Categories {
		categories = append(categories, ProductCategoryRef(ref))
	}

	return Product{
		ID:               p.ID,
		Name:             p.Name,
		Slug:             p.Slug,
		Permalink:        p.Permalink,
		Status:           p.Status,
		Description:      p.Description,
		ShortDescription: p.ShortDescription,
		Categories:       categories,
		MenuOrder:        p.MenuOrder,
		Modified:         parseWPTime(p.DateModifiedGMT),
	}
}

func (p productCategoryPayload) productCategory() ProductCategory {
	return ProductCategory(p)
}

func (c *Client) ListProducts(ctx context.Context, query ListQuery) (ProductPage, error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      "/products",
		query:     query.values(TypeProduct),
	})
	if err != nil {
		return ProductPage{}, err
	}

	var payload []productPayload
	if err := decodeJSON(body, &payload); err != nil {
		return ProductPage{}, err
	}

	products := make([]Product, 0, len(payload))
	for index := range payload {
		products = append(products, payload[index].product())
	}
	return newPage(resp, query.pageNumber(), products), nil
}

func (c *Client) GetProduct(ctx context.Context, id int64) (Product, error) {
	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      resourcePath("/products", id),
	})
	if err != nil {
		return Product{}, err
	}

	var payload productPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Product{}, err
	}
	return payload.product(), nil
}

func (c *Client) UpdateProduct(ctx context.Context, id int64, in UpdateProduct) (Product, error) {
	attributes := in.payload()
	if len(attributes) == 0 {
		return Product{}, errors.New(errors.Invalid, "the product update carries no fields")
	}

	body, err := encodeJSON(attributes)
	if err != nil {
		return Product{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   wooNamespace,
		path:        resourcePath("/products", id),
		body:        body,
		contentType: contentTypeJSON,
	})
	if err != nil {
		return Product{}, err
	}

	var payload productPayload
	if err := decodeJSON(raw, &payload); err != nil {
		return Product{}, err
	}
	return payload.product(), nil
}

func (c *Client) ListProductCategories(ctx context.Context, query ListQuery) (ProductCategoryPage, error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      "/products/categories",
		query:     query.values(TypeProductCategory),
	})
	if err != nil {
		return ProductCategoryPage{}, err
	}

	var payload []productCategoryPayload
	if err := decodeJSON(body, &payload); err != nil {
		return ProductCategoryPage{}, err
	}

	categories := make([]ProductCategory, 0, len(payload))
	for index := range payload {
		categories = append(categories, payload[index].productCategory())
	}
	return newPage(resp, query.pageNumber(), categories), nil
}
