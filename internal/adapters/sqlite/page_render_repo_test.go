package sqlite_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func renderOf(pageID string, root render.RootState, at time.Time) render.Record {
	return render.Record{
		PageID: pageID,
		Extraction: render.Extraction{
			Hash: "render-" + string(root),
			Root: root,
			Regions: []render.Region{
				{Kind: render.RegionBreadcrumb, Links: 2, Words: 3, Signal: render.SignalJSONLD},
				{Kind: render.RegionContent, Inside: true, Links: 1, Words: 120, Signal: render.SignalLinks},
			},
			Trails: []pagemap.Trail{{Source: pagemap.TrailJSONLD, Crumbs: []pagemap.Crumb{{Name: "Home", Path: "/"}, {Name: "Peptides", Path: "/peptides/"}}}},
			JSONLD: map[string][]json.RawMessage{"BreadcrumbList": {json.RawMessage(`{"@type":"BreadcrumbList"}`)}},
			Headings: []render.Heading{
				{Level: 1, Text: "TB-500"},
				{Level: 2, Text: "Related products", Generated: true},
			},
			WordsContent: 120, WordsGenerated: 40, Builder: "elementor", URLForm: render.URLFormAbsolute, Robots: "index, follow",
		},
		ExtractedAt: at,
	}
}

func TestPageRenderRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	product := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	term := sqlitetest.Page(t, store, owner.ID, "/peptides/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")
	repo := sqlite.NewPageRenderRepo(store)

	if _, err := repo.Get(t.Context(), product.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get before Put code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	want := renderOf(product.ID, render.RootFound, sqlitetest.Stamp)
	carried := want
	carried.Links = []render.Link{{Href: "/peptides/", Path: "/peptides/", Anchor: "Peptides", Type: pagemap.LinkBreadcrumb, Region: render.RegionBreadcrumb, Position: -1}}
	if err := repo.Put(t.Context(), carried); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := repo.Get(t.Context(), product.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Get = %+v, %v\nwant %+v without its links", got, err, want)
	}

	rescanned := renderOf(product.ID, render.RootDynamicOnly, sqlitetest.Stamp.Add(time.Hour))
	rescanned.Builder, rescanned.Robots = "", "noindex"
	if err = repo.Put(t.Context(), rescanned); err != nil {
		t.Fatalf("Put again: %v", err)
	}
	if got, err = repo.Get(t.Context(), product.ID); err != nil || !reflect.DeepEqual(got, rescanned) {
		t.Fatalf("Get after a second Put = %+v, %v\nwant %+v", got, err, rescanned)
	}

	sparse := render.Record{PageID: term.ID, Extraction: render.Extraction{Hash: "term", Root: render.RootNotFound}, ExtractedAt: sqlitetest.Stamp}
	for _, record := range []render.Record{sparse, renderOf(elsewhere.ID, render.RootFound, sqlitetest.Stamp)} {
		if err = repo.Put(t.Context(), record); err != nil {
			t.Fatalf("Put another page: %v", err)
		}
	}
	readSparse := sparse
	readSparse.Regions, readSparse.Trails, readSparse.Headings = []render.Region{}, []pagemap.Trail{}, []render.Heading{}
	readSparse.JSONLD = map[string][]json.RawMessage{}
	listed := []render.Record{rescanned, readSparse}
	slices.SortFunc(listed, func(a, b render.Record) int { return strings.Compare(a.PageID, b.PageID) })
	if got, err := repo.ListBySite(t.Context(), owner.ID); err != nil || !reflect.DeepEqual(got, listed) {
		t.Fatalf("ListBySite = %+v, %v\nwant %+v", got, err, listed)
	}

	unknownRoot := renderOf(product.ID, "partial", sqlitetest.Stamp)
	if err = repo.Put(t.Context(), unknownRoot); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a root state the model does not name code = %q, want INVALID", errors.CodeOf(err))
	}
	if err = repo.Put(t.Context(), renderOf("no-such-page", render.RootFound, sqlitetest.Stamp)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("the render of no page code = %q, want INVALID", errors.CodeOf(err))
	}

	if err = sqlite.NewPageRepo(store).Delete(t.Context(), product.ID); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if _, err = repo.Get(t.Context(), product.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get after the page is deleted code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}
