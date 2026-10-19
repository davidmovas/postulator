package sqlite_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestThePageModelIsReadByPageIDs(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	hub := sqlitetest.Page(t, store, owner.ID, "/peptides/")
	child := sqlitetest.Page(t, store, owner.ID, "/bpc-157/")
	other := sqlitetest.Page(t, store, owner.ID, "/lab-guides/")

	fields := sqlite.NewPageFieldRepo(store)
	for _, page := range []string{hub.ID, child.ID, other.ID} {
		if err := fields.Observe(t.Context(), page, []field.Value{field.NewValue(page, field.Title, "title of "+page, sqlitetest.Stamp)}); err != nil {
			t.Fatalf("Observe: %v", err)
		}
	}
	bodies := sqlite.NewPageBodyRepo(store)
	for _, page := range []string{hub.ID, child.ID} {
		if err := bodies.Observe(t.Context(), block.Snapshot{PageID: page, Side: block.Current, Raw: "<p>" + page + "</p>", Hash: block.BodyHash(page), Kind: block.Classic, ObservedAt: sqlitetest.Stamp}); err != nil {
			t.Fatalf("Observe body: %v", err)
		}
	}
	if err := bodies.Record(t.Context(), hub.ID, "<p>ours</p>", sqlitetest.Stamp); err != nil {
		t.Fatalf("Record: %v", err)
	}
	renders := sqlite.NewPageRenderRepo(store)
	if err := renders.Put(t.Context(), render.Record{PageID: hub.ID, Extraction: render.Extraction{Root: render.RootEmptyRaw}, ExtractedAt: sqlitetest.Stamp}); err != nil {
		t.Fatalf("Put render: %v", err)
	}
	links := sqlite.NewPageLinkRepo(store)
	link := func(from, to string) pagemap.PageLink {
		return pagemap.PageLink{ID: id.New(), SiteID: owner.ID, FromPageID: from, ToPageID: &to, ToURL: "/x/", Origin: pagemap.OriginObserved,
			Type: pagemap.LinkContextual, Source: pagemap.SourceRaw, ObservedAt: sqlitetest.Stamp}
	}
	if err := links.ReplaceForPage(t.Context(), child.ID, pagemap.SourceRaw, []pagemap.PageLink{link(child.ID, hub.ID)}); err != nil {
		t.Fatalf("ReplaceForPage child: %v", err)
	}
	if err := links.ReplaceForPage(t.Context(), other.ID, pagemap.SourceRaw, []pagemap.PageLink{link(other.ID, child.ID)}); err != nil {
		t.Fatalf("ReplaceForPage other: %v", err)
	}

	wanted := []string{hub.ID, child.ID, id.New()}

	byPage, err := fields.ListForPages(t.Context(), wanted)
	if err != nil || len(byPage) != 2 || len(byPage[hub.ID]) != 1 || byPage[other.ID] != nil {
		t.Errorf("fields = %+v, %v; want the two pages asked for", byPage, err)
	}
	current, err := bodies.ListCurrent(t.Context(), wanted)
	if err != nil || len(current) != 2 {
		t.Fatalf("bodies = %+v, %v", current, err)
	}
	for _, snapshot := range current {
		if snapshot.Side != block.Current {
			t.Errorf("body %+v is not the current side", snapshot)
		}
		if snapshot.PageID == hub.ID && snapshot.Raw != "<p>ours</p>" {
			t.Errorf("the hub's current body = %q, want what was recorded", snapshot.Raw)
		}
	}
	if got, err := renders.ListForPages(t.Context(), wanted); err != nil || len(got) != 1 || got[0].PageID != hub.ID {
		t.Errorf("renders = %+v, %v", got, err)
	}
	from, err := links.ListFromPages(t.Context(), []string{child.ID, other.ID})
	if err != nil || len(from) != 2 {
		t.Errorf("links from = %+v, %v", from, err)
	}
	to, err := links.ListToPage(t.Context(), hub.ID)
	if err != nil || len(to) != 1 || to[0].FromPageID != child.ID {
		t.Errorf("links to the hub = %+v, %v", to, err)
	}

	navigation := link(hub.ID, other.ID)
	navigation.Type, navigation.Source = pagemap.LinkNavigation, pagemap.SourceRender
	if err := links.ReplaceForPage(t.Context(), hub.ID, pagemap.SourceRender, []pagemap.PageLink{navigation}); err != nil {
		t.Fatalf("ReplaceForPage hub: %v", err)
	}
	if got, err := links.ListContextual(t.Context(), owner.ID); err != nil || len(got) != 2 {
		t.Errorf("contextual links = %+v, %v; want the two raw contextual links", got, err)
	}
	if err := renders.Put(t.Context(), render.Record{PageID: child.ID, Extraction: render.Extraction{Root: render.RootFound, URLForm: render.URLFormAbsolute}, ExtractedAt: sqlitetest.Stamp}); err != nil {
		t.Fatalf("Put render: %v", err)
	}
	if got, err := renders.URLForms(t.Context(), owner.ID); err != nil || !slices.Equal(got, []string{render.URLFormAbsolute}) {
		t.Errorf("url forms = %v, %v; want the one render that has one", got, err)
	}

	many := make([]string, 0, 1201)
	for range 1200 {
		many = append(many, id.New())
	}
	many = append(many, other.ID)
	slices.Reverse(many)
	if got, err := fields.ListForPages(t.Context(), many); err != nil || len(got) != 1 || len(got[other.ID]) != 1 {
		t.Errorf("a long id list = %+v, %v; want it read in chunks", got, err)
	}
	if got, err := fields.ListForPages(t.Context(), nil); err != nil || len(got) != 0 {
		t.Errorf("no ids = %+v, %v", got, err)
	}
}
