package contextpack_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
)

func addressRows(paths ...string) []contextpack.MapRow {
	rows := make([]contextpack.MapRow, 0, len(paths))
	for _, path := range paths {
		rows = append(rows, contextpack.MapRow{PageID: "page" + path, Path: path, Row: true})
	}
	return rows
}

func TestTheAddressTreeFoldsPagesByTheirPathSegments(t *testing.T) {
	t.Parallel()

	rows := addressRows(
		"/", "/products/", "/products/p1/", "/products/p2/", "/researchers/products/p1/",
		"/product/bpc-157/", "/product/tb-500/", "/guides/storage/cold/",
	)

	cases := []struct {
		name  string
		under string
		want  []contextpack.Folder
	}{
		{
			name: "the root is the front page holding everything",
			want: []contextpack.Folder{{Path: "/", PageID: "page/", Children: 4, Pages: 8}},
		},
		{
			name:  "the folders under the front page",
			under: "/",
			want: []contextpack.Folder{
				{Path: "/guides/", Children: 1, Pages: 1},
				{Path: "/product/", Children: 2, Pages: 2},
				{Path: "/products/", PageID: "page/products/", Children: 2, Pages: 3},
				{Path: "/researchers/", Children: 1, Pages: 1},
			},
		},
		{
			name:  "a folder with a page of its own",
			under: "/products/",
			want: []contextpack.Folder{
				{Path: "/products/p1/", PageID: "page/products/p1/", Pages: 1},
				{Path: "/products/p2/", PageID: "page/products/p2/", Pages: 1},
			},
		},
		{
			name:  "a folder with no page",
			under: "/guides/",
			want:  []contextpack.Folder{{Path: "/guides/storage/", Children: 1, Pages: 1}},
		},
		{name: "a leaf", under: "/product/tb-500/", want: []contextpack.Folder{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := contextpack.Folders(rows, tc.under); !slices.Equal(got, tc.want) {
				t.Errorf("Folders(%q) = %+v, want %+v", tc.under, got, tc.want)
			}
		})
	}
}

func TestASiteWithNoFrontPageStillHasItsRootFolder(t *testing.T) {
	t.Parallel()

	got := contextpack.Folders(addressRows("/shop/", "/shop/kettle/"), "")
	want := []contextpack.Folder{{Path: "/", Children: 1, Pages: 2}}
	if !slices.Equal(got, want) {
		t.Errorf("the root = %+v, want %+v", got, want)
	}
	if got = contextpack.Folders(nil, ""); len(got) != 0 {
		t.Errorf("an empty site's root = %+v, want nothing", got)
	}
}

func TestTheAddressTreeKeepsAFoldedStoreCategoryAndLeavesAnArchivedPageOut(t *testing.T) {
	t.Parallel()

	rows := addressRows("/", "/peptides/")
	rows = append(rows,
		contextpack.MapRow{PageID: "term", Path: "/product-category/peptides/", HubID: "page/peptides/"},
		contextpack.MapRow{PageID: "gone", Path: "/old/"},
	)
	got := contextpack.Folders(rows, "/")
	want := []contextpack.Folder{
		{Path: "/peptides/", PageID: "page/peptides/", Pages: 1},
		{Path: "/product-category/", Children: 1, Pages: 1},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Folders = %+v, want %+v: a store category sits at its own address, an archived page nowhere", got, want)
	}
	if root := contextpack.Folders(rows, ""); root[0].Pages != 3 {
		t.Errorf("the root holds %d pages, want 3", root[0].Pages)
	}
}

func TestOnlyAPathWithBothSlashesIsAFolder(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]bool{
		"/": true, "/products/": true, "/products/p1/": true,
		"": false, "products/": false, "/products": false, "//": false, "/a//b/": false,
	} {
		if got := contextpack.IsFolder(path); got != want {
			t.Errorf("IsFolder(%q) = %v, want %v", path, got, want)
		}
	}
}
