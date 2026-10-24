package wp_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestGetTermsReadsEveryTaxonomyThePostCarries(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "BPC-157", Terms: []wptest.Term{
			{Taxonomy: "product_cat", Name: "Healing", Slug: "healing", ID: 12, Parent: 11},
			{Taxonomy: "entity", Name: "BPC-157", Slug: "bpc-157", ID: 31},
		}},
		wptest.Item{Type: wptest.TypePage, Title: "Bare"},
	)
	client := newClient(t, server)

	terms, err := client.GetTerms(t.Context(), wp.TypeProduct, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetTerms: %v", err)
	}
	want := []wp.Term{
		{Taxonomy: "entity", Name: "BPC-157", Slug: "bpc-157", ID: 31},
		{Taxonomy: "product_cat", Name: "Healing", Slug: "healing", ID: 12, Parent: 11},
	}
	if !slices.Equal(terms, want) {
		t.Errorf("terms = %+v, want %+v", terms, want)
	}

	bare, err := client.GetTerms(t.Context(), wp.TypePage, seeded[1].ID)
	if err != nil || bare == nil || len(bare) != 0 {
		t.Errorf("a page's terms = %v (%v), want an empty list", bare, err)
	}
}

func TestGetTermsRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	if _, err := newClient(t, wptest.New(t)).GetTerms(t.Context(), wp.TypeProductCategory, 3); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a product category = %v, want Invalid", err)
	}

	withoutTerms := slices.DeleteFunc(slices.Clone(currentCapabilities), func(name string) bool { return name == wp.CapabilityTerms })
	old := wptest.New(t, wptest.WithCapabilities(withoutTerms...))
	page := old.Seed(wptest.Item{Type: wptest.TypePage, Title: "Bare"})[0]
	if _, err := newClient(t, old).GetTerms(t.Context(), wp.TypePage, page.ID); !wp.IsPluginOutdated(err) {
		t.Errorf("a plugin without terms = %v, want plugin_outdated", err)
	}
}
