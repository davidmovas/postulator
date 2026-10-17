package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func seedVariable(t *testing.T, server *wptest.Server) (product wptest.Item, variations []wptest.Item) {
	t.Helper()

	stock := int64(12)
	product = server.Seed(wptest.Item{
		Type: wptest.TypeProduct, Title: "BPC-157", Slug: "bpc-157", ProductType: "variable",
		RegularPrice: "19.90", SalePrice: "15.00", SKU: "BPC157", StockStatus: "instock",
		ManageStock: true, StockQuantity: &stock,
		MetaData: []wptest.MetaEntry{{ID: 91, Key: "origin", Value: "Peru"}},
	})[0]
	variations = server.Seed(
		wptest.Item{Type: wptest.TypeProductVariation, Parent: product.ID, SKU: "BPC157-5", RegularPrice: "19.90", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "5 mg"}}},
		wptest.Item{Type: wptest.TypeProductVariation, Parent: product.ID, SKU: "BPC157-10", RegularPrice: "29.90", ManageStock: true, Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "10 mg"}}},
	)
	return product, variations
}

func TestTheProductCarriesItsStoreFacts(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	product, variations := seedVariable(t, server)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/"+itoa(product.ID)+"?context=edit", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	var got struct {
		Price         string `json:"price"`
		SalePrice     string `json:"sale_price"`
		StockStatus   string `json:"stock_status"`
		StockQuantity *int64 `json:"stock_quantity"`
		MetaData      []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
			ID    int64  `json:"id"`
		} `json:"meta_data"`
		Variations  []int64 `json:"variations"`
		ManageStock bool    `json:"manage_stock"`
	}
	decode(t, payload, &got)
	if got.Price != "15.00" || got.SalePrice != "15.00" || got.StockStatus != "instock" || !got.ManageStock || got.StockQuantity == nil || *got.StockQuantity != 12 {
		t.Errorf("facts = %+v", got)
	}
	if len(got.MetaData) != 1 || got.MetaData[0].Key != "origin" || got.MetaData[0].Value != "Peru" || got.MetaData[0].ID != 91 {
		t.Errorf("meta data = %+v", got.MetaData)
	}
	if len(got.Variations) != 2 || got.Variations[0] != variations[0].ID || got.Variations[1] != variations[1].ID {
		t.Errorf("variations = %v", got.Variations)
	}

	response, payload = call(t, server, http.MethodGet, "/wp-json/wc/v3/products?type=variable", nil, true)
	var listed []map[string]any
	decode(t, payload, &listed)
	if response.StatusCode != http.StatusOK || len(listed) != 1 {
		t.Errorf("by type = %d, %d items", response.StatusCode, len(listed))
	}
}

func TestVariationsAreListedUnderTheirProduct(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	product, variations := seedVariable(t, server)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/"+itoa(product.ID)+"/variations?context=edit", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	if got := response.Header.Get("X-WP-Total"); got != "2" {
		t.Errorf("X-WP-Total = %q, want 2", got)
	}
	var listed []struct {
		ManageStock any    `json:"manage_stock"`
		SKU         string `json:"sku"`
		Attributes  []struct {
			Name   string `json:"name"`
			Option string `json:"option"`
			ID     int64  `json:"id"`
		} `json:"attributes"`
		ID       int64 `json:"id"`
		ParentID int64 `json:"parent_id"`
	}
	decode(t, payload, &listed)
	if len(listed) != 2 || listed[0].ID != variations[0].ID || listed[0].ParentID != product.ID || listed[0].SKU != "BPC157-5" {
		t.Fatalf("variations = %+v", listed)
	}
	if len(listed[0].Attributes) != 1 || listed[0].Attributes[0].Option != "5 mg" || listed[0].Attributes[0].Name != "Size" {
		t.Errorf("attributes = %+v", listed[0].Attributes)
	}
	if listed[0].ManageStock != "parent" || listed[1].ManageStock != true {
		t.Errorf("manage_stock = %v and %v, want parent and true", listed[0].ManageStock, listed[1].ManageStock)
	}

	response, _ = call(t, server, http.MethodGet, "/wp-json/wc/v3/products/9999/variations", nil, true)
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("an unknown product = %d, want 404", response.StatusCode)
	}
}

func TestTheStoreSettingsAnswerThePageIDs(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SetStorePages(wptest.StorePages{Shop: 5, Cart: 6, Checkout: 7, Account: 8})

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/settings/products/woocommerce_shop_page_id", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	var shop struct {
		Value string `json:"value"`
	}
	decode(t, payload, &shop)
	if shop.Value != "5" {
		t.Errorf("shop = %q, want 5", shop.Value)
	}

	response, payload = call(t, server, http.MethodGet, "/wp-json/wc/v3/settings/advanced", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	var advanced []struct {
		ID    string `json:"id"`
		Value string `json:"value"`
	}
	decode(t, payload, &advanced)
	values := make(map[string]string, len(advanced))
	for _, setting := range advanced {
		values[setting.ID] = setting.Value
	}
	for id, want := range map[string]string{"woocommerce_cart_page_id": "6", "woocommerce_checkout_page_id": "7", "woocommerce_myaccount_page_id": "8", "woocommerce_terms_page_id": ""} {
		if got, ok := values[id]; !ok || got != want {
			t.Errorf("%s = %q (%t), want %q", id, got, ok, want)
		}
	}

	refused := wptest.New(t, wptest.WithoutStoreSettingsAccess())
	response, _ = call(t, refused, http.MethodGet, "/wp-json/wc/v3/settings/advanced", nil, true)
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("status without manage_woocommerce = %d, want 403", response.StatusCode)
	}
}
