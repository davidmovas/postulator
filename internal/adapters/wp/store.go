package wp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type VariationAttribute struct {
	Name   string
	Option string
	ID     int64
}

type Variation struct {
	Modified      time.Time
	Description   string
	Permalink     string
	Status        string
	SKU           string
	Price         string
	RegularPrice  string
	SalePrice     string
	StockStatus   string
	StockQuantity *int64
	Attributes    []VariationAttribute
	MetaData      []MetaEntry
	ManageStock   StockManagement
	ID            int64
	ParentID      int64
}

type ProductQuery struct {
	ModifiedAfter *time.Time
	Status        string
	Type          string
	Page          int
	PerPage       int
}

type StorePages struct {
	Shop     int64
	Cart     int64
	Checkout int64
	Account  int64
	Terms    int64
}

type productCategoryPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	ID   int64  `json:"id"`
}

type metaEntryPayload struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	ID    int64           `json:"id"`
}

type variationAttributePayload struct {
	Name   string `json:"name"`
	Option string `json:"option"`
	ID     int64  `json:"id"`
}

type variationPayload struct {
	Description     string                      `json:"description"`
	Permalink       string                      `json:"permalink"`
	Status          string                      `json:"status"`
	SKU             string                      `json:"sku"`
	Price           string                      `json:"price"`
	RegularPrice    string                      `json:"regular_price"`
	SalePrice       string                      `json:"sale_price"`
	StockStatus     string                      `json:"stock_status"`
	DateModifiedGMT string                      `json:"date_modified_gmt"`
	StockQuantity   quantity                    `json:"stock_quantity"`
	ManageStock     StockManagement             `json:"manage_stock"`
	Attributes      []variationAttributePayload `json:"attributes"`
	MetaData        []metaEntryPayload          `json:"meta_data"`
	ID              int64                       `json:"id"`
	ParentID        int64                       `json:"parent_id"`
}

type settingPayload struct {
	ID    string          `json:"id"`
	Value json.RawMessage `json:"value"`
}

type quantity struct {
	value *int64
}

func (q *quantity) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		q.value = nil
		return nil
	}

	var raw any
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return err
	}
	switch typed := raw.(type) {
	case float64:
		amount := int64(typed)
		q.value = &amount
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			q.value = nil
			return nil
		}
		amount := int64(parsed)
		q.value = &amount
	default:
		q.value = nil
	}
	return nil
}

type StockManagement string

const (
	StockManaged         StockManagement = "true"
	StockNotManaged      StockManagement = "false"
	StockManagedByParent StockManagement = "parent"
)

func (m *StockManagement) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = StockNotManaged
	switch typed := raw.(type) {
	case bool:
		if typed {
			*m = StockManaged
		}
	case string:
		switch typed {
		case string(StockManagedByParent):
			*m = StockManagedByParent
		case string(StockManaged), "yes":
			*m = StockManaged
		}
	}
	return nil
}

func metaEntries(payload []metaEntryPayload) []MetaEntry {
	entries := make([]MetaEntry, 0, len(payload))
	for _, entry := range payload {
		entries = append(entries, MetaEntry(entry))
	}
	return entries
}

func (p variationPayload) variation() Variation {
	attributes := make([]VariationAttribute, 0, len(p.Attributes))
	for _, attribute := range p.Attributes {
		attributes = append(attributes, VariationAttribute(attribute))
	}
	return Variation{
		ID:            p.ID,
		ParentID:      p.ParentID,
		Description:   p.Description,
		Permalink:     p.Permalink,
		Status:        p.Status,
		SKU:           p.SKU,
		Price:         p.Price,
		RegularPrice:  p.RegularPrice,
		SalePrice:     p.SalePrice,
		StockStatus:   p.StockStatus,
		StockQuantity: p.StockQuantity.value,
		ManageStock:   p.ManageStock,
		Attributes:    attributes,
		MetaData:      metaEntries(p.MetaData),
		Modified:      parseWPTime(p.DateModifiedGMT),
	}
}

func storeWindow(page, perPage int) url.Values {
	query := url.Values{}
	query.Set("context", "edit")
	query.Set("page", strconv.Itoa(pageNumber(page)))
	query.Set("per_page", strconv.Itoa(perPageSize(perPage)))
	query.Set("orderby", "id")
	query.Set("order", "asc")
	return query
}

func (q ProductQuery) values() url.Values {
	query := storeWindow(q.Page, q.PerPage)
	if q.ModifiedAfter != nil {
		query.Set("modified_after", q.ModifiedAfter.UTC().Format(wpTimeLayout))
		query.Set("dates_are_gmt", "true")
	}
	if q.Status != "" {
		query.Set("status", q.Status)
	}
	if q.Type != "" {
		query.Set("type", q.Type)
	}
	return query
}

func (c *Client) ListProducts(ctx context.Context, query ProductQuery) (Page[Product], error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      "/products",
		query:     query.values(),
	})
	if err != nil {
		return Page[Product]{}, err
	}

	var payload []productPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Page[Product]{}, err
	}
	products := make([]Product, 0, len(payload))
	for index := range payload {
		products = append(products, payload[index].product())
	}
	return newPage(resp, pageNumber(query.Page), products), nil
}

func (c *Client) ListVariations(ctx context.Context, productID int64, page, perPage int) (Page[Variation], error) {
	resp, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      resourcePath("/products", productID) + "/variations",
		query:     storeWindow(page, perPage),
	})
	if err != nil {
		return Page[Variation]{}, err
	}

	var payload []variationPayload
	if err := decodeJSON(body, &payload); err != nil {
		return Page[Variation]{}, err
	}
	variations := make([]Variation, 0, len(payload))
	for index := range payload {
		variations = append(variations, payload[index].variation())
	}
	return newPage(resp, pageNumber(page), variations), nil
}

func (c *Client) StoreSettings(ctx context.Context) (StorePages, error) {
	_, body, err := c.do(ctx, request{
		method:    http.MethodGet,
		namespace: wooNamespace,
		path:      "/settings/products/woocommerce_shop_page_id",
	})
	if err != nil {
		return StorePages{}, err
	}
	var shop settingPayload
	if err := decodeJSON(body, &shop); err != nil {
		return StorePages{}, err
	}

	_, body, err = c.do(ctx, request{method: http.MethodGet, namespace: wooNamespace, path: "/settings/advanced"})
	if err != nil {
		return StorePages{}, err
	}
	var advanced []settingPayload
	if err := decodeJSON(body, &advanced); err != nil {
		return StorePages{}, err
	}

	pages := StorePages{Shop: settingPageID(shop.Value)}
	for _, setting := range advanced {
		switch setting.ID {
		case "woocommerce_cart_page_id":
			pages.Cart = settingPageID(setting.Value)
		case "woocommerce_checkout_page_id":
			pages.Checkout = settingPageID(setting.Value)
		case "woocommerce_myaccount_page_id":
			pages.Account = settingPageID(setting.Value)
		case "woocommerce_terms_page_id":
			pages.Terms = settingPageID(setting.Value)
		}
	}
	return pages, nil
}

func settingPageID(raw json.RawMessage) int64 {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return max(int64(typed), 0)
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0
		}
		return max(parsed, 0)
	default:
		return 0
	}
}
