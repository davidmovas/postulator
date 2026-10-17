//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
)

func newWPClient(t *testing.T) (*client, *wp.Client, environment, standFacts) {
	t.Helper()

	c, env := newClient(t)
	site, err := wp.New(wp.Config{BaseURL: env.baseURL, Username: env.user, AppPassword: env.pass, AllowInsecure: true},
		wp.WithRateLimit(0))
	if err != nil {
		t.Fatalf("wp.New: %v", err)
	}
	return c, site, env, standOf(t, env)
}

func TestCoreReadsTypesSettingsAndEmbeddedTerms(t *testing.T) {
	c, site, env, stand := newWPClient(t)
	ctx := t.Context()

	types, err := site.PostTypes(ctx)
	if err != nil {
		t.Fatalf("PostTypes: %v", err)
	}
	described := make(map[string]wp.PostType, len(types))
	for _, postType := range types {
		described[postType.Slug] = postType
	}
	for slug, base := range map[string]string{"page": "pages", "post": "posts", "attachment": "media"} {
		if got := described[slug]; got.RestBase != base || got.RestNamespace != "wp/v2" {
			t.Errorf("%s = %+v, want the rest base %s", slug, got, base)
		}
	}
	if env.woo {
		product := described["product"]
		if product.RestBase == "" || !slices.Contains(product.Taxonomies, "product_cat") {
			t.Errorf("product = %+v, want a rest base and product_cat", product)
		}
		if got := slices.Contains(product.Taxonomies, "entity"); got != stand.entityEmbedded {
			t.Errorf("core lists entity on products: %t, want %t", got, stand.entityEmbedded)
		}
	}

	settings, err := site.SiteSettings(ctx)
	if err != nil {
		t.Fatalf("SiteSettings: %v", err)
	}
	if settings.ShowOnFront != stand.frontPage {
		t.Errorf("show_on_front = %q, want %q", settings.ShowOnFront, stand.frontPage)
	}
	if stand.frontPage == "page" {
		front, err := site.GetItem(ctx, wp.TypePage, settings.FrontPageID)
		if err != nil || front.Slug != stand.frontPageSlug {
			t.Errorf("the front page = %+v (%v), want the slug %s", front, err, stand.frontPageSlug)
		}
	}

	slug := uniqueSlug("embedded")
	var created struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodPost, "/wp-json/wp/v2/posts", map[string]any{
		"title": "Embedded terms", "slug": slug, "status": "publish", "content": "<p>x</p>", "categories": []int{1},
	}, http.StatusCreated, &created)
	t.Cleanup(func() { c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/posts/%d?force=true", created.ID), nil) })

	page, err := site.ListItems(ctx, wp.TypePost, wp.ListQuery{Slug: slug, EmbedTerms: true, Fields: []string{"slug"}})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(page.Items) != 1 || !slices.ContainsFunc(page.Items[0].Terms, func(term wp.Term) bool {
		return term.Taxonomy == "category" && term.ID == 1
	}) {
		t.Errorf("embedded terms = %+v, want the default category", page.Items)
	}

	if !env.woo {
		return
	}
	var products []struct {
		Embedded struct {
			Terms [][]struct {
				Taxonomy string `json:"taxonomy"`
			} `json:"wp:term"`
		} `json:"_embedded"`
	}
	c.expect(t, http.MethodGet, "/wp-json/wp/v2/"+described["product"].RestBase+"?slug="+stand.variableProduct+"&_embed=wp:term", nil, http.StatusOK, &products)
	if len(products) != 1 {
		t.Fatalf("core lists %d products named %s", len(products), stand.variableProduct)
	}
	var taxonomies []string
	for _, group := range products[0].Embedded.Terms {
		for _, term := range group {
			taxonomies = append(taxonomies, term.Taxonomy)
		}
	}
	if !slices.Contains(taxonomies, "product_cat") {
		t.Errorf("core embeds %v, want product_cat", taxonomies)
	}
	if got := slices.Contains(taxonomies, "entity"); got != stand.entityEmbedded {
		t.Errorf("core embeds entity: %t, want %t", got, stand.entityEmbedded)
	}
}

func TestProductsListWithVariationsAndSKU(t *testing.T) {
	_, site, env, stand := newWPClient(t)
	requireWoo(t, env)
	ctx := t.Context()

	variable := findProduct(t, site, "variable", stand.variableProduct)
	if variable.SKU != stand.variableSKU || variable.Price == "" || len(variable.Categories) == 0 {
		t.Errorf("variable = SKU %q, price %q, categories %+v", variable.SKU, variable.Price, variable.Categories)
	}
	if len(variable.VariationIDs) != len(stand.variationSKUs) {
		t.Fatalf("variation ids = %v, want %d", variable.VariationIDs, len(stand.variationSKUs))
	}

	variations, err := site.ListVariations(ctx, variable.ID, 1, 100)
	if err != nil {
		t.Fatalf("ListVariations: %v", err)
	}
	skus := make([]string, 0, len(variations.Items))
	for _, variation := range variations.Items {
		skus = append(skus, variation.SKU)
		if variation.ParentID != variable.ID || variation.RegularPrice == "" || len(variation.Attributes) == 0 || variation.StockStatus == "" {
			t.Errorf("variation = %+v", variation)
		}
	}
	slices.Sort(skus)
	want := slices.Sorted(slices.Values(stand.variationSKUs))
	if !slices.Equal(skus, want) {
		t.Errorf("variation SKUs = %v, want %v", skus, want)
	}

	simple := findProduct(t, site, "simple", stand.simpleProduct)
	if simple.Name != stand.simpleProductTitle || len(simple.VariationIDs) != 0 {
		t.Errorf("simple = %+v", simple)
	}
}

func findProduct(t *testing.T, site *wp.Client, productType, slug string) wp.Product {
	t.Helper()

	for number := 1; ; number++ {
		page, err := site.ListProducts(t.Context(), wp.ProductQuery{Type: productType, Page: number, PerPage: 100})
		if err != nil {
			t.Fatalf("ListProducts: %v", err)
		}
		for _, product := range page.Items {
			if product.Slug == slug {
				return product
			}
		}
		if !page.HasMore {
			t.Fatalf("the store lists no %s product named %s", productType, slug)
		}
	}
}

func TestStoreSettingsNameTheShopPages(t *testing.T) {
	_, site, env, _ := newWPClient(t)
	requireWoo(t, env)
	ctx := t.Context()

	pages, err := site.StoreSettings(ctx)
	if err != nil {
		t.Fatalf("StoreSettings: %v", err)
	}
	if pages.Shop == 0 || pages.Cart == 0 || pages.Checkout == 0 || pages.Account == 0 {
		t.Fatalf("pages = %+v, want the shop, cart, checkout and account", pages)
	}
	for id, slug := range map[int64]string{pages.Shop: "shop", pages.Cart: "cart", pages.Checkout: "checkout", pages.Account: "my-account"} {
		page, err := site.GetItem(ctx, wp.TypePage, id)
		if err != nil || page.Slug != slug {
			t.Errorf("page %d = %+v (%v), want the slug %s", id, page, err, slug)
		}
	}
}
