package sync_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	siteapp "github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func seedGuide(server *wptest.Server) {
	guide := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Guide", Slug: "guide", Status: "publish", Content: "<p>The guide.</p>",
		Crumbs: []wptest.Crumb{{Name: "Home", Path: "/"}, {Name: "Guide", Path: "/guide/"}},
	})[0]
	server.Seed(
		wptest.Item{
			Type: wptest.TypePage, Title: "Setup", Slug: "setup", Parent: guide.ID, Status: "publish", Content: "<p>Set it up.</p>",
			Crumbs: []wptest.Crumb{{Name: "Home", Path: "/"}, {Name: "Guide", Path: "/guide/"}, {Name: "Setup", Path: "/guide/setup/"}},
		},
		wptest.Item{
			Type: wptest.TypePage, Title: "Care", Slug: "care", Parent: guide.ID, Status: "publish", Content: "<p>Care for it.</p>",
			Crumbs: []wptest.Crumb{{Name: "Home", Path: "/"}, {Name: "Guide", Path: "/guide/"}, {Name: "Care", Path: "/guide/care/"}},
		},
	)
}

func (r syncRig) entityNames(t *testing.T) []string {
	t.Helper()
	listed, err := sqlite.NewEntityRepo(r.store).ListBySite(t.Context(), r.owner.ID)
	if err != nil {
		t.Fatalf("list the entities: %v", err)
	}
	names := make([]string, 0, len(listed))
	for i := range listed {
		names = append(names, listed[i].Name)
	}
	slices.Sort(names)
	return names
}

func paths(pages []pagemap.Page) []string {
	out := make([]string, 0, len(pages))
	for i := range pages {
		out = append(out, pages[i].Path)
	}
	slices.Sort(out)
	return out
}

func mapped(pages []pagemap.Page) int {
	count := 0
	for i := range pages {
		if pages[i].EntityID != nil {
			count++
		}
	}
	return count
}

func TestASiteClearedOfItsDataIsReadAgainByTheNextSync(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		req  siteapp.ClearDataRequest
	}{
		{name: "the model and the graph", req: siteapp.ClearDataRequest{Model: true, Graph: true}},
		{name: "the model alone, the graph kept", req: siteapp.ClearDataRequest{Model: true}},
		{name: "the graph alone, the pages kept", req: siteapp.ClearDataRequest{Graph: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, wptest.WithoutCommerce())
			seedGuide(server)
			rig := newSyncRig(t, server, steps.DefaultBatchSize)
			first := rig.sync(t)
			built := rig.entityNames(t)
			if len(first) != 3 || len(built) == 0 || mapped(first) == 0 {
				t.Fatalf("the first sync read %v with entities %v and %d mapped, want three pages carrying the breadcrumbs' entities",
					paths(first), built, mapped(first))
			}

			service := siteapp.New(siteapp.Deps{
				Store: sqlite.NewSiteRepo(rig.store), UoW: rig.store, Data: sqlite.NewSiteDataRepo(rig.store),
				Publisher: silentBus{}, Clock: clock.System{},
			})
			req := tc.req
			req.ID = rig.owner.ID
			if _, err := service.ClearData(t.Context(), req); err != nil {
				t.Fatalf("ClearData = %v", err)
			}

			again := rig.sync(t)
			if !slices.Equal(paths(again), paths(first)) {
				t.Errorf("the next sync read %v, want %v again", paths(again), paths(first))
			}
			if rebuilt := rig.entityNames(t); !slices.Equal(rebuilt, built) {
				t.Errorf("the entities after the next sync are %v, want %v", rebuilt, built)
			}
			if mapped(again) != mapped(first) {
				t.Errorf("%d pages carry an entity after the next sync, want %d", mapped(again), mapped(first))
			}
		})
	}
}
