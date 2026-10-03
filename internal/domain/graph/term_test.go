package graph_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func flagged(id, name string, scope *string) graph.Entity {
	entity := named(id, name, scope)
	entity.SiteCategory = true
	return entity
}

func chainIDs(chain []graph.Entity) []string {
	ids := make([]string, 0, len(chain))
	for i := range chain {
		ids = append(ids, chain[i].ID)
	}
	return ids
}

func TestTaxonomiesAreTheTwoWordPressKnowsUsFor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		taxonomy graph.Taxonomy
		valid    bool
	}{
		{taxonomy: graph.TaxonomyCategory, valid: true},
		{taxonomy: graph.TaxonomyProductCategory, valid: true},
		{taxonomy: "post_tag"},
		{taxonomy: ""},
	}
	for _, tc := range cases {
		if got := tc.taxonomy.Valid(); got != tc.valid {
			t.Errorf("Taxonomy(%q).Valid() = %t, want %t", tc.taxonomy, got, tc.valid)
		}
	}
	if graph.TaxonomyCategory != "category" || graph.TaxonomyProductCategory != "product_cat" {
		t.Fatalf("the taxonomies are WordPress's own names, got %q and %q", graph.TaxonomyCategory, graph.TaxonomyProductCategory)
	}
}

func validTerm() graph.Term {
	return graph.Term{
		EntityID: entB, SiteID: siteA, Taxonomy: graph.TaxonomyCategory, TermID: 14, ParentTermID: 9,
		Name: " Healing ", RunID: "7f7f7f7f-7f7f-4f7f-8f7f-7f7f7f7f7f7f",
		SeenAt: time.Date(2026, time.October, 3, 9, 0, 0, 0, time.UTC),
	}
}

func TestNewTermTrimsTheName(t *testing.T) {
	t.Parallel()

	term, err := graph.NewTerm(validTerm())
	if err != nil {
		t.Fatalf("NewTerm: %v", err)
	}
	want := validTerm()
	want.Name = "Healing"
	if term != want {
		t.Fatalf("NewTerm = %+v, want %+v", term, want)
	}

	adopted := validTerm()
	adopted.ParentTermID = 0
	adopted.RunID = ""
	adopted.Taxonomy = graph.TaxonomyProductCategory
	if _, err = graph.NewTerm(adopted); err != nil {
		t.Fatalf("a top-level term no run created is refused: %v", err)
	}
}

func TestNewTermRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*graph.Term)
		field  string
	}{
		{name: "no entity", mutate: func(term *graph.Term) { term.EntityID = "" }, field: "entityId"},
		{name: "no site", mutate: func(term *graph.Term) { term.SiteID = "" }, field: "siteId"},
		{name: "a taxonomy we do not write", mutate: func(term *graph.Term) { term.Taxonomy = "post_tag" }, field: "taxonomy"},
		{name: "no term id", mutate: func(term *graph.Term) { term.TermID = 0 }, field: "termId"},
		{name: "a negative term id", mutate: func(term *graph.Term) { term.TermID = -3 }, field: "termId"},
		{name: "a negative parent", mutate: func(term *graph.Term) { term.ParentTermID = -1 }, field: "parentTermId"},
		{name: "a term under itself", mutate: func(term *graph.Term) { term.ParentTermID = term.TermID }, field: "parentTermId"},
		{name: "a blank name", mutate: func(term *graph.Term) { term.Name = "  " }, field: "name"},
		{name: "never seen", mutate: func(term *graph.Term) { term.SeenAt = time.Time{} }, field: "seenAt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			term := validTerm()
			tc.mutate(&term)
			_, err := graph.NewTerm(term)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestCategoryChainWalksUpToTheRootKeepingTheFlagged(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		entities []graph.Entity
		of       string
		want     []string
	}{
		{
			name: "no entity is a category",
			entities: []graph.Entity{
				named(entA, "Peptides", nil), named(entB, "Healing", new(entA)), named(entC, "Liquid", new(entB)),
			},
			of: entC, want: []string{},
		},
		{
			name: "the root group is not a category and the page's own entity is not either",
			entities: []graph.Entity{
				named(entA, "Peptides", nil), flagged(entB, "Healing", new(entA)),
				flagged(entC, "BPC-157", new(entB)), named(entD, "Liquid", new(entC)),
			},
			of: entD, want: []string{entB, entC},
		},
		{
			name: "a flagged entity is the last of its own chain",
			entities: []graph.Entity{
				named(entA, "Peptides", nil), flagged(entB, "Healing", new(entA)), flagged(entC, "BPC-157", new(entB)),
			},
			of: entC, want: []string{entB, entC},
		},
		{
			name: "a root that is flagged leads the chain",
			entities: []graph.Entity{
				flagged(entA, "Peptides", nil), named(entB, "Liquid", new(entA)),
			},
			of: entB, want: []string{entA},
		},
		{
			name: "an entity between two categories is passed over",
			entities: []graph.Entity{
				flagged(entA, "Healing", nil), named(entB, "Injectables", new(entA)),
				flagged(entC, "BPC-157", new(entB)), named(entD, "Liquid", new(entC)),
			},
			of: entD, want: []string{entA, entC},
		},
		{
			name: "a loop stops at the first entity seen twice",
			entities: []graph.Entity{
				flagged(entA, "Healing", new(entB)), flagged(entB, "BPC-157", new(entA)), named(entC, "Liquid", new(entA)),
			},
			of: entC, want: []string{entB, entA},
		},
		{
			name: "a parent that is not among the entities ends the walk",
			entities: []graph.Entity{
				flagged(entB, "Healing", new(entE)), flagged(entC, "BPC-157", new(entB)),
			},
			of: entC, want: []string{entB, entC},
		},
		{
			name: "an entity that is not among them has no chain",
			entities: []graph.Entity{
				flagged(entA, "Healing", nil),
			},
			of: entE, want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			chain := graph.CategoryChain(tc.entities, tc.of)
			if chain == nil {
				t.Fatal("CategoryChain = nil, want a list")
			}
			if got := chainIDs(chain); !slices.Equal(got, tc.want) {
				t.Fatalf("CategoryChain(%s) = %v, want %v", tc.of, got, tc.want)
			}

			chains := graph.CategoryChains(tc.entities)
			if len(chains) != len(tc.entities) {
				t.Fatalf("CategoryChains answered %d entities, want %d", len(chains), len(tc.entities))
			}
			for i := range tc.entities {
				id := tc.entities[i].ID
				listed, held := chains[id]
				if !held || listed == nil {
					t.Fatalf("CategoryChains has no list for %s", id)
				}
				if got, want := chainIDs(listed), chainIDs(graph.CategoryChain(tc.entities, id)); !slices.Equal(got, want) {
					t.Errorf("CategoryChains[%s] = %v, want %v as CategoryChain answers", id, got, want)
				}
			}
		})
	}
}

func TestCategoryChainCarriesTheEntitiesThemselves(t *testing.T) {
	t.Parallel()

	entities := []graph.Entity{flagged(entA, "Healing", nil), flagged(entB, "BPC-157", new(entA))}
	chain := graph.CategoryChain(entities, entB)
	if len(chain) != 2 || chain[0].Name != "Healing" || chain[1].Name != "BPC-157" || !chain[1].SiteCategory {
		t.Fatalf("CategoryChain = %+v, want the entities root first", chain)
	}

	chain[0].Name = "Renamed"
	if entities[0].Name != "Healing" {
		t.Fatal("writing to a chain changed the entities it was read from")
	}
}
