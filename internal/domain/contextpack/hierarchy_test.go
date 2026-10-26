package contextpack_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func TestThePageTreeNamesTheEntityAPageSitsUnder(t *testing.T) {
	t.Parallel()

	f := replicaLike()
	lone := typed(under(onSitePage("lone", "/lab-guides/storage/", "Storage", 40, pagemap.WPPage), "guides", pagemap.ParentJSONLD), pagetype.Entity)
	deep := typed(under(onSitePage("deep", "/bpc-157/dosing/", "Dosing", 41, pagemap.WPPage), "bpc", pagemap.ParentJSONLD), pagetype.Entity)
	f.pages = append(f.pages, lone, deep)
	tree := f.snapshot(t).Map(nil)

	cases := []struct {
		name   string
		pageID string
		onMap  bool
		want   contextpack.TreeParent
	}{
		{name: "a child of a hub that carries an entity", pageID: "bpc", onMap: true, want: contextpack.TreeParent{PageID: "hub", Path: "/peptides/", EntityID: "e-peptides"}},
		{name: "two levels down, the nearest entity wins", pageID: "deep", onMap: true, want: contextpack.TreeParent{PageID: "bpc", Path: "/bpc-157/", EntityID: "e-bpc"}},
		{name: "a hub under the front page is a root", pageID: "hub", onMap: true},
		{name: "a store category reads as its twin hub", pageID: "term", onMap: true},
		{name: "an ancestor without an entity is passed over up to the root", pageID: "lone", onMap: true},
		{name: "a product with no parent is a root", pageID: "tb-500", onMap: true},
		{name: "a page the model lacks is not on the map", pageID: "missing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, onMap := tree.TreeParentOf(tc.pageID)
			if onMap != tc.onMap || got != tc.want {
				t.Fatalf("TreeParentOf(%s) = %+v, %v; want %+v, %v", tc.pageID, got, onMap, tc.want, tc.onMap)
			}
		})
	}
}
