package application_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/graph"
)

const (
	categorySite = "0b6c2a4e-1f3d-4c8b-9a2e-5d7f8e9a0b1c"
	peptides     = "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a"
	healing      = "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b"
	bpc          = "3c3c3c3c-3c3c-4c3c-8c3c-3c3c3c3c3c3c"
	liquid       = "4d4d4d4d-4d4d-4d4d-8d4d-4d4d4d4d4d4d"
	lone         = "5e5e5e5e-5e5e-4e5e-8e5e-5e5e5e5e5e5e"
)

func categoryEntity(id, name string, flagged bool, scope *string) graph.Entity {
	return graph.Entity{
		ID: id, SiteID: categorySite, Name: name, Kind: graph.KindCategory, SiteCategory: flagged,
		ScopeID: scope, Source: graph.SourceImport,
	}
}

func categoryTerm(entityID string, taxonomy graph.Taxonomy, termID int64) graph.Term {
	return graph.Term{
		EntityID: entityID, SiteID: categorySite, Taxonomy: taxonomy, TermID: termID, Name: entityID,
		SeenAt: time.Date(2026, time.October, 3, 9, 0, 0, 0, time.UTC),
	}
}

func TestCategoryIndexGivesAPageItsChainWithTheTermsTheSiteHas(t *testing.T) {
	t.Parallel()

	index := application.NewCategoryIndex(
		[]graph.Entity{
			categoryEntity(peptides, "Peptides", false, nil),
			categoryEntity(healing, "Healing", true, new(peptides)),
			categoryEntity(bpc, "BPC-157", true, new(healing)),
			categoryEntity(liquid, "Liquid", false, new(bpc)),
			categoryEntity(lone, "Lone", false, nil),
		},
		[]graph.Term{
			categoryTerm(healing, graph.TaxonomyCategory, 5),
			categoryTerm(healing, graph.TaxonomyProductCategory, 31),
			categoryTerm(bpc, graph.TaxonomyCategory, 6),
			categoryTerm(lone, graph.TaxonomyCategory, 9),
		},
	)

	cases := []struct {
		name     string
		entityID string
		taxonomy graph.Taxonomy
		want     string
	}{
		{
			name: "a page's entity under two categories the site has", entityID: liquid, taxonomy: graph.TaxonomyCategory,
			want: `[{"entityId":"` + healing + `","name":"Healing","termId":5},{"entityId":"` + bpc + `","name":"BPC-157","termId":6}]`,
		},
		{
			name: "a product's chain, of which the store has only the first", entityID: liquid, taxonomy: graph.TaxonomyProductCategory,
			want: `[{"entityId":"` + healing + `","name":"Healing","termId":31},{"entityId":"` + bpc + `","name":"BPC-157"}]`,
		},
		{
			name: "a category is the last of its own chain", entityID: bpc, taxonomy: graph.TaxonomyCategory,
			want: `[{"entityId":"` + healing + `","name":"Healing","termId":5},{"entityId":"` + bpc + `","name":"BPC-157","termId":6}]`,
		},
		{name: "a root group that is not a category", entityID: peptides, taxonomy: graph.TaxonomyCategory, want: `[]`},
		{name: "an entity whose stored term is not a category any more", entityID: lone, taxonomy: graph.TaxonomyCategory, want: `[]`},
		{name: "an entity the index does not hold", entityID: "6f6f6f6f-6f6f-4f6f-8f6f-6f6f6f6f6f6f", taxonomy: graph.TaxonomyCategory, want: `[]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(index.Of(tc.entityID, tc.taxonomy))
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Of = %s\nwant %s", encoded, tc.want)
			}
		})
	}
}

func TestCategoryIndexHandsOutTermIDsOfItsOwn(t *testing.T) {
	t.Parallel()

	index := application.NewCategoryIndex(
		[]graph.Entity{categoryEntity(healing, "Healing", true, nil)},
		[]graph.Term{categoryTerm(healing, graph.TaxonomyCategory, 5)},
	)

	first := index.Of(healing, graph.TaxonomyCategory)
	*first[0].TermID = 99
	if again := index.Of(healing, graph.TaxonomyCategory); again[0].TermID == nil || *again[0].TermID != 5 {
		t.Fatalf("writing to one answer changed the next: %+v", again)
	}

	var empty application.CategoryIndex
	if got := empty.Of(healing, graph.TaxonomyCategory); got == nil || len(got) != 0 {
		t.Fatalf("an empty index answered %#v, want an empty list", got)
	}
}
