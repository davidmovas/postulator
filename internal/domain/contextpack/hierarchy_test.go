package contextpack_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func TestThePageTreeAndTheGraphAreComparedByTheirPagedAncestors(t *testing.T) {
	t.Parallel()

	group := graph.Entity{ID: "e-group", SiteID: siteID, Name: "Research", Source: graph.SourceUser}
	lab := graph.Entity{ID: "e-lab", SiteID: siteID, Name: "Lab Guides", CanonicalPageID: new("guides"), Source: graph.SourceUser}
	proposed := parentEdge("g9", "e-bpc", "e-peptides")
	proposed.Status = graph.StatusProposed

	cases := []struct {
		name    string
		edges   []graph.Edge
		pageIDs []string
		want    map[string]contextpack.ParentDisagreement
	}{
		{name: "the graph says what the tree says", edges: []graph.Edge{parentEdge("g1", "e-bpc", "e-peptides"), parentEdge("g2", "e-healing", "e-peptides")}},
		{
			name: "a flat graph under a deep tree",
			want: map[string]contextpack.ParentDisagreement{
				"e-bpc":     {EntityID: "e-bpc", PageID: "bpc", Path: "/bpc-157/", Tree: contextpack.TreeParent{PageID: "hub", Path: "/peptides/", EntityID: "e-peptides"}},
				"e-healing": {EntityID: "e-healing", PageID: "healing", Path: "/healing-peptides/", Tree: contextpack.TreeParent{PageID: "hub", Path: "/peptides/", EntityID: "e-peptides"}},
			},
		},
		{
			name:    "only the chosen pages are compared",
			pageIDs: []string{"healing"},
			want: map[string]contextpack.ParentDisagreement{
				"e-healing": {EntityID: "e-healing", PageID: "healing", Path: "/healing-peptides/", Tree: contextpack.TreeParent{PageID: "hub", Path: "/peptides/", EntityID: "e-peptides"}},
			},
		},
		{
			name:  "the graph names another parent than the tree",
			edges: []graph.Edge{parentEdge("g1", "e-bpc", "e-healing"), parentEdge("g2", "e-healing", "e-peptides")},
			want: map[string]contextpack.ParentDisagreement{
				"e-bpc": {EntityID: "e-bpc", PageID: "bpc", Path: "/bpc-157/", GraphParents: []string{"e-healing"}, Tree: contextpack.TreeParent{PageID: "hub", Path: "/peptides/", EntityID: "e-peptides"}},
			},
		},
		{
			name:  "a proposed parent edge is not yet the graph's",
			edges: []graph.Edge{proposed, parentEdge("g2", "e-healing", "e-peptides")},
			want: map[string]contextpack.ParentDisagreement{
				"e-bpc": {EntityID: "e-bpc", PageID: "bpc", Path: "/bpc-157/", Tree: contextpack.TreeParent{PageID: "hub", Path: "/peptides/", EntityID: "e-peptides"}},
			},
		},
		{
			name: "a parent without a page is passed through to the one above it",
			edges: []graph.Edge{
				parentEdge("g1", "e-bpc", "e-group"), parentEdge("g3", "e-group", "e-peptides"), parentEdge("g2", "e-healing", "e-peptides"),
			},
		},
		{
			name: "a parent the tree shows elsewhere for a root of the tree",
			edges: []graph.Edge{
				parentEdge("g1", "e-bpc", "e-peptides"), parentEdge("g2", "e-healing", "e-peptides"), parentEdge("g4", "e-peptides", "e-lab"),
			},
			want: map[string]contextpack.ParentDisagreement{
				"e-peptides": {EntityID: "e-peptides", PageID: "hub", Path: "/peptides/", GraphParents: []string{"e-lab"}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := replicaLike()
			f.entities = append(f.entities, group, lab)
			f.edges = tc.edges
			got := f.snapshot(t).Map(nil).ParentDisagreements(tc.pageIDs)
			if len(got) != len(tc.want) {
				t.Fatalf("the disagreements = %+v, want %d", got, len(tc.want))
			}
			if !slices.IsSortedFunc(got, func(a, b contextpack.ParentDisagreement) int { return cmpPath(a.Path, b.Path) }) {
				t.Errorf("the disagreements are not in path order: %+v", got)
			}
			for _, held := range got {
				want, named := tc.want[held.EntityID]
				if !named || held.PageID != want.PageID || held.Path != want.Path || held.Tree != want.Tree ||
					!slices.Equal(held.GraphParents, want.GraphParents) {
					t.Errorf("%s disagrees as %+v, want %+v", held.EntityID, held, want)
				}
			}
		})
	}
}

func cmpPath(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

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
