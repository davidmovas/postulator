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
	"github.com/davidmovas/postulator/internal/domain/pagetype"
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

	byRule := pagetype.Assignment{TypeID: "hub", Source: pagemap.TypeSourceAuto, Reason: pagemap.TypeByRule, RuleID: "rule-6"}
	if _, err = repo.Classify(t.Context(), page.ID, pagetype.Assignment{}, byRule, at); err != nil {
		t.Fatalf("Classify: %v", err)
	}
	expect("the first automatic type", "hub", pagemap.TypeSourceAuto, pagemap.TypeByRule)
	unmatched := pagetype.Assignment{TypeID: "entity", Source: pagemap.TypeSourceAuto, Reason: pagemap.TypeByUnmatched}
	if _, err = repo.Classify(t.Context(), page.ID, pagetype.Assignment{}, unmatched, at); err != nil {
		t.Fatalf("Classify again: %v", err)
	}
	expect("a second automatic type read before the first", "hub", pagemap.TypeSourceAuto, pagemap.TypeByRule)

	if err = repo.SetType(t.Context(), page.ID, "pillar", pagemap.TypeByUser, at); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	expect("a person's type", "pillar", pagemap.TypeSourceManual, pagemap.TypeByUser)
	if _, err = repo.Classify(t.Context(), page.ID, byRule, unmatched, at); err != nil {
		t.Fatalf("Classify over a manual type: %v", err)
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
	want.PlannedTrail = []string{"Shop", "Footwear"}
	want.RowKey = "row:Groups:running shoes"
	want.ParentSource = pagemap.ParentFromWorkbook
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

	pinned := want.ParentSource
	want.ParentSource = pagemap.ParentFromSite
	want.PlannedTrail = nil
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
	want.ParentSource = pinned
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get after update = %+v\nwant %+v, the parent source a person set kept by a whole-row update", got, want)
	}
	if err = repo.SetParentSource(t.Context(), want.ID, pagemap.ParentFromSite, sqlitetest.Stamp.Add(2*time.Minute)); err != nil {
		t.Fatalf("SetParentSource: %v", err)
	}
	if got, err = repo.Get(t.Context(), want.ID); err != nil || got.ParentSource != pagemap.ParentFromSite {
		t.Errorf("parent source after SetParentSource = %q, %v; want the site's", got.ParentSource, err)
	}
	if err = repo.SetParentSource(t.Context(), want.ID, "elsewhere", sqlitetest.Stamp); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown parent source code = %q, want INVALID", errors.CodeOf(err))
	}
	want.ParentSource = pagemap.ParentFromSite

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

func TestPageRepoSetRowKeyMovesTheKeyOntoOnePage(t *testing.T) {
	t.Parallel()

	const key = "row:Catalog:bpc-157 guide"
	later := sqlitetest.Stamp.Add(time.Hour)
	cases := []struct {
		name    string
		trailed []string
		given   []string
		want    []string
	}{
		{name: "a page without a trail takes the one it is given", given: []string{"Peptides", "BPC-157"}, want: []string{"Peptides", "BPC-157"}},
		{name: "a page keeps the trail it carries", trailed: []string{"Guides"}, given: []string{"Peptides"}, want: []string{"Guides"}},
		{name: "no trail given leaves none", want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := sqlitetest.Open(t)
			owner := sqlitetest.Site(t, store, "shop")
			other := sqlitetest.Site(t, store, "other")
			repo := sqlite.NewPageRepo(store)
			held := fullPage(owner.ID, "/bpc-157-guide/", sqlitetest.Stamp)
			held.WPID, held.Status, held.RowKey = nil, pagemap.StatusPlanned, key
			chosen := fullPage(owner.ID, "/guides/bpc-157/", sqlitetest.Stamp)
			chosen.PlannedTrail = tc.trailed
			elsewhere := fullPage(other.ID, "/bpc-157-guide/", sqlitetest.Stamp)
			elsewhere.RowKey = key
			for _, page := range []pagemap.Page{held, chosen, elsewhere} {
				if err := repo.Insert(t.Context(), page); err != nil {
					t.Fatalf("Insert %s: %v", page.Path, err)
				}
			}

			if err := repo.SetRowKey(t.Context(), owner.ID, chosen.ID, key, tc.given, later); err != nil {
				t.Fatalf("SetRowKey: %v", err)
			}
			want := chosen
			want.RowKey, want.PlannedTrail, want.UpdatedAt = key, tc.want, later
			if got, err := repo.Get(t.Context(), chosen.ID); err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("the chosen page = %+v, %v\nwant %+v", got, err, want)
			}
			released := held
			released.RowKey, released.UpdatedAt = "", later
			if got, err := repo.Get(t.Context(), held.ID); err != nil || !reflect.DeepEqual(got, released) {
				t.Fatalf("the page that held the key = %+v, %v\nwant %+v", got, err, released)
			}
			if got, err := repo.Get(t.Context(), elsewhere.ID); err != nil || !reflect.DeepEqual(got, elsewhere) {
				t.Fatalf("another site's page = %+v, %v; want it untouched", got, err)
			}
		})
	}
}

func TestPageRepoSetRowKeyRefusesWhatItCannotMove(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	repo := sqlite.NewPageRepo(store)
	held := fullPage(owner.ID, "/guide/", sqlitetest.Stamp)
	held.RowKey = "row::guide"
	stranger := fullPage(other.ID, "/guide/", sqlitetest.Stamp)
	for _, page := range []pagemap.Page{held, stranger} {
		if err := repo.Insert(t.Context(), page); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	cases := []struct {
		name   string
		pageID string
		key    string
		code   errors.Code
	}{
		{name: "an empty key", pageID: held.ID, key: " ", code: errors.Invalid},
		{name: "a page of another site", pageID: stranger.ID, key: "row::other", code: errors.NotFound},
		{name: "an unknown page", pageID: id.New(), key: "row::other", code: errors.NotFound},
	}
	for _, tc := range cases {
		if err := repo.SetRowKey(t.Context(), owner.ID, tc.pageID, tc.key, nil, sqlitetest.Stamp); !errors.IsCode(err, tc.code) {
			t.Errorf("%s: SetRowKey = %v, want %s", tc.name, err, tc.code)
		}
	}
	if got, err := repo.Get(t.Context(), held.ID); err != nil || got.RowKey != "row::guide" {
		t.Fatalf("the page = %+v, %v; want its key kept after the refusals", got, err)
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

func TestPageRepoFollowsTheObservedH1OnlyWhereNoPlanHoldsIt(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewPageRepo(store)

	mirrored := fullPage(owner.ID, "/bpc-157/", sqlitetest.Stamp)
	mirrored.H1 = ""
	planned := fullPage(owner.ID, "/tb-500/", sqlitetest.Stamp)
	planned.H1 = "TB-500 Research Peptide"
	for _, page := range []pagemap.Page{mirrored, planned} {
		if err := repo.Insert(t.Context(), page); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	later := sqlitetest.Stamp.Add(time.Hour)
	for _, page := range []pagemap.Page{mirrored, planned} {
		if err := repo.FollowObservedH1(t.Context(), page.ID, "", page.Title, later); err != nil {
			t.Fatalf("FollowObservedH1: %v", err)
		}
	}

	got, err := repo.Get(t.Context(), mirrored.ID)
	want := mirrored
	want.H1, want.UpdatedAt = mirrored.Title, later
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("the mirrored page = %+v, %v\nwant %+v", got, err, want)
	}
	if got, err = repo.Get(t.Context(), planned.ID); err != nil || !reflect.DeepEqual(got, planned) {
		t.Errorf("the planned page = %+v, %v\nwant it untouched %+v", got, err, planned)
	}
}

func TestPageRepoMapsAnEntityOnlyOntoAPageThatCarriesNone(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewPageRepo(store)
	built := sqlitetest.Entity(t, store, owner.ID, "Healing Peptides")
	chosen := sqlitetest.Entity(t, store, owner.ID, "Chosen by a person")

	open := fullPage(owner.ID, "/healing-peptides/", sqlitetest.Stamp)
	held := fullPage(owner.ID, "/tb-500/", sqlitetest.Stamp)
	held.EntityID = &chosen.ID
	for _, page := range []pagemap.Page{open, held} {
		if err := repo.Insert(t.Context(), page); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	later := sqlitetest.Stamp.Add(time.Hour)
	cases := []struct {
		name   string
		pageID string
		mapped bool
		want   pagemap.Page
	}{
		{name: "a page with no entity takes it", pageID: open.ID, mapped: true, want: func() pagemap.Page {
			want := open
			want.EntityID, want.UpdatedAt = &built.ID, later
			return want
		}()},
		{name: "a page a person mapped keeps its entity", pageID: held.ID, want: held},
		{name: "an unknown page maps nothing", pageID: id.New()},
	}
	for _, tc := range cases {
		mapped, err := repo.MapEntity(t.Context(), tc.pageID, built.ID, later)
		if err != nil || mapped != tc.mapped {
			t.Errorf("%s: MapEntity = %v, %v; want %v", tc.name, mapped, err, tc.mapped)
		}
		if tc.want.ID == "" {
			continue
		}
		if got, getErr := repo.Get(t.Context(), tc.pageID); getErr != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: the page = %+v, %v\nwant %+v", tc.name, got, getErr, tc.want)
		}
	}
	if mapped, err := repo.MapEntity(t.Context(), open.ID, chosen.ID, later); err != nil || mapped {
		t.Errorf("a second mapping = %v, %v; want the first entity kept", mapped, err)
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
