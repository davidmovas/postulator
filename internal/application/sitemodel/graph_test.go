package sitemodel_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func graphEntity(id, name, pageID string) graph.Entity {
	entity := graph.Entity{ID: id, SiteID: siteID, Name: name, Source: graph.SourceUser, CreatedAt: stamp, UpdatedAt: stamp}
	if pageID != "" {
		entity.CanonicalPageID = new(pageID)
	}
	return entity
}

func carrying(page pagemap.Page, entityID string) pagemap.Page {
	page.EntityID = new(entityID)
	return page
}

func twoHierarchies() *sitemodeltest.Memory {
	related := graph.Edge{
		ID: "r-oils-serums", SiteID: siteID, FromEntityID: "e-oils", ToEntityID: "e-serums", Kind: graph.EdgeRelated, Weight: 0.7,
		Source: graph.SourceUser, Status: graph.StatusApproved,
	}
	proposed := graph.Edge{
		ID: "r-argan-vitc", SiteID: siteID, FromEntityID: "e-argan", ToEntityID: "e-vitc", Kind: graph.EdgeRelated, Weight: 0.4,
		Source: graph.SourceAI, Status: graph.StatusProposed,
	}
	return &sitemodeltest.Memory{
		Site: site.Site{ID: siteID, Name: "Shop", BaseURL: "https://shop.example.com", Settings: site.DefaultSettings()},
		Pages: []pagemap.Page{
			sitePage("home", "/", "Home", 1, pagemap.WPPage, pagetype.Entity, ""),
			carrying(sitePage("serums", "/serums/", "Serums", 2, pagemap.WPPage, pagetype.Hub, "home"), "e-serums"),
			carrying(sitePage("vitc", "/serums/vitamin-c/", "Vitamin C", 3, pagemap.WPPage, pagetype.Entity, "serums"), "e-vitc"),
			carrying(sitePage("oils", "/oils/", "Oils", 4, pagemap.WPPage, pagetype.Hub, "home"), "e-oils"),
			carrying(sitePage("argan", "/oils/argan/", "Argan", 5, pagemap.WPPage, pagetype.Entity, "oils"), "e-argan"),
		},
		Entities: []graph.Entity{
			graphEntity("e-serums", "Serums", "serums"), graphEntity("e-vitc", "Vitamin C", "vitc"),
			graphEntity("e-oils", "Oils", "oils"), graphEntity("e-argan", "Argan Oil", "argan"),
			graphEntity("e-loose", "Loose Notes", ""),
		},
		Edges: []graph.Edge{edge("p-vitc", "e-vitc", "e-serums"), related, proposed},
		Types: presetSet(),
	}
}

func entityRows(t *testing.T, m *sitemodeltest.Memory, req sitemodel.EntitiesRequest) []sitemodel.EntityRow {
	t.Helper()

	req.SiteID = m.Site.ID
	var out []sitemodel.EntityRow
	for {
		got, err := m.Service().Entities(t.Context(), req)
		if err != nil {
			t.Fatalf("Entities(%+v): %v", req, err)
		}
		out = append(out, got.Items...)
		if !got.HasMore {
			return out
		}
		req.Cursor = got.Next
	}
}

func names(rows []sitemodel.EntityRow) []string {
	out := make([]string, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].Name)
	}
	return out
}

func TestEveryEntityReadsWithItsParentOrWhyItHasNone(t *testing.T) {
	t.Parallel()

	m := twoHierarchies()
	rows := entityRows(t, m, sitemodel.EntitiesRequest{Limit: 2})
	if got := names(rows); !slices.Equal(got, []string{"Argan Oil", "Loose Notes", "Oils", "Serums", "Vitamin C"}) {
		t.Fatalf("the entities = %v, want every one by name across pages", got)
	}
	byName := map[string]sitemodel.EntityRow{}
	for i := range rows {
		byName[rows[i].Name] = rows[i]
	}
	switch vitc := byName["Vitamin C"]; {
	case vitc.Parent == nil || vitc.Parent.EntityID != "e-serums" || vitc.Parent.Name != "Serums" || vitc.Parent.Status != string(graph.StatusApproved):
		t.Errorf("Vitamin C's parent = %+v", vitc.Parent)
	case vitc.ParentWhy != "":
		t.Errorf("an entity with a parent says why it has none: %q", vitc.ParentWhy)
	case vitc.CanonicalPage == nil || vitc.CanonicalPage.Path != "/serums/vitamin-c/":
		t.Errorf("Vitamin C's page = %+v", vitc.CanonicalPage)
	}
	if argan := byName["Argan Oil"]; argan.Parent != nil || argan.ParentWhy != sitemodel.NoParentEdge {
		t.Errorf("a flat entity reads parent %+v, why %q", argan.Parent, argan.ParentWhy)
	}
	if serums := byName["Serums"]; serums.Children != 1 || serums.Related != 1 {
		t.Errorf("Serums counts %d children and %d related", serums.Children, serums.Related)
	}
	if loose := byName["Loose Notes"]; loose.CanonicalPage != nil {
		t.Errorf("an entity without a page names one: %+v", loose.CanonicalPage)
	}
}

func TestTheEntitiesFilterByNameRootAndParent(t *testing.T) {
	t.Parallel()

	m := twoHierarchies()
	cases := []struct {
		name string
		req  sitemodel.EntitiesRequest
		want []string
	}{
		{name: "a part of the name", req: sitemodel.EntitiesRequest{Query: "ARGAN"}, want: []string{"Argan Oil"}},
		{name: "the roots", req: sitemodel.EntitiesRequest{Roots: true}, want: []string{"Argan Oil", "Loose Notes", "Oils", "Serums"}},
		{name: "a parent's children", req: sitemodel.EntitiesRequest{ParentID: "e-serums"}, want: []string{"Vitamin C"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := names(entityRows(t, m, tc.req)); !slices.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAnEntityReadsBesideThePageTreesParent(t *testing.T) {
	t.Parallel()

	m := twoHierarchies()
	argan, err := m.Service().Entity(t.Context(), sitemodel.EntityRequest{SiteID: siteID, EntityID: "e-argan"})
	if err != nil {
		t.Fatalf("Entity: %v", err)
	}
	switch {
	case len(argan.Parents) != 0 || argan.ParentWhy != sitemodel.NoParentEdge:
		t.Errorf("the flat entity's parents = %+v, why %q", argan.Parents, argan.ParentWhy)
	case argan.Tree == nil || argan.Tree.EntityID != "e-oils" || argan.Tree.Path != "/oils/":
		t.Errorf("the page tree's parent = %+v, want the Oils hub's entity", argan.Tree)
	case argan.CanonicalPage == nil || argan.CanonicalPage.Path != "/oils/argan/" || argan.CanonicalPage.ParentPath != "/oils/":
		t.Errorf("the canonical page = %+v", argan.CanonicalPage)
	case len(argan.Related) != 1 || argan.Related[0].EntityID != "e-vitc" || argan.Related[0].Status != string(graph.StatusProposed):
		t.Errorf("the related edges = %+v, want the proposed one to Vitamin C", argan.Related)
	}

	serums, err := m.Service().Entity(t.Context(), sitemodel.EntityRequest{SiteID: siteID, EntityID: "e-serums"})
	if err != nil {
		t.Fatalf("Entity: %v", err)
	}
	if serums.Tree == nil || serums.Tree.EntityID != "" || serums.Children != 1 {
		t.Errorf("a hub under the front page reads tree %+v and %d children; want a root of the tree", serums.Tree, serums.Children)
	}

	if _, err = m.Service().Entity(t.Context(), sitemodel.EntityRequest{SiteID: siteID, EntityID: "e-missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("an unknown entity answered %v", err)
	}
}
