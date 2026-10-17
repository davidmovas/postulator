package wptest_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func recipeSite(t *testing.T, opts ...wptest.Option) *wptest.Server {
	t.Helper()

	return wptest.New(t, append([]wptest.Option{
		wptest.WithPostType(wptest.PostType{Slug: "recipe", Name: "Recipes", RestBase: "recipes", Taxonomies: []string{"cuisine", "secret"}, Viewable: true}),
		wptest.WithTaxonomy(wptest.Taxonomy{Name: "cuisine", RestBase: "cuisines", ObjectTypes: []string{"recipe"}, ShowInREST: true}),
		wptest.WithTaxonomy(wptest.Taxonomy{Name: "secret", ObjectTypes: []string{"recipe"}}),
	}, opts...)...)
}

func TestTheTypesRouteDescribesEveryPostType(t *testing.T) {
	t.Parallel()

	server := recipeSite(t)
	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/types?context=edit", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}

	var types map[string]struct {
		Name          string   `json:"name"`
		Slug          string   `json:"slug"`
		RestBase      string   `json:"rest_base"`
		RestNamespace string   `json:"rest_namespace"`
		Taxonomies    []string `json:"taxonomies"`
		Hierarchical  bool     `json:"hierarchical"`
		Viewable      bool     `json:"viewable"`
	}
	decode(t, payload, &types)

	for slug, want := range map[string]string{"attachment": "media", "page": "pages", "post": "posts", "product": "product", "recipe": "recipes"} {
		described, ok := types[slug]
		if !ok {
			t.Errorf("the types lack %s", slug)
			continue
		}
		if described.RestBase != want || described.RestNamespace != "wp/v2" || described.Slug != slug {
			t.Errorf("%s = %+v, want the rest base %s under wp/v2", slug, described, want)
		}
	}
	if recipe := types["recipe"]; recipe.Name != "Recipes" || !recipe.Viewable || !slices.Equal(recipe.Taxonomies, []string{"cuisine", "secret"}) {
		t.Errorf("recipe = %+v", recipe)
	}
	if !types["page"].Hierarchical || types["post"].Hierarchical {
		t.Errorf("hierarchy: page %t, post %t", types["page"].Hierarchical, types["post"].Hierarchical)
	}
}

func TestACustomTypeIsServedAtItsRestBase(t *testing.T) {
	t.Parallel()

	server := recipeSite(t)
	seeded := server.Seed(wptest.Item{Type: "recipe", Title: "Soup", Slug: "soup", Status: "publish"})

	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/recipes", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d: %s", response.StatusCode, payload)
	}
	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 1 || items[0]["slug"] != "soup" || items[0]["type"] != "recipe" {
		t.Fatalf("items = %v", items)
	}

	response, payload = call(t, server, http.MethodGet, "/wp-json/wp/v2/recipes/"+itoa(seeded[0].ID), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d: %s", response.StatusCode, payload)
	}
}

func TestTheEmbedCarriesTheTermsOfRestTaxonomiesOnly(t *testing.T) {
	t.Parallel()

	server := recipeSite(t)
	server.Seed(wptest.Item{Type: "recipe", Title: "Soup", Slug: "soup", Status: "publish", Terms: []wptest.Term{
		{Taxonomy: "cuisine", Name: "Thai", Slug: "thai", ID: 7, Parent: 3},
		{Taxonomy: "secret", Name: "Hidden", Slug: "hidden", ID: 8},
	}})

	cases := []struct {
		name  string
		query string
		embed bool
	}{
		{name: "the term embed", query: "?_embed=wp:term", embed: true},
		{name: "every embed", query: "?_embed=1", embed: true},
		{name: "fields that keep the embed", query: "?_embed=wp:term&_fields=id,_links,_embedded", embed: true},
		{name: "fields that drop the embed", query: "?_embed=wp:term&_fields=id"},
		{name: "no embed", query: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/recipes"+tc.query, nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d: %s", response.StatusCode, payload)
			}
			var items []struct {
				Embedded *struct {
					Terms [][]struct {
						Taxonomy string `json:"taxonomy"`
						Name     string `json:"name"`
						Slug     string `json:"slug"`
						ID       int64  `json:"id"`
					} `json:"wp:term"`
				} `json:"_embedded"`
			}
			decode(t, payload, &items)
			if len(items) != 1 {
				t.Fatalf("items = %d", len(items))
			}
			if !tc.embed {
				if items[0].Embedded != nil {
					t.Errorf("the embed arrived unasked: %s", payload)
				}
				return
			}
			if items[0].Embedded == nil || len(items[0].Embedded.Terms) != 1 || len(items[0].Embedded.Terms[0]) != 1 {
				t.Fatalf("embedded = %s, want one group with the cuisine term", payload)
			}
			got := items[0].Embedded.Terms[0][0]
			if got.Taxonomy != "cuisine" || got.Name != "Thai" || got.Slug != "thai" || got.ID != 7 {
				t.Errorf("term = %+v", got)
			}
		})
	}
}

func TestTheSettingsAnswerTheReadingOptions(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SetReading("page", 12, 14)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/settings", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	var settings struct {
		ShowOnFront  string `json:"show_on_front"`
		PageOnFront  int64  `json:"page_on_front"`
		PageForPosts int64  `json:"page_for_posts"`
	}
	decode(t, payload, &settings)
	if settings.ShowOnFront != "page" || settings.PageOnFront != 12 || settings.PageForPosts != 14 {
		t.Errorf("settings = %+v", settings)
	}

	refused := wptest.New(t, wptest.WithoutSettingsAccess())
	response, payload = call(t, refused, http.MethodGet, "/wp-json/wp/v2/settings", nil, true)
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("status without manage_options = %d: %s", response.StatusCode, payload)
	}
}
