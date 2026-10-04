package application_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	categorySite = "0b6c2a4e-1f3d-4c8b-9a2e-5d7f8e9a0b1c"
	peptides     = "1a1a1a1a-1a1a-4a1a-8a1a-1a1a1a1a1a1a"
	healing      = "2b2b2b2b-2b2b-4b2b-8b2b-2b2b2b2b2b2b"
	bpc          = "3c3c3c3c-3c3c-4c3c-8c3c-3c3c3c3c3c3c"
	liquid       = "4d4d4d4d-4d4d-4d4d-8d4d-4d4d4d4d4d4d"
	lone         = "5e5e5e5e-5e5e-4e5e-8e5e-5e5e5e5e5e5e"
	unknown      = "6f6f6f6f-6f6f-4f6f-8f6f-6f6f6f6f6f6f"
)

func treeCategory(id, name, parentID string) category.Category {
	return category.Category{ID: id, SiteID: categorySite, Name: name, Key: category.Key(name), ParentID: parentID}
}

func treeTerm(categoryID string, taxonomy category.Taxonomy, termID int64) category.Term {
	return category.Term{
		CategoryID: categoryID, SiteID: categorySite, Taxonomy: taxonomy, TermID: termID, Name: categoryID,
		SeenAt: time.Date(2026, time.October, 4, 9, 0, 0, 0, time.UTC),
	}
}

func treeCategories() []category.Category {
	return []category.Category{
		treeCategory(peptides, "Peptides", ""),
		treeCategory(lone, "Lone", ""),
		treeCategory(healing, "Healing", peptides),
		treeCategory(bpc, "BPC-157", healing),
		treeCategory(liquid, "Liquid", bpc),
	}
}

func treeTerms() []category.Term {
	return []category.Term{
		treeTerm(peptides, category.TaxonomyCategory, 5),
		treeTerm(peptides, category.TaxonomyProductCategory, 31),
		treeTerm(bpc, category.TaxonomyCategory, 6),
		treeTerm(lone, category.TaxonomyCategory, 9),
	}
}

func TestCategoryTreeGivesALeafItsChainWithTheTermsOfOneTaxonomy(t *testing.T) {
	t.Parallel()

	tree := application.NewCategoryTree(treeCategories(), treeTerms())

	cases := []struct {
		name     string
		leafID   string
		taxonomy category.Taxonomy
		want     string
	}{
		{
			name: "a leaf three levels down", leafID: liquid, taxonomy: category.TaxonomyCategory,
			want: `[{"id":"` + peptides + `","name":"Peptides","termId":5},{"id":"` + healing + `","name":"Healing"},` +
				`{"id":"` + bpc + `","name":"BPC-157","termId":6},{"id":"` + liquid + `","name":"Liquid"}]`,
		},
		{
			name: "the same leaf for a product, of which the store has only the root", leafID: liquid, taxonomy: category.TaxonomyProductCategory,
			want: `[{"id":"` + peptides + `","name":"Peptides","termId":31},{"id":"` + healing + `","name":"Healing"},` +
				`{"id":"` + bpc + `","name":"BPC-157"},{"id":"` + liquid + `","name":"Liquid"}]`,
		},
		{
			name: "a category in the middle is the last of its own chain", leafID: healing, taxonomy: category.TaxonomyCategory,
			want: `[{"id":"` + peptides + `","name":"Peptides","termId":5},{"id":"` + healing + `","name":"Healing"}]`,
		},
		{name: "a root alone", leafID: lone, taxonomy: category.TaxonomyCategory, want: `[{"id":"` + lone + `","name":"Lone","termId":9}]`},
		{name: "a category the tree does not hold", leafID: unknown, taxonomy: category.TaxonomyCategory, want: `[]`},
		{name: "no category", taxonomy: category.TaxonomyCategory, want: `[]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tree.Chain(tc.leafID, tc.taxonomy))
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Chain = %s\nwant %s", encoded, tc.want)
			}
		})
	}
}

func TestCategoryTreeNamesEveryCategoryUnderOne(t *testing.T) {
	t.Parallel()

	tree := application.NewCategoryTree(treeCategories(), nil)

	cases := []struct {
		name       string
		tree       application.CategoryTree
		categoryID string
		want       []string
	}{
		{name: "a root with a branch of three", tree: tree, categoryID: peptides, want: []string{peptides, healing, bpc, liquid}},
		{name: "a category in the middle", tree: tree, categoryID: bpc, want: []string{bpc, liquid}},
		{name: "a leaf", tree: tree, categoryID: liquid, want: []string{liquid}},
		{name: "a root with nothing under it", tree: tree, categoryID: lone, want: []string{lone}},
		{name: "a category the tree does not hold", tree: tree, categoryID: unknown, want: []string{unknown}},
		{name: "an empty tree", categoryID: peptides, want: []string{peptides}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.tree.Subtree(tc.categoryID)
			if len(got) == 0 || got[0] != tc.categoryID {
				t.Fatalf("Subtree = %v, want it to open with %s", got, tc.categoryID)
			}
			slices.Sort(got)
			if want := slices.Sorted(slices.Values(tc.want)); !slices.Equal(got, want) {
				t.Fatalf("Subtree = %v, want %v", got, want)
			}
		})
	}
}

func TestCategoryTreeTellsTheTermOfACategoryInOneTaxonomy(t *testing.T) {
	t.Parallel()

	tree := application.NewCategoryTree(treeCategories(), treeTerms())

	cases := []struct {
		name       string
		categoryID string
		taxonomy   category.Taxonomy
		want       int64
		held       bool
	}{
		{name: "a post category", categoryID: peptides, taxonomy: category.TaxonomyCategory, want: 5, held: true},
		{name: "a product category of the same record", categoryID: peptides, taxonomy: category.TaxonomyProductCategory, want: 31, held: true},
		{name: "a taxonomy the store has not made", categoryID: bpc, taxonomy: category.TaxonomyProductCategory},
		{name: "a category the site does not have yet", categoryID: healing, taxonomy: category.TaxonomyCategory},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, held := tree.TermID(tc.categoryID, tc.taxonomy)
			if got != tc.want || held != tc.held {
				t.Fatalf("TermID = %d, %t; want %d, %t", got, held, tc.want, tc.held)
			}
		})
	}
}

func TestCategoryTreeHandsOutTermIDsOfItsOwn(t *testing.T) {
	t.Parallel()

	tree := application.NewCategoryTree(treeCategories(), treeTerms())

	first := tree.Chain(lone, category.TaxonomyCategory)
	*first[0].TermID = 99
	if again := tree.Chain(lone, category.TaxonomyCategory); again[0].TermID == nil || *again[0].TermID != 9 {
		t.Fatalf("writing to one answer changed the next: %+v", again)
	}

	var empty application.CategoryTree
	if got := empty.Chain(lone, category.TaxonomyCategory); got == nil || len(got) != 0 {
		t.Fatalf("an empty tree answered %#v, want an empty list", got)
	}
	if !empty.Empty() || tree.Empty() || !application.NewCategoryTree(nil, treeTerms()).Empty() {
		t.Fatal("a tree is empty exactly when it holds no category")
	}
}

func TestCategoryTreeListsItsCategoriesInTheOrderItWasGiven(t *testing.T) {
	t.Parallel()

	given := treeCategories()
	tree := application.NewCategoryTree(given, nil)
	given[0].Name = "Renamed by the caller"

	listed := tree.Categories()
	if len(listed) != len(treeCategories()) || !slices.Equal(listed, treeCategories()) {
		t.Fatalf("Categories = %+v, want %+v", listed, treeCategories())
	}
	listed[1].Name = "Renamed by a reader"
	if again := tree.Categories(); !slices.Equal(again, treeCategories()) {
		t.Fatalf("writing to one answer changed the next: %+v", again)
	}

	var empty application.CategoryTree
	if got := empty.Categories(); len(got) != 0 {
		t.Fatalf("an empty tree listed %+v", got)
	}
}

type categoryShelf struct {
	listed []category.Category
	err    error
}

func (s categoryShelf) ListBySite(context.Context, string) ([]category.Category, error) {
	return s.listed, s.err
}

type termShelf struct {
	listed []category.Term
	err    error
	asked  *int
}

func (s termShelf) ListBySite(context.Context, string) ([]category.Term, error) {
	*s.asked++
	return s.listed, s.err
}

type categoryReader interface {
	ListBySite(ctx context.Context, siteID string) ([]category.Category, error)
}

type categoryTermReader interface {
	ListBySite(ctx context.Context, siteID string) ([]category.Term, error)
}

func TestLoadCategoryTreeReadsTheTermsOnlyOfASiteWithCategories(t *testing.T) {
	t.Parallel()

	failed := errors.New(errors.Internal, "the database is gone")

	cases := []struct {
		name       string
		categories categoryReader
		terms      []category.Term
		termErr    error
		noTerms    bool
		want       string
		asked      int
		err        error
	}{
		{name: "no category reader", terms: treeTerms(), want: `[]`},
		{name: "a site with no category", categories: categoryShelf{listed: []category.Category{}}, terms: treeTerms(), want: `[]`},
		{
			name: "a site with categories and terms", categories: categoryShelf{listed: treeCategories()}, terms: treeTerms(), asked: 1,
			want: `[{"id":"` + peptides + `","name":"Peptides","termId":5},{"id":"` + healing + `","name":"Healing"}]`,
		},
		{
			name: "no term reader", categories: categoryShelf{listed: treeCategories()}, noTerms: true,
			want: `[{"id":"` + peptides + `","name":"Peptides"},{"id":"` + healing + `","name":"Healing"}]`,
		},
		{name: "the categories cannot be read", categories: categoryShelf{err: failed}, err: failed},
		{name: "the terms cannot be read", categories: categoryShelf{listed: treeCategories()}, termErr: failed, asked: 1, err: failed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			asked := 0
			var terms categoryTermReader
			if !tc.noTerms {
				terms = termShelf{listed: tc.terms, err: tc.termErr, asked: &asked}
			}
			tree, err := application.LoadCategoryTree(t.Context(), categorySite, tc.categories, terms)
			if asked != tc.asked {
				t.Errorf("the terms were read %d times, want %d", asked, tc.asked)
			}
			if tc.err != nil {
				if !stderrors.Is(err, tc.err) {
					t.Fatalf("LoadCategoryTree error = %v, want %v", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadCategoryTree: %v", err)
			}
			encoded, err := json.Marshal(tree.Chain(healing, category.TaxonomyCategory))
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Chain = %s, want %s", encoded, tc.want)
			}
		})
	}
}
