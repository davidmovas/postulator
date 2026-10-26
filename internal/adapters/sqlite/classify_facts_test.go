package sqlite_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestTheStoreSummarizesARenderAsTheDomainDoes(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	repo := sqlite.NewPageRenderRepo(store)

	listing := renderOf(sqlitetest.Page(t, store, owner.ID, "/peptides/").ID, render.RootEmptyRaw, sqlitetest.Stamp)
	listing.Regions = append(listing.Regions, render.Region{Kind: render.RegionListing, Inside: true, Links: 8})
	listing.JSONLD["CollectionPage"] = []json.RawMessage{json.RawMessage(`{"@type":"CollectionPage"}`)}
	listing.JSONLD["ItemList"] = []json.RawMessage{json.RawMessage(`{"@type":"ItemList"}`)}

	article := renderOf(sqlitetest.Page(t, store, owner.ID, "/guide/").ID, render.RootFound, sqlitetest.Stamp)
	article.Regions = append(article.Regions, render.Region{Kind: render.RegionListing, Links: 4})
	article.JSONLD = map[string][]json.RawMessage{"Article": {json.RawMessage(`{"@type":"Article"}`)}}
	article.Headings = append(article.Headings, render.Heading{Level: 2, Text: "Storage"}, render.Heading{Level: 2, Text: "Dosing"},
		render.Heading{Level: 3, Text: "Notes"})
	article.WordsContent = 1400

	bare := render.Record{
		PageID: sqlitetest.Page(t, store, owner.ID, "/bare/").ID, Extraction: render.Extraction{Hash: "bare", Root: render.RootNotFound},
		ExtractedAt: sqlitetest.Stamp,
	}
	records := []render.Record{listing, article, bare}
	for i := range records {
		if err := repo.Put(t.Context(), records[i]); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	got, err := repo.ListRenderedBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListRenderedBySite: %v", err)
	}
	if len(got) != len(records) {
		t.Fatalf("summaries = %d, want %d", len(got), len(records))
	}
	for i := range records {
		want := pagetype.RenderedOf(&records[i])
		if !reflect.DeepEqual(got[records[i].PageID], want) {
			t.Errorf("the store summarizes %s as %+v, the domain as %+v", records[i].PageID, got[records[i].PageID], want)
		}
	}
	if !got[listing.PageID].Listing || got[article.PageID].Listing || got[article.PageID].H2 != 2 {
		t.Errorf("the summaries = %+v, want the listing inside its content and the article's two H2s", got)
	}
}

func TestTheStoreCountsTheProductsAPageLists(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	pages := sqlite.NewPageRepo(store)
	hub := sqlitetest.Page(t, store, owner.ID, "/peptides/")
	guide := sqlitetest.Page(t, store, owner.ID, "/guide/")
	products := make([]pagemap.Page, 0, 3)
	for _, path := range []string{"/product/a/", "/product/b/", "/product/c/"} {
		product := sqlitetest.Page(t, store, owner.ID, path)
		product.WPType = pagemap.WPProduct
		if err := pages.Update(t.Context(), product); err != nil {
			t.Fatalf("make %s a product: %v", path, err)
		}
		products = append(products, product)
	}
	link := func(from, to pagemap.Page, kind pagemap.LinkType) pagemap.PageLink {
		return pagemap.PageLink{
			ID: id.New(), SiteID: owner.ID, FromPageID: from.ID, ToPageID: &to.ID, ToURL: to.Path, Type: kind,
			Region: string(render.RegionListing), Source: pagemap.SourceRender, ObservedAt: sqlitetest.Stamp,
		}
	}
	links := sqlite.NewPageLinkRepo(store)
	if err := links.ReplaceForPage(t.Context(), hub.ID, pagemap.SourceRender, []pagemap.PageLink{
		link(hub, products[0], pagemap.LinkListing), link(hub, products[1], pagemap.LinkListing), link(hub, products[1], pagemap.LinkListing),
		link(hub, guide, pagemap.LinkListing), link(hub, products[2], pagemap.LinkContextual),
	}); err != nil {
		t.Fatalf("ReplaceForPage: %v", err)
	}

	got, err := links.ListedProductsBySite(t.Context(), owner.ID)
	if err != nil {
		t.Fatalf("ListedProductsBySite: %v", err)
	}
	if !reflect.DeepEqual(got, map[string]int{hub.ID: 2}) {
		t.Errorf("listed products = %v, want the hub's two distinct products", got)
	}
}
