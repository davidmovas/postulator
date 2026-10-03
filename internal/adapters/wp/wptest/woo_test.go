package wptest_test

import (
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

type productBody struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Permalink        string `json:"permalink"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	Description      string `json:"description"`
	ShortDescription string `json:"short_description"`
	RegularPrice     string `json:"regular_price"`
	SKU              string `json:"sku"`
	DateModifiedGMT  string `json:"date_modified_gmt"`
	Attributes       []struct {
		ID        int64    `json:"id"`
		Name      string   `json:"name"`
		Options   []string `json:"options"`
		Position  int      `json:"position"`
		Visible   bool     `json:"visible"`
		Variation bool     `json:"variation"`
	} `json:"attributes"`
	Images []struct {
		ID int64 `json:"id"`
	} `json:"images"`
	Categories []struct {
		ID   int64  `json:"id"`
		Slug string `json:"slug"`
	} `json:"categories"`
}

func TestProductsUseTheWooCommerceFieldNames(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{
		Type:         wptest.TypeProduct,
		Title:        "Koffein Powder",
		Content:      "<p>long</p>",
		Excerpt:      "short",
		Status:       "publish",
		RegularPrice: "9.90",
		SKU:          "kp-1",
	})

	cases := []struct {
		name        string
		query       string
		description string
		short       string
	}{
		{name: "the edit context answers what is stored", query: "?context=edit", description: "<p>long</p>", short: "short"},
		{name: "the view context answers the rendering", query: "", description: "<p>long</p>\n", short: "short\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/"+itoa(seeded[0].ID)+tc.query, nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}

			var product productBody
			decode(t, payload, &product)
			if product.Name != "Koffein Powder" || product.Description != tc.description || product.ShortDescription != tc.short {
				t.Errorf("product = %+v", product)
			}
			if product.Type != "simple" || product.RegularPrice != "9.90" || product.SKU != "kp-1" {
				t.Errorf("product = %+v", product)
			}
			if product.Permalink != server.URL()+"/product/koffein-powder/" || product.DateModifiedGMT == "" {
				t.Errorf("product = %+v", product)
			}
		})
	}
}

func TestADraftProductHasNoPrettyAddress(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Draft", Status: "draft"})[0]

	_, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/"+itoa(seeded.ID), nil, true)
	var product productBody
	decode(t, payload, &product)
	if want := server.URL() + "/?post_type=product&p=" + itoa(seeded.ID); product.Permalink != want {
		t.Errorf("permalink = %q, want %q", product.Permalink, want)
	}
}

func TestProductsPageWithoutTheCoreEndOfListError(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "One"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Two"},
	)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products?page=9&per_page=1", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; WooCommerce answers an empty page", response.StatusCode)
	}
	if response.Header.Get("X-WP-TotalPages") != "2" {
		t.Errorf("X-WP-TotalPages = %q, want 2", response.Header.Get("X-WP-TotalPages"))
	}

	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 0 {
		t.Errorf("got %d items, want none", len(items))
	}
}

func TestAProductListLeavesTheTrashOutUnlessAsked(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "Live", Status: "publish"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Draft", Status: "draft"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Gone", Status: "trash"},
	)

	cases := []struct {
		name   string
		query  string
		status int
		count  int
	}{
		{name: "any status but the trash", query: "", status: http.StatusOK, count: 2},
		{name: "one status", query: "?status=draft", status: http.StatusOK, count: 1},
		{name: "the trash when asked", query: "?status=trash", status: http.StatusOK, count: 1},
		{name: "a comma list", query: "?status=publish,draft", status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products"+tc.query, nil, true)
			if response.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, tc.status)
			}
			if tc.status != http.StatusOK {
				return
			}
			var items []map[string]any
			decode(t, payload, &items)
			if len(items) != tc.count {
				t.Errorf("got %d products, want %d", len(items), tc.count)
			}
		})
	}
}

func TestUpdatingAProductWritesTheWooCommerceFields(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	product := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", RegularPrice: "9.90"})[0]

	body := []byte(`{"short_description":"<p>brief<br/>line</p><script>x()</script>","status":"draft",` +
		`"categories":[{"id":` + itoa(category.ID) + `}],"images":[{"id":41}],` +
		`"attributes":[{"id":7,"name":"Color","options":["<b>Blue</b>"],"visible":true},` +
		`{"name":"Form","options":["Liquid"],"position":1,"visible":true},{"options":["nameless"]}]}`)
	_, payload := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(product.ID), body, true)

	var updated productBody
	decode(t, payload, &updated)

	if updated.ShortDescription != "<p>brief<br />line</p>" || updated.Status != "draft" || updated.RegularPrice != "9.90" {
		t.Errorf("updated = %+v", updated)
	}
	if len(updated.Categories) != 1 || updated.Categories[0].ID != category.ID || updated.Categories[0].Slug != "koffein" {
		t.Errorf("categories = %+v", updated.Categories)
	}
	if len(updated.Images) != 1 || updated.Images[0].ID != 41 {
		t.Errorf("images = %+v", updated.Images)
	}
	if len(updated.Attributes) != 2 {
		t.Fatalf("attributes = %+v; an entry with neither id nor name is skipped", updated.Attributes)
	}
	if color := updated.Attributes[0]; color.ID != 7 || !slices.Equal(color.Options, []string{"Blue"}) || !color.Visible {
		t.Errorf("color = %+v; option values lose their tags", color)
	}
	if form := updated.Attributes[1]; form.ID != 0 || form.Name != "Form" || form.Position != 1 {
		t.Errorf("form = %+v", form)
	}
}

func TestSendingAttributesReplacesTheWholeList(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	product := server.Seed(wptest.Item{
		Type:  wptest.TypeProduct,
		Title: "Powder",
		Attributes: []wptest.Attribute{
			{Name: "Origin", Options: []string{"Client"}, Visible: true},
			{Name: "Form", Options: []string{"Powder"}, Visible: true},
		},
	})[0]

	_, payload := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(product.ID),
		[]byte(`{"attributes":[{"name":"Form","options":["Liquid"],"visible":true}]}`), true)

	var updated productBody
	decode(t, payload, &updated)
	if len(updated.Attributes) != 1 || updated.Attributes[0].Name != "Form" {
		t.Errorf("attributes = %+v; the list the store keeps is the one it was sent", updated.Attributes)
	}
}

func TestSavingAProductPostFiltersTheDescriptionForAUserWithoutUnfilteredHTML(t *testing.T) {
	t.Parallel()

	const stored = "<p>one<br/>two</p>"
	cases := []struct {
		name    string
		options []wptest.Option
		body    string
		want    string
	}{
		{name: "a short description saved by an administrator", body: `{"short_description":"<p>s</p>"}`, want: stored},
		{
			name: "a short description saved by a shop manager", options: []wptest.Option{wptest.WithFilteredHTML()},
			body: `{"short_description":"<p>s</p>"}`, want: "<p>one<br />two</p>",
		},
		{
			name: "attributes saved by a shop manager", options: []wptest.Option{wptest.WithFilteredHTML()},
			body: `{"attributes":[{"name":"Form","options":["Liquid"]}]}`, want: stored,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			product := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Content: stored})[0]

			call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(product.ID), []byte(tc.body), true)

			if held, _ := server.Lookup(product.ID); held.Content != tc.want {
				t.Errorf("description = %q, want %q", held.Content, tc.want)
			}
		})
	}
}

func TestAVisitorSeesAProductPageAsTheThemeLaysItOut(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		options     []wptest.Option
		status      string
		code        int
		description bool
	}{
		{name: "a theme that prints the description", status: "publish", code: http.StatusOK, description: true},
		{name: "a page builder that leaves it out", options: []wptest.Option{wptest.WithBuilderLayout()}, status: "publish", code: http.StatusOK},
		{name: "a product that is not published", status: "draft", code: http.StatusNotFound},
		{name: "a storefront that is down", options: []wptest.Option{wptest.WithStorefrontDown()}, status: "publish", code: http.StatusServiceUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Slug: "powder", Status: tc.status, Content: "<p>Made by hand.</p>"})

			response, err := http.Get(server.URL() + "/product/powder/")
			if err != nil {
				t.Fatalf("GET: %v", err)
			}
			raw, err := io.ReadAll(response.Body)
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Errorf("close: %v", closeErr)
			}
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if response.StatusCode != tc.code {
				t.Fatalf("status = %d, want %d", response.StatusCode, tc.code)
			}
			if tc.code != http.StatusOK {
				return
			}
			page := string(raw)
			if !strings.Contains(page, "<h1>Powder</h1>") || !strings.Contains(page, "application/ld+json") {
				t.Errorf("the page = %s, want the name and the structured data", page)
			}
			if strings.Contains(page, "<p>Made by hand.</p>") != tc.description {
				t.Errorf("the page = %s, want the description shown %t", page, tc.description)
			}
		})
	}
}

func TestTheNameIsFilteredOnTheWayIn(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	product := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder"})[0]

	_, payload := call(t, server, http.MethodPut, "/wp-json/wc/v3/products/"+itoa(product.ID),
		[]byte(`{"name":"Salt & Pepper"}`), true)

	var updated productBody
	decode(t, payload, &updated)
	if updated.Name != "Salt &amp; Pepper" {
		t.Errorf("name = %q, want the bare ampersand turned into an entity", updated.Name)
	}
}

func TestProductCategoriesCarryNoModificationDate(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/categories", nil, true)

	var categories []map[string]any
	decode(t, payload, &categories)
	if len(categories) != 1 {
		t.Fatalf("got %d categories, want 1", len(categories))
	}
	if _, present := categories[0]["date_modified_gmt"]; present {
		t.Error("a WooCommerce product category is a term and has no modification date")
	}
	if categories[0]["name"] != "Koffein" || categories[0]["description"] != "the hub" {
		t.Errorf("category = %v", categories[0])
	}
}

func TestTheWooCommerceRoutesGuardTheirIds(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]

	response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/categories/"+itoa(seeded.ID), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var category struct {
		Slug string `json:"slug"`
	}
	decode(t, payload, &category)
	if category.Slug != "koffein" {
		t.Errorf("slug = %q, want koffein", category.Slug)
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
		status int
	}{
		{name: "product zero", method: http.MethodGet, path: "/wp-json/wc/v3/products/0", status: http.StatusNotFound},
		{name: "missing product", method: http.MethodGet, path: "/wp-json/wc/v3/products/404", status: http.StatusNotFound},
		{name: "a category is not a product", method: http.MethodGet, path: "/wp-json/wc/v3/products/" + itoa(seeded.ID), status: http.StatusNotFound},
		{name: "missing product category", method: http.MethodGet, path: "/wp-json/wc/v3/products/categories/404", status: http.StatusNotFound},
		{name: "update a missing product", method: http.MethodPost, path: "/wp-json/wc/v3/products/404", body: []byte(`{"status":"draft"}`), status: http.StatusNotFound},
		{name: "update with a malformed id", method: http.MethodPost, path: "/wp-json/wc/v3/products/none", body: []byte(`{"status":"draft"}`), status: http.StatusNotFound},
		{name: "update with a broken body", method: http.MethodPost, path: "/wp-json/wc/v3/products/" + itoa(seeded.ID), body: []byte("not json"), status: http.StatusBadRequest},
		{name: "invalid window", method: http.MethodGet, path: "/wp-json/wc/v3/products?per_page=0", status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, _ := call(t, server, tc.method, tc.path, tc.body, true)
			if got.StatusCode != tc.status {
				t.Errorf("status = %d, want %d", got.StatusCode, tc.status)
			}
		})
	}
}

func TestUpdatingAProductRenamesAndReslugsIt(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder"})[0]

	_, payload := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(seeded.ID), []byte(`{"name":"Koffein Powder","slug":"koffein-powder"}`), true)

	var updated struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	decode(t, payload, &updated)
	if updated.Name != "Koffein Powder" || updated.Slug != "koffein-powder" {
		t.Errorf("updated = %+v", updated)
	}
}

func TestASiteWithoutAStoreHasNoStoreRoutes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutCommerce())
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder"})[0]

	for _, path := range []string{"/wp-json/wc/v3/products", "/wp-json/wc/v3/products/" + itoa(seeded.ID)} {
		response, payload := call(t, server, http.MethodGet, path, nil, true)
		var failure struct {
			Code string `json:"code"`
		}
		decode(t, payload, &failure)
		if response.StatusCode != http.StatusNotFound || failure.Code != "rest_no_route" {
			t.Errorf("%s: status %d, code %q", path, response.StatusCode, failure.Code)
		}
	}

	_, payload := call(t, server, http.MethodGet, "/wp-json", nil, true)
	var root struct {
		Namespaces []string `json:"namespaces"`
	}
	decode(t, payload, &root)
	if slices.Contains(root.Namespaces, "wc/v3") {
		t.Errorf("namespaces = %v, want no wc/v3", root.Namespaces)
	}
}

func TestAUserWithoutProductRightsReadsButCannotEdit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		status  int
		allowed bool
	}{
		{name: "an administrator", status: http.StatusOK, allowed: true},
		{name: "an editor without product rights", options: []wptest.Option{wptest.WithoutProductEdit()}, status: http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			seeded := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder"})[0]

			got, _ := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(seeded.ID), []byte(`{"short_description":"x"}`), true)
			if got.StatusCode != tc.status {
				t.Errorf("update status = %d, want %d", got.StatusCode, tc.status)
			}

			_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/users/me?context=edit", nil, true)
			var me struct {
				Capabilities map[string]bool `json:"capabilities"`
			}
			decode(t, payload, &me)
			if me.Capabilities["edit_products"] != tc.allowed || !me.Capabilities["edit_pages"] {
				t.Errorf("capabilities = %v", me.Capabilities)
			}
		})
	}
}

func TestTheCurrentUserShowsItsCapabilitiesOnlyInTheEditContext(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/users/me", nil, true)

	var me map[string]any
	decode(t, payload, &me)
	if _, present := me["capabilities"]; present {
		t.Errorf("me = %v; the view context carries no capabilities", me)
	}
}
