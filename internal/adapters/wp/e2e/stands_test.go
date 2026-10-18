//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

type standFacts struct {
	name                string
	seoPlugin           string
	frontPage           string
	frontPageSlug       string
	hubEmptyRaw         string
	hubShortcode        string
	hubWithText         string
	termCrumb           string
	redirectedCrumb     string
	redirectTarget      string
	simpleProduct       string
	simpleProductTitle  string
	simpleProductLink   link
	variableProduct     string
	variableSKU         string
	variationSKUs       []string
	category            string
	categoryTitle       string
	categoryDescription string
	categoryChain       []string
	entityTerm          string
	breadcrumbInContent bool
	entityTaxonomy      bool
	entityEmbedded      bool
	nestedPages         bool
}

func standOf(t *testing.T, env environment) standFacts {
	t.Helper()

	switch env.stand {
	case "vanilla":
		return standFacts{
			name:                "vanilla",
			seoPlugin:           env.seo,
			frontPage:           "posts",
			simpleProduct:       "powder",
			simpleProductTitle:  "Powder",
			simpleProductLink:   link{Href: "/product-category/koffein/", Anchor: "Koffein"},
			variableProduct:     "coffee-beans",
			variableSKU:         "BEANS",
			variationSKUs:       []string{"BEANS-250G", "BEANS-1KG"},
			category:            "koffein",
			categoryTitle:       "Koffein",
			categoryDescription: "Caffeine products.",
			categoryChain:       []string{"koffein"},
			nestedPages:         true,
		}
	case "replica":
		if env.seo != "rankmath" {
			t.Fatalf("the replica stand runs Rank Math and its env file says E2E_SEO=%s", env.seo)
		}
		return standFacts{
			name:                "replica",
			seoPlugin:           "rankmath",
			frontPage:           "page",
			frontPageSlug:       "home",
			hubEmptyRaw:         "peptides",
			hubShortcode:        "bpc-157",
			hubWithText:         "tb-500",
			termCrumb:           "/product-category/peptides/healing-peptides/",
			redirectedCrumb:     "/category/healing-peptides/",
			redirectTarget:      "/healing-peptides/",
			simpleProduct:       "bacteriostatic-water",
			simpleProductTitle:  "Bacteriostatic Water",
			simpleProductLink:   link{Href: "/peptides/", Anchor: "Peptides"},
			variableProduct:     "bpc-157-peptide",
			variableSKU:         "BPC157",
			variationSKUs:       []string{"BPC157-5MG", "BPC157-10MG"},
			category:            "peptides",
			categoryTitle:       "Peptides",
			categoryDescription: "Research peptides for the laboratory.",
			categoryChain:       []string{"peptides", "healing-peptides"},
			entityTerm:          "bpc-157",
			breadcrumbInContent: true,
			entityTaxonomy:      true,
		}
	}
	t.Fatalf("E2E_STAND is %q, and the suites know the vanilla and the replica stands", env.stand)
	return standFacts{}
}

type corePage struct {
	ID      int    `json:"id"`
	Slug    string `json:"slug"`
	Parent  int    `json:"parent"`
	Link    string `json:"link"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
}

type storeCategory struct {
	ID     int    `json:"id"`
	Slug   string `json:"slug"`
	Parent int    `json:"parent"`
}

var loneShortcode = regexp.MustCompile(`^\[[a-z_]+( [^\]]*)?\]$`)

func pageBySlug(t *testing.T, c *client, slug string) (corePage, bool) {
	t.Helper()

	var listed []corePage
	c.expect(t, http.MethodGet, "/wp-json/wp/v2/pages?context=edit&status=publish&slug="+url.QueryEscape(slug),
		nil, http.StatusOK, &listed)
	if len(listed) == 0 {
		return corePage{}, false
	}
	return listed[0], true
}

func requirePage(t *testing.T, c *client, slug string) corePage {
	t.Helper()

	page, found := pageBySlug(t, c, slug)
	if !found {
		t.Fatalf("the stand holds no published page %q", slug)
	}
	return page
}

func TestTheStandHoldsItsFixture(t *testing.T) {
	env := loadEnvironment(t)
	facts := standOf(t, env)
	c := &client{base: env.baseURL, user: env.user, pass: env.pass, http: &http.Client{Timeout: 60 * time.Second}}

	t.Run("front page", func(t *testing.T) {
		var settings struct {
			ShowOnFront string `json:"show_on_front"`
			PageOnFront int    `json:"page_on_front"`
		}
		c.expect(t, http.MethodGet, "/wp-json/wp/v2/settings", nil, http.StatusOK, &settings)
		if settings.ShowOnFront != facts.frontPage {
			t.Fatalf("show_on_front = %q, want %q", settings.ShowOnFront, facts.frontPage)
		}
		if facts.frontPage == "page" && settings.PageOnFront != requirePage(t, c, facts.frontPageSlug).ID {
			t.Errorf("page_on_front = %d, want the page %q", settings.PageOnFront, facts.frontPageSlug)
		}
	})

	t.Run("pages", func(t *testing.T) {
		if facts.nestedPages {
			guide, setup := requirePage(t, c, "guide"), requirePage(t, c, "setup")
			if setup.Parent != guide.ID || !strings.HasSuffix(setup.Link, "/guide/setup/") {
				t.Errorf("setup hangs off %d at %s, want the guide %d at /guide/setup/", setup.Parent, setup.Link, guide.ID)
			}
		}
		if facts.hubEmptyRaw == "" {
			return
		}
		for _, slug := range []string{facts.frontPageSlug, facts.hubEmptyRaw, facts.hubShortcode, facts.hubWithText} {
			if page := requirePage(t, c, slug); page.Parent != 0 {
				t.Errorf("%s hangs off %d, want the replica's pages flat", slug, page.Parent)
			}
		}
		if raw := requirePage(t, c, facts.hubEmptyRaw).Content.Raw; raw != "" {
			t.Errorf("the hub %s holds %q, want an empty raw", facts.hubEmptyRaw, raw)
		}
		if raw := requirePage(t, c, facts.hubShortcode).Content.Raw; !loneShortcode.MatchString(raw) {
			t.Errorf("the hub %s holds %q, want one shortcode alone", facts.hubShortcode, raw)
		}
		if raw := requirePage(t, c, facts.hubWithText).Content.Raw; strings.TrimSpace(raw) == "" || strings.Contains(raw, "[") {
			t.Errorf("the hub %s holds %q, want paragraphs of text", facts.hubWithText, raw)
		}
		if _, found := pageBySlug(t, c, strings.Trim(facts.redirectTarget, "/")); !found {
			t.Errorf("no page answers %s, the hub a redirected crumb must land on", facts.redirectTarget)
		}
	})

	t.Run("render", func(t *testing.T) {
		probe := facts.hubWithText
		if probe == "" {
			probe = "guide"
		}
		body := c.fetchPage(t, env.baseURL+"/"+probe+"/")
		nav := strings.Index(body, `aria-label="Breadcrumb"`)
		if !facts.breadcrumbInContent {
			if nav >= 0 {
				t.Errorf("the %s page carries a breadcrumb, and no page of this stand does", probe)
			}
			return
		}
		if nav < 0 {
			t.Fatalf("the hub %s carries no breadcrumb", probe)
		}
		if !strings.Contains(body, `href="`+env.baseURL+facts.redirectedCrumb+`"`) {
			t.Errorf("the breadcrumb of %s names no crumb at %s", probe, facts.redirectedCrumb)
		}
		unfollowed := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := unfollowed.Get(env.baseURL + facts.redirectedCrumb)
		if err != nil {
			t.Fatalf("GET %s: %v", facts.redirectedCrumb, err)
		}
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatalf("close %s: %v", facts.redirectedCrumb, closeErr)
		}
		if resp.StatusCode != http.StatusMovedPermanently || !strings.HasSuffix(resp.Header.Get("Location"), facts.redirectTarget) {
			t.Errorf("%s answers %d to %q, want a 301 to the hub %s", facts.redirectedCrumb, resp.StatusCode, resp.Header.Get("Location"), facts.redirectTarget)
		}
		if !strings.Contains(body[nav:], "<h2>") {
			t.Errorf("the hub %s carries no listing heading after its breadcrumb", probe)
		}
		for _, schema := range []string{`"@type":"BreadcrumbList"`, `"@type":"ItemList"`, `"@type":"CollectionPage"`} {
			if !strings.Contains(body, schema) {
				t.Errorf("the hub %s prints no %s", probe, schema)
			}
		}
	})

	t.Run("store", func(t *testing.T) {
		requireWoo(t, env)

		var simple []storeProduct
		c.expect(t, http.MethodGet, "/wp-json/wc/v3/products?slug="+facts.simpleProduct, nil, http.StatusOK, &simple)
		if len(simple) != 1 || simple[0].Type != "simple" || simple[0].Name != facts.simpleProductTitle {
			t.Errorf("the simple product %s reads %+v", facts.simpleProduct, simple)
		}

		var variable []struct {
			storeProduct
			Variations []int           `json:"variations"`
			Categories []storeCategory `json:"categories"`
		}
		c.expect(t, http.MethodGet, "/wp-json/wc/v3/products?sku="+facts.variableSKU, nil, http.StatusOK, &variable)
		if len(variable) != 1 || variable[0].Type != "variable" || variable[0].Slug != facts.variableProduct {
			t.Fatalf("the variable product %s reads %+v", facts.variableSKU, variable)
		}
		var variations []struct {
			SKU string `json:"sku"`
		}
		c.expect(t, http.MethodGet, fmt.Sprintf("/wp-json/wc/v3/products/%d/variations?per_page=100", variable[0].ID),
			nil, http.StatusOK, &variations)
		skus := make([]string, 0, len(variations))
		for _, variation := range variations {
			skus = append(skus, variation.SKU)
		}
		if !sameMembers(skus, facts.variationSKUs) {
			t.Errorf("the variations carry the SKUs %v, want %v", skus, facts.variationSKUs)
		}

		parent := 0
		for _, slug := range facts.categoryChain {
			var listed []storeCategory
			c.expect(t, http.MethodGet, "/wp-json/wc/v3/products/categories?slug="+slug, nil, http.StatusOK, &listed)
			if len(listed) != 1 || listed[0].Parent != parent {
				t.Fatalf("the category %s reads %+v, want it under %d", slug, listed, parent)
			}
			parent = listed[0].ID
		}
		leaf := facts.categoryChain[len(facts.categoryChain)-1]
		if !slices.ContainsFunc(variable[0].Categories, func(category storeCategory) bool { return category.Slug == leaf }) {
			t.Errorf("the variable product sits in %+v, want %s", variable[0].Categories, leaf)
		}

		taxonomies := strings.Fields(wpCLI(t, env, "taxonomy", "list", "--field=name"))
		if slices.Contains(taxonomies, "entity") != facts.entityTaxonomy {
			t.Fatalf("the stand registers the taxonomies %v, and the entity taxonomy is expected: %t", taxonomies, facts.entityTaxonomy)
		}
		var exposed map[string]any
		c.expect(t, http.MethodGet, "/wp-json/wp/v2/taxonomies?type=product", nil, http.StatusOK, &exposed)
		if _, held := exposed["entity"]; held != facts.entityEmbedded {
			t.Errorf("core REST exposes the entity taxonomy: %t, want %t", held, facts.entityEmbedded)
		}
		if !facts.entityTaxonomy {
			return
		}
		carried := strings.Fields(wpCLI(t, env, "post", "term", "list", fmt.Sprint(variable[0].ID), "entity", "--field=slug"))
		if !slices.Contains(carried, facts.entityTerm) {
			t.Errorf("the variable product carries the entities %v, want %s", carried, facts.entityTerm)
		}
	})
}

func sameMembers(got, want []string) bool {
	return len(got) == len(want) && !slices.ContainsFunc(want, func(item string) bool { return !slices.Contains(got, item) })
}
