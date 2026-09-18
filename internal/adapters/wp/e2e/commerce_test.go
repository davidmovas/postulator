//go:build e2e

package e2e_test

import (
	"net/url"
	"testing"
	"time"
)

func requireWoo(t *testing.T, env environment) {
	t.Helper()

	if !env.woo {
		t.Fatalf("this suite needs WooCommerce; run `task e2e:reset` with E2E_WOO=1")
	}
}

func TestProductCategoryIsListed(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	item := findBySlug(t, c, "product_cat", "postulator-koffein")

	if item.Type != "product_cat" {
		t.Errorf("type = %q, want product_cat", item.Type)
	}
	if item.Title != "Koffein" {
		t.Errorf("title = %q, want Koffein", item.Title)
	}
	if item.Path != "/product-category/postulator-koffein/" {
		t.Errorf("path = %q, want /product-category/postulator-koffein/", item.Path)
	}
	if item.Parent != 0 {
		t.Errorf("parent = %d, want 0", item.Parent)
	}
	if item.Status != "publish" {
		t.Errorf("status = %q, want publish", item.Status)
	}
	if item.H1 != "" {
		t.Errorf("h1 = %q, want empty for a term", item.H1)
	}
	if len(item.Links) != 0 {
		t.Errorf("links = %v, want none for a term", item.Links)
	}
	if item.ContentHash != hashOf("Caffeine products.") {
		t.Errorf("contentHash = %q, want the hash of the term description", item.ContentHash)
	}
	if _, err := time.Parse(time.RFC3339, item.Modified); err != nil {
		t.Errorf("modified = %q is not RFC3339: %v", item.Modified, err)
	}
}

func TestProductIsListedWithItsLinks(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	item := findBySlug(t, c, "product", "postulator-powder")

	if item.Type != "product" {
		t.Errorf("type = %q, want product", item.Type)
	}
	if item.Title != "Powder" {
		t.Errorf("title = %q, want Powder", item.Title)
	}
	want := link{Href: "/product-category/postulator-koffein/", Anchor: "Koffein"}
	if len(item.Links) != 1 || item.Links[0] != want {
		t.Fatalf("links = %v, want exactly [%v]", item.Links, want)
	}
}

func TestMixedTypesPageThroughBothPhases(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	seen := make(map[string]bool)
	cursor := ""
	for range 200 {
		query := "types=product,product_cat&limit=1"
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		page := listContent(t, c, query)
		for _, item := range page.Items {
			seen[item.Type] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	if !seen["product"] || !seen["product_cat"] {
		t.Fatalf("walking types=product,product_cat saw %v, want both phases", seen)
	}
}

func TestTermsOnlyListingStartsInTheTermPhase(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	page := listContent(t, c, "types=product_cat&limit=500")
	if len(page.Items) == 0 {
		t.Fatalf("types=product_cat returned no items")
	}
	for _, item := range page.Items {
		if item.Type != "product_cat" {
			t.Fatalf("types=product_cat returned a %q item", item.Type)
		}
	}
	if page.NextCursor != nil {
		t.Errorf("nextCursor = %q, want null", *page.NextCursor)
	}
}
