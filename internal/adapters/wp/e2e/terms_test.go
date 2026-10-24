//go:build e2e

package e2e_test

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"testing"
)

type termEntry struct {
	Taxonomy string `json:"taxonomy"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ID       int    `json:"id"`
	Parent   int    `json:"parent"`
}

type postTerms struct {
	Terms []termEntry `json:"terms"`
	ID    int         `json:"id"`
}

func termsPath(id int) string {
	return fmt.Sprintf("/wp-json/postulator/v1/terms/%d", id)
}

func readTerms(t *testing.T, c *client, id int) []termEntry {
	t.Helper()

	var got postTerms
	c.expect(t, http.MethodGet, termsPath(id), nil, http.StatusOK, &got)
	if got.ID != id || got.Terms == nil {
		t.Fatalf("terms of %d = %+v, want the id and a list", id, got)
	}
	ordered := slices.IsSortedFunc(got.Terms, func(a, b termEntry) int {
		return cmp.Or(cmp.Compare(a.Taxonomy, b.Taxonomy), cmp.Compare(a.ID, b.ID))
	})
	if !ordered {
		t.Errorf("terms of %d are not ordered by taxonomy then id: %+v", id, got.Terms)
	}
	return got.Terms
}

func termNamed(terms []termEntry, taxonomy, slug string) (termEntry, bool) {
	for _, held := range terms {
		if held.Taxonomy == taxonomy && held.Slug == slug {
			return held, true
		}
	}
	return termEntry{}, false
}

func TestTermsListEveryTaxonomy(t *testing.T) {
	c, env := newClient(t)
	stand := standOf(t, env)

	tag := struct {
		ID int `json:"id"`
	}{}
	c.expect(t, http.MethodPost, "/wp-json/wp/v2/tags", map[string]any{"name": uniqueSlug("roast")}, http.StatusCreated, &tag)
	t.Cleanup(func() { c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/tags/%d?force=true", tag.ID), nil) })
	var post struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodPost, "/wp-json/wp/v2/posts", map[string]any{
		"title": "Tagged", "status": "publish", "content": "<p>x</p>", "categories": []int{1}, "tags": []int{tag.ID},
	}, http.StatusCreated, &post)
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/posts/%d?force=true", post.ID), nil)
	})

	posted := readTerms(t, c, post.ID)
	if !slices.ContainsFunc(posted, func(held termEntry) bool { return held.Taxonomy == "category" && held.ID == 1 }) ||
		!slices.ContainsFunc(posted, func(held termEntry) bool { return held.Taxonomy == "post_tag" && held.ID == tag.ID }) {
		t.Errorf("the post's terms = %+v, want its category and its tag", posted)
	}

	page := createPage(t, c, pageSpec{title: "No terms", slug: uniqueSlug("bare"), content: "<p>x</p>"})
	if bare := readTerms(t, c, page); len(bare) != 0 {
		t.Errorf("a page's terms = %+v, want none", bare)
	}

	if !env.woo {
		return
	}
	product := findBySlug(t, c, "product", stand.variableProduct)
	terms := readTerms(t, c, product.ID)
	leaf, found := termNamed(terms, "product_cat", stand.categoryChain[len(stand.categoryChain)-1])
	if !found {
		t.Fatalf("the product's terms = %+v, want its category %s", terms, stand.categoryChain[len(stand.categoryChain)-1])
	}
	if hasParent := leaf.Parent != 0; hasParent != (len(stand.categoryChain) > 1) {
		t.Errorf("the category %s has the parent %d, want one only under a chain %v", leaf.Slug, leaf.Parent, stand.categoryChain)
	}
	if _, typed := termNamed(terms, "product_type", "variable"); !typed {
		t.Errorf("the product's terms = %+v, want its product_type", terms)
	}
	if _, tagged := termNamed(terms, "entity", stand.entityTerm); tagged != stand.entityTaxonomy {
		t.Errorf("the product carries the entity %q: %t, want %t", stand.entityTerm, tagged, stand.entityTaxonomy)
	}
}
