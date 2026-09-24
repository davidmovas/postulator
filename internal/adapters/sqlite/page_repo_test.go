package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func fullPage(siteID, path string, at time.Time) pagemap.Page {
	wpID := int64(42)
	modified := at.Add(-time.Hour)
	return pagemap.Page{
		ID: id.New(), SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPost, WPID: &wpID,
		Title: "Title", H1: "Heading", MetaTitle: "Meta", MetaDescription: "Description", Canonical: "https://shop.example.com" + path,
		PrimaryKeyword: "title keyword", Keywords: []string{"one", "two"},
		Status: pagemap.StatusExists, ContentHash: "abc", WPModifiedAt: &modified, LastSyncedAt: &at, Drift: true, CreatedAt: at, UpdatedAt: at,
		Observed: pagemap.Observed{
			Link: "https://shop.example.com" + path, Slug: pagemap.Slug(path), Status: "draft",
			Title: "Title", H1: "Heading",
		},
	}
}

func pagePaths(pages []pagemap.Page) []string {
	out := make([]string, 0, len(pages))
	for i := range pages {
		out = append(out, pages[i].Path)
	}
	return out
}

func TestPageRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	entity := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	repo := sqlite.NewPageRepo(store)

	parent := fullPage(owner.ID, "/shop/", sqlitetest.Stamp)
	if err := repo.Insert(t.Context(), parent); err != nil {
		t.Fatalf("Insert parent: %v", err)
	}
	want := fullPage(owner.ID, "/shop/shoes/", sqlitetest.Stamp)
	want.ParentPageID = &parent.ID
	want.EntityID = &entity.ID
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

	if err = repo.Insert(t.Context(), fullPage(owner.ID, "/shop/shoes/", sqlitetest.Stamp)); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate path code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if err = repo.Insert(t.Context(), fullPage("no-such-site", "/x/", sqlitetest.Stamp)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown site code = %q, want INVALID", errors.CodeOf(err))
	}
	stranger := fullPage(owner.ID, "/y/", sqlitetest.Stamp)
	missing := "no-such-entity"
	stranger.EntityID = &missing
	if err = repo.Insert(t.Context(), stranger); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown entity code = %q, want INVALID", errors.CodeOf(err))
	}

	want.Title = "Renamed"
	want.Status = pagemap.StatusPublished
	want.WPID = nil
	want.LastSyncedAt = nil
	want.Drift = false
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

	if err = repo.Delete(t.Context(), parent.ID); err != nil {
		t.Fatalf("Delete parent: %v", err)
	}
	got, err = repo.Get(t.Context(), want.ID)
	if err != nil || got.ParentPageID != nil {
		t.Errorf("child after parent delete = %+v, %v; want a NULL parent", got, err)
	}
	if err = repo.Delete(t.Context(), parent.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Update(t.Context(), parent); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Update missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = repo.Get(t.Context(), parent.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestPageRepoListAndFilters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "blog")
	entity := sqlitetest.Entity(t, store, owner.ID, "Shoes")
	repo := sqlite.NewPageRepo(store)

	paths := []string{"/shop/", "/blog/", "/shop/shoes/", "/about/", "/shop/bags/"}
	for i, path := range paths {
		record := fullPage(owner.ID, path, sqlitetest.Stamp.Add(time.Duration(i)*time.Minute))
		if i%2 == 0 {
			record.Status = pagemap.StatusPlanned
		}
		if path == "/shop/shoes/" {
			record.EntityID = &entity.ID
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert %s: %v", path, err)
		}
	}
	if err := repo.Insert(t.Context(), fullPage(other.ID, "/shop/", sqlitetest.Stamp)); err != nil {
		t.Fatalf("Insert on the other site: %v", err)
	}

	bySite, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListBySite: %v", err)
	}
	if got := pagePaths(bySite); !reflect.DeepEqual(got, []string{"/about/", "/blog/", "/shop/", "/shop/bags/", "/shop/shoes/"}) {
		t.Errorf("ListBySite = %v", got)
	}

	byPath := pagemap.Query{SiteID: owner.ID, Sort: pagemap.SortPath}
	first, err := repo.List(t.Context(), byPath, paging.Request{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := pagePaths(first.Items); !reflect.DeepEqual(got, []string{"/about/", "/blog/"}) || !first.HasMore {
		t.Fatalf("first page = %v", got)
	}
	second, err := repo.List(t.Context(), byPath, paging.Request{After: first.Next, Limit: 2})
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if got := pagePaths(second.Items); !reflect.DeepEqual(got, []string{"/shop/", "/shop/bags/"}) {
		t.Fatalf("second page = %v", got)
	}
	back, err := repo.List(t.Context(), byPath, paging.Request{Before: second.Prev, Limit: 2})
	if err != nil {
		t.Fatalf("List before: %v", err)
	}
	if got := pagePaths(back.Items); !reflect.DeepEqual(got, []string{"/about/", "/blog/"}) {
		t.Fatalf("backward page = %v", got)
	}

	newest, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, Sort: pagemap.SortCreatedAt, Desc: true}, paging.Request{Limit: 1})
	if err != nil || len(newest.Items) != 1 || newest.Items[0].Path != "/shop/bags/" {
		t.Errorf("newest = %+v, %v", newest.Items, err)
	}

	planned := pagemap.StatusPlanned
	byStatus, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, Status: &planned, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List by status: %v", err)
	}
	if got := pagePaths(byStatus.Items); !reflect.DeepEqual(got, []string{"/shop/", "/shop/bags/", "/shop/shoes/"}) {
		t.Errorf("planned = %v", got)
	}
	byEntity, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, EntityID: &entity.ID, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(byEntity.Items) != 1 || byEntity.Items[0].Path != "/shop/shoes/" {
		t.Errorf("by entity = %+v, %v", byEntity.Items, err)
	}
	unmapped, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, Unmapped: true, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(unmapped.Items) != 4 {
		t.Errorf("unmapped = %d, %v", len(unmapped.Items), err)
	}
	prefixed, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, PathPrefix: "/shop/", Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List by prefix: %v", err)
	}
	if got := pagePaths(prefixed.Items); !reflect.DeepEqual(got, []string{"/shop/", "/shop/bags/", "/shop/shoes/"}) {
		t.Errorf("prefixed = %v", got)
	}
	wild, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, PathPrefix: "/sh_p/", Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(wild.Items) != 0 {
		t.Errorf("an underscore must be escaped, got %d, %v", len(wild.Items), err)
	}
}

func TestPageFixture(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	fixture := sqlitetest.Page(t, store, owner.ID, "/shop/")
	got, err := sqlite.NewPageRepo(store).Get(t.Context(), fixture.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !reflect.DeepEqual(got, fixture) {
		t.Errorf("fixture round trip = %+v\nwant %+v", got, fixture)
	}
}
