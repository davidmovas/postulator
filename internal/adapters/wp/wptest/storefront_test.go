package wptest_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

var (
	structuredPattern = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)
	articlePattern    = regexp.MustCompile(`(?s)<main><article>(.*)</article></main>`)
)

func visit(t *testing.T, server *wptest.Server, path string) (status int, page string) {
	t.Helper()

	response, payload := call(t, server, http.MethodGet, path, nil, false)
	return response.StatusCode, string(payload)
}

func TestThePublicRenderOfAPageCarriesItsBreadcrumb(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	guide := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Guide", Slug: "guide"})[0]
	server.Seed(
		wptest.Item{
			Type: wptest.TypePage, Title: "Setup", Slug: "setup", Parent: guide.ID, Content: "<p>Set it up.</p>",
			Crumbs: []wptest.Crumb{{Name: "Home", Path: "/"}, {Name: "Guide", Path: "/guide/"}, {Name: "Setup", Path: "/guide/setup/"}},
		},
		wptest.Item{Type: wptest.TypePage, Title: "Plain", Slug: "plain", Content: "<p>Plain.</p>"},
	)
	site := server.URL()

	status, page := visit(t, server, "/guide/setup/")
	if status != http.StatusOK {
		t.Fatalf("status = %d: %s", status, page)
	}
	scripts := structuredPattern.FindAllStringSubmatch(page, -1)
	if len(scripts) != 1 {
		t.Fatalf("the page carries %d ld+json scripts, want one: %s", len(scripts), page)
	}
	var list struct {
		Context  string `json:"@context"`
		Type     string `json:"@type"`
		Elements []struct {
			Type     string `json:"@type"`
			Name     string `json:"name"`
			Item     string `json:"item"`
			Position int    `json:"position"`
		} `json:"itemListElement"`
	}
	if err := json.Unmarshal([]byte(scripts[0][1]), &list); err != nil {
		t.Fatalf("decode the BreadcrumbList: %v", err)
	}
	if list.Context != "https://schema.org" || list.Type != "BreadcrumbList" || len(list.Elements) != 3 {
		t.Fatalf("list = %+v", list)
	}
	for index, want := range []struct{ name, item string }{{"Home", site + "/"}, {"Guide", site + "/guide/"}, {"Setup", site + "/guide/setup/"}} {
		got := list.Elements[index]
		if got.Type != "ListItem" || got.Position != index+1 || got.Name != want.name || got.Item != want.item {
			t.Errorf("crumb %d = %+v, want %s at %s", index, got, want.name, want.item)
		}
	}

	article := articlePattern.FindStringSubmatch(page)
	if article == nil {
		t.Fatalf("no main > article in %s", page)
	}
	nav := `<nav aria-label="Breadcrumb"><ol>` +
		`<li><a href="` + site + `/">Home</a></li>` +
		`<li><a href="` + site + `/guide/">Guide</a></li>` +
		`<li><span aria-current="page">Setup</span></li>` +
		`</ol></nav>`
	if want := nav + "<h1>Setup</h1><p>Set it up.</p>"; article[1] != want {
		t.Errorf("article = %q, want %q", article[1], want)
	}
	for _, landmark := range []string{
		`<header><nav aria-label="Primary"><a href="` + site + `/">Home</a><a href="` + site + `/shop/">Shop</a></nav></header>`,
		`<footer><nav><a href="` + site + `/privacy-policy/">Privacy Policy</a></nav></footer>`,
		"<title>Setup</title>",
	} {
		if !strings.Contains(page, landmark) {
			t.Errorf("the page lacks %q: %s", landmark, page)
		}
	}

	status, page = visit(t, server, "/plain/")
	if status != http.StatusOK {
		t.Fatalf("plain status = %d: %s", status, page)
	}
	if structuredPattern.MatchString(page) || strings.Contains(page, `aria-label="Breadcrumb"`) {
		t.Errorf("a page without crumbs carries a breadcrumb: %s", page)
	}
	if article := articlePattern.FindStringSubmatch(page); len(article) < 2 || article[1] != "<h1>Plain</h1><p>Plain.</p>" {
		t.Errorf("plain article = %v", article)
	}
}

func TestEveryPublishedItemHasAPublicRenderOnItsPath(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	guide := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Guide", Slug: "guide"})[0]
	peptides := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Peptides", Slug: "peptides"})[0]
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Setup", Slug: "setup", Parent: guide.ID, Content: "<p>Set it up.</p>"},
		wptest.Item{Type: wptest.TypePost, Title: "News", Slug: "news", Content: "<p>Fresh.</p>"},
		wptest.Item{Type: wptest.TypePage, Title: "Hub", Slug: "hub", Filtered: "<section><h2>Products</h2></section>"},
		wptest.Item{Type: wptest.TypeProductCategory, Title: "Healing", Slug: "healing", Parent: peptides.ID, Content: "<p>Healing peptides.</p>"},
		wptest.Item{
			Type: wptest.TypeProduct, Title: "Powder", Slug: "powder", Content: "<p>Made by hand.</p>",
			Crumbs: []wptest.Crumb{{Name: "Home", Path: "/"}, {Name: "Healing", Path: "/product-category/peptides/healing/"}, {Name: "Powder"}},
		},
	)

	cases := []struct {
		name string
		path string
		want []string
	}{
		{name: "a nested page", path: "/guide/setup/", want: []string{"<h1>Setup</h1><p>Set it up.</p>"}},
		{name: "a path asked without its trailing slash", path: "/guide/setup", want: []string{"<h1>Setup</h1>"}},
		{name: "a post", path: "/news/", want: []string{"<h1>News</h1><p>Fresh.</p>"}},
		{name: "a page the site adds to", path: "/hub/", want: []string{"<h1>Hub</h1><section><h2>Products</h2></section>"}},
		{name: "a product category with its description", path: "/product-category/peptides/healing/", want: []string{"<h1>Healing</h1><p>Healing peptides.</p>"}},
		{name: "a product with its breadcrumb", path: "/product/powder/", want: []string{
			`"@type":"BreadcrumbList"`,
			`<li><a href="` + server.URL() + `/product-category/peptides/healing/">Healing</a></li><li><span aria-current="page">Powder</span></li>`,
			"<h1>Powder</h1><p>Made by hand.</p>",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, page := visit(t, server, tc.path)
			if status != http.StatusOK {
				t.Fatalf("status = %d: %s", status, page)
			}
			for _, want := range tc.want {
				if !strings.Contains(page, want) {
					t.Errorf("the render lacks %q: %s", want, page)
				}
			}
		})
	}
}

func TestADraftHasNoPublicRender(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Soon", Slug: "soon", Status: "draft", Content: "<p>Soon.</p>"},
		wptest.Item{Type: wptest.TypePage, Title: "Waiting", Slug: "waiting", Status: "pending", Content: "<p>Waiting.</p>"},
		wptest.Item{Type: wptest.TypePage, Title: "Live", Slug: "live", Content: "<p>Live.</p>"},
	)

	cases := []struct {
		name   string
		path   string
		status int
	}{
		{name: "a draft", path: "/soon/", status: http.StatusNotFound},
		{name: "a pending page", path: "/waiting/", status: http.StatusNotFound},
		{name: "a path no page answers", path: "/nowhere/", status: http.StatusNotFound},
		{name: "a published page", path: "/live/", status: http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, page := visit(t, server, tc.path)
			if status != tc.status {
				t.Errorf("status = %d, want %d: %s", status, tc.status, page)
			}
			if tc.status != http.StatusOK && strings.Contains(page, "<h1>") {
				t.Errorf("a refused page still renders: %s", page)
			}
		})
	}

	down := wptest.New(t, wptest.WithStorefrontDown())
	down.Seed(wptest.Item{Type: wptest.TypePage, Title: "Live", Slug: "live"})
	if status, _ := visit(t, down, "/live/"); status != http.StatusServiceUnavailable {
		t.Errorf("a page while the storefront is down = %d, want 503", status)
	}
}
