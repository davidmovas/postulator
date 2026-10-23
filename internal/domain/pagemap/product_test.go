package pagemap_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
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
	at := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	fields := map[string][]field.Value{
		"powder": {field.NewValue("powder", field.Title, "Old Powder", at)},
		"page":   {field.NewValue("page", field.Title, "Mak Capsule", at)},
	}

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

			got, claimed := pagemap.ClaimProduct(tc.store, pages, fields)
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
