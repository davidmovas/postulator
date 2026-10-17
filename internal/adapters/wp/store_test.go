package wp_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type storeFixture struct {
	server     *wptest.Server
	category   wptest.Item
	simple     wptest.Item
	variable   wptest.Item
	variations []wptest.Item
}

func seedStore(t *testing.T, opts ...wptest.Option) storeFixture {
	t.Helper()

	server := wptest.New(t, opts...)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Peptides", Slug: "peptides"})[0]
	stock := int64(12)
	products := server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Slug: "powder", RegularPrice: "9.50", SKU: "PWD", StockStatus: "outofstock"},
		wptest.Item{
			Type: wptest.TypeProduct, Title: "BPC-157", Slug: "bpc-157", ProductType: "variable",
			RegularPrice: "19.90", SalePrice: "15.00", SKU: "BPC157", StockStatus: "instock",
			ManageStock: true, StockQuantity: &stock, Categories: []int64{category.ID},
			MetaData: []wptest.MetaEntry{{ID: 91, Key: "origin", Value: "Peru"}},
		},
		wptest.Item{Type: wptest.TypeProduct, Title: "TB-500", Slug: "tb-500", RegularPrice: "25.00", SKU: "TB500", Status: "draft"},
	)
	variable := products[1]
	variations := server.Seed(
		wptest.Item{Type: wptest.TypeProductVariation, Parent: variable.ID, Title: "BPC-157 5 mg", SKU: "BPC157-5", RegularPrice: "19.90", StockStatus: "instock", Content: "<p>Five.</p>", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "5 mg"}}},
		wptest.Item{Type: wptest.TypeProductVariation, Parent: variable.ID, Title: "BPC-157 10 mg", SKU: "BPC157-10", RegularPrice: "29.90", SalePrice: "25.00", StockStatus: "onbackorder", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "10 mg"}}},
	)
	return storeFixture{server: server, category: category, simple: products[0], variable: variable, variations: variations}
}

func TestListProductsWalksTheStoreWithItsFacts(t *testing.T) {
	t.Parallel()

	store := seedStore(t)
	client := newClient(t, store.server)

	first, err := client.ListProducts(t.Context(), wp.ProductQuery{PerPage: 2})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(first.Items) != 2 || !first.HasMore || first.Total != 3 {
		t.Fatalf("first page = %d items, more %t, total %d", len(first.Items), first.HasMore, first.Total)
	}
	recorded, _ := store.server.LastRequest()
	for key, want := range map[string]string{"context": "edit", "page": "1", "per_page": "2", "orderby": "id", "order": "asc"} {
		if got := recorded.Query.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	variable := first.Items[1]
	if variable.ID != store.variable.ID || variable.Type != "variable" || variable.SKU != "BPC157" {
		t.Fatalf("variable = %+v", variable)
	}
	if variable.Price != "15.00" || variable.RegularPrice != "19.90" || variable.SalePrice != "15.00" {
		t.Errorf("prices = %q %q %q", variable.Price, variable.RegularPrice, variable.SalePrice)
	}
	if variable.StockStatus != "instock" || variable.ManageStock != wp.StockManaged || variable.StockQuantity == nil || *variable.StockQuantity != 12 {
		t.Errorf("stock = %q %q %v", variable.StockStatus, variable.ManageStock, variable.StockQuantity)
	}
	if want := []wp.ProductCategory{{ID: store.category.ID, Name: "Peptides", Slug: "peptides"}}; !slices.Equal(variable.Categories, want) {
		t.Errorf("categories = %+v, want %+v", variable.Categories, want)
	}
	if len(variable.MetaData) != 1 || variable.MetaData[0].Key != "origin" || variable.MetaData[0].ID != 91 || string(variable.MetaData[0].Value) != `"Peru"` {
		t.Errorf("meta data = %+v", variable.MetaData)
	}
	if want := []int64{store.variations[0].ID, store.variations[1].ID}; !slices.Equal(variable.VariationIDs, want) {
		t.Errorf("variations = %v, want %v", variable.VariationIDs, want)
	}
	if simple := first.Items[0]; simple.Price != "9.50" || simple.StockQuantity != nil || simple.ManageStock != wp.StockNotManaged || simple.StockStatus != "outofstock" {
		t.Errorf("simple = %+v", simple)
	}

	second, err := client.ListProducts(t.Context(), wp.ProductQuery{PerPage: 2, Page: 2})
	if err != nil {
		t.Fatalf("ListProducts page 2: %v", err)
	}
	if len(second.Items) != 1 || second.HasMore || second.Items[0].SKU != "TB500" {
		t.Errorf("second page = %+v", second)
	}
}

func TestListProductsNarrowsByTypeStatusAndTime(t *testing.T) {
	t.Parallel()

	store := seedStore(t)
	client := newClient(t, store.server)

	variable, err := client.ListProducts(t.Context(), wp.ProductQuery{Type: "variable"})
	if err != nil {
		t.Fatalf("ListProducts by type: %v", err)
	}
	if len(variable.Items) != 1 || variable.Items[0].ID != store.variable.ID {
		t.Errorf("by type = %+v", variable.Items)
	}

	drafts, err := client.ListProducts(t.Context(), wp.ProductQuery{Status: "draft"})
	if err != nil {
		t.Fatalf("ListProducts by status: %v", err)
	}
	if len(drafts.Items) != 1 || drafts.Items[0].SKU != "TB500" {
		t.Errorf("by status = %+v", drafts.Items)
	}

	after := store.variable.Modified
	recent, err := client.ListProducts(t.Context(), wp.ProductQuery{ModifiedAfter: &after})
	if err != nil {
		t.Fatalf("ListProducts after a time: %v", err)
	}
	if len(recent.Items) != 1 || recent.Items[0].SKU != "TB500" {
		t.Errorf("after %s = %+v", after.Format(time.RFC3339), recent.Items)
	}
	recorded, _ := store.server.LastRequest()
	if recorded.Query.Get("modified_after") == "" || recorded.Query.Get("dates_are_gmt") != "true" {
		t.Errorf("query = %v, want modified_after read as GMT", recorded.Query)
	}
}

func TestListVariationsCarriesSKUPriceStockAndAttributes(t *testing.T) {
	t.Parallel()

	store := seedStore(t)
	client := newClient(t, store.server)

	page, err := client.ListVariations(t.Context(), store.variable.ID, 1, 0)
	if err != nil {
		t.Fatalf("ListVariations: %v", err)
	}
	if len(page.Items) != 2 || page.HasMore {
		t.Fatalf("variations = %+v", page)
	}

	five, ten := page.Items[0], page.Items[1]
	if five.ID != store.variations[0].ID || five.ParentID != store.variable.ID || five.SKU != "BPC157-5" || five.Description != "<p>Five.</p>" {
		t.Errorf("five = %+v", five)
	}
	if want := []wp.VariationAttribute{{ID: 3, Name: "Size", Option: "5 mg"}}; !slices.Equal(five.Attributes, want) {
		t.Errorf("attributes = %+v, want %+v", five.Attributes, want)
	}
	if five.ManageStock != wp.StockManagedByParent {
		t.Errorf("manage stock = %q, want a variation whose parent manages the stock to say parent", five.ManageStock)
	}
	if ten.Price != "25.00" || ten.RegularPrice != "29.90" || ten.SalePrice != "25.00" || ten.StockStatus != "onbackorder" {
		t.Errorf("ten = %+v", ten)
	}

	if _, err := client.ListVariations(t.Context(), store.simple.ID+1000, 1, 10); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("an unknown product = %v, want NotFound", err)
	}
}

func TestAVariationManagedByItsParentDecodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		parent    bool
		own       bool
		product   wp.StockManagement
		variation wp.StockManagement
	}{
		{name: "the parent manages the stock", parent: true, product: wp.StockManaged, variation: wp.StockManagedByParent},
		{name: "the variation manages its own", parent: true, own: true, product: wp.StockManaged, variation: wp.StockManaged},
		{name: "only the variation manages", own: true, product: wp.StockNotManaged, variation: wp.StockManaged},
		{name: "nobody manages the stock", product: wp.StockNotManaged, variation: wp.StockNotManaged},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			product := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "BPC-157", ProductType: "variable", ManageStock: tc.parent})[0]
			server.Seed(wptest.Item{Type: wptest.TypeProductVariation, Parent: product.ID, SKU: "BPC157-5", ManageStock: tc.own})
			client := newClient(t, server)

			products, err := client.ListProducts(t.Context(), wp.ProductQuery{})
			if err != nil {
				t.Fatalf("ListProducts: %v", err)
			}
			if len(products.Items) != 1 || products.Items[0].ManageStock != tc.product {
				t.Errorf("product manage stock = %+v, want %q", products.Items, tc.product)
			}
			variations, err := client.ListVariations(t.Context(), product.ID, 1, 0)
			if err != nil {
				t.Fatalf("ListVariations: %v", err)
			}
			if len(variations.Items) != 1 || variations.Items[0].ManageStock != tc.variation {
				t.Errorf("variation manage stock = %+v, want %q", variations.Items, tc.variation)
			}
		})
	}
}

func TestStockManagementDecodesEveryShapeWooCommerceSends(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw  string
		want wp.StockManagement
	}{
		{raw: `true`, want: wp.StockManaged},
		{raw: `false`, want: wp.StockNotManaged},
		{raw: `"parent"`, want: wp.StockManagedByParent},
		{raw: `"yes"`, want: wp.StockManaged},
		{raw: `"no"`, want: wp.StockNotManaged},
		{raw: `null`, want: wp.StockNotManaged},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()

			var got wp.StockManagement
			if err := json.Unmarshal([]byte(tc.raw), &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tc.raw, err)
			}
			if got != tc.want {
				t.Errorf("Unmarshal(%s) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestStoreSettingsNameTheShopPages(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SetStorePages(wptest.StorePages{Shop: 5, Cart: 6, Checkout: 7, Account: 8})

	pages, err := newClient(t, server).StoreSettings(t.Context())
	if err != nil {
		t.Fatalf("StoreSettings: %v", err)
	}
	if want := (wp.StorePages{Shop: 5, Cart: 6, Checkout: 7, Account: 8}); pages != want {
		t.Errorf("pages = %+v, want %+v", pages, want)
	}

	refused := wptest.New(t, wptest.WithoutStoreSettingsAccess())
	if _, err := newClient(t, refused).StoreSettings(t.Context()); !wp.Forbidden(err) {
		t.Errorf("StoreSettings without manage_woocommerce = %v, want Forbidden", err)
	}
}

func TestTheStoreFactsAreNeverSent(t *testing.T) {
	t.Parallel()

	store := seedStore(t)
	client := newClient(t, store.server)

	short := "<p>short</p>"
	attributes := []wp.ProductAttribute{{ID: 3, Name: "Size", Options: []string{"5 mg", "10 mg"}, Variation: true}}
	images := []int64{41}
	if _, err := client.UpdateProduct(t.Context(), store.variable.ID, wp.UpdateProduct{ShortDescription: &short, Attributes: &attributes, Images: &images}); err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	recorded, _ := store.server.LastRequest()
	var sent map[string]json.RawMessage
	if err := json.Unmarshal(recorded.Body, &sent); err != nil {
		t.Fatalf("decode the update: %v", err)
	}
	for _, field := range []string{"price", "regular_price", "sale_price", "sku", "stock_status", "stock_quantity", "manage_stock", "categories", "meta_data", "variations", "status", "slug", "name"} {
		if _, present := sent[field]; present {
			t.Errorf("the update sent %s, which the store owns", field)
		}
	}
}
