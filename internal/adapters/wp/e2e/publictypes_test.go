//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func TestContentCarriesExcerptTermsAndEveryPublicType(t *testing.T) {
	c, env := newClient(t)
	stand := standOf(t, env)

	described := readManifest(t, c)
	for _, want := range []string{"page", "post"} {
		if !slices.Contains(described.PostTypes, want) {
			t.Errorf("postTypes = %v, want %s", described.PostTypes, want)
		}
	}
	if slices.Contains(described.PostTypes, "attachment") || slices.Contains(described.PostTypes, "wp_block") {
		t.Errorf("postTypes = %v, want content types only", described.PostTypes)
	}
	for _, want := range []string{"category", "post_tag"} {
		if !slices.Contains(described.Taxonomies, want) {
			t.Errorf("taxonomies = %v, want %s", described.Taxonomies, want)
		}
	}
	if env.woo && (!slices.Contains(described.PostTypes, "product") || !slices.Contains(described.Taxonomies, "product_cat")) {
		t.Errorf("the store's types are missing: %v and %v", described.PostTypes, described.Taxonomies)
	}
	if got := slices.Contains(described.Taxonomies, "entity"); got != stand.entityTaxonomy {
		t.Errorf("taxonomies list entity: %t, want %t", got, stand.entityTaxonomy)
	}

	var post struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodPost, "/wp-json/wp/v2/posts", map[string]any{
		"title": "Summarized", "status": "publish", "content": "<p>Body.</p>", "excerpt": "Short summary.", "categories": []int{1},
	}, http.StatusCreated, &post)
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/posts/%d?force=true", post.ID), nil)
	})
	listed := findItem(t, c, "post", post.ID)
	if listed.Excerpt != "Short summary." || !slices.ContainsFunc(listed.Terms, func(held termEntry) bool {
		return held.Taxonomy == "category" && held.ID == 1
	}) {
		t.Errorf("the post is listed as %+v, want its excerpt and its category", listed)
	}

	page := findItem(t, c, "page", createPage(t, c, pageSpec{title: "Bare page", slug: uniqueSlug("bare"), content: "<p>x</p>"}))
	if page.Terms == nil || len(page.Terms) != 0 || page.Excerpt != "" {
		t.Errorf("a bare page is listed with excerpt %q and terms %v, want none and an empty list", page.Excerpt, page.Terms)
	}

	walkContent(t, c, "", 500, func(item contentItem) bool {
		if item.Type == "entity" {
			t.Errorf("the default listing holds the %s term %s, want only the four types it always listed", item.Type, item.Slug)
			return false
		}
		return true
	})

	if stand.entityTaxonomy {
		entity := findBySlug(t, c, "entity", stand.entityTerm)
		if entity.Type != "entity" || entity.Terms == nil || len(entity.Terms) != 0 || entity.Path == "" {
			t.Errorf("the entity term is listed as %+v", entity)
		}
	}
	if env.woo {
		product := findBySlug(t, c, "product", stand.variableProduct)
		leaf := stand.categoryChain[len(stand.categoryChain)-1]
		if _, found := termNamed(product.Terms, "product_cat", leaf); !found {
			t.Errorf("the product is listed with %+v, want its category %s", product.Terms, leaf)
		}
		if _, tagged := termNamed(product.Terms, "entity", stand.entityTerm); tagged != stand.entityTaxonomy {
			t.Errorf("the product is listed with the entity %q: %t, want %t", stand.entityTerm, tagged, stand.entityTaxonomy)
		}
	}

	var refused struct {
		Code string `json:"code"`
	}
	c.expect(t, http.MethodGet, "/wp-json/postulator/v1/content?types=wp_block", nil, http.StatusBadRequest, &refused)
	if refused.Code != "invalid_type" {
		t.Errorf("types=wp_block answers %q, want invalid_type", refused.Code)
	}
}
