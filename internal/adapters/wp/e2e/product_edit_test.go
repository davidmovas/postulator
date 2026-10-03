//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
)

type storeAttribute struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Position  int      `json:"position"`
	Visible   bool     `json:"visible"`
	Variation bool     `json:"variation"`
	Options   []string `json:"options"`
}

type storeProduct struct {
	ID               int              `json:"id"`
	Name             string           `json:"name"`
	Slug             string           `json:"slug"`
	Permalink        string           `json:"permalink"`
	Type             string           `json:"type"`
	Status           string           `json:"status"`
	Description      string           `json:"description"`
	ShortDescription string           `json:"short_description"`
	RegularPrice     string           `json:"regular_price"`
	SKU              string           `json:"sku"`
	Attributes       []storeAttribute `json:"attributes"`
	Categories       []struct {
		ID int `json:"id"`
	} `json:"categories"`
}

func (p storeProduct) categoryIDs() []int64 {
	ids := make([]int64, 0, len(p.Categories))
	for _, category := range p.Categories {
		ids = append(ids, int64(category.ID))
	}
	return ids
}

func storeProductPath(id int) string {
	return fmt.Sprintf("/wp-json/wc/v3/products/%d", id)
}

func readStoreProduct(t *testing.T, c *client, id int) storeProduct {
	t.Helper()

	var product storeProduct
	c.expect(t, http.MethodGet, storeProductPath(id)+"?context=edit", nil, http.StatusOK, &product)
	return product
}

func createStoreProduct(t *testing.T, c *client, body map[string]any) storeProduct {
	t.Helper()

	var created storeProduct
	c.expect(t, http.MethodPost, "/wp-json/wc/v3/products", body, http.StatusCreated, &created)
	if created.ID == 0 {
		t.Fatalf("the created product has id 0")
	}
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, storeProductPath(created.ID)+"?force=true", nil)
	})
	return created
}

func createGlobalAttribute(t *testing.T, c *client, name, slug, term string) int {
	t.Helper()

	var attribute struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodPost, "/wp-json/wc/v3/products/attributes",
		map[string]any{"name": name, "slug": slug}, http.StatusCreated, &attribute)
	if attribute.ID == 0 {
		t.Fatalf("the created attribute has id 0")
	}
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wc/v3/products/attributes/%d?force=true", attribute.ID), nil)
	})
	c.expect(t, http.MethodPost, fmt.Sprintf("/wp-json/wc/v3/products/attributes/%d/terms", attribute.ID),
		map[string]any{"name": term}, http.StatusCreated, nil)
	return attribute.ID
}

func attributeNamed(attributes []storeAttribute, name string) (storeAttribute, bool) {
	for _, attribute := range attributes {
		if strings.EqualFold(attribute.Name, name) {
			return attribute, true
		}
	}
	return storeAttribute{}, false
}

func requireUntouched(t *testing.T, step string, before, after storeProduct) {
	t.Helper()

	if after.Name != before.Name {
		t.Errorf("after %s the name is %q, want %q", step, after.Name, before.Name)
	}
	if after.Status != before.Status {
		t.Errorf("after %s the status is %q, want %q", step, after.Status, before.Status)
	}
	if after.Slug != before.Slug {
		t.Errorf("after %s the slug is %q, want %q", step, after.Slug, before.Slug)
	}
	if after.RegularPrice != before.RegularPrice {
		t.Errorf("after %s the price is %q, want %q", step, after.RegularPrice, before.RegularPrice)
	}
	if after.SKU != before.SKU {
		t.Errorf("after %s the SKU is %q, want %q", step, after.SKU, before.SKU)
	}
}

func TestAnApplicationPasswordEditsAProductThroughTheStoreAPI(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	var listed []struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodGet, "/wp-json/wc/v3/products?per_page=1&_fields=id", nil, http.StatusOK, &listed)

	var me struct {
		Capabilities map[string]bool `json:"capabilities"`
	}
	c.expect(t, http.MethodGet, "/wp-json/wp/v2/users/me?context=edit", nil, http.StatusOK, &me)
	for _, capability := range []string{"edit_products", "edit_published_products", "edit_others_products"} {
		if !me.Capabilities[capability] {
			t.Errorf("the application password's user lacks %s", capability)
		}
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	colorID := createGlobalAttribute(t, c, "Color "+suffix, "pc"+suffix, "Blue")
	var clientCategory storedTerm
	c.expect(t, http.MethodPost, productCategoriesRoute, map[string]any{"name": "Client Shelf " + suffix}, http.StatusCreated, &clientCategory)
	forgetTerm(t, c, wp.TaxonomyProductCategory, int64(clientCategory.ID))
	created := createStoreProduct(t, c, map[string]any{
		"name":              "Spike & Co " + suffix,
		"slug":              "postulator-spike-" + suffix,
		"type":              "simple",
		"status":            "publish",
		"regular_price":     "19.90",
		"sku":               "postulator-spike-" + suffix,
		"description":       "<p>Original description.</p>",
		"short_description": "<p>Original short.</p>",
		"attributes": []map[string]any{
			{"id": colorID, "options": []string{"Blue"}, "visible": true, "variation": false, "position": 0},
			{"name": "Origin", "options": []string{"Client"}, "visible": true, "variation": false, "position": 1},
		},
		"categories": []map[string]any{{"id": clientCategory.ID}},
	})
	before := readStoreProduct(t, c, created.ID)
	if before.Type != "simple" {
		t.Fatalf("the seeded product is a %q product, want simple", before.Type)
	}

	t.Run("the short description changes and nothing else does", func(t *testing.T) {
		short := `<p>Generated <strong>short</strong> text with a "quote" and an &amp; sign.</p>`
		c.expect(t, http.MethodPut, storeProductPath(created.ID),
			map[string]any{"short_description": short}, http.StatusOK, nil)

		after := readStoreProduct(t, c, created.ID)
		requireUntouched(t, "the short description write", before, after)
		if after.Description != before.Description {
			t.Errorf("the description became %q, want %q", after.Description, before.Description)
		}
		if !strings.Contains(after.ShortDescription, "<strong>short</strong>") {
			t.Errorf("the short description is stored as %q", after.ShortDescription)
		}
		t.Logf("sent short description %q, stored %q", short, after.ShortDescription)
	})

	t.Run("an attribute is added while the shop's own come back untouched", func(t *testing.T) {
		current := readStoreProduct(t, c, created.ID)
		attributes := make([]storeAttribute, 0, len(current.Attributes)+1)
		attributes = append(attributes, current.Attributes...)
		attributes = append(attributes, storeAttribute{
			Name: "Form", Options: []string{"Liquid"}, Visible: true, Position: len(current.Attributes),
		})
		c.expect(t, http.MethodPut, storeProductPath(created.ID),
			map[string]any{"attributes": attributes}, http.StatusOK, nil)

		after := readStoreProduct(t, c, created.ID)
		requireUntouched(t, "the attribute write", before, after)
		if len(after.Attributes) != 3 {
			t.Fatalf("the product carries %d attributes, want 3: %+v", len(after.Attributes), after.Attributes)
		}
		color, held := attributeNamed(after.Attributes, "Color "+suffix)
		if !held || color.ID != colorID || !slices.Equal(color.Options, []string{"Blue"}) {
			t.Errorf("the global attribute reads %+v, want id %d with Blue", color, colorID)
		}
		origin, held := attributeNamed(after.Attributes, "Origin")
		if !held || origin.ID != 0 || !slices.Equal(origin.Options, []string{"Client"}) {
			t.Errorf("the shop's own attribute reads %+v, want Origin with Client", origin)
		}
		form, held := attributeNamed(after.Attributes, "Form")
		if !held || form.ID != 0 || !slices.Equal(form.Options, []string{"Liquid"}) || !form.Visible || form.Variation {
			t.Errorf("the added attribute reads %+v, want a visible Form with Liquid", form)
		}
	})

	t.Run("the plugin writes the description byte for byte", func(t *testing.T) {
		var raw rawContent
		c.expect(t, http.MethodGet, rawPath(created.ID), nil, http.StatusOK, &raw)
		if raw.Type != "product" {
			t.Errorf("the raw route reads a %q, want product", raw.Type)
		}
		if raw.Content != before.Description {
			t.Errorf("the raw route reads %q, the store API %q", raw.Content, before.Description)
		}

		body := `<h2>Overview</h2><p>Rewritten with a <a href="/koffein/">link</a>, a "quote" and an &amp; sign.</p>` +
			`<img src="/wp-content/uploads/powder.png" alt="Powder" width="800" height="600" /><br/>`
		var written struct {
			ContentHash string `json:"contentHash"`
		}
		c.expect(t, http.MethodPut, rawPath(created.ID),
			map[string]string{"content": body, "expectedHash": raw.ContentHash}, http.StatusOK, &written)
		if written.ContentHash != hashOf(body) {
			t.Fatalf("the stored description hashes to %q, want %q", written.ContentHash, hashOf(body))
		}

		status, conflict := c.request(t, http.MethodPut, rawPath(created.ID),
			map[string]string{"content": "<p>late</p>", "expectedHash": raw.ContentHash})
		if status != http.StatusConflict {
			t.Errorf("a write over a stale hash answers %d, want 409, body %s", status, conflict)
		}

		after := readStoreProduct(t, c, created.ID)
		requireUntouched(t, "the raw write", before, after)
		if after.Description != body {
			t.Errorf("the store API reads the description as %q, want %q", after.Description, body)
		}

		page := c.fetchPage(t, after.Permalink)
		if !strings.Contains(page, "Rewritten with a") {
			t.Errorf("the product page at %s does not show the description", after.Permalink)
		}
	})

	t.Run("our category joins the client's and leaves alone on a revert", func(t *testing.T) {
		adapter := newAdapter(t, env)
		client := int64(clientCategory.ID)
		ours, made := ensureTerm(t, c, adapter, wp.TaxonomyProductCategory, "Postulator Shelf "+suffix, 0)
		if !made {
			t.Fatalf("the category %+v already existed", ours)
		}

		product, err := adapter.GetProduct(t.Context(), int64(created.ID))
		if err != nil {
			t.Fatalf("GetProduct: %v", err)
		}
		if !slices.Equal(product.Categories, []int64{client}) {
			t.Fatalf("the product starts with %v, want the client's %d", product.Categories, client)
		}

		union := append(slices.Clone(product.Categories), ours.ID)
		joined, err := adapter.UpdateProduct(t.Context(), int64(created.ID), wp.UpdateProduct{Categories: &union})
		if err != nil {
			t.Fatalf("UpdateProduct: %v", err)
		}
		if !sameIDs(joined.Categories, []int64{client, ours.ID}) {
			t.Errorf("after the union the product carries %v, want %d and %d", joined.Categories, client, ours.ID)
		}
		after := readStoreProduct(t, c, created.ID)
		requireUntouched(t, "the category write", before, after)
		if !sameIDs(after.categoryIDs(), []int64{client, ours.ID}) {
			t.Errorf("the store reads %v after the union", after.categoryIDs())
		}

		previous := slices.DeleteFunc(slices.Clone(joined.Categories), func(id int64) bool { return id == ours.ID })
		reverted, err := adapter.UpdateProduct(t.Context(), int64(created.ID), wp.UpdateProduct{Categories: &previous})
		if err != nil {
			t.Fatalf("UpdateProduct: %v", err)
		}
		if !slices.Equal(reverted.Categories, []int64{client}) {
			t.Errorf("after the revert the product carries %v, want only the client's %d", reverted.Categories, client)
		}
		requireUntouched(t, "the category revert", before, readStoreProduct(t, c, created.ID))
	})

	t.Run("the plugin writes the product's SEO meta", func(t *testing.T) {
		written := map[string]string{
			"title":       "Spike title " + suffix,
			"description": "Spike description " + suffix,
		}
		writeSEO(t, c, created.ID, written)

		got := readSEO(t, c, created.ID)
		if got.Title != written["title"] || got.Description != written["description"] {
			t.Errorf("the product's SEO meta reads %+v, want %v", got, written)
		}
		requireUntouched(t, "the SEO write", before, readStoreProduct(t, c, created.ID))
	})
}
