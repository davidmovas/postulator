package graph_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type filedGraph struct {
	harness
	peptides, bpc, liquid                                                 category.Category
	hub, topic, product, several, chosen, shelf, unfiled, bare, untouched graph.Entity
}

func (h harness) category(t *testing.T, name string, parent *category.Category) category.Category {
	t.Helper()
	record := category.Category{ID: id.New(), SiteID: h.siteID, Name: name, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp}
	if parent != nil {
		record.ParentID = parent.ID
	}
	record, err := category.New(record)
	if err != nil {
		t.Fatalf("category %s: %v", name, err)
	}
	if err = sqlite.NewCategoryRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the category %s: %v", name, err)
	}
	return record
}

func (h harness) term(t *testing.T, filedUnder category.Category, taxonomy category.Taxonomy, termID int64) {
	t.Helper()
	err := sqlite.NewCategoryTermRepo(h.store).Upsert(t.Context(), category.Term{
		CategoryID: filedUnder.ID, SiteID: h.siteID, Taxonomy: taxonomy, TermID: termID,
		Name: "term " + strconv.FormatInt(termID, 10), SeenAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("Upsert the term %d: %v", termID, err)
	}
}

func (h harness) filedPage(t *testing.T, path string, wpType pagemap.WPType, entity graph.Entity, filedUnder *category.Category) pagemap.Page {
	t.Helper()
	record := pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: path, Slug: pagemap.Slug(path), WPType: wpType, Title: path, H1: path,
		Keywords: keyword.Of(), Notes: []pagemap.Note{}, Status: pagemap.StatusPlanned, EntityID: &entity.ID,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if filedUnder != nil {
		record.CategoryID = filedUnder.ID
	}
	if err := sqlite.NewPageRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert %s: %v", path, err)
	}
	return record
}

func (h harness) canonical(t *testing.T, entity graph.Entity, page pagemap.Page) {
	t.Helper()
	if err := sqlite.NewEntityRepo(h.store).SetCanonicalPage(t.Context(), entity.ID, &page.ID, sqlitetest.Stamp); err != nil {
		t.Fatalf("SetCanonicalPage %s: %v", page.Path, err)
	}
}

func newFiledGraph(t *testing.T) filedGraph {
	t.Helper()

	h := newHarness(t)
	g := filedGraph{harness: h}
	g.peptides = h.category(t, "Peptides", nil)
	g.bpc = h.category(t, "BPC-157", &g.peptides)
	g.liquid = h.category(t, "Liquid", &g.bpc)
	h.term(t, g.peptides, category.TaxonomyCategory, 5)
	h.term(t, g.peptides, category.TaxonomyProductCategory, 31)
	h.term(t, g.bpc, category.TaxonomyProductCategory, 32)

	g.hub = h.entity(t, "Peptides", "hub")
	g.topic = h.under(t, "Liquid BPC", g.hub.ID)
	g.product = h.under(t, "BPC Capsules", g.hub.ID)
	g.several = h.under(t, "BPC Powder", g.hub.ID)
	g.chosen = h.under(t, "BPC Dosing", g.hub.ID)
	g.shelf = h.under(t, "BPC Shelf", g.hub.ID)
	g.unfiled = h.under(t, "BPC Stories", g.hub.ID)
	g.bare = h.under(t, "BPC Research", g.hub.ID)
	g.untouched = h.entity(t, "Company", "hub")

	h.canonical(t, g.hub, h.filedPage(t, "/peptides/", pagemap.WPPage, g.hub, &g.peptides))
	h.filedPage(t, "/peptides/liquid/", pagemap.WPPage, g.topic, &g.liquid)
	h.filedPage(t, "/shop/capsules/", pagemap.WPProduct, g.product, &g.liquid)
	h.filedPage(t, "/blog/powder/", pagemap.WPPost, g.several, &g.bpc)
	h.filedPage(t, "/blog/powder-dosing/", pagemap.WPPost, g.several, &g.liquid)
	h.filedPage(t, "/dosing/table/", pagemap.WPPage, g.chosen, &g.liquid)
	h.canonical(t, g.chosen, h.filedPage(t, "/dosing/", pagemap.WPPost, g.chosen, &g.bpc))
	h.filedPage(t, "/product-category/bpc/", pagemap.WPProductCategory, g.shelf, &g.bpc)
	h.filedPage(t, "/stories/", pagemap.WPPost, g.unfiled, nil)
	h.recorder.Reset()
	return g
}

func filedAs(record category.Category, termID int64) dto.Category {
	out := dto.Category{ID: record.ID, Name: record.Name}
	if termID > 0 {
		out.TermID = &termID
	}
	return out
}

func (g filedGraph) wanted(withTerms bool) map[string][]dto.Category {
	term := func(termID int64) int64 {
		if withTerms {
			return termID
		}
		return 0
	}
	return map[string][]dto.Category{
		g.hub.ID:       {filedAs(g.peptides, term(5))},
		g.topic.ID:     {filedAs(g.peptides, term(5)), filedAs(g.bpc, 0), filedAs(g.liquid, 0)},
		g.product.ID:   {filedAs(g.peptides, term(31)), filedAs(g.bpc, term(32)), filedAs(g.liquid, 0)},
		g.several.ID:   {},
		g.chosen.ID:    {filedAs(g.peptides, term(5)), filedAs(g.bpc, 0)},
		g.shelf.ID:     {},
		g.unfiled.ID:   {},
		g.bare.ID:      {},
		g.untouched.ID: {},
	}
}

func (g filedGraph) unfiledEverywhere() map[string][]dto.Category {
	out := g.wanted(true)
	for entityID := range out {
		out[entityID] = []dto.Category{}
	}
	return out
}

func (g filedGraph) serviceWith(categories, terms bool) *graph.Service {
	deps := graph.Deps{
		Entities: sqlite.NewEntityRepo(g.store), Edges: sqlite.NewEdgeRepo(g.store),
		Sites: sqlite.NewSiteRepo(g.store), Pages: sqlite.NewPageRepo(g.store),
		UnitOfWork: g.store, Publisher: &applicationtest.Recorder{}, Clock: g.clock,
	}
	if categories {
		deps.Categories = sqlite.NewCategoryRepo(g.store)
	}
	if terms {
		deps.CategoryTerms = sqlite.NewCategoryTermRepo(g.store)
	}
	return graph.New(deps)
}

func encoded(t *testing.T, v any) string {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(out)
}

func wantCategories(t *testing.T, got graph.Entity, want []dto.Category) {
	t.Helper()
	if encoded(t, got.Categories) != encoded(t, want) {
		t.Errorf("%s categories = %s, want %s", got.Name, encoded(t, got.Categories), encoded(t, want))
	}
	if strings.Contains(encoded(t, got), "siteCategory") {
		t.Errorf("%s still says whether it is a WordPress category: %s", got.Name, encoded(t, got))
	}
}

func TestEveryReadOfAnEntityCarriesTheCategoriesOfItsPage(t *testing.T) {
	t.Parallel()

	g := newFiledGraph(t)
	ids := []string{g.hub.ID, g.topic.ID, g.product.ID, g.several.ID, g.chosen.ID, g.shelf.ID, g.unfiled.ID, g.bare.ID, g.untouched.ID}
	loadedBy := func(service *graph.Service) func(t *testing.T) []graph.Entity {
		return func(t *testing.T) []graph.Entity {
			loaded, err := service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: g.siteID})
			if err != nil {
				t.Fatalf("LoadGraph: %v", err)
			}
			return loaded.Entities
		}
	}

	cases := []struct {
		name string
		read func(t *testing.T) []graph.Entity
		want map[string][]dto.Category
	}{
		{name: "the whole graph", read: loadedBy(g.service), want: g.wanted(true)},
		{
			name: "a page of the entity list", want: g.wanted(true),
			read: func(t *testing.T) []graph.Entity {
				listed, err := g.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: g.siteID})
				if err != nil {
					t.Fatalf("ListEntities: %v", err)
				}
				return listed.Items
			},
		},
		{
			name: "one entity at a time", want: g.wanted(true),
			read: func(t *testing.T) []graph.Entity {
				out := make([]graph.Entity, 0, len(ids))
				for _, entityID := range ids {
					got, err := g.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: entityID})
					if err != nil {
						t.Fatalf("GetEntity: %v", err)
					}
					out = append(out, got.Entity)
				}
				return out
			},
		},
		{name: "a service that reads no terms names the chain without term ids", read: loadedBy(g.serviceWith(true, false)), want: g.wanted(false)},
		{name: "a service that reads no categories files nothing", read: loadedBy(g.serviceWith(false, true)), want: g.unfiledEverywhere()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.read(t)
			if len(got) != len(ids) {
				t.Fatalf("read %d entities, want %d", len(got), len(ids))
			}
			for i := range got {
				wantCategories(t, got[i], tc.want[got[i].ID])
			}
		})
	}
}

func TestASiteWithoutCategoriesFilesNoEntity(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Peptides", "hub")
	h.canonical(t, hub, h.filedPage(t, "/peptides/", pagemap.WPPage, hub, nil))

	loaded, err := h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if len(loaded.Entities) != 1 || len(loaded.Pages) != 1 {
		t.Fatalf("LoadGraph = %d entities and %d pages, want one of each", len(loaded.Entities), len(loaded.Pages))
	}
	wantCategories(t, loaded.Entities[0], []dto.Category{})
}

func TestEveryWriteAnswersWithTheCategoriesOfWhatItWrote(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		write func(t *testing.T, g filedGraph) []graph.Entity
		want  func(g filedGraph) []dto.Category
	}{
		{
			name: "an entity created, which has no page yet",
			write: func(t *testing.T, g filedGraph) []graph.Entity {
				return []graph.Entity{g.under(t, "Nasal Spray", g.topic.ID)}
			},
			want: func(filedGraph) []dto.Category { return []dto.Category{} },
		},
		{
			name: "an entity renamed",
			write: func(t *testing.T, g filedGraph) []graph.Entity {
				updated, err := g.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: g.topic.ID, Name: new("Liquid BPC-157")})
				if err != nil {
					t.Fatalf("UpdateEntity: %v", err)
				}
				return []graph.Entity{updated.Entity}
			},
			want: func(g filedGraph) []dto.Category {
				return []dto.Category{filedAs(g.peptides, 5), filedAs(g.bpc, 0), filedAs(g.liquid, 0)}
			},
		},
		{
			name: "the anchors of an entity whose page is a product",
			write: func(t *testing.T, g filedGraph) []graph.Entity {
				anchored, err := g.service.SetAnchors(t.Context(), graph.SetAnchorsRequest{
					EntityID: g.product.ID, Anchors: []graph.Anchor{{Text: "bpc capsules", Source: "user"}},
				})
				if err != nil {
					t.Fatalf("SetAnchors: %v", err)
				}
				return []graph.Entity{anchored.Entity}
			},
			want: func(g filedGraph) []dto.Category {
				return []dto.Category{filedAs(g.peptides, 31), filedAs(g.bpc, 32), filedAs(g.liquid, 0)}
			},
		},
		{
			name: "a batch hung under a filed entity",
			write: func(t *testing.T, g filedGraph) []graph.Entity {
				created, err := g.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
					SiteID: g.siteID, Entities: []graph.EntityInput{
						{Name: "Liquid Storage", Kind: "topic", ParentName: "Liquid BPC"},
						{Name: "Liquid Storage Tips", Kind: "topic", ParentName: "Liquid Storage"},
					},
				})
				if err != nil {
					t.Fatalf("CreateEntities: %v", err)
				}
				return created.Entities
			},
			want: func(filedGraph) []dto.Category { return []dto.Category{} },
		},
		{
			name: "a proposal written as a new root",
			write: func(t *testing.T, g filedGraph) []graph.Entity {
				applied, err := g.service.ApplyProposals(t.Context(), graph.ApplyProposalsRequest{
					SiteID: g.siteID, Entities: []graph.ProposedEntity{{Name: "Nootropics", Kind: "hub", Parent: "Peptides"}},
				})
				if err != nil {
					t.Fatalf("ApplyProposals: %v", err)
				}
				return applied.Entities
			},
			want: func(filedGraph) []dto.Category { return []dto.Category{} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			g := newFiledGraph(t)
			written := tc.write(t, g)
			if len(written) == 0 {
				t.Fatal("the write answered with no entity")
			}
			for i := range written {
				wantCategories(t, written[i], tc.want(g))
			}
		})
	}
}
