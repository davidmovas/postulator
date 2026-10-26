//go:build !race

package scale_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	appgraph "github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const differentialBlocks = 6

func (s *scaled) stored(t *testing.T, q contextpack.MapRowQuery) []string {
	t.Helper()

	q.SiteID = s.siteID
	repo := sqlite.NewPageMapRepo(s.core.Store)
	out := make([]string, 0)
	var cursor paging.Cursor
	for {
		listed, err := repo.List(t.Context(), q, paging.Request{After: cursor, Limit: paging.MaxLimit})
		if err != nil {
			t.Fatalf("List(%+v): %v", q, err)
		}
		for i := range listed.Items {
			out = append(out, listed.Items[i].Path)
		}
		if !listed.HasMore {
			return out
		}
		cursor = listed.Next
	}
}

func refPaths(nodes []contextpack.MapNode) []string {
	out := make([]string, 0, len(nodes))
	for i := range nodes {
		out = append(out, nodes[i].Page.Path)
	}
	return out
}

func (s *scaled) agrees(t *testing.T, step string) {
	t.Helper()

	if err := s.core.SiteModel.Refresh(t.Context(), s.siteID); err != nil {
		t.Fatalf("%s: refresh the map: %v", step, err)
	}
	tree, err := s.core.SiteModel.Map(t.Context(), s.siteID)
	if err != nil {
		t.Fatalf("%s: read the model's map: %v", step, err)
	}
	listed, err := sqlite.NewPageMapRepo(s.core.Store).ListBySite(t.Context(), s.siteID)
	if err != nil {
		t.Fatalf("%s: list the stored map: %v", step, err)
	}
	rows := make(map[string]contextpack.MapRow, len(listed))
	for i := range listed {
		rows[listed[i].PageID] = listed[i]
	}
	visited := 0
	var walk func(parent string)
	walk = func(parent string) {
		children := tree.Children(parent)
		if got, want := s.stored(t, contextpack.MapRowQuery{ParentID: &parent}), refPaths(children); !slices.Equal(got, want) {
			t.Errorf("%s: the stored children of %q = %v, the model's %v", step, parent, got, want)
		}
		for i := range children {
			node := &children[i]
			visited++
			row := rows[node.Page.PageID]
			twin := ""
			if node.Twin != nil {
				twin = node.Twin.PageID
			}
			if row.Children != node.Children || row.Descendants != node.Descendants || row.TwinID != twin || row.Title != node.Page.Title {
				t.Errorf("%s: the stored row of %s = %+v, the model's node %+v", step, node.Page.Path, row, node)
			}
			trail := make([]string, 0)
			refs := tree.Trail(node.Page.PageID)
			for i := range refs {
				trail = append(trail, refs[i].PageID)
			}
			if !slices.Equal(row.Ancestors(), trail) {
				t.Errorf("%s: the stored trail of %s = %v, the model's %v", step, node.Page.Path, row.Ancestors(), trail)
			}
			if node.Children > 0 {
				under := tree.Under(node.Page.PageID)
				inside := make([]string, 0, len(under))
				for i := range listed {
					if strings.HasPrefix(listed[i].Trail, row.Trail) {
						inside = append(inside, listed[i].PageID)
					}
				}
				slices.Sort(under)
				slices.Sort(inside)
				if !slices.Equal(inside, under) {
					t.Errorf("%s: under %s the stored map holds %d pages, the model %d", step, node.Page.Path, len(inside), len(under))
				}
			}
			walk(node.Page.PageID)
		}
	}
	walk("")
	for i := range listed {
		place, found := tree.Place(listed[i].PageID)
		if !found || place.TypeID != listed[i].TypeID || place.Ref.Title != listed[i].Title {
			t.Errorf("%s: the stored row of %s is typed %q and named %q, the model's place %+v", step, listed[i].Path, listed[i].TypeID, listed[i].Title, place)
		}
	}
	nodes := 0
	for i := range listed {
		if listed[i].Row {
			nodes++
		}
	}
	if nodes != visited || len(listed) != len(rows) {
		t.Errorf("%s: the stored map holds %d nodes, the model %d", step, nodes, visited)
	}
	hub := s.id(t, "/c001/")
	for _, q := range []contextpack.MapQuery{
		{Text: "Article 001 0"}, {Text: "product-category"}, {TypeID: s.typeOf(t, pagetype.Product)},
		{TypeID: s.typeOf(t, pagetype.StoreCategory)}, {Under: hub, Text: "item"}, {Under: hub, TypeID: s.typeOf(t, pagetype.Guide)},
	} {
		query := contextpack.MapRowQuery{Text: graph.Key(q.Text), TypeID: q.TypeID}
		if q.Under != "" {
			query.Under = rows[q.Under].Trail
		}
		if got, want := s.stored(t, query), refPaths(tree.Matches(q)); !slices.Equal(got, want) {
			t.Errorf("%s: the stored map finds %v for %+v, the model %v", step, got, q, want)
		}
	}
}

func TestTheStoredMapAnswersAsTheModelDoes(t *testing.T) {
	t.Parallel()

	s := generate(t, differentialBlocks)
	s.agrees(t, "the generated site")

	sheet := filepath.Join(t.TempDir(), "more.csv")
	if err := os.WriteFile(sheet, []byte("path,title,entity\n/c002/s1/t1/later/,Later,Later\n/c003/s0/fresh/,Fresh,Fresh\n"), 0o600); err != nil {
		t.Fatalf("write the workbook: %v", err)
	}
	if _, err := s.core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: s.siteID, Path: sheet, Mapping: imports.Mapping{
		SiteID: s.siteID, Name: "more", Columns: map[string]string{
			string(importmap.FieldPath): "path", string(importmap.FieldTitle): "title", string(importmap.FieldEntity): "entity",
		},
	}}); err != nil {
		t.Fatalf("import: %v", err)
	}
	s.agrees(t, "an import")

	s.walk(t, s.plan.touched(false))
	if generation, built, err := sqlite.NewPageMapRepo(s.core.Store).State(t.Context(), s.siteID); err != nil || built < generation {
		t.Errorf("a walk left the map at generation %d built %d, %v; want derive to rebuild it", generation, built, err)
	}
	s.agrees(t, "a derive")

	archived := "Archived"
	if _, err := s.core.Pages.Update(t.Context(), pages.UpdateRequest{ID: s.id(t, "/c002/s0/"), Status: ref(string(pagemap.StatusArchived))}); err != nil {
		t.Fatalf("archive a hub: %v", err)
	}
	s.agrees(t, archived+" a hub")

	pinned := s.id(t, "/c004/s0/t0/p0/")
	repo := sqlite.NewPageRepo(s.core.Store)
	if err := repo.SetRowKey(t.Context(), s.siteID, pinned, "pinned-row", []string{"Topic 004 0 1"}, observedAt); err != nil {
		t.Fatalf("give the page a workbook trail: %v", err)
	}
	if err := repo.SetParentSource(t.Context(), pinned, pagemap.ParentFromWorkbook, observedAt); err != nil {
		t.Fatalf("pin the workbook's parent: %v", err)
	}
	s.walk(t, nil)
	page, err := repo.Get(t.Context(), pinned)
	if err != nil || page.ParentPageID == nil || *page.ParentPageID != s.id(t, "/c004/s0/t1/") {
		t.Fatalf("the pinned page sits under %v, %v; want the workbook's topic", page.ParentPageID, err)
	}
	s.agrees(t, "a parent pin")

	entity, err := s.core.Graph.CreateEntity(t.Context(), appgraph.CreateEntityRequest{SiteID: s.siteID, Name: "Article Five Two"})
	if err != nil {
		t.Fatalf("create an entity: %v", err)
	}
	if _, err = s.core.Pages.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: s.id(t, "/c005/s1/t0/p2/"), EntityID: entity.Entity.ID}); err != nil {
		t.Fatalf("map a page: %v", err)
	}
	s.agrees(t, "a map-to-page")

	created, err := s.core.Pages.Create(t.Context(), pages.CreateRequest{SiteID: s.siteID, Path: "/c005/s1/t0/planned/", Title: "Planned"})
	if err != nil {
		t.Fatalf("create a page: %v", err)
	}
	s.pageIDs[created.Page.Path] = created.Page.ID
	s.agrees(t, "a page created")
	if _, err = s.core.Pages.Delete(t.Context(), pages.DeleteRequest{ID: created.Page.ID}); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	delete(s.pageIDs, created.Page.Path)
	s.agrees(t, "a page deleted")

	hub := s.id(t, "/c003/")
	retitle := func(name string) {
		t.Helper()
		if _, updateErr := s.core.Pages.Update(t.Context(), pages.UpdateRequest{ID: hub, Title: &name, H1: &name}); updateErr != nil {
			t.Fatalf("retitle the hub: %v", updateErr)
		}
		if observeErr := sqlite.NewPageFieldRepo(s.core.Store).Observe(t.Context(), hub, []field.Value{
			field.NewValue(hub, field.Title, name, observedAt), field.NewValue(hub, field.H1, name, observedAt),
		}); observeErr != nil {
			t.Fatalf("observe the hub's new name: %v", observeErr)
		}
	}
	retitle("Something else entirely")
	if node, found := mustMap(t, s).Node(hub); !found || node.Twin != nil {
		t.Fatalf("the retitled hub = %+v, want its store category unfolded", node)
	}
	s.agrees(t, "a retitle that breaks a twin")
	retitle("Category 003")
	if node, found := mustMap(t, s).Node(hub); !found || node.Twin == nil {
		t.Fatalf("the hub named back = %+v, want its store category folded again", node)
	}
	s.agrees(t, "a retitle that makes a twin")
}

func mustMap(t *testing.T, s *scaled) contextpack.Map {
	t.Helper()

	tree, err := s.core.SiteModel.Map(t.Context(), s.siteID)
	if err != nil {
		t.Fatalf("read the model's map: %v", err)
	}
	return tree
}
