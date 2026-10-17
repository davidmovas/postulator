//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type renderedContent struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
	ID          int    `json:"id"`
}

func renderedPath(id int) string {
	return fmt.Sprintf("/wp-json/postulator/v1/content/%d/rendered", id)
}

func readRendered(t *testing.T, c *client, id int) renderedContent {
	t.Helper()

	var got renderedContent
	c.expect(t, http.MethodGet, renderedPath(id), nil, http.StatusOK, &got)
	if got.ID != id || got.ContentHash != hashOf(got.Content) {
		t.Fatalf("rendered %d = id %d, hash %s for %d bytes", id, got.ID, got.ContentHash, len(got.Content))
	}
	return got
}

func TestRenderedContentCarriesTheInjectedBreadcrumbOnTheReplica(t *testing.T) {
	c, env := newClient(t)
	stand := standOf(t, env)

	if stand.breadcrumbInContent {
		for _, slug := range []string{stand.hubEmptyRaw, stand.hubShortcode, stand.hubWithText} {
			hub := findBySlug(t, c, "page", slug)
			got := readRendered(t, c, hub.ID)
			nav := strings.Index(got.Content, `aria-label="Breadcrumb"`)
			section := strings.Index(got.Content, "<section")
			if got.Type != "page" || nav < 0 || section < nav || !strings.Contains(got.Content, "<h2") {
				t.Errorf("hub %s renders %q, want the breadcrumb nav, then the product section with its heading", slug, got.Content)
			}
		}
		shortcode := readRendered(t, c, findBySlug(t, c, "page", stand.hubShortcode).ID)
		if strings.Contains(shortcode.Content, "[products") {
			t.Errorf("the shortcode hub renders its shortcode unexpanded: %q", shortcode.Content)
		}
		if !strings.Contains(shortcode.Content, stand.unresolvedCrumb) {
			t.Errorf("the shortcode hub renders no crumb to %s, which has no page", stand.unresolvedCrumb)
		}
	}

	page := createPage(t, c, pageSpec{title: "Plain page", slug: uniqueSlug("plain"), content: "First line.\n\nSecond paragraph."})
	plain := readRendered(t, c, page)
	if !strings.Contains(plain.Content, "<p>First line.</p>") || !strings.Contains(plain.Content, "<p>Second paragraph.</p>") {
		t.Errorf("a plain page renders %q, want its paragraphs through the_content", plain.Content)
	}
	if strings.Contains(plain.Content, "Breadcrumb") {
		t.Errorf("a page that is no hub renders a breadcrumb: %q", plain.Content)
	}

	if status, _ := c.request(t, http.MethodGet, renderedPath(999999999), nil); status != http.StatusNotFound {
		t.Errorf("an unknown id = %d, want 404", status)
	}
	if status, _, _ := c.anonymous(t, env.baseURL+renderedPath(page)); status != http.StatusUnauthorized {
		t.Errorf("an anonymous read = %d, want 401", status)
	}
}
