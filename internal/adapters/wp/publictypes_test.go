package wp_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestListContentCarriesExcerptTermsAndCustomTypes(t *testing.T) {
	t.Parallel()

	server := recipeServer(t)
	recipe := server.Seed(wptest.Item{Type: "recipe", Title: "Soup", Excerpt: "Hot soup.", Terms: []wptest.Term{
		{Taxonomy: "cuisine", Name: "Thai", Slug: "thai", ID: 7},
	}})[0]
	cuisine := server.Seed(wptest.Item{Type: "cuisine", Title: "Thai", Slug: "thai"})[0]
	client := newClient(t, server)

	page, err := client.ListContent(t.Context(), wp.ContentQuery{Types: []wp.ItemType{"recipe", "cuisine"}})
	if err != nil {
		t.Fatalf("ListContent: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %+v", page.Items)
	}
	soup, thai := page.Items[0], page.Items[1]
	if soup.ID != recipe.ID || soup.Excerpt != "Hot soup." || !slices.Equal(soup.Terms, []wp.Term{{Taxonomy: "cuisine", Name: "Thai", Slug: "thai", ID: 7}}) {
		t.Errorf("the recipe = %+v", soup)
	}
	if thai.ID != cuisine.ID || thai.Type != "cuisine" || thai.Terms == nil || len(thai.Terms) != 0 || thai.Excerpt != "" {
		t.Errorf("the cuisine term = %+v", thai)
	}

	capabilities, err := client.Capabilities(t.Context())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if !slices.Contains(capabilities.PostTypes, "recipe") || !slices.Contains(capabilities.Taxonomies, "cuisine") || slices.Contains(capabilities.PostTypes, "attachment") {
		t.Errorf("types = %v and %v", capabilities.PostTypes, capabilities.Taxonomies)
	}
}

func TestThePostRoutesTakeTheTypesTheManifestNames(t *testing.T) {
	t.Parallel()

	server := recipeServer(t)
	recipe := server.Seed(wptest.Item{Type: "recipe", Title: "Soup", Content: "<p>Hot.</p>"})[0]
	cuisine := server.Seed(wptest.Item{Type: "cuisine", Title: "Thai"})[0]
	client := newClient(t, server)

	raw, err := client.GetRaw(t.Context(), "recipe", recipe.ID)
	if err != nil || raw.Content != "<p>Hot.</p>" {
		t.Fatalf("GetRaw(recipe) = %+v, %v", raw, err)
	}
	if _, err := client.GetRaw(t.Context(), "cuisine", cuisine.ID); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a taxonomy = %v, want Invalid", err)
	}
	if _, err := client.GetRaw(t.Context(), "nope", 1); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown type = %v, want Invalid", err)
	}
}
