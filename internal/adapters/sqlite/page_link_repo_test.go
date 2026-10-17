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

	position := 2
	links := []pagemap.PageLink{
		{ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToPageID: &to.ID, ToURL: "/shop/shoes/", AnchorText: "shoes", Origin: pagemap.OriginGenerated,
			Type: pagemap.LinkContextual, Region: "content", Source: pagemap.SourceRaw, Position: &position, Ours: true, ObservedAt: sqlitetest.Stamp},
		{ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToURL: "https://elsewhere.example.com/", AnchorText: "elsewhere", Origin: pagemap.OriginObserved,
			Type: pagemap.LinkContextual, Source: pagemap.SourceRaw, ObservedAt: sqlitetest.Stamp.Add(time.Second)},
	}
	if err = repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, links); err != nil {
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
	if err = repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, replacement); err != nil {
		t.Fatalf("ReplaceForPage again: %v", err)
	}
	got, err = repo.ListForPage(t.Context(), from.ID)
	if err != nil || !reflect.DeepEqual(got, replacement) {
		t.Errorf("after replacement = %+v, %v", got, err)
	}

	if err = repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, nil); err != nil {
		t.Fatalf("ReplaceForPage with nothing: %v", err)
	}
	got, err = repo.ListForPage(t.Context(), from.ID)
	if err != nil || len(got) != 0 {
		t.Errorf("after clearing = %+v, %v", got, err)
	}

	foreign := links[0]
	foreign.FromPageID = to.ID
	if err = repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, []pagemap.PageLink{foreign}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link from another page code = %q, want INVALID", errors.CodeOf(err))
	}
	dangling := links[0]
	unknown := "no-such-page"
	dangling.ToPageID = &unknown
	if err = repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, []pagemap.PageLink{dangling}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a link to an unknown page code = %q, want INVALID", errors.CodeOf(err))
	}
	if got, err = repo.ListForPage(t.Context(), from.ID); err != nil || len(got) != 0 {
		t.Errorf("a refused replace left %+v, %v", got, err)
	}
}

func TestPageLinkRepoReplacesOneSourceAndKeepsTheOther(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	from := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	hub := sqlitetest.Page(t, store, owner.ID, "/peptides/")
	repo := sqlite.NewPageLinkRepo(store)

	link := func(anchor string, kind pagemap.LinkType, region string, source pagemap.LinkSource, at time.Duration) pagemap.PageLink {
		return pagemap.PageLink{
			ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToPageID: &hub.ID, ToURL: "/peptides/", AnchorText: anchor,
			Origin: pagemap.OriginObserved, Type: kind, Region: region, Source: source, ObservedAt: sqlitetest.Stamp.Add(at),
		}
	}
	raw := []pagemap.PageLink{link("peptides", pagemap.LinkContextual, "content", pagemap.SourceRaw, 0)}
	rendered := []pagemap.PageLink{
		link("Peptides", pagemap.LinkBreadcrumb, "breadcrumb", pagemap.SourceRender, time.Second),
		link("All peptides", pagemap.LinkNavigation, "header", pagemap.SourceRender, 2*time.Second),
	}

	if err := repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, raw); err != nil {
		t.Fatalf("replace the raw links: %v", err)
	}
	if err := repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRender, rendered); err != nil {
		t.Fatalf("replace the render links: %v", err)
	}
	both := append(append([]pagemap.PageLink{}, raw...), rendered...)
	if got, err := repo.ListForPage(t.Context(), from.ID); err != nil || !reflect.DeepEqual(got, both) {
		t.Fatalf("ListForPage = %+v, %v\nwant %+v", got, err, both)
	}
	if got, err := repo.ListBySite(t.Context(), owner.ID); err != nil || len(got) != len(both) {
		t.Fatalf("ListBySite = %d links, %v; want %d", len(got), err, len(both))
	}

	if err := repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, nil); err != nil {
		t.Fatalf("clear the raw links: %v", err)
	}
	if got, err := repo.ListForPage(t.Context(), from.ID); err != nil || !reflect.DeepEqual(got, rendered) {
		t.Fatalf("clearing the raw links left %+v, %v\nwant the render links %+v", got, err, rendered)
	}
	if err := repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, raw); err != nil {
		t.Fatalf("restore the raw links: %v", err)
	}
	if err := repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRender, rendered[:1]); err != nil {
		t.Fatalf("replace the render links with fewer: %v", err)
	}
	kept := []pagemap.PageLink{raw[0], rendered[0]}
	if got, err := repo.ListForPage(t.Context(), from.ID); err != nil || !reflect.DeepEqual(got, kept) {
		t.Fatalf("ListForPage = %+v, %v\nwant %+v", got, err, kept)
	}

	if err := repo.ReplaceForPage(t.Context(), from.ID, pagemap.SourceRaw, rendered); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("render links replaced as raw code = %q, want INVALID", errors.CodeOf(err))
	}
	if err := repo.ReplaceForPage(t.Context(), from.ID, "html", nil); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a source the model does not name code = %q, want INVALID", errors.CodeOf(err))
	}
	if got, err := repo.ListForPage(t.Context(), from.ID); err != nil || !reflect.DeepEqual(got, kept) {
		t.Errorf("a refused replace changed the links to %+v, %v", got, err)
	}
}
