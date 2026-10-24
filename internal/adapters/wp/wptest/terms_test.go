package wptest_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestTheTermsRouteListsEveryTaxonomyInOrder(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "BPC-157", Terms: []wptest.Term{
			{Taxonomy: "product_type", Name: "variable", Slug: "variable", ID: 4},
			{Taxonomy: "entity", Name: "BPC-157", Slug: "bpc-157", ID: 31},
			{Taxonomy: "product_cat", Name: "Healing", Slug: "healing", ID: 12, Parent: 11},
			{Taxonomy: "product_cat", Name: "Peptides", Slug: "peptides", ID: 11},
		}},
		wptest.Item{Type: wptest.TypePage, Title: "Bare"},
	)

	response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/terms/"+itoa(seeded[0].ID), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	var got struct {
		Terms []struct {
			Taxonomy string `json:"taxonomy"`
			Slug     string `json:"slug"`
			ID       int64  `json:"id"`
			Parent   int64  `json:"parent"`
		} `json:"terms"`
		ID int64 `json:"id"`
	}
	decode(t, payload, &got)
	order := make([]string, 0, len(got.Terms))
	for _, held := range got.Terms {
		order = append(order, held.Taxonomy+"/"+held.Slug)
	}
	want := []string{"entity/bpc-157", "product_cat/peptides", "product_cat/healing", "product_type/variable"}
	if got.ID != seeded[0].ID || len(order) != len(want) {
		t.Fatalf("terms = %v, want %v", order, want)
	}
	for index := range want {
		if order[index] != want[index] {
			t.Errorf("terms = %v, want %v", order, want)
			break
		}
	}
	if got.Terms[2].Parent != 11 {
		t.Errorf("the child category's parent = %d, want 11", got.Terms[2].Parent)
	}

	response, payload = call(t, server, http.MethodGet, "/wp-json/postulator/v1/terms/"+itoa(seeded[1].ID), nil, true)
	if response.StatusCode != http.StatusOK || !strings.Contains(string(payload), `"terms":[]`) {
		t.Errorf("a page without terms answers %d %s", response.StatusCode, payload)
	}
}
