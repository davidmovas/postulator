//go:build e2e

package e2e_test

import (
	"net/http"
	"strings"
	"testing"
)

const programName = "postulator"

func TestTheFrontEndNamesNothingOfTheProgram(t *testing.T) {
	c, env := newClient(t)
	stand := standOf(t, env)

	page := findItem(t, c, "page", createPage(t, c, pageSpec{title: "Opening hours", slug: uniqueSlug("hours"), content: "<p>Open daily.</p>"}))
	draft := createPage(t, c, pageSpec{title: "Coming soon", slug: uniqueSlug("soon"), content: "<p>Soon.</p>", status: "draft"})
	preview := issuePreview(t, c, draft)
	if strings.Contains(strings.ToLower(preview.URL), programName) {
		t.Errorf("the preview link %s names the program", preview.URL)
	}

	visits := map[string]string{
		"the home page":   env.baseURL + "/",
		"a page":          env.baseURL + page.Path,
		"a missing page":  env.baseURL + "/" + uniqueSlug("missing") + "/",
		"a draft preview": preview.URL,
		"the feed":        env.baseURL + "/feed/",
		"the sitemap":     env.baseURL + "/wp-sitemap.xml",
	}
	if env.woo {
		visits["a product"] = env.baseURL + findBySlug(t, c, "product", stand.simpleProduct).Path
		visits["a product category"] = env.baseURL + findBySlug(t, c, "product_cat", stand.category).Path
	}

	for name, target := range visits {
		status, header, body := c.anonymous(t, target)
		if status >= http.StatusInternalServerError {
			t.Errorf("%s (%s) answers %d", name, target, status)
			continue
		}
		if strings.Contains(strings.ToLower(body), programName) {
			at := strings.Index(strings.ToLower(body), programName)
			t.Errorf("%s (%s) names the program in its body: …%s…", name, target, body[max(0, at-80):min(len(body), at+80)])
		}
		for key, values := range header {
			for _, value := range values {
				if strings.Contains(strings.ToLower(key+": "+value), programName) {
					t.Errorf("%s (%s) names the program in its header %s: %s", name, target, key, value)
				}
			}
		}
	}
}
