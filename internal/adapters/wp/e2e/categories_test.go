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

const (
	categoriesRoute        = "/wp-json/wp/v2/categories"
	productCategoriesRoute = "/wp-json/wc/v3/products/categories"
)

type termRefusal struct {
	Code string `json:"code"`
	Data struct {
		Status     int `json:"status"`
		TermID     int `json:"term_id"`
		ResourceID int `json:"resource_id"`
	} `json:"data"`
}

type storedTerm struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Parent int    `json:"parent"`
}

func suffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
}

func termRouteOf(taxonomy wp.Taxonomy) string {
	if taxonomy == wp.TaxonomyProductCategory {
		return productCategoriesRoute
	}
	return categoriesRoute
}

func forgetTerm(t *testing.T, c *client, taxonomy wp.Taxonomy, id int64) {
	t.Helper()

	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("%s/%d?force=true", termRouteOf(taxonomy), id), nil)
	})
}

func ensureTerm(t *testing.T, c *client, adapter *wp.Client, taxonomy wp.Taxonomy, name string, parent int64) (wp.Term, bool) {
	t.Helper()

	term, created, err := adapter.EnsureTerm(t.Context(), taxonomy, name, parent)
	if err != nil {
		t.Fatalf("ensure %s %q under %d: %v", taxonomy, name, parent, err)
	}
	if created {
		forgetTerm(t, c, taxonomy, term.ID)
	}
	return term, created
}

func createItem(t *testing.T, c *client, adapter *wp.Client, itemType wp.ItemType, in wp.CreateItem) wp.Item {
	t.Helper()

	created, err := adapter.CreateItem(t.Context(), itemType, in)
	if err != nil {
		t.Fatalf("create the %s %q: %v", itemType, in.Title, err)
	}
	route := "/wp-json/wp/v2/pages"
	if itemType == wp.TypePost {
		route = "/wp-json/wp/v2/posts"
	}
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("%s/%d?force=true", route, created.ID), nil)
	})
	return created
}

func TestEnsuringACategoryTwiceAnswersOneTerm(t *testing.T) {
	c, env := newClient(t)
	adapter := newAdapter(t, env)

	for _, taxonomy := range []wp.Taxonomy{wp.TaxonomyCategory, wp.TaxonomyProductCategory} {
		t.Run(string(taxonomy), func(t *testing.T) {
			if taxonomy == wp.TaxonomyProductCategory {
				requireWoo(t, env)
			}
			name := "Postulator Ensure " + suffix()

			first, created := ensureTerm(t, c, adapter, taxonomy, name, 0)
			if !created || first.ID == 0 || first.Name != name || first.Parent != 0 {
				t.Fatalf("the first ensure answered %+v, created %t; want a new top level term", first, created)
			}

			for _, again := range []string{name, strings.ToUpper(name), "  " + name + " "} {
				second, created := ensureTerm(t, c, adapter, taxonomy, again, 0)
				if created || second.ID != first.ID {
					t.Errorf("ensuring %q answered %+v, created %t; want the term %d again", again, second, created, first.ID)
				}
			}
		})
	}
}

func TestADuplicateCategoryIsRefusedWithTheTermItFound(t *testing.T) {
	c, env := newClient(t)
	adapter := newAdapter(t, env)

	cases := []struct {
		taxonomy wp.Taxonomy
		idOf     func(termRefusal) int
	}{
		{taxonomy: wp.TaxonomyCategory, idOf: func(refusal termRefusal) int { return refusal.Data.TermID }},
		{taxonomy: wp.TaxonomyProductCategory, idOf: func(refusal termRefusal) int { return refusal.Data.ResourceID }},
	}

	for _, tc := range cases {
		t.Run(string(tc.taxonomy), func(t *testing.T) {
			if tc.taxonomy == wp.TaxonomyProductCategory {
				requireWoo(t, env)
			}
			name := "Postulator Duplicate " + suffix()

			var created storedTerm
			c.expect(t, http.MethodPost, termRouteOf(tc.taxonomy), map[string]any{"name": name}, http.StatusCreated, &created)
			forgetTerm(t, c, tc.taxonomy, int64(created.ID))

			for _, again := range []string{name, strings.ToLower(name)} {
				var refused termRefusal
				c.expect(t, http.MethodPost, termRouteOf(tc.taxonomy), map[string]any{"name": again}, http.StatusBadRequest, &refused)
				if refused.Code != "term_exists" || tc.idOf(refused) != created.ID {
					t.Errorf("a second %q answered %+v, want term_exists naming %d", again, refused, created.ID)
				}
			}

			_, err := adapter.CreateTerm(t.Context(), tc.taxonomy, name, 0)
			if id, exists := wp.TermExists(err); !exists || id != int64(created.ID) {
				t.Errorf("CreateTerm answered %v, which names %d (%t); want term_exists naming %d", err, id, exists, created.ID)
			}
		})
	}
}

func TestACategoryNameWithAnAmpersandRoundTrips(t *testing.T) {
	c, env := newClient(t)
	adapter := newAdapter(t, env)

	name := "R&D " + suffix()
	term, created := ensureTerm(t, c, adapter, wp.TaxonomyCategory, name, 0)
	if !created || term.Name != name {
		t.Fatalf("the ensure answered %+v, created %t; want %q decoded", term, created, name)
	}

	escaped := strings.Replace(name, "&", "&amp;", 1)
	var stored storedTerm
	c.expect(t, http.MethodGet, fmt.Sprintf("%s/%d", categoriesRoute, term.ID), nil, http.StatusOK, &stored)
	if stored.Name != escaped {
		t.Errorf("WordPress returns %q as %q, want %q, the escaped form the fake returns", name, stored.Name, escaped)
	}

	for _, again := range []string{name, escaped, strings.ToLower(name)} {
		found, created := ensureTerm(t, c, adapter, wp.TaxonomyCategory, again, 0)
		if created || found.ID != term.ID || found.Name != name {
			t.Errorf("ensuring %q answered %+v, created %t; want the term %d named %q", again, found, created, term.ID, name)
		}
	}
}

func TestAChildCategoryLivesUnderItsParent(t *testing.T) {
	c, env := newClient(t)
	adapter := newAdapter(t, env)
	tag := suffix()

	parent, _ := ensureTerm(t, c, adapter, wp.TaxonomyCategory, "Postulator Parent "+tag, 0)
	child, created := ensureTerm(t, c, adapter, wp.TaxonomyCategory, "Powder "+tag, parent.ID)
	if !created || child.Parent != parent.ID {
		t.Fatalf("the child reads %+v, created %t; want it under %d", child, created, parent.ID)
	}

	again, created := ensureTerm(t, c, adapter, wp.TaxonomyCategory, "powder "+tag, parent.ID)
	if created || again.ID != child.ID {
		t.Errorf("the child again reads %+v, created %t; want %d", again, created, child.ID)
	}

	top, created := ensureTerm(t, c, adapter, wp.TaxonomyCategory, "Powder "+tag, 0)
	if !created || top.ID == child.ID || top.Parent != 0 {
		t.Errorf("the same name at the top level reads %+v, created %t; want a term of its own", top, created)
	}
	t.Logf("the child took the slug %q and its namesake at the top level %q", child.Slug, top.Slug)

	children, err := adapter.ListTerms(t.Context(), wp.TaxonomyCategory, wp.TermQuery{Parent: &parent.ID})
	if err != nil {
		t.Fatalf("ListTerms: %v", err)
	}
	if ids := termIDs(children.Items); !slices.Equal(ids, []int64{child.ID}) {
		t.Errorf("the parent's children are %v, want only %d", ids, child.ID)
	}

	var refused termRefusal
	c.expect(t, http.MethodPost, categoriesRoute, map[string]any{"name": "Postulator Orphan " + tag, "parent": 999999999},
		http.StatusBadRequest, &refused)
	if refused.Code != "rest_term_invalid" {
		t.Errorf("a missing parent answered %+v, want rest_term_invalid", refused)
	}
}

func TestAPageReadsItsCategoriesBack(t *testing.T) {
	c, env := newClient(t)
	adapter := newAdapter(t, env)

	capabilities, err := adapter.Capabilities(t.Context())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if !capabilities.Has(wp.CapabilityPageCategories) {
		t.Fatalf("the plugin lists %v, without %s", capabilities.Names, wp.CapabilityPageCategories)
	}

	tag := suffix()
	first, _ := ensureTerm(t, c, adapter, wp.TaxonomyCategory, "Postulator Page "+tag, 0)
	second, _ := ensureTerm(t, c, adapter, wp.TaxonomyCategory, "Postulator Page Child "+tag, first.ID)

	page := createItem(t, c, adapter, wp.TypePage, wp.CreateItem{
		Title: "Filed page " + tag, Content: "<p>Filed.</p>", Status: "draft", Categories: []int64{first.ID, second.ID},
	})
	if !sameIDs(page.Categories, []int64{first.ID, second.ID}) {
		t.Fatalf("the created page carries %v, want %d and %d", page.Categories, first.ID, second.ID)
	}

	read, err := adapter.GetItem(t.Context(), wp.TypePage, page.ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if !sameIDs(read.Categories, []int64{first.ID, second.ID}) {
		t.Errorf("the page reads back %v", read.Categories)
	}

	kept, err := adapter.UpdateItem(t.Context(), wp.TypePage, page.ID, wp.UpdateItem{Categories: []int64{first.ID}})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if !sameIDs(kept.Categories, []int64{first.ID}) {
		t.Errorf("after replacing the list the page carries %v, want only %d", kept.Categories, first.ID)
	}

	cleared, err := adapter.UpdateItem(t.Context(), wp.TypePage, page.ID, wp.UpdateItem{Categories: []int64{}})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if cleared.Categories == nil || len(cleared.Categories) != 0 {
		t.Errorf("after an empty list the page carries %#v, want none", cleared.Categories)
	}
}

func TestACategoryArchiveListsPagesBesidePosts(t *testing.T) {
	c, env := newClient(t)
	adapter := newAdapter(t, env)
	tag := suffix()

	category, _ := ensureTerm(t, c, adapter, wp.TaxonomyCategory, "Postulator Archive "+tag, 0)
	pageTitle := "Archived page " + tag
	postTitle := "Archived post " + tag
	createItem(t, c, adapter, wp.TypePage, wp.CreateItem{
		Title: pageTitle, Content: "<p>A page.</p>", Status: "publish", Categories: []int64{category.ID},
	})
	createItem(t, c, adapter, wp.TypePost, wp.CreateItem{
		Title: postTitle, Content: "<p>A post.</p>", Status: "publish", Categories: []int64{category.ID},
	})

	archive := c.fetchPage(t, env.baseURL+"/category/"+category.Slug+"/")
	for _, title := range []string{pageTitle, postTitle} {
		if !strings.Contains(archive, title) {
			t.Errorf("the archive of %q does not list %q", category.Slug, title)
		}
	}
}

func termIDs(terms []wp.Term) []int64 {
	ids := make([]int64, 0, len(terms))
	for _, term := range terms {
		ids = append(ids, term.ID)
	}
	return ids
}

func sameIDs(got, want []int64) bool {
	left, right := slices.Clone(got), slices.Clone(want)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}
