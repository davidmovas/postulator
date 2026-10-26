package sitemodel_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func tree(t *testing.T, m *sitemodeltest.Memory, req sitemodel.TreeRequest) sitemodel.TreeAnswer {
	t.Helper()

	req.SiteID = m.Site.ID
	got, err := m.Service().Tree(t.Context(), req)
	if err != nil {
		t.Fatalf("Tree(%+v): %v", req, err)
	}
	return got
}

func walkAll(t *testing.T, m *sitemodeltest.Memory, req sitemodel.TreeRequest) (nodes []contextpack.MapNode, calls int) {
	t.Helper()

	for {
		got := tree(t, m, req)
		calls++
		if got.Stale {
			t.Fatalf("the walk went stale on call %d with nothing changed", calls)
		}
		if req.Generation != 0 && got.Generation != req.Generation {
			t.Fatalf("the generation moved from %d to %d with nothing changed", req.Generation, got.Generation)
		}
		nodes = append(nodes, got.Nodes.Items...)
		if !got.Nodes.HasMore {
			return nodes, calls
		}
		req.Cursor, req.Generation = got.Nodes.Next, got.Generation
	}
}

func TestTheWholeTreeWalksInPagesWithNoGapAndNoDuplicate(t *testing.T) {
	t.Parallel()

	m := wide()
	nodes, calls := walkAll(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeWalk, Limit: 97})
	if len(nodes) != len(m.Pages) || calls != (len(m.Pages)+96)/97 {
		t.Fatalf("the walk read %d nodes in %d calls, want %d", len(nodes), calls, len(m.Pages))
	}
	seen := map[string]int{}
	for i, node := range nodes {
		if _, twice := seen[node.Page.PageID]; twice {
			t.Fatalf("%s came back twice", node.Page.Path)
		}
		if node.ParentID != "" {
			if _, before := seen[node.ParentID]; !before {
				t.Fatalf("%s came before its parent %s", node.Page.Path, node.ParentID)
			}
		}
		seen[node.Page.PageID] = i
	}
	for i := range m.Pages {
		if _, read := seen[m.Pages[i].ID]; !read {
			t.Fatalf("the walk missed %s", m.Pages[i].Path)
		}
	}
}

func TestASubtreeStopsAtTheDepthAsked(t *testing.T) {
	t.Parallel()

	m := wide()
	cases := []struct {
		name   string
		pageID string
		depth  int
		want   int
	}{
		{name: "the hubs under the front page", pageID: "home", depth: 1, want: hubs},
		{name: "two levels under the front page", pageID: "home", depth: 2, want: hubs + hubs*childrenOf},
		{name: "a hub's children and never the hub", pageID: "hub-07", depth: 1, want: childrenOf},
		{name: "deeper than the tree goes", pageID: "hub-07", depth: 5, want: childrenOf},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			nodes, _ := walkAll(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeSubtree, PageID: tc.pageID, Depth: tc.depth, Limit: 200})
			if len(nodes) != tc.want {
				t.Fatalf("the subtree holds %d nodes, want %d", len(nodes), tc.want)
			}
			if slices.ContainsFunc(nodes, func(node contextpack.MapNode) bool { return node.Page.PageID == tc.pageID }) {
				t.Fatalf("the subtree of %s holds the page itself", tc.pageID)
			}
		})
	}
}

func TestAPageReadsItsAncestorsSiblingsAndChildren(t *testing.T) {
	t.Parallel()

	m := wide()
	ancestors := tree(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeAncestors, PageID: "page-07-03"})
	if got := nodePaths(ancestors.Nodes.Items); !slices.Equal(got, []string{"/", "/hub-07/"}) || ancestors.Nodes.HasMore {
		t.Fatalf("the ancestors = %v, want the front page then the hub", got)
	}
	if ancestors.Node == nil || ancestors.Node.Page.PageID != "page-07-03" || ancestors.Node.ParentID != "hub-07" || ancestors.Node.Depth != 2 {
		t.Fatalf("the asked page = %+v", ancestors.Node)
	}

	siblings, _ := walkAll(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeSiblings, PageID: "page-07-03", Limit: 40})
	if len(siblings) != childrenOf-1 || slices.ContainsFunc(siblings, func(node contextpack.MapNode) bool { return node.Page.PageID == "page-07-03" }) {
		t.Fatalf("the siblings = %d, want the hub's other %d children", len(siblings), childrenOf-1)
	}

	children, _ := walkAll(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeChildren, PageID: "hub-07", Limit: 40})
	if len(children) != childrenOf {
		t.Fatalf("the children = %d, want %d", len(children), childrenOf)
	}

	roots := tree(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeRoots})
	if got := nodePaths(roots.Nodes.Items); !slices.Equal(got, []string{"/"}) {
		t.Fatalf("the roots = %v", got)
	}

	hubsOnly, _ := walkAll(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeWalk, PageTypeID: typeOf(pagetype.Hub), Limit: 200})
	if len(hubsOnly) != hubs {
		t.Fatalf("the walk of the hub type read %d nodes, want %d", len(hubsOnly), hubs)
	}
}

func TestATreeNodeWithoutAParentSaysWhy(t *testing.T) {
	t.Parallel()

	m := wide()
	m.Pages = append(m.Pages,
		sitePage("loose", "/loose/", "Loose", 9001, pagemap.WPPage, pagetype.Entity, ""),
		sitePage("crumbed", "/crumbed/", "Crumbed", 9002, pagemap.WPPage, pagetype.Entity, ""),
		sitePage("rendered", "/rendered/", "Rendered", 9003, pagemap.WPPage, pagetype.Entity, ""),
	)
	m.Renders = append(m.Renders,
		render.Record{PageID: "crumbed", Extraction: render.Extraction{Root: render.RootFound}},
		render.Record{PageID: "rendered", Extraction: render.Extraction{Root: render.RootFound}},
	)
	m.Findings = append(m.Findings, finding.Finding{
		ID: "unresolved", SiteID: siteID, PageID: new("crumbed"), Code: finding.ParentUnresolved, Status: finding.Open,
		Subject: json.RawMessage(`{"name":"Skin care","path":"/skin-care/"}`),
	})

	roots := tree(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeRoots})
	want := map[string]contextpack.WhyReason{
		"/": contextpack.WhyRoot, "/loose/": contextpack.WhyNotDerived, "/crumbed/": contextpack.WhyCrumbUnresolved,
		"/rendered/": contextpack.WhyNoSignal,
	}
	if len(roots.Nodes.Items) != len(want) {
		t.Fatalf("the roots = %v", nodePaths(roots.Nodes.Items))
	}
	for _, node := range roots.Nodes.Items {
		if node.ParentID != "" || node.Why == nil || node.Why.Reason != want[node.Page.Path] {
			t.Errorf("%s: parent %q, why %+v, want %s", node.Page.Path, node.ParentID, node.Why, want[node.Page.Path])
		}
	}
	for _, node := range tree(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeChildren, PageID: "hub-07"}).Nodes.Items {
		if node.Why != nil {
			t.Fatalf("%s has a parent and still answers why not: %+v", node.Page.Path, node.Why)
		}
	}
}

func TestAWalkSaysWhenTheMapMovedUnderIt(t *testing.T) {
	t.Parallel()

	m := wide()
	first := tree(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeWalk, Limit: 50})
	m.MarkStale()
	next := tree(t, m, sitemodel.TreeRequest{Shape: sitemodel.TreeWalk, Limit: 50, Cursor: first.Nodes.Next, Generation: first.Generation})
	if !next.Stale || next.Generation == first.Generation {
		t.Fatalf("a walk across a moved map answered stale %v, generation %d then %d", next.Stale, first.Generation, next.Generation)
	}
}

func TestATreeShapeThatNeedsAPageRefusesWithoutOne(t *testing.T) {
	t.Parallel()

	m := wide()
	for _, shape := range []sitemodel.TreeShape{sitemodel.TreeChildren, sitemodel.TreeSubtree, sitemodel.TreeAncestors, sitemodel.TreeSiblings} {
		_, err := m.Service().Tree(t.Context(), sitemodel.TreeRequest{SiteID: m.Site.ID, Shape: shape})
		if !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s without a page answered %v", shape, err)
		}
	}
	if _, err := m.Service().Tree(t.Context(), sitemodel.TreeRequest{SiteID: m.Site.ID, Shape: "sideways"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown shape answered %v", err)
	}
}
