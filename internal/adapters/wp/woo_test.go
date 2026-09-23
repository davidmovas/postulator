package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestProductsAreListedAndRead(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	seeded := server.Seed(wptest.Item{
		Type:       wptest.TypeProduct,
		Title:      "Powder",
		Content:    "<p>long</p>",
		Excerpt:    "short",
		Status:     "publish",
		Categories: []int64{category.ID},
	})
	client := newClient(t, server)

	listed, err := client.ListProducts(t.Context(), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if listed.Total != 1 || len(listed.Items) != 1 {
		t.Fatalf("listed = %+v", listed)
	}

	product, err := client.GetProduct(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if product.Name != "Powder" || product.Description != "<p>long</p>" || product.ShortDescription != "short" {
		t.Errorf("product = %+v", product)
	}
	if product.Permalink == "" || product.Modified.IsZero() {
		t.Errorf("product = %+v", product)
	}
	if len(product.Categories) != 1 || product.Categories[0].Slug != "koffein" {
		t.Errorf("categories = %+v", product.Categories)
	}
}

func TestUpdateProductWritesTheWooCommerceFieldsOnly(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Content: "<p>old</p>", Status: "publish"})

	updated, err := newClient(t, server).UpdateProduct(t.Context(), seeded[0].ID, wp.UpdateProduct{
		Description:      pointerTo("<p>new</p>"),
		ShortDescription: pointerTo("brief"),
		Status:           pointerTo("draft"),
		Categories:       []int64{category.ID},
	})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	if updated.Description != "<p>new</p>" || updated.ShortDescription != "brief" || updated.Status != "draft" {
		t.Errorf("updated = %+v", updated)
	}
	if updated.Name != "Powder" {
		t.Errorf("name = %q; an absent field must not be overwritten", updated.Name)
	}
	if len(updated.Categories) != 1 || updated.Categories[0].ID != category.ID {
		t.Errorf("categories = %+v", updated.Categories)
	}

	recorded, _ := server.LastRequest()
	if len(recorded.Body) == 0 {
		t.Fatal("the update sent no body")
	}
	if _, err = newClient(t, server).UpdateProduct(t.Context(), seeded[0].ID, wp.UpdateProduct{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q for an empty update", errors.CodeOf(err), errors.Invalid)
	}
}

func TestProductCategoriesAreListed(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})[0]
	server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Powder", Parent: parent.ID})

	listed, err := newClient(t, server).ListProductCategories(t.Context(), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListProductCategories: %v", err)
	}
	if listed.Total != 2 || len(listed.Items) != 2 {
		t.Fatalf("listed = %+v", listed)
	}
	if listed.Items[0].Name != "Koffein" || listed.Items[0].Description != "the hub" {
		t.Errorf("first = %+v", listed.Items[0])
	}
	if listed.Items[1].Parent != parent.ID {
		t.Errorf("second = %+v", listed.Items[1])
	}
}
