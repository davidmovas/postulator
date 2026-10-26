package sitemodel_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func findAll(t *testing.T, m *sitemodeltest.Memory, req sitemodel.FindRequest) []contextpack.MapNode {
	t.Helper()

	req.SiteID = m.Site.ID
	var out []contextpack.MapNode
	for {
		got, err := m.Service().Find(t.Context(), req)
		if err != nil {
			t.Fatalf("Find(%+v): %v", req, err)
		}
		out = append(out, got.Items...)
		if !got.HasMore {
			return out
		}
		req.Cursor = got.Next
	}
}

func TestManyPagesAreFoundInOneCallWithTheirKeyFacts(t *testing.T) {
	t.Parallel()

	m := wide()
	for i := range m.Pages {
		if m.Pages[i].ID == "page-03-04" {
			m.Pages[i].Keywords = keyword.New([]keyword.Keyword{{Text: "Cold Storage Serum"}})
		}
	}
	cases := []struct {
		name string
		req  sitemodel.FindRequest
		want int
	}{
		{name: "by ids", req: sitemodel.FindRequest{PageIDs: []string{"page-01-01", "hub-02", "home", "not-a-page"}}, want: 3},
		{name: "by title", req: sitemodel.FindRequest{Query: "page 07 1"}, want: 10},
		{name: "by path", req: sitemodel.FindRequest{Query: "/hub-07/page-0"}, want: 10},
		{name: "by keyword", req: sitemodel.FindRequest{Query: "storage serum"}, want: 1},
		{name: "by type", req: sitemodel.FindRequest{PageTypeID: typeOf(pagetype.Hub)}, want: hubs},
		{name: "by subtree", req: sitemodel.FindRequest{Under: "hub-07"}, want: childrenOf},
		{name: "by subtree and title", req: sitemodel.FindRequest{Under: "hub-07", Query: "page 07 9"}, want: childrenOf - 90},
		{name: "by a gap", req: sitemodel.FindRequest{Gap: contextpack.GapEmpty}, want: 1 + hubs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.req.Limit = 7
			got := findAll(t, m, tc.req)
			if len(got) != tc.want {
				t.Fatalf("found %d pages, want %d", len(got), tc.want)
			}
			paths := make([]string, 0, len(got))
			for _, node := range got {
				paths = append(paths, node.Page.Path)
				if node.ParentID == "" && node.Why == nil {
					t.Errorf("%s has no parent and no reason", node.Page.Path)
				}
				if node.Page.Type == "" {
					t.Errorf("%s came back without its type", node.Page.Path)
				}
			}
			if !slices.IsSorted(paths) {
				t.Errorf("the pages are not in path order: %v", paths)
			}
		})
	}

	if _, err := m.Service().Find(t.Context(), sitemodel.FindRequest{SiteID: m.Site.ID, Gap: "everything"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown gap answered %v", err)
	}
}

func TestPagesAreFoundByTheirEntityAndTheirState(t *testing.T) {
	t.Parallel()

	m := wide()
	for i := range m.Pages {
		switch m.Pages[i].ID {
		case "page-03-04", "page-05-01":
			m.Pages[i].EntityID = new("entity-serum")
		case "page-06-02":
			m.Pages[i].EntityID = new("entity-serum")
			m.Pages[i].Status = pagemap.StatusExists
		}
	}
	cases := []struct {
		name string
		req  sitemodel.FindRequest
		want []string
	}{
		{name: "by entity", req: sitemodel.FindRequest{EntityID: "entity-serum"}, want: []string{"page-03-04", "page-05-01", "page-06-02"}},
		{name: "by entity and state", req: sitemodel.FindRequest{EntityID: "entity-serum", Status: pagemap.StatusExists}, want: []string{"page-06-02"}},
		{name: "by state", req: sitemodel.FindRequest{Status: pagemap.StatusExists}, want: []string{"page-06-02"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := findAll(t, m, tc.req)
			ids := make([]string, 0, len(got))
			for _, node := range got {
				ids = append(ids, node.Page.PageID)
			}
			if !slices.Equal(ids, tc.want) {
				t.Fatalf("found %v, want %v", ids, tc.want)
			}
		})
	}

	if _, err := m.Service().Find(t.Context(), sitemodel.FindRequest{SiteID: m.Site.ID, Status: "lost"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown state answered %v", err)
	}
}
