package graph_test

import (
	"encoding/json"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type categoryTree struct {
	harness
	peptides, healing, bpc, liquid graph.Entity
}

func (h harness) file(t *testing.T, entityID string, filed bool) graph.Entity {
	t.Helper()
	updated, err := h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: entityID, SiteCategory: &filed})
	if err != nil {
		t.Fatalf("UpdateEntity siteCategory=%t: %v", filed, err)
	}
	return updated.Entity
}

func (h harness) term(t *testing.T, entityID string, taxonomy graphdomain.Taxonomy, termID int64) {
	t.Helper()
	err := sqlite.NewTermRepo(h.store).Upsert(t.Context(), graphdomain.Term{
		EntityID: entityID, SiteID: h.siteID, Taxonomy: taxonomy, TermID: termID, Name: "term " + strconv.FormatInt(termID, 10),
		SeenAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("Upsert the term %d: %v", termID, err)
	}
}

func newCategoryTree(t *testing.T) categoryTree {
	t.Helper()

	h := newHarness(t)
	tree := categoryTree{harness: h, peptides: h.entity(t, "Peptides", "hub")}
	tree.healing = h.under(t, "Healing", tree.peptides.ID)
	tree.bpc = h.under(t, "BPC-157", tree.healing.ID)
	tree.liquid = h.under(t, "Liquid", tree.bpc.ID)
	h.file(t, tree.peptides.ID, true)
	h.file(t, tree.bpc.ID, true)
	h.term(t, tree.peptides.ID, graphdomain.TaxonomyCategory, 5)
	h.term(t, tree.bpc.ID, graphdomain.TaxonomyProductCategory, 31)
	h.recorder.Reset()
	return tree
}

func (tree categoryTree) chain(withTerms bool, entityIDs ...string) []dto.Category {
	names := map[string]string{tree.peptides.ID: "Peptides", tree.bpc.ID: "BPC-157"}
	out := make([]dto.Category, 0, len(entityIDs))
	for _, entityID := range entityIDs {
		category := dto.Category{EntityID: entityID, Name: names[entityID]}
		if withTerms && entityID == tree.peptides.ID {
			category.TermID = new(int64(5))
		}
		out = append(out, category)
	}
	return out
}

func (tree categoryTree) wanted(withTerms bool) map[string][]dto.Category {
	return map[string][]dto.Category{
		tree.peptides.ID: tree.chain(withTerms, tree.peptides.ID),
		tree.healing.ID:  tree.chain(withTerms, tree.peptides.ID),
		tree.bpc.ID:      tree.chain(withTerms, tree.peptides.ID, tree.bpc.ID),
		tree.liquid.ID:   tree.chain(withTerms, tree.peptides.ID, tree.bpc.ID),
	}
}

func (tree categoryTree) withoutTerms() *graph.Service {
	return graph.New(graph.Deps{
		Entities: sqlite.NewEntityRepo(tree.store), Edges: sqlite.NewEdgeRepo(tree.store),
		Sites: sqlite.NewSiteRepo(tree.store), Pages: sqlite.NewPageRepo(tree.store),
		UnitOfWork: tree.store, Publisher: &applicationtest.Recorder{}, Clock: tree.clock,
	})
}

func encoded(t *testing.T, v any) string {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(out)
}

func wantCategories(t *testing.T, got graph.Entity, want []dto.Category, flagged bool) {
	t.Helper()
	if encoded(t, got.Categories) != encoded(t, want) {
		t.Errorf("%s categories = %s, want %s", got.Name, encoded(t, got.Categories), encoded(t, want))
	}
	if got.SiteCategory != flagged {
		t.Errorf("%s siteCategory = %t, want %t", got.Name, got.SiteCategory, flagged)
	}
}

func TestEveryReadOfAnEntityCarriesTheCategoriesItsPagesAreFiledIn(t *testing.T) {
	t.Parallel()

	tree := newCategoryTree(t)
	flagged := map[string]bool{tree.peptides.ID: true, tree.bpc.ID: true}
	ids := []string{tree.peptides.ID, tree.healing.ID, tree.bpc.ID, tree.liquid.ID}

	cases := []struct {
		name      string
		read      func(t *testing.T) []graph.Entity
		withTerms bool
	}{
		{
			name: "the whole graph", withTerms: true,
			read: func(t *testing.T) []graph.Entity {
				loaded, err := tree.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: tree.siteID})
				if err != nil {
					t.Fatalf("LoadGraph: %v", err)
				}
				return loaded.Entities
			},
		},
		{
			name: "a page of the entity list", withTerms: true,
			read: func(t *testing.T) []graph.Entity {
				listed, err := tree.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: tree.siteID})
				if err != nil {
					t.Fatalf("ListEntities: %v", err)
				}
				return listed.Items
			},
		},
		{
			name: "one entity at a time", withTerms: true,
			read: func(t *testing.T) []graph.Entity {
				out := make([]graph.Entity, 0, len(ids))
				for _, entityID := range ids {
					got, err := tree.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: entityID})
					if err != nil {
						t.Fatalf("GetEntity: %v", err)
					}
					out = append(out, got.Entity)
				}
				return out
			},
		},
		{
			name: "a service that reads no terms names the chain without term ids",
			read: func(t *testing.T) []graph.Entity {
				loaded, err := tree.withoutTerms().LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: tree.siteID})
				if err != nil {
					t.Fatalf("LoadGraph: %v", err)
				}
				return loaded.Entities
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want := tree.wanted(tc.withTerms)
			got := tc.read(t)
			if len(got) != len(ids) {
				t.Fatalf("read %d entities, want %d", len(got), len(ids))
			}
			for i := range got {
				wantCategories(t, got[i], want[got[i].ID], flagged[got[i].ID])
			}
		})
	}
}

func TestEveryWriteAnswersWithTheCategoriesOfWhatItWrote(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		write func(t *testing.T, tree categoryTree) []graph.Entity
		want  func(tree categoryTree) []dto.Category
	}{
		{
			name: "an entity created under a category",
			write: func(t *testing.T, tree categoryTree) []graph.Entity {
				return []graph.Entity{tree.under(t, "Capsules", tree.bpc.ID)}
			},
			want: func(tree categoryTree) []dto.Category { return tree.chain(true, tree.peptides.ID, tree.bpc.ID) },
		},
		{
			name: "an entity renamed below a category",
			write: func(t *testing.T, tree categoryTree) []graph.Entity {
				updated, err := tree.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: tree.liquid.ID, Name: new("Liquid BPC")})
				if err != nil {
					t.Fatalf("UpdateEntity: %v", err)
				}
				return []graph.Entity{updated.Entity}
			},
			want: func(tree categoryTree) []dto.Category { return tree.chain(true, tree.peptides.ID, tree.bpc.ID) },
		},
		{
			name: "the anchors of an entity between two categories",
			write: func(t *testing.T, tree categoryTree) []graph.Entity {
				anchored, err := tree.service.SetAnchors(t.Context(), graph.SetAnchorsRequest{
					EntityID: tree.healing.ID, Anchors: []graph.Anchor{{Text: "healing peptides", Source: "user"}},
				})
				if err != nil {
					t.Fatalf("SetAnchors: %v", err)
				}
				return []graph.Entity{anchored.Entity}
			},
			want: func(tree categoryTree) []dto.Category { return tree.chain(true, tree.peptides.ID) },
		},
		{
			name: "a batch hung under a category",
			write: func(t *testing.T, tree categoryTree) []graph.Entity {
				created, err := tree.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
					SiteID: tree.siteID, Entities: []graph.EntityInput{
						{Name: "Powder", Kind: "topic", ParentName: "BPC-157"},
						{Name: "Powder Dosing", Kind: "topic", ParentName: "Powder"},
					},
				})
				if err != nil {
					t.Fatalf("CreateEntities: %v", err)
				}
				return created.Entities
			},
			want: func(tree categoryTree) []dto.Category { return tree.chain(true, tree.peptides.ID, tree.bpc.ID) },
		},
		{
			name: "a proposal written as a new root",
			write: func(t *testing.T, tree categoryTree) []graph.Entity {
				applied, err := tree.service.ApplyProposals(t.Context(), graph.ApplyProposalsRequest{
					SiteID: tree.siteID, Entities: []graph.ProposedEntity{{Name: "Nootropics", Kind: "hub", Parent: "Peptides"}},
				})
				if err != nil {
					t.Fatalf("ApplyProposals: %v", err)
				}
				return applied.Entities
			},
			want: func(categoryTree) []dto.Category { return []dto.Category{} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tree := newCategoryTree(t)
			written := tc.write(t, tree)
			if len(written) == 0 {
				t.Fatal("the write answered with no entity")
			}
			for i := range written {
				wantCategories(t, written[i], tc.want(tree), false)
			}
		})
	}
}

func TestTheWordPressCategoryOfAnEntityIsSwitchedAlone(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	parent := h.entity(t, "Peptides", "hub")
	created, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: h.siteID, Name: "Healing", Kind: "category", Intent: "informational", ParentID: parent.ID,
		Keywords: []dto.Keyword{{Text: "healing peptides", Volume: new(900)}, {Text: "peptides for healing"}},
		Anchors:  []graph.Anchor{{Text: "healing peptides", Source: "user", Weight: 1}},
	})
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	page := sqlitetest.Page(t, h.store, h.siteID, "/peptides/healing/")
	if err = sqlite.NewEntityRepo(h.store).SetCanonicalPage(t.Context(), created.Entity.ID, &page.ID, sqlitetest.Stamp); err != nil {
		t.Fatalf("SetCanonicalPage: %v", err)
	}
	before, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: created.Entity.ID})
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	h.recorder.Reset()

	itself := []dto.Category{{EntityID: created.Entity.ID, Name: "Healing"}}
	steps := []struct {
		name    string
		request graph.UpdateEntityRequest
		flagged bool
		want    []dto.Category
	}{
		{name: "filed", request: graph.UpdateEntityRequest{SiteCategory: new(true)}, flagged: true, want: itself},
		{name: "an update that leaves it out keeps it", request: graph.UpdateEntityRequest{Name: new("Healing")}, flagged: true, want: itself},
		{name: "filed twice", request: graph.UpdateEntityRequest{SiteCategory: new(true)}, flagged: true, want: itself},
		{name: "taken off", request: graph.UpdateEntityRequest{SiteCategory: new(false)}, flagged: false, want: []dto.Category{}},
	}

	for _, step := range steps {
		h.clock.Advance(time.Minute)
		step.request.ID = created.Entity.ID
		updated, updateErr := h.service.UpdateEntity(t.Context(), step.request)
		if updateErr != nil {
			t.Fatalf("%s: UpdateEntity: %v", step.name, updateErr)
		}
		h.wantEvents(t, events.GraphChanged)
		stored, getErr := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: created.Entity.ID})
		if getErr != nil {
			t.Fatalf("%s: GetEntity: %v", step.name, getErr)
		}

		for _, got := range []graph.Entity{updated.Entity, stored.Entity} {
			wantCategories(t, got, step.want, step.flagged)
			kept := got
			kept.SiteCategory, kept.Categories, kept.UpdatedAt = before.Entity.SiteCategory, before.Entity.Categories, before.Entity.UpdatedAt
			if encoded(t, kept) != encoded(t, before.Entity) {
				t.Errorf("%s changed more than the category:\n got %s\nwant %s", step.name, encoded(t, kept), encoded(t, before.Entity))
			}
		}
	}

	if !slices.Equal(texts(before.Entity.Keywords), []string{"healing peptides", "peptides for healing"}) ||
		before.Entity.ScopeEntityID == nil || before.Entity.CanonicalPageID == nil {
		t.Fatalf("the entity was not set up with everything a switch must keep: %+v", before.Entity)
	}
}
