package category_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestTaxonomiesAreTheTwoWordPressKnowsUsFor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		taxonomy category.Taxonomy
		valid    bool
	}{
		{taxonomy: category.TaxonomyCategory, valid: true},
		{taxonomy: category.TaxonomyProductCategory, valid: true},
		{taxonomy: "post_tag"},
		{taxonomy: "Category"},
		{taxonomy: ""},
	}
	for _, tc := range cases {
		if got := tc.taxonomy.Valid(); got != tc.valid {
			t.Errorf("Taxonomy(%q).Valid() = %t, want %t", tc.taxonomy, got, tc.valid)
		}
	}
	if category.TaxonomyCategory != "category" || category.TaxonomyProductCategory != "product_cat" {
		t.Fatalf("the taxonomies are WordPress's own names, got %q and %q", category.TaxonomyCategory, category.TaxonomyProductCategory)
	}
}

func validTerm() category.Term {
	return category.Term{
		CategoryID: catB, SiteID: siteA, Taxonomy: category.TaxonomyCategory, TermID: 14, ParentTermID: 9,
		Name: " Healing ", RunID: "7f7f7f7f-7f7f-4f7f-8f7f-7f7f7f7f7f7f",
		SeenAt: time.Date(2026, time.October, 4, 9, 0, 0, 0, time.UTC),
	}
}

func TestNewTermTrimsTheName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*category.Term)
	}{
		{name: "a term a run created under a parent", mutate: func(*category.Term) {}},
		{name: "a top-level term no run created", mutate: func(term *category.Term) {
			term.ParentTermID = 0
			term.RunID = ""
		}},
		{name: "a product category", mutate: func(term *category.Term) { term.Taxonomy = category.TaxonomyProductCategory }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			in := validTerm()
			tc.mutate(&in)
			got, err := category.NewTerm(in)
			if err != nil {
				t.Fatalf("NewTerm: %v", err)
			}
			want := in
			want.Name = "Healing"
			if got != want {
				t.Fatalf("NewTerm = %+v, want %+v", got, want)
			}
		})
	}
}

func TestNewTermRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*category.Term)
		field  string
	}{
		{name: "no category", mutate: func(term *category.Term) { term.CategoryID = "" }, field: "categoryId"},
		{name: "no site", mutate: func(term *category.Term) { term.SiteID = "" }, field: "siteId"},
		{name: "a taxonomy we do not write", mutate: func(term *category.Term) { term.Taxonomy = "post_tag" }, field: "taxonomy"},
		{name: "no term id", mutate: func(term *category.Term) { term.TermID = 0 }, field: "termId"},
		{name: "a negative term id", mutate: func(term *category.Term) { term.TermID = -3 }, field: "termId"},
		{name: "a negative parent", mutate: func(term *category.Term) { term.ParentTermID = -1 }, field: "parentTermId"},
		{name: "a term under itself", mutate: func(term *category.Term) { term.ParentTermID = term.TermID }, field: "parentTermId"},
		{name: "a blank name", mutate: func(term *category.Term) { term.Name = "  " }, field: "name"},
		{name: "never seen", mutate: func(term *category.Term) { term.SeenAt = time.Time{} }, field: "seenAt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			term := validTerm()
			tc.mutate(&term)
			_, err := category.NewTerm(term)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}
