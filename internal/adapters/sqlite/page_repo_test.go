package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func fullPage(siteID, path string, at time.Time) pagemap.Page {
	wpID := int64(42)
	modified := at.Add(-time.Hour)
	pageType := "page-type-guide"
	return pagemap.Page{
		ID: id.New(), SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPost, WPID: &wpID,
		ParentReason: pagemap.ParentJSONLD, ParentMethod: pagemap.MethodRedirect, PageTypeID: &pageType, TypeSource: pagemap.TypeSourceAuto, TypeReason: pagemap.TypeByKind,
		TechnicalReason: pagemap.TechnicalShop, ReadOnlyReason: pagemap.ReadOnlyBuilder("elementor"),
		Title: "Title", H1: "Heading", MetaTitle: "Meta", MetaDescription: "Description",
		Keywords: keyword.New([]keyword.Keyword{{Text: "title keyword", Volume: new(5400)}, {Text: "one", Volume: new(0)}, {Text: "two"}}),
		Notes:    []note.Note{{Label: "Intent Owner", Text: "GEO Commercial"}, {Label: "Notes", Text: "the liquid form"}},
		Status:   pagemap.StatusExists, WPModifiedAt: &modified, LastSyncedAt: &at, CreatedAt: at, UpdatedAt: at,
	}
}

func pagePaths(pages []pagemap.Page) []string {
	out := make([]string, 0, len(pages))
	for i := range pages {
		out = append(out, pages[i].Path)
	}
	return out
}

func TestAPageTypeHasItsOwnWriters(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewPageRepo(store)
	page := sqlitetest.Page(t, store, owner.ID, "/peptides/")
	at := sqlitetest.Stamp.Add(time.Minute)
	typeOf := func() (string, pagemap.TypeSource, pagemap.TypeReason) {
		t.Helper()
		got, err := repo.Get(t.Context(), page.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		typeID := ""
		if got.PageTypeID != nil {
			typeID = *got.PageTypeID
		}
		return typeID, got.TypeSource, got.TypeReason
	}
	expect := func(step, typeID string, source pagemap.TypeSource, reason pagemap.TypeReason) {
		t.Helper()
		if gotID, gotSource, gotReason := typeOf(); gotID != typeID || gotSource != source || gotReason != reason {
			t.Fatalf("%s: type = %q %s %s, want %q %s %s", step, gotID, gotSource, gotReason, typeID, source, reason)
		}
	}

	snapshot, err := repo.Get(t.Context(), page.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	stale := "stale-type"
	snapshot.PageTypeID, snapshot.TypeSource, snapshot.TypeReason = &stale, pagemap.TypeSourceAuto, pagemap.TypeByDefault
	if err = repo.Update(t.Context(), snapshot); err != nil {
		t.Fatalf("Update: %v", err)
	}
	expect("a whole-row update", "", pagemap.TypeSourceNone, pagemap.TypeByNone)

	if err = repo.AutoType(t.Context(), page.ID, "hub", pagemap.TypeByListing, at); err != nil {
		t.Fatalf("AutoType: %v", err)
	}
	expect("the first automatic type", "hub", pagemap.TypeSourceAuto, pagemap.TypeByListing)
	if err = repo.AutoType(t.Context(), page.ID, "entity", pagemap.TypeByDefault, at); err != nil {
		t.Fatalf("AutoType again: %v", err)
	}
	expect("a second automatic type", "hub", pagemap.TypeSourceAuto, pagemap.TypeByListing)

	if err = repo.SetType(t.Context(), page.ID, "pillar", pagemap.TypeByUser, at); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	expect("a person's type", "pillar", pagemap.TypeSourceManual, pagemap.TypeByUser)
	if err = repo.AutoType(t.Context(), page.ID, "entity", pagemap.TypeByDefault, at); err != nil {
		t.Fatalf("AutoType over a manual type: %v", err)
	}
	expect("an automatic type over a person's", "pillar", pagemap.TypeSourceManual, pagemap.TypeByUser)
	if err = repo.Update(t.Context(), snapshot); err != nil {
		t.Fatalf("Update from an old snapshot: %v", err)
	}
	expect("an old snapshot written back", "pillar", pagemap.TypeSourceManual, pagemap.TypeByUser)

	moved, err := repo.ReassignType(t.Context(), "pillar", "hub", pagemap.TypeByUser, at)
	if err != nil || moved != 1 {
		t.Fatalf("ReassignType = %d %v, want one page moved", moved, err)
	}
	expect("a reassignment", "hub", pagemap.TypeSourceManual, pagemap.TypeByUser)

	if err = repo.SetType(t.Context(), "no-such-page", "hub", pagemap.TypeByAI, at); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("SetType on a missing page = %q, want NOT_FOUND", errors.CodeOf(err))
	}
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
	want.PlannedPath = "/sneakers/running/"
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
	want.ParentReason = pagemap.ParentWorkbook
	want.ParentMethod = pagemap.MethodNone
	want.TechnicalReason = pagemap.TechnicalNone
	want.ReadOnlyReason = pagemap.ReadOnlyTerm
	want.LastSyncedAt = nil
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

func TestPageRepoUpdatePublishedTouchesOnlyWhatAPublishOwns(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewPageRepo(store)

	read := fullPage(owner.ID, "/bpc-157/", sqlitetest.Stamp)
	if err := repo.Insert(t.Context(), read); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	edited := read
	edited.Keywords = keyword.New([]keyword.Keyword{{Text: "edited while the run wrote"}})
	edited.Title = "Edited by a person"
	edited.Notes = []note.Note{{Label: "Notes", Text: "a person's note, kept"}}
	edited.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err := repo.Update(t.Context(), edited); err != nil {
		t.Fatalf("Update: %v", err)
	}

	wpID := int64(4242)
	synced := sqlitetest.Stamp.Add(3 * time.Minute)
	published := read
	published.WPID = &wpID
	published.Status = pagemap.StatusPublished
	published.LastSyncedAt = &synced
	published.WPModifiedAt = &synced
	published.UpdatedAt = sqlitetest.Stamp.Add(2 * time.Minute)
	if err := repo.UpdatePublished(t.Context(), published); err != nil {
		t.Fatalf("UpdatePublished: %v", err)
	}

	got, err := repo.Get(t.Context(), read.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := edited
	want.WPID, want.Status = &wpID, pagemap.StatusPublished
	want.LastSyncedAt, want.WPModifiedAt, want.UpdatedAt = &synced, &synced, published.UpdatedAt
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	missing := read
	missing.ID = id.New()
	if err = repo.UpdatePublished(t.Context(), missing); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("UpdatePublished of an unknown page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestPageRepoUpdateDerivedTouchesOnlyTheDerivedColumns(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewPageRepo(store)

	parent := fullPage(owner.ID, "/peptides/", sqlitetest.Stamp)
	if err := repo.Insert(t.Context(), parent); err != nil {
		t.Fatalf("Insert parent: %v", err)
	}
	read := fullPage(owner.ID, "/bpc-157/", sqlitetest.Stamp)
	if err := repo.Insert(t.Context(), read); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	edited := read
	edited.Keywords = keyword.New([]keyword.Keyword{{Text: "edited while the walk ran"}})
	edited.Title = "Edited"
	edited.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err := repo.Update(t.Context(), edited); err != nil {
		t.Fatalf("Update: %v", err)
	}

	derived := read
	derived.ParentPageID = &parent.ID
	derived.ParentReason = pagemap.ParentARIA
	derived.ParentMethod = pagemap.MethodName
	derived.TechnicalReason = pagemap.TechnicalNone
	derived.ReadOnlyReason = pagemap.ReadOnlyNone
	derived.Status = pagemap.StatusArchived
	derived.UpdatedAt = sqlitetest.Stamp.Add(2 * time.Minute)
	if err := repo.UpdateDerived(t.Context(), derived); err != nil {
		t.Fatalf("UpdateDerived: %v", err)
	}

	got, err := repo.Get(t.Context(), read.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := edited
	want.ParentPageID = &parent.ID
	want.ParentReason = pagemap.ParentARIA
	want.ParentMethod = pagemap.MethodName
	want.TechnicalReason = pagemap.TechnicalNone
	want.ReadOnlyReason = pagemap.ReadOnlyNone
	want.Status = pagemap.StatusArchived
	want.UpdatedAt = derived.UpdatedAt
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %+v\nwant %+v", got, want)
	}

	missing := read
	missing.ID = id.New()
	if err = repo.UpdateDerived(t.Context(), missing); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("UpdateDerived of an unknown page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestPageRepoReadsNoParentReasonAsNone(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewPageRepo(store)

	for _, reason := range []pagemap.ParentReason{"", pagemap.ParentNone} {
		record := fullPage(owner.ID, "/reason-"+string(reason)+"/", sqlitetest.Stamp)
		record.ParentReason = reason
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("Insert with the reason %q: %v", reason, err)
		}
		got, err := repo.Get(t.Context(), record.ID)
		if err != nil || got.ParentReason != pagemap.ParentNone {
			t.Errorf("the reason %q reads back as %q, %v; want %q", reason, got.ParentReason, err, pagemap.ParentNone)
		}
		record.ParentReason = pagemap.ParentURL
		if err = repo.Update(t.Context(), record); err != nil {
			t.Fatalf("Update to a reason: %v", err)
		}
		record.ParentReason = reason
		if err = repo.Update(t.Context(), record); err != nil {
			t.Fatalf("Update back to the reason %q: %v", reason, err)
		}
		if got, err = repo.Get(t.Context(), record.ID); err != nil || got.ParentReason != pagemap.ParentNone {
			t.Errorf("the reason %q updated reads back as %q, %v; want %q", reason, got.ParentReason, err, pagemap.ParentNone)
		}
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
	byEntity, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, EntityIDs: []string{entity.ID}, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(byEntity.Items) != 1 || byEntity.Items[0].Path != "/shop/shoes/" {
		t.Errorf("by entity = %+v, %v", byEntity.Items, err)
	}
	none, err := repo.List(t.Context(), pagemap.Query{SiteID: owner.ID, EntityIDs: []string{}, Sort: pagemap.SortPath}, paging.Request{Limit: 10})
	if err != nil || len(none.Items) != 0 {
		t.Errorf("an empty entity list = %+v, %v, want no page", none.Items, err)
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
