package wp_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func recipeServer(t *testing.T, opts ...wptest.Option) *wptest.Server {
	t.Helper()

	return wptest.New(t, append([]wptest.Option{
		wptest.WithPostType(wptest.PostType{Slug: "recipe", Name: "Recipes", RestBase: "recipes", Taxonomies: []string{"cuisine", "secret"}, Viewable: true}),
		wptest.WithTaxonomy(wptest.Taxonomy{Name: "cuisine", RestBase: "cuisines", ObjectTypes: []string{"recipe"}, ShowInREST: true}),
		wptest.WithTaxonomy(wptest.Taxonomy{Name: "secret", ObjectTypes: []string{"recipe"}}),
	}, opts...)...)
}

func countPath(server *wptest.Server, path string) int {
	count := 0
	for _, recorded := range server.Requests() {
		if recorded.Path == path {
			count++
		}
	}
	return count
}

func TestPostTypesNameTheRestBaseOfEveryType(t *testing.T) {
	t.Parallel()

	server := recipeServer(t)
	client := newClient(t, server)

	types, err := client.PostTypes(t.Context())
	if err != nil {
		t.Fatalf("PostTypes: %v", err)
	}

	slugs := make([]string, 0, len(types))
	for _, postType := range types {
		slugs = append(slugs, postType.Slug)
	}
	if want := []string{"attachment", "page", "post", "product", "recipe"}; !slices.Equal(slugs, want) {
		t.Fatalf("slugs = %v, want %v", slugs, want)
	}

	byName := make(map[string]wp.PostType, len(types))
	for _, postType := range types {
		byName[postType.Slug] = postType
	}
	recipe := byName["recipe"]
	if recipe.RestBase != "recipes" || recipe.RestNamespace != "wp/v2" || !recipe.Viewable || recipe.Name != "Recipes" {
		t.Errorf("recipe = %+v", recipe)
	}
	if !byName["page"].Hierarchical || byName["post"].Hierarchical {
		t.Errorf("hierarchy: page %t, post %t", byName["page"].Hierarchical, byName["post"].Hierarchical)
	}
	if got := byName["post"].Taxonomies; !slices.Equal(got, []string{"category", "post_tag"}) {
		t.Errorf("post taxonomies = %v", got)
	}

	if _, err := client.PostTypes(t.Context()); err != nil {
		t.Fatalf("PostTypes again: %v", err)
	}
	if got := countPath(server, "/wp-json/wp/v2/types"); got != 1 {
		t.Errorf("the types were read %d times, want once", got)
	}
	client.InvalidateManifest()
	if _, err := client.PostTypes(t.Context()); err != nil {
		t.Fatalf("PostTypes after invalidation: %v", err)
	}
	if got := countPath(server, "/wp-json/wp/v2/types"); got != 2 {
		t.Errorf("the types were read %d times after invalidation, want twice", got)
	}
}

func TestACustomTypeIsListedByItsRestBase(t *testing.T) {
	t.Parallel()

	server := recipeServer(t)
	seeded := server.Seed(wptest.Item{Type: "recipe", Title: "Soup", Slug: "soup", Status: "publish", Content: "<p>Hot.</p>"})
	client := newClient(t, server)

	page, err := client.ListItems(t.Context(), wp.ItemType("recipe"), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Soup" || page.Items[0].Type != "recipe" {
		t.Fatalf("items = %+v", page.Items)
	}
	if got := countPath(server, "/wp-json/wp/v2/recipes"); got != 1 {
		t.Errorf("the list reached /recipes %d times, want once", got)
	}

	item, err := client.GetItem(t.Context(), wp.ItemType("recipe"), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if item.Slug != "soup" || item.Content != "<p>Hot.</p>" {
		t.Errorf("item = %+v", item)
	}
}

func TestTheStoreAndMediaTypesStayOnTheirOwnRoutes(t *testing.T) {
	t.Parallel()

	server := recipeServer(t)
	client := newClient(t, server)

	for _, itemType := range []wp.ItemType{wp.TypeProduct, wp.TypeProductCategory, "product_variation", "attachment"} {
		if _, err := client.ListItems(t.Context(), itemType, wp.ListQuery{}); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("ListItems(%s) code = %q, want %q", itemType, errors.CodeOf(err), errors.Invalid)
		}
		if _, err := client.GetItem(t.Context(), itemType, 1); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("GetItem(%s) code = %q, want %q", itemType, errors.CodeOf(err), errors.Invalid)
		}
	}
	if len(server.Requests()) != 0 {
		t.Errorf("a refused type reached the site %d times", len(server.Requests()))
	}
}

func TestAnUnknownTypeIsRefusedOnceTheTypesAreRead(t *testing.T) {
	t.Parallel()

	server := recipeServer(t)
	client := newClient(t, server)

	_, err := client.ListItems(t.Context(), wp.ItemType("nope"), wp.ListQuery{})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if got := countPath(server, "/wp-json/wp/v2/types"); got != 1 {
		t.Errorf("the types were read %d times, want once", got)
	}
}

func TestEmbeddedTermsArriveWithTheItem(t *testing.T) {
	t.Parallel()

	server := recipeServer(t)
	server.Seed(wptest.Item{Type: "recipe", Title: "Soup", Slug: "soup", Status: "publish", Terms: []wptest.Term{
		{Taxonomy: "cuisine", Name: "Thai", Slug: "thai", ID: 7, Parent: 3},
		{Taxonomy: "secret", Name: "Hidden", Slug: "hidden", ID: 8},
	}})
	client := newClient(t, server)

	want := []wp.Term{{Taxonomy: "cuisine", Name: "Thai", Slug: "thai", ID: 7}}
	cases := []struct {
		name  string
		query wp.ListQuery
	}{
		{name: "every field", query: wp.ListQuery{EmbedTerms: true}},
		{name: "narrowed fields", query: wp.ListQuery{EmbedTerms: true, Fields: []string{"title"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			page, err := client.ListItems(t.Context(), wp.ItemType("recipe"), tc.query)
			if err != nil {
				t.Fatalf("ListItems: %v", err)
			}
			if len(page.Items) != 1 || !slices.Equal(page.Items[0].Terms, want) {
				t.Fatalf("terms = %+v, want %+v", page.Items, want)
			}
		})
	}

	for _, recorded := range server.Requests() {
		if recorded.Path != "/wp-json/wp/v2/recipes" {
			continue
		}
		if got := recorded.Query.Get("_embed"); got != "wp:term" {
			t.Errorf("_embed = %q, want wp:term", got)
		}
		if fields := recorded.Query.Get("_fields"); fields != "" && (!strings.Contains(fields, "_links") || !strings.Contains(fields, "_embedded")) {
			t.Errorf("_fields = %q drops the embed", fields)
		}
	}

	server.ResetRequests()
	plain, err := client.ListItems(t.Context(), wp.ItemType("recipe"), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListItems without terms: %v", err)
	}
	if len(plain.Items) != 1 || len(plain.Items[0].Terms) != 0 {
		t.Errorf("terms without asking = %+v", plain.Items)
	}
	if recorded, ok := server.LastRequest(); !ok || recorded.Query.Has("_embed") {
		t.Errorf("a plain list asked for the embed: %+v", recorded.Query)
	}
}

func TestSiteSettingsNameTheFrontAndPostsPages(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SetReading("page", 12, 14)

	settings, err := newClient(t, server).SiteSettings(t.Context())
	if err != nil {
		t.Fatalf("SiteSettings: %v", err)
	}
	if want := (wp.SiteSettings{ShowOnFront: "page", FrontPageID: 12, PostsPageID: 14}); settings != want {
		t.Errorf("settings = %+v, want %+v", settings, want)
	}
}

func TestARefusedSettingsReadIsForbidden(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutSettingsAccess())

	_, err := newClient(t, server).SiteSettings(t.Context())
	if !wp.Forbidden(err) {
		t.Fatalf("SiteSettings = %v, want a refusal Forbidden tells apart", err)
	}
}
