//go:build e2e

package e2e_test

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestTheSameHrefIsClassifiedAlikeOnBothSides(t *testing.T) {
	c, env := newClient(t)

	base, err := url.Parse(env.baseURL)
	if err != nil || base.Port() == "" {
		t.Fatalf("the stand %s runs on no explicit port: %v", env.baseURL, err)
	}
	host, port := base.Hostname(), base.Port()
	other := "8088"
	if port == other {
		other = "8087"
	}

	hrefs := []string{
		env.baseURL + "/same-port/",
		"http://" + host + ":" + other + "/other-port/",
		"http://" + host + "/no-port/",
		"//" + host + ":" + port + "/protocol-relative/",
		"//" + host + ":" + other + "/protocol-relative-other/",
		"HTTP://" + strings.ToUpper(host) + ":" + port + "/Upper/Case/",
		env.baseURL,
		"/root-relative/",
		"/Café/",
		"/caf%C3%A9-encoded/",
		"/a%zz/",
		"/a/../b/",
		"#top",
		"?page=2",
		"mailto:shop@example.test",
		"ftp://" + host + ":" + port + "/files/",
		"a/",
	}
	var content strings.Builder
	for index, href := range hrefs {
		fmt.Fprintf(&content, "<p><a href=\"%s\">anchor %d</a></p>\n", href, index)
	}
	parent := createPage(t, c, pageSpec{title: "Href parent", slug: uniqueSlug("href-parent")})
	id := createPage(t, c, pageSpec{title: "Href table", slug: uniqueSlug("hrefs"), content: content.String(), parent: parent})
	item := findItem(t, c, "page", id)
	if strings.Count(strings.Trim(item.Path, "/"), "/") != 1 {
		t.Fatalf("the href page sits at %s, want one level under its parent", item.Path)
	}

	listed := make([]string, 0, len(item.Links))
	for _, found := range item.Links {
		listed = append(listed, found.Href)
	}
	site := pagemap.NewSite(env.baseURL)
	var want []string
	for _, href := range hrefs {
		path, kind := site.ResolveAt(item.Path, href)
		if kind == pagemap.LinkPath {
			want = append(want, path)
		}
	}
	slices.Sort(listed)
	slices.Sort(want)
	if !slices.Equal(listed, want) {
		t.Errorf("on %s the plugin lists %v, Site.ResolveAt(%s) says %v", env.baseURL, listed, item.Path, want)
	}

	relative, kind := site.ResolveAt(item.Path, "a/")
	if kind != pagemap.LinkPath || relative != item.Path+"a/" {
		t.Fatalf("Site.ResolveAt(%s, a/) = %s %s, want the path under the page", item.Path, relative, kind)
	}
	if !slices.Contains(listed, relative) {
		t.Errorf("on %s the plugin does not list the relative anchor at %s: %v", env.baseURL, relative, listed)
	}
}
