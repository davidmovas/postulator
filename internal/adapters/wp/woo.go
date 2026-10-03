package wp

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Commerce string

const (
	CommerceAbsent    Commerce = "absent"
	CommerceForbidden Commerce = "forbidden"
	CommerceReady     Commerce = "ready"
)

var productEditCapabilities = []string{"edit_products", "edit_published_products", "edit_others_products"}

type ProductAttribute struct {
	Name      string
	Options   []string
	ID        int64
	Position  int
	Visible   bool
	Variation bool
}

type ProductImage struct {
	Src string
	Alt string
	ID  int64
}

type Product struct {
	Modified         time.Time
	Name             string
	Slug             string
	Permalink        string
	Type             string
	Status           string
	Description      string
	ShortDescription string
	Attributes       []ProductAttribute
	Images           []ProductImage
	ID               int64
}

type UpdateProduct struct {
	ShortDescription *string
	Attributes       *[]ProductAttribute
	Images           *[]int64
}

func (in UpdateProduct) payload() map[string]any {
	fields := make(map[string]any)
	if in.ShortDescription != nil {
		fields["short_description"] = *in.ShortDescription
	}
	if in.Attributes != nil {
		attributes := make([]map[string]any, 0, len(*in.Attributes))
		for _, attribute := range *in.Attributes {
			entry := map[string]any{
				"position":  attribute.Position,
				"visible":   attribute.Visible,
				"variation": attribute.Variation,
				"options":   append([]string{}, attribute.Options...),
			}
			if attribute.ID != 0 {
				entry["id"] = attribute.ID
			}
			if attribute.Name != "" {
				entry["name"] = attribute.Name
			}
			attributes = append(attributes, entry)
		}
		fields["attributes"] = attributes
	}
	if in.Images != nil {
		images := make([]map[string]any, 0, len(*in.Images))
		for _, id := range *in.Images {
			images = append(images, map[string]any{"id": id})
		}
		fields["images"] = images
	}
	return fields
}

type productAttributePayload struct {
	Name      string   `json:"name"`
	Options   []string `json:"options"`
	ID        int64    `json:"id"`
	Position  int      `json:"position"`
	Visible   bool     `json:"visible"`
	Variation bool     `json:"variation"`
}

type productImagePayload struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
	ID  int64  `json:"id"`
}

type productPayload struct {
	Name             string                    `json:"name"`
	Slug             string                    `json:"slug"`
	Permalink        string                    `json:"permalink"`
	Type             string                    `json:"type"`
	Status           string                    `json:"status"`
	Description      string                    `json:"description"`
	ShortDescription string                    `json:"short_description"`
	DateModifiedGMT  string                    `json:"date_modified_gmt"`
	Attributes       []productAttributePayload `json:"attributes"`
	Images           []productImagePayload     `json:"images"`
	ID               int64                     `json:"id"`
}

func (p productPayload) product() Product {
	attributes := make([]ProductAttribute, 0, len(p.Attributes))
	for _, attribute := range p.Attributes {
		options := attribute.Options
		if options == nil {
			options = []string{}
		}
		attributes = append(attributes, ProductAttribute{
			ID:        attribute.ID,
			Name:      attribute.Name,
			Options:   options,
			Position:  attribute.Position,
			Visible:   attribute.Visible,
			Variation: attribute.Variation,
		})
	}

	images := make([]ProductImage, 0, len(p.Images))
	for _, image := range p.Images {
		images = append(images, ProductImage(image))
	}

	return Product{
		ID:               p.ID,
		Name:             p.Name,
		Slug:             p.Slug,
		Permalink:        p.Permalink,
		Type:             p.Type,
		Status:           p.Status,
		Description:      p.Description,
		ShortDescription: p.ShortDescription,
		Attributes:       attributes,
		Images:           images,
		Modified:         parseWPTime(p.DateModifiedGMT),
	}
}

func (c *Client) GetProduct(ctx context.Context, id int64) (Product, error) {
	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      resourcePath("/products", id),
		query:     url.Values{"context": {"edit"}},
	})
	if err != nil {
		return Product{}, err
	}

	var payload productPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Product{}, err
	}
	if payload.Status == "trash" {
		return Product{}, errors.New(errors.NotFound, "the product is in the shop's trash").WithDetail("id", id)
	}
	return payload.product(), nil
}

func (c *Client) UpdateProduct(ctx context.Context, id int64, in UpdateProduct) (Product, error) {
	fields := in.payload()
	if len(fields) == 0 {
		return Product{}, errors.New(errors.Invalid, "the product update carries no fields")
	}

	body, err := encodeJSON(fields)
	if err != nil {
		return Product{}, err
	}

	_, raw, err := c.do(ctx, request{
		method:      http.MethodPost,
		namespace:   wooNamespace,
		path:        resourcePath("/products", id),
		query:       url.Values{"context": {"edit"}},
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

func (c *Client) Commerce(ctx context.Context) (Commerce, error) {
	_, _, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      "/products",
		query:     url.Values{"per_page": {"1"}, "_fields": {"id"}},
	})
	switch {
	case err == nil:
	case StoreAbsent(err):
		return CommerceAbsent, nil
	case StoreForbidden(err):
		return CommerceForbidden, nil
	default:
		return "", err
	}

	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: coreNamespace,
		path:      "/users/me",
		query:     url.Values{"context": {"edit"}, "_fields": {"capabilities"}},
	})
	if err != nil {
		return "", err
	}

	var me struct {
		Capabilities map[string]bool `json:"capabilities"`
	}
	if err := decodeJSON(body, &me); err != nil {
		return "", err
	}
	for _, capability := range productEditCapabilities {
		if !me.Capabilities[capability] {
			return CommerceForbidden, nil
		}
	}
	return CommerceReady, nil
}

func (c *Client) CommerceOr(ctx context.Context, known Commerce) (Commerce, error) {
	found, err := c.Commerce(ctx)
	switch {
	case err == nil:
		return found, nil
	case errors.IsCode(err, errors.External), errors.IsCode(err, errors.Unauthorized):
		return known, nil
	default:
		return "", err
	}
}

func StoreAbsent(err error) bool {
	return errors.IsCode(err, errors.NotFound) && detailString(err, "code") == "rest_no_route"
}

func StoreForbidden(err error) bool {
	status, ok := detailValue(err, "status")
	return errors.IsCode(err, errors.Unauthorized) && ok && status == http.StatusForbidden
}
