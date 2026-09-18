//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type seoResult struct {
	Applied   []string `json:"applied"`
	SEOPlugin string   `json:"seoPlugin"`
}

func writeSEO(t *testing.T, c *client, id int, body map[string]string) seoResult {
	t.Helper()

	var out seoResult
	c.expect(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", id), body, http.StatusOK, &out)
	return out
}

func TestSEOMetaRoundTripsAndRenders(t *testing.T) {
	c, env := newClient(t)

	slug := uniqueSlug("seo")
	id := createPage(t, c, pageSpec{title: "SEO", slug: slug, content: "<h1>SEO</h1><p>body</p>"})

	title := "Postulator SEO title " + slug
	description := "Postulator SEO description " + slug
	canonical := env.baseURL + "/" + slug + "/"

	result := writeSEO(t, c, id, map[string]string{
		"title":         title,
		"description":   description,
		"canonical":     canonical,
		"ogTitle":       "OG " + title,
		"ogDescription": "OG " + description,
	})

	want := []string{"title", "description", "canonical", "ogTitle", "ogDescription"}
	if !slices.Equal(result.Applied, want) {
		t.Errorf("applied = %v, want %v", result.Applied, want)
	}
	if result.SEOPlugin != env.seo {
		t.Errorf("seoPlugin = %q, want %q", result.SEOPlugin, env.seo)
	}

	item := findItem(t, c, "page", id)
	if item.Meta.Title != title {
		t.Errorf("meta.title = %q, want %q", item.Meta.Title, title)
	}
	if item.Meta.Description != description {
		t.Errorf("meta.description = %q, want %q", item.Meta.Description, description)
	}
	if item.Meta.Canonical != canonical {
		t.Errorf("meta.canonical = %q, want %q", item.Meta.Canonical, canonical)
	}

	html := c.fetchPage(t, env.baseURL+item.Path)
	if !strings.Contains(html, "<title>"+title+"</title>") {
		t.Errorf("rendered page has no <title>%s</title>", title)
	}
	if !strings.Contains(html, description) {
		t.Errorf("rendered page has no meta description %q", description)
	}
	if !strings.Contains(html, canonical) {
		t.Errorf("rendered page has no canonical %q", canonical)
	}
	if got := strings.Count(html, `rel="canonical"`); got != 1 {
		t.Errorf("rendered page has %d canonical tags, want exactly 1", got)
	}
	if got := strings.Count(html, "<title>"); got != 1 {
		t.Errorf("rendered page has %d title tags, want exactly 1", got)
	}
	if got := strings.Count(html, `name="description"`); got > 1 {
		t.Errorf("rendered page has %d description tags, want at most 1", got)
	}
}

func TestSEOMetaAppliesOnlyPresentFieldsAndRemovesOnEmpty(t *testing.T) {
	c, _ := newClient(t)

	slug := uniqueSlug("partial")
	id := createPage(t, c, pageSpec{title: "Partial", slug: slug, content: "<p>p</p>"})

	writeSEO(t, c, id, map[string]string{"title": "first", "description": "kept"})

	result := writeSEO(t, c, id, map[string]string{"title": "second"})
	if !slices.Equal(result.Applied, []string{"title"}) {
		t.Errorf("applied = %v, want [title]", result.Applied)
	}

	item := findItem(t, c, "page", id)
	if item.Meta.Title != "second" {
		t.Errorf("meta.title = %q, want second", item.Meta.Title)
	}
	if item.Meta.Description != "kept" {
		t.Errorf("meta.description = %q, want kept", item.Meta.Description)
	}

	writeSEO(t, c, id, map[string]string{"description": ""})
	item = findItem(t, c, "page", id)
	if item.Meta.Description != "" {
		t.Errorf("meta.description = %q after an empty write, want empty", item.Meta.Description)
	}
}

func TestSEOMetaRejectsAnUnknownID(t *testing.T) {
	c, _ := newClient(t)

	status, body := c.request(t, http.MethodPut, "/wp-json/postulator/v1/seo-meta/98765432", map[string]string{"title": "x"})
	if status != http.StatusNotFound {
		t.Fatalf("status %d, want 404, body %s", status, body)
	}
	if !strings.Contains(string(body), `"not_found"`) {
		t.Fatalf("body %s does not carry the not_found code", body)
	}
}

func TestSEOMetaRejectsATermID(t *testing.T) {
	c, env := newClient(t)
	requireWoo(t, env)

	term := findBySlug(t, c, "product_cat", "postulator-koffein")

	status, body := c.request(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", term.ID), map[string]string{"title": "x"})
	if status != http.StatusNotFound {
		t.Fatalf("term id %d: status %d, want 404, body %s", term.ID, status, body)
	}
}

func TestSEOMetaEscapesRenderedHead(t *testing.T) {
	c, env := newClient(t)

	if env.seo != "none" {
		t.Skipf("head rendering belongs to the companion only when no SEO plugin is active, mode is %q", env.seo)
	}

	slug := uniqueSlug("xss")
	id := createPage(t, c, pageSpec{title: "XSS", slug: slug, content: "<p>x</p>"})

	payload := `Pwn</title><script>alert(1)</script>`
	attrPayload := `a" onload="alert(1)" x="<b>`
	writeSEO(t, c, id, map[string]string{
		"title":         payload,
		"description":   attrPayload,
		"canonical":     env.baseURL + `/" onmouseover="alert(1)`,
		"ogTitle":       attrPayload,
		"ogDescription": attrPayload,
	})

	item := findItem(t, c, "page", id)
	html := c.fetchPage(t, env.baseURL+item.Path)

	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatalf("rendered page carries an unescaped script tag")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("rendered page does not carry the escaped script text")
	}
	if strings.Contains(html, `onload="alert(1)"`) {
		t.Errorf("rendered page carries an unescaped onload attribute")
	}
	if strings.Contains(html, `onmouseover="alert(1)"`) {
		t.Errorf("rendered page carries an unescaped onmouseover attribute")
	}
	if got := strings.Count(html, "<title>"); got != 1 {
		t.Errorf("rendered page has %d title tags, want exactly 1", got)
	}
	if got := strings.Count(html, `rel="canonical"`); got != 1 {
		t.Errorf("rendered page has %d canonical tags, want exactly 1", got)
	}
}

func TestSEOMetaRejectsANonStringValue(t *testing.T) {
	c, _ := newClient(t)

	slug := uniqueSlug("nonstring")
	id := createPage(t, c, pageSpec{title: "NonString", slug: slug, content: "<p>n</p>"})
	writeSEO(t, c, id, map[string]string{"description": "must survive"})

	status, body := c.request(t, http.MethodPut,
		fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", id),
		map[string]any{"description": []string{"a", "b"}})

	if status != http.StatusBadRequest {
		t.Fatalf("status %d, want 400, body %s", status, body)
	}
	if !strings.Contains(string(body), `"invalid_param"`) {
		t.Errorf("body %s does not carry the invalid_param code", body)
	}

	item := findItem(t, c, "page", id)
	if item.Meta.Description != "must survive" {
		t.Fatalf("meta.description = %q; a rejected write must not delete the key", item.Meta.Description)
	}
}
