package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func storedTerm(entity graph.Entity, taxonomy graph.Taxonomy, termID, parentTermID int64) graph.Term {
	return graph.Term{
		EntityID: entity.ID, SiteID: entity.SiteID, Taxonomy: taxonomy, TermID: termID, ParentTermID: parentTermID,
		Name: entity.Name, SeenAt: sqlitetest.Stamp,
	}
}

func TestTermRepoUpsertsListsAndDeletes(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	healing := sqlitetest.Entity(t, store, owner.ID, "Healing")
	bpc := sqlitetest.Entity(t, store, owner.ID, "BPC-157")
	elsewhere := sqlitetest.Entity(t, store, other.ID, "Healing")
	repo := sqlite.NewTermRepo(store)

	created := storedTerm(healing, graph.TaxonomyCategory, 5, 0)
	created.RunID = id.New()
	product := storedTerm(healing, graph.TaxonomyProductCategory, 31, 0)
	child := storedTerm(bpc, graph.TaxonomyCategory, 6, 5)
	for _, term := range []graph.Term{child, product, created, storedTerm(elsewhere, graph.TaxonomyCategory, 5, 0)} {
		if err := repo.Upsert(t.Context(), term); err != nil {
			t.Fatalf("Upsert %s %s: %v", term.Name, term.Taxonomy, err)
		}
	}

	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if want := []graph.Term{created, child, product}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListBySite = %+v\nwant %+v", listed, want)
	}

	moved := child
	moved.TermID = 8
	moved.ParentTermID = 0
	moved.Name = "BPC 157"
	moved.SeenAt = sqlitetest.Stamp.Add(time.Hour)
	if err = repo.Upsert(t.Context(), moved); err != nil {
		t.Fatalf("Upsert the moved term: %v", err)
	}
	if listed, err = repo.ListBySite(t.Context(), owner.ID); err != nil {
		t.Fatalf("ListBySite after the move: %v", err)
	}
	if want := []graph.Term{created, moved, product}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListBySite after the move = %+v\nwant %+v", listed, want)
	}

	if err = repo.Delete(t.Context(), healing.ID, graph.TaxonomyProductCategory); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), healing.ID, graph.TaxonomyProductCategory); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Delete twice = %v, want NOT_FOUND", err)
	}
	if listed, err = repo.ListBySite(t.Context(), owner.ID); err != nil || !reflect.DeepEqual(listed, []graph.Term{created, moved}) {
		t.Fatalf("ListBySite after the delete = %+v, %v", listed, err)
	}

	if listed, err = repo.ListBySite(t.Context(), other.ID); err != nil || len(listed) != 1 || listed[0].EntityID != elsewhere.ID {
		t.Fatalf("the other site's terms = %+v, %v", listed, err)
	}
	if listed, err = repo.ListBySite(t.Context(), id.New()); err != nil || listed == nil || len(listed) != 0 {
		t.Fatalf("a site with no terms = %#v, %v; want an empty list", listed, err)
	}
}

func TestTermRepoRefusesWhatTheSchemaRefuses(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	healing := sqlitetest.Entity(t, store, owner.ID, "Healing")
	repo := sqlite.NewTermRepo(store)

	cases := []struct {
		name   string
		mutate func(*graph.Term)
	}{
		{name: "an entity that does not exist", mutate: func(term *graph.Term) { term.EntityID = id.New() }},
		{name: "a site that does not exist", mutate: func(term *graph.Term) { term.SiteID = id.New() }},
		{name: "a taxonomy Postulator does not write", mutate: func(term *graph.Term) { term.Taxonomy = "post_tag" }},
		{name: "a term id of zero", mutate: func(term *graph.Term) { term.TermID = 0 }},
		{name: "a negative parent", mutate: func(term *graph.Term) { term.ParentTermID = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			term := storedTerm(healing, graph.TaxonomyCategory, 5, 0)
			tc.mutate(&term)
			if err := repo.Upsert(t.Context(), term); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Upsert = %v, want INVALID", err)
			}
		})
	}

	if listed, err := repo.ListBySite(t.Context(), owner.ID); err != nil || len(listed) != 0 {
		t.Fatalf("refused terms were stored: %+v, %v", listed, err)
	}
}
