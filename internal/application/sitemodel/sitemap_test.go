package sitemodel_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	hubs        = 30
	childrenOf  = 99
	pageOfNodes = 50
)

func wide() *sitemodeltest.Memory {
	pages := make([]pagemap.Page, 0, 1+hubs*(1+childrenOf))
	pages = append(pages, sitePage("home", "/", "Home", 1, pagemap.WPPage, pagetype.Entity, ""))
	bodies := make([]block.Snapshot, 0, hubs*childrenOf)
	for h := range hubs {
		hubID := fmt.Sprintf("hub-%02d", h)
		pages = append(pages, sitePage(hubID, fmt.Sprintf("/hub-%02d/", h), fmt.Sprintf("Hub %02d", h), int64(100+h), pagemap.WPPage, pagetype.Hub, "home"))
		for c := range childrenOf {
			id := fmt.Sprintf("page-%02d-%02d", h, c)
			pages = append(pages, sitePage(id, fmt.Sprintf("/hub-%02d/page-%02d/", h, c), fmt.Sprintf("Page %02d %02d", h, c),
				int64(10000+h*100+c), pagemap.WPPage, pagetype.Entity, hubID))
			bodies = append(bodies, block.Snapshot{PageID: id, Side: block.Current, Raw: "<p>The client's words.</p>", Kind: block.Classic})
		}
	}
	links := make([]pagemap.PageLink, 0, 7)
	contextual := func(from, to string) pagemap.PageLink {
		return pagemap.PageLink{
			ID: from + "-" + to, SiteID: siteID, FromPageID: from, ToPageID: &to, ToURL: "/" + to + "/",
			Type: pagemap.LinkContextual, Source: pagemap.SourceRaw,
		}
	}
	for c := 1; c <= 4; c++ {
		links = append(links, contextual(fmt.Sprintf("page-07-%02d", c), "page-07-00"))
	}
	links = append(links, contextual("page-07-00", "page-07-05"), contextual("page-07-00", "page-07-06"))
	navigation := contextual("page-07-07", "page-07-00")
	navigation.Type = pagemap.LinkNavigation
	links = append(links, navigation)
	return &sitemodeltest.Memory{
		Site:  site.Site{ID: siteID, Name: "Shop", BaseURL: "https://shop.example.com", Settings: site.DefaultSettings()},
		Pages: pages, Types: presetSet(), Bodies: bodies, Links: links,
	}
}

func siteMap(t *testing.T, m *sitemodeltest.Memory, req sitemodel.MapRequest) sitemodel.MapView {
	t.Helper()

	req.SiteID = m.Site.ID
	got, err := m.Service().SiteMap(t.Context(), req)
	if err != nil {
		t.Fatalf("SiteMap(%+v): %v", req, err)
	}
	return got
}

func nodePaths(nodes []contextpack.MapNode) []string {
	out := make([]string, 0, len(nodes))
	for i := range nodes {
		out = append(out, nodes[i].Page.Path)
	}
	return out
}

func TestTheSiteMapReadsAPageOfChildrenAtATime(t *testing.T) {
	t.Parallel()

	m := wide()
	if len(m.Pages) != 1+hubs+hubs*childrenOf {
		t.Fatalf("the store holds %d pages", len(m.Pages))
	}
	root := siteMap(t, m, sitemodel.MapRequest{})
	if got := nodePaths(root.Nodes.Items); !slices.Equal(got, []string{"/"}) || root.Parent != nil {
		t.Fatalf("the roots = %v, want the front page alone", got)
	}
	if root.Nodes.Items[0].Descendants != hubs+hubs*childrenOf {
		t.Errorf("the front page counts %d descendants", root.Nodes.Items[0].Descendants)
	}

	first := siteMap(t, m, sitemodel.MapRequest{ParentID: "hub-07", Limit: pageOfNodes})
	if len(first.Nodes.Items) != pageOfNodes || !first.Nodes.HasMore || first.Parent == nil || first.Parent.Page.PageID != "hub-07" {
		t.Fatalf("the first page = %d nodes, more %v, parent %+v", len(first.Nodes.Items), first.Nodes.HasMore, first.Parent)
	}
	if trail := first.Trail; len(trail) != 1 || trail[0].Path != "/" {
		t.Errorf("the parent's trail = %+v, want the front page", trail)
	}
	asked := m.Asked()
	if len(asked) != 2 || len(asked[1]) != pageOfNodes {
		t.Fatalf("bodies were asked for %v, want one read per call holding that call's nodes", asked)
	}
	for _, id := range asked[1] {
		if !slices.ContainsFunc(first.Nodes.Items, func(node contextpack.MapNode) bool { return node.Page.PageID == id }) {
			t.Errorf("a body outside the page of nodes was read: %s", id)
		}
	}
	if node := first.Nodes.Items[0]; node.Body != contextpack.BodyClient || node.LinksIn != 4 || node.LinksOut != 2 {
		t.Errorf("the first child = %+v, want the client's text and the audit's links", node)
	}

	rest := siteMap(t, m, sitemodel.MapRequest{ParentID: "hub-07", Limit: pageOfNodes, Cursor: first.Nodes.Next})
	if len(rest.Nodes.Items) != childrenOf-pageOfNodes || rest.Nodes.HasMore {
		t.Errorf("the second page = %d nodes, more %v", len(rest.Nodes.Items), rest.Nodes.HasMore)
	}
	all := append(nodePaths(first.Nodes.Items), nodePaths(rest.Nodes.Items)...)
	if !slices.IsSorted(all) || len(all) != childrenOf {
		t.Errorf("the hub's children come back %d, sorted %v", len(all), slices.IsSorted(all))
	}
	if m.Reads() != 3 {
		t.Errorf("three calls read in %d units, want one each", m.Reads())
	}
}

func TestTheSiteMapSearchesWithEachMatchsTrail(t *testing.T) {
	t.Parallel()

	m := wide()
	found := siteMap(t, m, sitemodel.MapRequest{Search: "page 07 1"})
	if len(found.Nodes.Items) != 10 {
		t.Fatalf("found %v, want the ten pages 07 1x", nodePaths(found.Nodes.Items))
	}
	trail := found.Nodes.Items[0].Trail
	if len(trail) != 2 || trail[0].Path != "/" || trail[1].Path != "/hub-07/" {
		t.Errorf("the first match's trail = %+v, want Home › Hub 07", trail)
	}
	typed := siteMap(t, m, sitemodel.MapRequest{PageTypeID: typeOf(pagetype.Hub), Limit: 200})
	if len(typed.Nodes.Items) != hubs {
		t.Errorf("hubs = %d, want %d", len(typed.Nodes.Items), hubs)
	}
	inside := siteMap(t, m, sitemodel.MapRequest{ParentID: "hub-03", Search: "page"})
	for _, node := range inside.Nodes.Items {
		if len(node.Trail) < 2 || node.Trail[1].PageID != "hub-03" {
			t.Errorf("%s was found outside the subtree", node.Page.Path)
		}
	}
	if len(inside.Nodes.Items) != childrenOf || inside.Nodes.HasMore {
		t.Errorf("the subtree search found %d, want the hub's %d children on one default page", len(inside.Nodes.Items), childrenOf)
	}
}

func TestTheMapPlacesPagesAndScopesTheirSubtrees(t *testing.T) {
	t.Parallel()

	m := wide()
	service := m.Service()
	hub, err := service.Scope(t.Context(), siteID, "hub-07")
	if err != nil || !strings.HasSuffix(hub, "hub-07/") || !strings.HasPrefix(hub, "home/") {
		t.Fatalf("the scope of a hub = %q, %v; want its root-first trail", hub, err)
	}
	if _, err = service.Scope(t.Context(), siteID, "nobody"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("the scope of a page the map lacks = %v, want NotFound", err)
	}
	places, err := service.Places(t.Context(), siteID, []string{"page-07-00", "nobody"})
	if err != nil || len(places) != 1 {
		t.Fatalf("Places = %+v, %v; want the one page the site holds", places, err)
	}
	if place := places["page-07-00"]; place.Ref.Path != "/hub-07/page-00/" || place.TypeID != typeOf(pagetype.Entity) || !place.Ref.OnSite {
		t.Errorf("the place of a page = %+v", place)
	}
	if none, noneErr := service.Places(t.Context(), siteID, nil); noneErr != nil || len(none) != 0 {
		t.Errorf("the places of no page = %+v, %v", none, noneErr)
	}
	kind, found, err := service.PageType(t.Context(), siteID, typeOf(pagetype.Hub))
	if err != nil || !found || kind.Preset != pagetype.Hub {
		t.Errorf("PageType = %+v, %v, %v", kind, found, err)
	}
	if _, found, err = service.PageType(t.Context(), siteID, "type-nobody"); err != nil || found {
		t.Errorf("a type the site lacks = %v, %v", found, err)
	}
	if err = service.Refresh(t.Context(), siteID); err != nil {
		t.Errorf("refreshing a fresh map = %v", err)
	}
}

func TestTheSiteMapRefusesWhatItCannotPlace(t *testing.T) {
	t.Parallel()

	m := wide()
	if _, err := m.Service().SiteMap(t.Context(), sitemodel.MapRequest{SiteID: siteID, ParentID: "nobody"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("a parent the map lacks = %v, want NotFound", err)
	}
	if _, err := m.Service().SiteMap(t.Context(), sitemodel.MapRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("no site = %v, want Invalid", err)
	}
	if _, err := m.Service().SiteMap(t.Context(), sitemodel.MapRequest{SiteID: siteID, Cursor: paging.Cursor("garbage")}); err == nil {
		t.Error("a cursor nobody issued was accepted")
	}
}

func TestEveryStandPageIsOneNodeOfItsMap(t *testing.T) {
	t.Parallel()

	for _, name := range sitemodeltest.Stands() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			m, err := sitemodeltest.Stand(name)
			if err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
			walked := func(limit int) []contextpack.MapNode {
				out := []contextpack.MapNode{}
				var walk func(parent string)
				walk = func(parent string) {
					var cursor paging.Cursor
					for {
						view := siteMap(t, m, sitemodel.MapRequest{ParentID: parent, Cursor: cursor, Limit: limit})
						for _, node := range view.Nodes.Items {
							out = append(out, node)
							walk(node.Page.PageID)
						}
						if !view.Nodes.HasMore {
							return
						}
						cursor = view.Nodes.Next
					}
				}
				walk("")
				return out
			}
			whole := walked(200)
			if one := walked(1); !slices.Equal(nodePaths(one), nodePaths(whole)) {
				t.Errorf("a node at a time walks %v, a page at a time %v", nodePaths(one), nodePaths(whole))
			}
			seen := map[string]int{}
			folded := 0
			for _, node := range whole {
				seen[node.Page.PageID]++
				if node.Twin != nil {
					seen[node.Twin.PageID]++
					folded++
				}
			}
			live := 0
			for i := range m.Pages {
				if m.Pages[i].Status != pagemap.StatusArchived {
					live++
				}
			}
			if len(seen) != live {
				t.Errorf("the map holds %d pages and their twins, want every one of the %d live pages", len(seen), live)
			}
			for id, times := range seen {
				if times != 1 {
					t.Errorf("%s is in the map %d times", id, times)
				}
			}
			root := siteMap(t, m, sitemodel.MapRequest{})
			t.Logf("%s: %d live pages, %d store categories folded into their hubs, roots %v", name, live, folded, nodePaths(root.Nodes.Items))
		})
	}
}
