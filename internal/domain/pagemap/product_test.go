package pagemap_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func storeProduct(id string, wpID int64, path, slug string) pagemap.Page {
	return pagemap.Page{
		ID: id, SiteID: "site", Path: path, Slug: slug, WPType: pagemap.WPProduct, WPID: &wpID,
		Status: pagemap.StatusPublished,
	}
}

func inTheStore(wpID int64, path, slug, name string) pagemap.Page {
	product := storeProduct("", wpID, path, slug)
	product.Title = name
	return product
}

func TestMatchProductFindsTheStoreProductAFileRowMeans(t *testing.T) {
	t.Parallel()

	liquid := storeProduct("liquid", 11, "/product/bpc-157-liquid/", "bpc-157-liquid")
	liquid.Title = "Buy Liquid BPC"
	capsules := storeProduct("capsules", 12, "/product/capsules/", "capsules")
	tbLiquid := storeProduct("tb-liquid", 13, "/product/tb-500-liquid/", "tb-500-liquid")
	sameSlugA := storeProduct("powder-a", 14, "/product/powder/", "powder")
	sameSlugB := storeProduct("powder-b", 15, "/product/powder-2/", "powder")
	unread := storeProduct("unread", 16, "/product/gel-caps/", "gel-caps")
	unread.Title = "Gel Caps"
	page := pagemap.Page{ID: "page", SiteID: "site", Path: "/bpc-157/", WPType: pagemap.WPPage}
	unsynced := pagemap.Page{ID: "planned", SiteID: "site", Path: "/x/", WPType: pagemap.WPProduct}
	products := []pagemap.Page{liquid, capsules, tbLiquid, sameSlugA, sameSlugB, unread, page, unsynced}
	names := map[string][]string{
		"liquid": {"BPC-157 Liquid"}, "capsules": {"BPC-157 Capsules"}, "tb-liquid": {"TB-500 Liquid", "TB-500 in Liquid Form"},
		"powder-a": {"Powder"}, "powder-b": {"Powder"}, "page": {"BPC-157"}, "planned": {"BPC-157 Liquid"},
	}

	cases := []struct {
		name      string
		row       pagemap.ProductRow
		want      string
		by        pagemap.MatchedBy
		ambiguous bool
	}{
		{name: "the exact address", row: pagemap.ProductRow{Path: "/product/bpc-157-liquid/"}, want: "liquid", by: pagemap.MatchedByPath},
		{name: "the address without a trailing slash", row: pagemap.ProductRow{Path: "/product/capsules"}, want: "capsules", by: pagemap.MatchedByPath},
		{name: "the slug the URL ends in", row: pagemap.ProductRow{Path: "/bpc-157/capsules/"}, want: "capsules", by: pagemap.MatchedBySlug},
		{name: "the store name the H1 names", row: pagemap.ProductRow{Path: "/bpc-157/liquid/", H1: "bpc-157 liquid"}, want: "liquid", by: pagemap.MatchedByName},
		{name: "the store name the title names", row: pagemap.ProductRow{Path: "/tb-500/liquid/", Title: "TB-500 Liquid"}, want: "tb-liquid", by: pagemap.MatchedByName},
		{name: "a slug two products share", row: pagemap.ProductRow{Path: "/forms/powder/"}, ambiguous: true},
		{name: "a name two products share", row: pagemap.ProductRow{Path: "/forms/x/", H1: "Powder"}, ambiguous: true},
		{name: "nothing alike", row: pagemap.ProductRow{Path: "/forms/gel/", H1: "Gel"}},
		{name: "a page is never a product", row: pagemap.ProductRow{Path: "/bpc-157/", H1: "BPC-157"}},
		{name: "a product the store does not hold yet", row: pagemap.ProductRow{Path: "/elsewhere/", H1: "BPC-157 Liquid"}, want: "liquid", by: pagemap.MatchedByName},
		{name: "the H1 the store's page shows", row: pagemap.ProductRow{Path: "/tb-500/form/", H1: "TB-500 in liquid form"}, want: "tb-liquid", by: pagemap.MatchedByName},
		{name: "a plan's title the site does not show", row: pagemap.ProductRow{Path: "/bpc/liquid/", Title: "Buy Liquid BPC"}},
		{name: "the plan's title of a product whose name was never read", row: pagemap.ProductRow{Path: "/gels/", Title: "Gel Caps"}, want: "unread", by: pagemap.MatchedByName},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := pagemap.MatchProduct(tc.row, products, names)
			if got.Ambiguous != tc.ambiguous {
				t.Fatalf("ambiguous = %t, want %t (%+v)", got.Ambiguous, tc.ambiguous, got)
			}
			if tc.want == "" {
				if got.Found {
					t.Fatalf("found %s, want none", got.Page.ID)
				}
				return
			}
			if !got.Found || got.Page.ID != tc.want || got.By != tc.by {
				t.Fatalf("match = %s by %s (found %t), want %s by %s", got.Page.ID, got.By, got.Found, tc.want, tc.by)
			}
		})
	}
}

func TestANewStoreProductClaimsTheRowThatWaitedForIt(t *testing.T) {
	t.Parallel()

	waiting := func(id, path, h1 string) pagemap.Page {
		return pagemap.Page{ID: id, SiteID: "site", Path: path, WPType: pagemap.WPProduct, Status: pagemap.StatusPlanned, H1: h1}
	}
	pages := []pagemap.Page{
		waiting("liquid-row", "/mak/mak-liquid/", "Mak Liquid"),
		waiting("capsule-row", "/mak/capsule/", "Mak Capsule"),
		waiting("gel-a", "/a/gel/", ""),
		waiting("gel-b", "/b/gel/", ""),
		waiting("powder-row", "/mak/powder/", "Powder"),
		storeProduct("powder", 20, "/product/powder/", "powder"),
		{ID: "page", SiteID: "site", Path: "/mak/", WPType: pagemap.WPPage, H1: "Mak Liquid"},
	}
	names := map[string][]string{"powder": {"Old Powder"}, "page": {"Mak Capsule"}}

	cases := []struct {
		name  string
		store pagemap.Page
		want  string
	}{
		{name: "the slug the row's URL ends in", store: inTheStore(31, "/product/mak-liquid/", "mak-liquid", "Liquid"), want: "liquid-row"},
		{name: "the name the row's H1 gives", store: inTheStore(32, "/product/capsule-x/", "capsule-x", "Mak Capsule"), want: "capsule-row"},
		{name: "a product two rows wait for", store: inTheStore(33, "/product/gel/", "gel", "Gel")},
		{name: "a product a row would rather match elsewhere", store: inTheStore(34, "/product/powder-2/", "powder-2", "Powder")},
		{name: "a product no row waits for", store: inTheStore(35, "/product/other/", "other", "Other")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, claimed := pagemap.ClaimProduct(tc.store, pages, names)
			if tc.want == "" {
				if claimed {
					t.Fatalf("claimed by %s, want no row", got.ID)
				}
				return
			}
			if !claimed || got.ID != tc.want {
				t.Fatalf("claimed by %s (%t), want %s", got.ID, claimed, tc.want)
			}
		})
	}
}

func TestOnlyTheShopsTypesAreAddressedByTheStore(t *testing.T) {
	t.Parallel()

	cases := map[pagemap.WPType]bool{
		pagemap.WPPage: false, pagemap.WPPost: false, pagemap.WPProduct: true, pagemap.WPProductCategory: true,
	}
	for wpType, want := range cases {
		if got := wpType.StoreAddressed(); got != want {
			t.Errorf("%s.StoreAddressed() = %t, want %t", wpType, got, want)
		}
	}
}

func TestNewPageKeepsThePlannedPathOnlyWhenItDiffers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		planned string
		want    string
	}{
		{name: "a file URL beside the store's", planned: "/BPC-157/Liquid", want: "/bpc-157/liquid/"},
		{name: "the same address", planned: "/product/liquid/", want: ""},
		{name: "none", planned: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			page, err := pagemap.NewPage(pagemap.Page{
				ID: "p", SiteID: "s", Path: "/product/liquid/", PlannedPath: tc.planned,
				WPType: pagemap.WPProduct, Status: pagemap.StatusPublished,
			})
			if err != nil {
				t.Fatalf("NewPage: %v", err)
			}
			if page.PlannedPath != tc.want {
				t.Errorf("planned path = %q, want %q", page.PlannedPath, tc.want)
			}
		})
	}

	if _, err := pagemap.NewPage(pagemap.Page{
		ID: "p", SiteID: "s", Path: "/product/liquid/", PlannedPath: "/a/../b/",
		WPType: pagemap.WPProduct, Status: pagemap.StatusPlanned,
	}); err == nil {
		t.Error("a planned path with a dot segment must be refused")
	}
}
