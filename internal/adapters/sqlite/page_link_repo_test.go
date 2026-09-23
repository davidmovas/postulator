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
)

func TestPageLinkRepoReplaceForPage(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	from := sqlitetest.Page(t, store, owner.ID, "/shop/")
	to := sqlitetest.Page(t, store, owner.ID, "/shop/shoes/")
	repo := sqlite.NewPageLinkRepo(store)

	empty, err := repo.ListForPage(t.Context(), from.ID)
	if err != nil || len(empty) != 0 {
		t.Fatalf("ListForPage on a fresh page = %v, %v", empty, err)
	}

	links := []pagemap.PageLink{
		{ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToPageID: &to.ID, ToURL: "/shop/shoes/", AnchorText: "shoes", Origin: pagemap.OriginGenerated, ObservedAt: sqlitetest.Stamp},
		{ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToURL: "https://elsewhere.example.com/", AnchorText: "elsewhere", Origin: pagemap.OriginObserved, ObservedAt: sqlitetest.Stamp.Add(time.Second)},
	}
	if err = repo.ReplaceForPage(t.Context(), from.ID, links); err != nil {
		t.Fatalf("ReplaceForPage: %v", err)
	}
	got, err := repo.ListForPage(t.Context(), from.ID)
	if err != nil {
		t.Fatalf("ListForPage: %v", err)
	}
	if !reflect.DeepEqual(got, links) {
		t.Errorf("ListForPage = %+v\nwant %+v", got, links)
	}

	replacement := []pagemap.PageLink{links[1]}
	if err = repo.ReplaceForPage(t.Context(), from.ID, replacement); err != nil {
		t.Fatalf("ReplaceForPage again: %v", err)
	}
	got, err = repo.ListForPage(t.Context(), from.ID)
	if err != nil || !reflect.DeepEqual(got, replacement) {
		t.Errorf("after replacement = %+v, %v", got, err)
	}

	if err = repo.ReplaceForPage(t.Context(), from.ID, nil); err != nil {
		t.Fatalf("ReplaceForPage with nothing: %v", err)
	}
	got, err = repo.ListForPage(t.Context(), from.ID)
	if err != nil || len(got) != 0 {
		t.Errorf("after clearing = %+v, %v", got, err)
	}

	foreign := links[0]
	foreign.FromPageID = to.ID
	if err = repo.ReplaceForPage(t.Context(), from.ID, []pagemap.PageLink{foreign}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link from another page code = %q, want INVALID", errors.CodeOf(err))
	}
	dangling := links[0]
	unknown := "no-such-page"
	dangling.ToPageID = &unknown
	if err = repo.ReplaceForPage(t.Context(), from.ID, []pagemap.PageLink{dangling}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link to an unknown page code = %q, want INVALID", errors.CodeOf(err))
	}
}
