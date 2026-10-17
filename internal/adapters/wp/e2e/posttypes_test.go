//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestThePostRoutesServeContentTypesOnly(t *testing.T) {
	c, _ := newClient(t)

	var block struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodPost, "/wp-json/wp/v2/blocks", map[string]any{
		"title":   "Shared footer note",
		"content": "<!-- wp:paragraph --><p>Shared.</p><!-- /wp:paragraph -->",
		"status":  "publish",
	}, http.StatusCreated, &block)
	if block.ID == 0 {
		t.Fatal("the reusable block has id 0")
	}
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/blocks/%d?force=true", block.ID), nil)
	})

	routes := []struct {
		method string
		path   string
		body   any
	}{
		{method: http.MethodGet, path: rawPath(block.ID)},
		{method: http.MethodGet, path: renderedPath(block.ID)},
		{method: http.MethodPut, path: rawPath(block.ID), body: map[string]any{"content": "<p>Taken.</p>"}},
		{method: http.MethodGet, path: fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", block.ID)},
		{method: http.MethodPut, path: fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", block.ID), body: map[string]any{"title": "Taken"}},
		{method: http.MethodPost, path: previewPath(block.ID)},
	}
	for _, route := range routes {
		var failure struct {
			Code string `json:"code"`
		}
		c.expect(t, route.method, route.path, route.body, http.StatusNotFound, &failure)
		if failure.Code != "not_found" {
			t.Errorf("%s %s code = %q, want not_found", route.method, route.path, failure.Code)
		}
	}

	var stored struct {
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
	}
	c.expect(t, http.MethodGet, fmt.Sprintf("/wp-json/wp/v2/blocks/%d?context=edit", block.ID), nil, http.StatusOK, &stored)
	if stored.Content.Raw != "<!-- wp:paragraph --><p>Shared.</p><!-- /wp:paragraph -->" {
		t.Errorf("the reusable block was written through a post route: %q", stored.Content.Raw)
	}

	page := createPage(t, c, pageSpec{title: "Still served", slug: uniqueSlug("served"), content: "<p>Here.</p>"})
	c.expect(t, http.MethodGet, rawPath(page), nil, http.StatusOK, nil)
}
