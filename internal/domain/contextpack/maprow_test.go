package contextpack_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func rowsByID(rows []contextpack.MapRow) map[string]contextpack.MapRow {
	out := make(map[string]contextpack.MapRow, len(rows))
	for i := range rows {
		out[rows[i].PageID] = rows[i]
	}
	return out
}

func held(rows []contextpack.MapRow, q contextpack.MapRowQuery) []string {
	out := make([]string, 0)
	for i := range rows {
		if q.Holds(rows[i]) {
			out = append(out, rows[i].Path)
		}
	}
	slices.Sort(out)
	return out
}

func TestTheRowsSayWhatTheMapAnswers(t *testing.T) {
	t.Parallel()

	for name, snapshot := range map[string]contextpack.Snapshot{"5 mg": mapped5mg(t), "replica": replicaLike().snapshot(t)} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			m := snapshot.Map(nil)
			rows := m.Rows()
			byID := rowsByID(rows)
			if len(rows) != len(snapshot.Index.Pages()) {
				t.Fatalf("%d rows for %d pages, want one each", len(rows), len(snapshot.Index.Pages()))
			}
			var walk func(parent string)
			walk = func(parent string) {
				children := m.Children(parent)
				parentID := parent
				want := make([]string, 0, len(children))
				for _, node := range children {
					want = append(want, node.Page.Path)
				}
				slices.Sort(want)
				if got := held(rows, contextpack.MapRowQuery{SiteID: siteID, ParentID: &parentID}); !slices.Equal(got, want) {
					t.Errorf("the rows under %q = %v, the map's children %v", parent, got, want)
				}
				for _, node := range children {
					row := byID[node.Page.PageID]
					if !row.Row || row.Children != node.Children || row.Descendants != node.Descendants || row.Title != node.Page.Title {
						t.Errorf("the row of %s = %+v, the node %+v", node.Page.Path, row, node)
					}
					if (node.Twin == nil) != (row.TwinID == "") || (node.Twin != nil && node.Twin.PageID != row.TwinID) {
						t.Errorf("the row of %s pairs %q, the node %+v", node.Page.Path, row.TwinID, node.Twin)
					}
					refs := m.Trail(node.Page.PageID)
					trail := make([]string, 0, len(refs))
					for _, ref := range refs {
						trail = append(trail, ref.PageID)
					}
					if !slices.Equal(row.Ancestors(), trail) || row.Depth != len(trail) {
						t.Errorf("the row of %s descends %v at depth %d, the map's trail %v", node.Page.Path, row.Ancestors(), row.Depth, trail)
					}
					under := m.Under(node.Page.PageID)
					inside := make([]string, 0, len(under))
					for i := range rows {
						if strings.HasPrefix(rows[i].Trail, row.Trail) {
							inside = append(inside, rows[i].PageID)
						}
					}
					slices.Sort(under)
					slices.Sort(inside)
					if !slices.Equal(inside, under) {
						t.Errorf("the trail of %s holds %v, the map's subtree %v", node.Page.Path, inside, under)
					}
					walk(node.Page.PageID)
				}
			}
			walk("")
			for i := range rows {
				place, _ := m.Place(rows[i].PageID)
				if rows[i].TypeID != place.TypeID || rows[i].Title != place.Ref.Title {
					t.Errorf("the row of %s is typed %q and named %q, its place %+v", rows[i].Path, rows[i].TypeID, rows[i].Title, place)
				}
				if _, isNode := m.Node(rows[i].PageID); rows[i].Row != (isNode && rows[i].HubID == "") {
					t.Errorf("the row of %s says row %v", rows[i].Path, rows[i].Row)
				}
			}
			for _, q := range []contextpack.MapQuery{
				{Text: "bpc"}, {Text: "product-category"}, {TypeID: typeIDOf(pagetype.Product)}, {TypeID: typeIDOf(pagetype.StoreCategory)},
				{Under: "hub", Text: "BPC"}, {Under: "home", TypeID: typeIDOf(pagetype.Hub)},
			} {
				want := pathsOf(m.Matches(q))
				slices.Sort(want)
				query := contextpack.MapRowQuery{SiteID: siteID, Text: graph.Key(q.Text), TypeID: q.TypeID}
				if q.Under != "" {
					query.Under = byID[q.Under].Trail
				}
				if got := held(rows, query); !slices.Equal(got, want) {
					t.Errorf("the rows holding %+v = %v, the map's matches %v", q, got, want)
				}
			}
		})
	}
}

func TestARowOutsideTheMapHasNoTrail(t *testing.T) {
	t.Parallel()

	rows := rowsByID(mapped5mg(t).Map(nil).Rows())
	if gone := rows["gone"]; gone.Row || gone.Trail != "" || gone.Title != "Old page" {
		t.Errorf("an archived page's row = %+v, want no place in the tree but its name", gone)
	}
	term, hub := rows["term"], rows["hub"]
	if term.Row || term.HubID != "hub" || !strings.HasPrefix(term.Trail, hub.Trail) || hub.TwinID != "term" {
		t.Errorf("the folded store category = %+v under %+v, want it inside its hub's trail", term, hub)
	}
	if ancestors := (contextpack.MapRow{Trail: "a/b/c/"}).Ancestors(); !slices.Equal(ancestors, []string{"a", "b"}) {
		t.Errorf("the ancestors of a/b/c/ = %v", ancestors)
	}
	if root := (contextpack.MapRow{Trail: "a/"}).Ancestors(); len(root) != 0 {
		t.Errorf("a root's ancestors = %v", root)
	}
}
