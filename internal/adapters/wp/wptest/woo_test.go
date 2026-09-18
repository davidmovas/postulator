package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestProductsUseTheWooCommerceFieldNames(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{
		Type:    wptest.TypeProduct,
		Title:   "Koffein Powder",
		Content: "<p>long</p>",
		Excerpt: "short",
		Status:  "publish",
	})

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/"+itoa(seeded[0].ID), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var product struct {
		ID               int64  `json:"id"`
		Name             string `json:"name"`
		Slug             string `json:"slug"`
		Permalink        string `json:"permalink"`
		Status           string `json:"status"`
		Description      string `json:"description"`
		ShortDescription string `json:"short_description"`
		DateModifiedGMT  string `json:"date_modified_gmt"`
	}
	decode(t, payload, &product)

	if product.Name != "Koffein Powder" || product.Description != "<p>long</p>" || product.ShortDescription != "short" {
		t.Errorf("product = %+v", product)
	}
	if product.Permalink == "" || product.DateModifiedGMT == "" || product.Slug != "koffein-powder" {
		t.Errorf("product = %+v", product)
	}
}

func TestProductsPageWithoutTheCoreEndOfListError(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "One"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Two"},
	)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products?page=9&per_page=1", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; WooCommerce answers an empty page", response.StatusCode)
	}
	if response.Header.Get("X-WP-TotalPages") != "2" {
		t.Errorf("X-WP-TotalPages = %q, want 2", response.Header.Get("X-WP-TotalPages"))
	}

	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 0 {
		t.Errorf("got %d items, want none", len(items))
	}
}

func TestAProductStatusFilterTakesOneValue(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "Live", Status: "publish"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Draft", Status: "draft"},
	)

	response, _ := call(t, server, http.MethodGet, "/wp-json/wc/v3/products?status=publish,draft", nil, true)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a comma list", response.StatusCode)
	}

	_, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products?status=draft", nil, true)
	var drafts []map[string]any
	decode(t, payload, &drafts)
	if len(drafts) != 1 {
		t.Errorf("got %d drafts, want 1", len(drafts))
	}
}

func TestUpdatingAProductWritesTheWooCommerceFields(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	product := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder"})[0]

	body := []byte(`{"description":"<p>new</p>","short_description":"brief","status":"draft","categories":[{"id":` + itoa(category.ID) + `}]}`)
	_, payload := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(product.ID), body, true)

	var updated struct {
		Description      string `json:"description"`
		ShortDescription string `json:"short_description"`
		Status           string `json:"status"`
		Categories       []struct {
			ID   int64  `json:"id"`
			Slug string `json:"slug"`
		} `json:"categories"`
	}
	decode(t, payload, &updated)

	if updated.Description != "<p>new</p>" || updated.ShortDescription != "brief" || updated.Status != "draft" {
		t.Errorf("updated = %+v", updated)
	}
	if len(updated.Categories) != 1 || updated.Categories[0].ID != category.ID || updated.Categories[0].Slug != "koffein" {
		t.Errorf("categories = %+v", updated.Categories)
	}
}

func TestProductCategoriesCarryNoModificationDate(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/categories", nil, true)

	var categories []map[string]any
	decode(t, payload, &categories)
	if len(categories) != 1 {
		t.Fatalf("got %d categories, want 1", len(categories))
	}
	if _, present := categories[0]["date_modified_gmt"]; present {
		t.Error("a WooCommerce product category is a term and has no modification date")
	}
	if categories[0]["name"] != "Koffein" || categories[0]["description"] != "the hub" {
		t.Errorf("category = %v", categories[0])
	}
}
