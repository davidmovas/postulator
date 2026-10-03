package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func storedCategoryTerm(record category.Category, taxonomy category.Taxonomy, termID, parentTermID int64) category.Term {
	return category.Term{
		CategoryID: record.ID, SiteID: record.SiteID, Taxonomy: taxonomy, TermID: termID, ParentTermID: parentTermID,
		Name: record.Name, SeenAt: sqlitetest.Stamp,
	}
}

func TestCategoryTermRepoUpsertsListsAndDeletes(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	categories := sqlite.NewCategoryRepo(store)
	healing := newCategory(t, owner.ID, "Healing", "")
	bpc := newCategory(t, owner.ID, "BPC-157", healing.ID)
	elsewhere := newCategory(t, other.ID, "Healing", "")
	insertCategories(t, categories, healing, bpc, elsewhere)
	repo := sqlite.NewCategoryTermRepo(store)

	created := storedCategoryTerm(healing, category.TaxonomyCategory, 5, 0)
	created.RunID = id.New()
	product := storedCategoryTerm(healing, category.TaxonomyProductCategory, 31, 0)
	child := storedCategoryTerm(bpc, category.TaxonomyCategory, 6, 5)
	for _, term := range []category.Term{child, product, created, storedCategoryTerm(elsewhere, category.TaxonomyCategory, 5, 0)} {
		if err := repo.Upsert(t.Context(), term); err != nil {
			t.Fatalf("Upsert %s %s: %v", term.Name, term.Taxonomy, err)
		}
	}

	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if want := []category.Term{created, child, product}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListBySite = %+v\nwant %+v", listed, want)
	}

	moved := child
	moved.TermID = 8
	moved.ParentTermID = 0
	moved.Name = "BPC 157"
	moved.RunID = id.New()
	moved.SeenAt = sqlitetest.Stamp.Add(time.Hour)
	if err = repo.Upsert(t.Context(), moved); err != nil {
		t.Fatalf("Upsert the moved term: %v", err)
	}
	if listed, err = repo.ListBySite(t.Context(), owner.ID); err != nil {
		t.Fatalf("ListBySite after the move: %v", err)
	}
	if want := []category.Term{created, moved, product}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListBySite after the move = %+v\nwant %+v", listed, want)
	}

	if err = repo.Delete(t.Context(), healing.ID, category.TaxonomyProductCategory); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), healing.ID, category.TaxonomyProductCategory); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Delete twice = %v, want NOT_FOUND", err)
	}
	if listed, err = repo.ListBySite(t.Context(), owner.ID); err != nil || !reflect.DeepEqual(listed, []category.Term{created, moved}) {
		t.Fatalf("ListBySite after the delete = %+v, %v", listed, err)
	}

	if listed, err = repo.ListBySite(t.Context(), other.ID); err != nil || len(listed) != 1 || listed[0].CategoryID != elsewhere.ID {
		t.Fatalf("the other site's terms = %+v, %v", listed, err)
	}
	if listed, err = repo.ListBySite(t.Context(), id.New()); err != nil || listed == nil || len(listed) != 0 {
		t.Fatalf("a site with no terms = %#v, %v; want an empty list", listed, err)
	}
}

func TestCategoryTermRepoRefusesWhatTheSchemaRefuses(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	healing := newCategory(t, owner.ID, "Healing", "")
	foreign := newCategory(t, other.ID, "Foreign", "")
	insertCategories(t, sqlite.NewCategoryRepo(store), healing, foreign)
	repo := sqlite.NewCategoryTermRepo(store)

	cases := []struct {
		name   string
		mutate func(*category.Term)
	}{
		{name: "a category that does not exist", mutate: func(term *category.Term) { term.CategoryID = id.New() }},
		{name: "a site that does not exist", mutate: func(term *category.Term) { term.SiteID = id.New() }},
		{name: "a category of another site", mutate: func(term *category.Term) { term.CategoryID = foreign.ID }},
		{name: "a taxonomy Postulator does not write", mutate: func(term *category.Term) { term.Taxonomy = "post_tag" }},
		{name: "a term id of zero", mutate: func(term *category.Term) { term.TermID = 0 }},
		{name: "a negative parent", mutate: func(term *category.Term) { term.ParentTermID = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			term := storedCategoryTerm(healing, category.TaxonomyCategory, 5, 0)
			tc.mutate(&term)
			if err := repo.Upsert(t.Context(), term); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Upsert = %v, want INVALID", err)
			}
		})
	}

	for _, siteID := range []string{owner.ID, other.ID} {
		if listed, err := repo.ListBySite(t.Context(), siteID); err != nil || len(listed) != 0 {
			t.Fatalf("refused terms were stored: %+v, %v", listed, err)
		}
	}
}
