package sqlite_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func mapRow(page pagemap.Page, parent, trail string) contextpack.MapRow {
	return contextpack.MapRow{
		PageID: page.ID, SiteID: page.SiteID, ParentID: parent, Path: page.Path, Trail: trail, Row: true,
		Title: page.Path, TitleKey: page.Path, PathKey: page.Path,
	}
}

func paths(rows []contextpack.MapRow) []string {
	out := make([]string, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].Path)
	}
	return out
}

func TestPageMapRepoKeepsTheMapAndSaysWhenItMoved(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	repo := sqlite.NewPageMapRepo(store)

	generation, built, err := repo.State(t.Context(), owner.ID)
	if err != nil || built >= generation {
		t.Fatalf("a new site's map = generation %d built %d, %v; want it stale", generation, built, err)
	}
	home := sqlitetest.Page(t, store, owner.ID, "/")
	hub := sqlitetest.Page(t, store, owner.ID, "/hub/")
	child := sqlitetest.Page(t, store, owner.ID, "/hub/child/")
	term := sqlitetest.Page(t, store, owner.ID, "/product-category/hub/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")
	moved, _, err := repo.State(t.Context(), owner.ID)
	if err != nil || moved <= generation {
		t.Fatalf("four new pages left the generation at %d (was %d), %v", moved, generation, err)
	}

	rows := []contextpack.MapRow{
		mapRow(home, "", home.ID+"/"),
		mapRow(hub, home.ID, home.ID+"/"+hub.ID+"/"),
		mapRow(child, hub.ID, home.ID+"/"+hub.ID+"/"+child.ID+"/"),
		{PageID: term.ID, SiteID: owner.ID, ParentID: hub.ID, HubID: hub.ID, Path: term.Path, Trail: home.ID + "/" + hub.ID + "/" + term.ID + "/"},
	}
	rows[1].TwinID, rows[1].TwinTitleKey, rows[1].TypeID, rows[1].Children, rows[1].Descendants = term.ID, "category", "hub-type", 1, 1
	rows[2].TypeID = "entity-type"
	if err = repo.Write(t.Context(), owner.ID, rows, nil, moved); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err = repo.Write(t.Context(), other.ID, []contextpack.MapRow{mapRow(elsewhere, "", elsewhere.ID+"/")}, nil, 1); err != nil {
		t.Fatalf("Write the other site: %v", err)
	}
	if generation, built, err = repo.State(t.Context(), owner.ID); err != nil || built != generation {
		t.Fatalf("a written map = generation %d built %d, %v; want it fresh", generation, built, err)
	}

	stored, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || len(stored) != len(rows) {
		t.Fatalf("ListBySite = %d rows, %v", len(stored), err)
	}
	byID := map[string]contextpack.MapRow{}
	for i := range stored {
		byID[stored[i].PageID] = stored[i]
	}
	for i := range rows {
		if got := byID[rows[i].PageID]; got != rows[i] {
			t.Errorf("the stored row = %+v, want %+v", got, rows[i])
		}
	}

	list := func(q contextpack.MapRowQuery, page paging.Request) paging.List[contextpack.MapRow] {
		t.Helper()
		q.SiteID = owner.ID
		got, listErr := repo.List(t.Context(), q, page)
		if listErr != nil {
			t.Fatalf("List(%+v): %v", q, listErr)
		}
		return got
	}
	root := ""
	if got := paths(list(contextpack.MapRowQuery{ParentID: &root}, paging.Request{}).Items); !slices.Equal(got, []string{"/"}) {
		t.Errorf("the roots = %v", got)
	}
	under := home.ID
	first := list(contextpack.MapRowQuery{Under: home.ID + "/"}, paging.Request{Limit: 2})
	if got := paths(first.Items); !slices.Equal(got, []string{"/", "/hub/"}) || !first.HasMore {
		t.Errorf("the first page under home = %v, more %v", got, first.HasMore)
	}
	if got := paths(list(contextpack.MapRowQuery{Under: home.ID + "/"}, paging.Request{After: first.Next, Limit: 2}).Items); !slices.Equal(got, []string{"/hub/child/"}) {
		t.Errorf("the second page under home = %v, want the child and never the folded store category", got)
	}
	if got := paths(list(contextpack.MapRowQuery{ParentID: &under}, paging.Request{}).Items); !slices.Equal(got, []string{"/hub/"}) {
		t.Errorf("the front page's children = %v", got)
	}
	if got := paths(list(contextpack.MapRowQuery{Text: "category"}, paging.Request{}).Items); !slices.Equal(got, []string{"/hub/"}) {
		t.Errorf("the twin's title finds %v, want the hub", got)
	}
	if got := paths(list(contextpack.MapRowQuery{Text: "child"}, paging.Request{}).Items); !slices.Equal(got, []string{"/hub/child/"}) {
		t.Errorf("a path finds %v", got)
	}
	if got := paths(list(contextpack.MapRowQuery{TypeID: "entity-type"}, paging.Request{}).Items); !slices.Equal(got, []string{"/hub/child/"}) {
		t.Errorf("a type finds %v", got)
	}

	held, err := repo.ListByIDs(t.Context(), []string{term.ID, child.ID, elsewhere.ID})
	if err != nil || len(held) != 3 {
		t.Fatalf("ListByIDs = %+v, %v", held, err)
	}

	if err = repo.Write(t.Context(), owner.ID, []contextpack.MapRow{rows[0]}, []string{child.ID}, moved); err != nil {
		t.Fatalf("Write a removal: %v", err)
	}
	if stored, err = repo.ListBySite(t.Context(), owner.ID); err != nil || len(stored) != 3 {
		t.Errorf("after the removal the map holds %d rows, %v", len(stored), err)
	}
	if _, built, err = repo.State(t.Context(), owner.ID); err != nil || built != moved {
		t.Errorf("an older write moved built to %d, %v; want it kept at %d", built, err, moved)
	}
}

func TestEveryWriterThatMovesTheMapMarksItStale(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/hub/")
	repo := sqlite.NewPageMapRepo(store)
	pages := sqlite.NewPageRepo(store)
	fields := sqlite.NewPageFieldRepo(store)
	types := sqlite.NewPageTypeRepo(store)

	fresh := func() int64 {
		t.Helper()
		generation, _, err := repo.State(t.Context(), owner.ID)
		if err != nil {
			t.Fatalf("State: %v", err)
		}
		if err = repo.Write(t.Context(), owner.ID, nil, nil, generation); err != nil {
			t.Fatalf("Write: %v", err)
		}
		return generation
	}
	stale := func(what string, write func() error, want bool) {
		t.Helper()
		before := fresh()
		if err := write(); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		generation, built, err := repo.State(t.Context(), owner.ID)
		if err != nil {
			t.Fatalf("State after %s: %v", what, err)
		}
		if got := built < generation; got != want {
			t.Errorf("%s left the map stale %v (generation %d, built %d, was %d), want %v", what, got, generation, built, before, want)
		}
	}

	stale("a retitle", func() error { page.Title = "Hub"; return pages.Update(t.Context(), page) }, true)
	stale("an unchanged update", func() error { return pages.Update(t.Context(), page) }, false)
	stale("a meta description", func() error { page.MetaDescription = "Read me"; return pages.Update(t.Context(), page) }, false)
	stale("an archive", func() error { page.Status = pagemap.StatusArchived; return pages.Update(t.Context(), page) }, true)
	stale("a title field", func() error {
		return fields.Observe(t.Context(), page.ID, []field.Value{field.NewValue(page.ID, field.Title, "Hub", sqlitetest.Stamp)})
	}, true)
	stale("the same title field", func() error {
		return fields.Observe(t.Context(), page.ID, []field.Value{field.NewValue(page.ID, field.Title, "Hub", sqlitetest.Stamp)})
	}, false)
	stale("an excerpt", func() error {
		return fields.Observe(t.Context(), page.ID, []field.Value{field.NewValue(page.ID, field.Excerpt, "Short", sqlitetest.Stamp)})
	}, false)
	if named, err := fields.ListNamesBySite(t.Context(), owner.ID); err != nil || len(named) != 1 || len(named[page.ID]) != 1 ||
		named[page.ID][0].Key != field.Title {
		t.Errorf("ListNamesBySite = %+v, %v; want the title alone, never the excerpt", named, err)
	}
	stale("a new page", func() error { sqlitetest.Page(t, store, owner.ID, "/hub/child/"); return nil }, true)
	kind := pagetype.Type{ID: id.New(), SiteID: owner.ID, Name: "Hubs", Preset: pagetype.Hub, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	stale("a new type", func() error { return types.Insert(t.Context(), kind) }, true)
	stale("a type", func() error {
		return pages.SetType(t.Context(), page.ID, kind.ID, pagemap.TypeByUser, sqlitetest.Stamp)
	}, true)
	stale("a deletion", func() error { return pages.Delete(t.Context(), page.ID) }, true)
}
