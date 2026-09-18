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
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func fullEntity(siteID, name string, at time.Time) graph.Entity {
	return graph.Entity{
		ID: id.New(), SiteID: siteID, Name: name, Kind: graph.KindTopic, Intent: "informational", PrimaryKeyword: name + " keyword",
		SecondaryKeywords: []string{name + " one", name + " two"},
		Anchors:           []graph.Anchor{{Text: name, Source: graph.AnchorUser, Weight: 1}, {Text: "best " + name, Source: graph.AnchorAI, Weight: 0.4}},
		Score:             0.5, Source: graph.SourceImport, CreatedAt: at, UpdatedAt: at,
	}
}

func entityNames(list []graph.Entity) []string {
	out := make([]string, 0, len(list))
	for i := range list {
		out = append(out, list[i].Name)
	}
	return out
}

func TestEntityRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewEntityRepo(store)
	want := fullEntity(owner.ID, "Shoes", sqlitetest.Stamp)

	if err := repo.Insert(t.Context(), want); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	got, err := repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	duplicate := fullEntity(owner.ID, "shoes", sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), duplicate); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("case-insensitive duplicate code = %q, want CONFLICT", errors.CodeOf(err))
	}
	orphan := fullEntity("no-such-site", "Orphan", sqlitetest.Stamp)
	if err = repo.Insert(t.Context(), orphan); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown site code = %q, want INVALID", errors.CodeOf(err))
	}

	want.Name = "Running Shoes"
	want.Anchors = []graph.Anchor{{Text: "running shoes", Source: graph.AnchorUser, Weight: 0.9}}
	want.SecondaryKeywords = []string{}
	want.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = repo.Update(t.Context(), want); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v\nwant %+v", got, want)
	}

	if err = repo.SetScore(t.Context(), want.ID, 0.25); err != nil {
		t.Fatalf("SetScore: %v", err)
	}
	pageID := "not-a-page"
	if err = repo.SetCanonicalPage(t.Context(), want.ID, &pageID, want.UpdatedAt); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("canonical to an unknown page code = %q, want INVALID", errors.CodeOf(err))
	}
	if err = repo.SetCanonicalPage(t.Context(), want.ID, nil, want.UpdatedAt); err != nil {
		t.Fatalf("SetCanonicalPage nil: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil {
		t.Fatalf("Get after score: %v", err)
	}
	if got.Score != 0.25 || got.CanonicalPageID != nil {
		t.Errorf("score = %v canonical = %v", got.Score, got.CanonicalPageID)
	}

	if err = repo.SetScore(t.Context(), "missing", 1); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("SetScore unknown code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Delete(t.Context(), want.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err = repo.Delete(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), want); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), want.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestEntityRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	repo := sqlite.NewEntityRepo(store)

	names := []string{"Boots", "sandals", "Shoes", "slippers", "Trainers"}
	for i, name := range names {
		record := fullEntity(owner.ID, name, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))
		if i%2 == 1 {
			record.Kind = graph.KindProduct
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", name, err)
		}
	}
	if err := repo.Insert(t.Context(), fullEntity(other.ID, "Shoes", sqlitetest.Stamp)); err != nil {
		t.Fatalf("Insert on the other site: %v", err)
	}

	bySite, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if got := entityNames(bySite); !reflect.DeepEqual(got, []string{"Boots", "sandals", "Shoes", "slippers", "Trainers"}) {
		t.Errorf("ListBySite = %v", got)
	}
	for i := range bySite {
		if len(bySite[i].Anchors) != 2 {
			t.Errorf("%s carries %d anchors, want 2", bySite[i].Name, len(bySite[i].Anchors))
		}
	}

	byCreation := graph.EntityQuery{SiteID: owner.ID, Sort: graph.EntitySortCreatedAt}
	first, err := repo.List(t.Context(), byCreation, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := entityNames(first.Items); !reflect.DeepEqual(got, []string{"Boots", "sandals"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	if len(first.Items[0].Anchors) != 2 {
		t.Errorf("list pages must carry anchors, got %d", len(first.Items[0].Anchors))
	}
	second, err := repo.List(t.Context(), byCreation, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := entityNames(second.Items); !reflect.DeepEqual(got, []string{"Shoes", "slippers"}) || !second.HasMore {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), byCreation, paging.Request{Before: second.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := entityNames(back.Items); !reflect.DeepEqual(got, []string{"Boots", "sandals"}) {
		t.Fatalf("backward page = %v", got)
	}

	product := graph.KindProduct
	products, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, Kind: &product, Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List products: %v", err)
	}
	if got := entityNames(products.Items); !reflect.DeepEqual(got, []string{"sandals", "slippers"}) {
		t.Errorf("products = %v", got)
	}

	prefixed, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, NamePrefix: "s", Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List prefixed: %v", err)
	}
	if got := entityNames(prefixed.Items); !reflect.DeepEqual(got, []string{"sandals", "Shoes", "slippers"}) {
		t.Errorf("prefix search must be case-insensitive and name-ordered, got %v", got)
	}
	wild, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, NamePrefix: "%", Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(wild.Items) != 0 {
		t.Errorf("a literal percent must be escaped and match nothing, got %d items, %v", len(wild.Items), err)
	}

	everySite, err := repo.List(t.Context(), graph.EntityQuery{NamePrefix: "sho", Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(everySite.Items) != 2 {
		t.Errorf("an empty site id spans every site, got %d, %v", len(everySite.Items), err)
	}

	withCanonical := true
	none, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, HasCanonicalPage: &withCanonical, Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(none.Items) != 0 {
		t.Errorf("no entity has a canonical page yet, got %d, %v", len(none.Items), err)
	}
	withoutCanonical := false
	all, err := repo.List(t.Context(), graph.EntityQuery{SiteID: owner.ID, HasCanonicalPage: &withoutCanonical, Sort: graph.EntitySortName}, paging.Request{Limit: 10})
	if err != nil || len(all.Items) != 5 {
		t.Errorf("every entity lacks a canonical page, got %d, %v", len(all.Items), err)
	}
}

func TestEntityFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	fixture := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	got, err := sqlite.NewEntityRepo(store).Get(t.Context(), fixture.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v\nwant %+v", got, fixture)
	}
}
