package pages_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	peptidesPath = "/peptides/"
	liquidPath   = "/peptides/liquid/"
	powderPath   = "/blog/powder/"
	capsulesPath = "/shop/capsules/"
	shelfPath    = "/product-category/bpc/"
	aboutPath    = "/about/"
	contactPath  = "/contact/"
)

type shelf struct {
	harness
	peptides, bpc, liquid, powder, capsules, company graph.Entity
	pages                                            map[string]pagemap.Page
}

type filed struct {
	categories  []dto.Category
	needsPlugin bool
}

func (h harness) categoryEntity(t *testing.T, name string, siteCategory bool, scope *graph.Entity) graph.Entity {
	t.Helper()
	record := graph.Entity{
		ID: id.New(), SiteID: h.siteID, Name: name, Kind: graph.KindCategory, SiteCategory: siteCategory,
		Keywords: keyword.Of(name), Anchors: []graph.Anchor{}, Source: graph.SourceImport,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if scope != nil {
		record.ScopeID = &scope.ID
	}
	if err := sqlite.NewEntityRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert %s: %v", name, err)
	}
	return record
}

func (h harness) storedPage(t *testing.T, path string, wpType pagemap.WPType, entity *graph.Entity, parent *pagemap.Page) pagemap.Page {
	t.Helper()
	record := pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: path, Slug: pagemap.Slug(path), WPType: wpType, Title: path, H1: path,
		Keywords: keyword.Of(), Notes: []pagemap.Note{}, Status: pagemap.StatusPlanned,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if entity != nil {
		record.EntityID = &entity.ID
	}
	if parent != nil {
		record.ParentPageID = &parent.ID
	}
	if err := sqlite.NewPageRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert %s: %v", path, err)
	}
	return record
}

func (h harness) term(t *testing.T, entity graph.Entity, taxonomy graph.Taxonomy, termID int64) {
	t.Helper()
	err := sqlite.NewTermRepo(h.store).Upsert(t.Context(), graph.Term{
		EntityID: entity.ID, SiteID: h.siteID, Taxonomy: taxonomy, TermID: termID,
		Name: "term " + strconv.FormatInt(termID, 10), SeenAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("Upsert the term %d: %v", termID, err)
	}
}

func (h harness) plugin(t *testing.T, state site.PluginState) {
	t.Helper()
	sites := sqlite.NewSiteRepo(h.store)
	record, err := sites.Get(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("Get the site: %v", err)
	}
	record.Plugin = state
	if err = sites.Update(t.Context(), record); err != nil {
		t.Fatalf("Update the site: %v", err)
	}
}

func newShelf(t *testing.T) shelf {
	t.Helper()

	h := newHarness(t)
	s := shelf{harness: h, pages: map[string]pagemap.Page{}}
	s.peptides = h.categoryEntity(t, "Peptides", true, nil)
	s.bpc = h.categoryEntity(t, "BPC-157", true, &s.peptides)
	s.liquid = h.categoryEntity(t, "Liquid", false, &s.bpc)
	s.powder = h.categoryEntity(t, "Powder", false, &s.bpc)
	s.capsules = h.categoryEntity(t, "Capsules", false, &s.bpc)
	s.company = h.categoryEntity(t, "Company", false, nil)
	h.term(t, s.peptides, graph.TaxonomyCategory, 5)
	h.term(t, s.peptides, graph.TaxonomyProductCategory, 31)
	h.term(t, s.bpc, graph.TaxonomyProductCategory, 32)

	parent := h.storedPage(t, peptidesPath, pagemap.WPPage, &s.peptides, nil)
	s.pages[peptidesPath] = parent
	s.pages[liquidPath] = h.storedPage(t, liquidPath, pagemap.WPPage, &s.liquid, &parent)
	s.pages[powderPath] = h.storedPage(t, powderPath, pagemap.WPPost, &s.powder, nil)
	s.pages[capsulesPath] = h.storedPage(t, capsulesPath, pagemap.WPProduct, &s.capsules, nil)
	s.pages[shelfPath] = h.storedPage(t, shelfPath, pagemap.WPProductCategory, &s.bpc, nil)
	s.pages[aboutPath] = h.storedPage(t, aboutPath, pagemap.WPPage, nil, nil)
	s.pages[contactPath] = h.storedPage(t, contactPath, pagemap.WPPage, &s.company, nil)
	return s
}

func (s shelf) category(entity graph.Entity, termID int64) dto.Category {
	category := dto.Category{EntityID: entity.ID, Name: entity.Name}
	if termID > 0 {
		category.TermID = &termID
	}
	return category
}

func (s shelf) wanted(withTerms bool) map[string]filed {
	term := func(termID int64) int64 {
		if withTerms {
			return termID
		}
		return 0
	}
	return map[string]filed{
		peptidesPath: {categories: []dto.Category{s.category(s.peptides, term(5))}, needsPlugin: true},
		liquidPath:   {categories: []dto.Category{s.category(s.peptides, term(5)), s.category(s.bpc, 0)}, needsPlugin: true},
		powderPath:   {categories: []dto.Category{s.category(s.peptides, term(5)), s.category(s.bpc, 0)}},
		capsulesPath: {categories: []dto.Category{s.category(s.peptides, term(31)), s.category(s.bpc, term(32))}},
		shelfPath:    {categories: []dto.Category{}},
		aboutPath:    {categories: []dto.Category{}},
		contactPath:  {categories: []dto.Category{}},
	}
}

func (s shelf) withoutTerms() *pages.Service {
	return pages.New(pages.Deps{
		Pages: sqlite.NewPageRepo(s.store), Links: sqlite.NewPageLinkRepo(s.store), Entities: sqlite.NewEntityRepo(s.store),
		Edges: sqlite.NewEdgeRepo(s.store), Sites: sqlite.NewSiteRepo(s.store), UnitOfWork: s.store,
		Publisher: &applicationtest.Recorder{}, Clock: s.clock, Preview: &recordingIssuer{},
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

func wantFiled(t *testing.T, got pages.Page, want filed) {
	t.Helper()
	if encoded(t, got.Categories) != encoded(t, want.categories) {
		t.Errorf("%s categories = %s, want %s", got.Path, encoded(t, got.Categories), encoded(t, want.categories))
	}
	if got.CategoriesNeedPlugin != want.needsPlugin {
		t.Errorf("%s categoriesNeedPlugin = %t, want %t", got.Path, got.CategoriesNeedPlugin, want.needsPlugin)
	}
}

func flatten(nodes []pages.TreeNode) []pages.Page {
	out := make([]pages.Page, 0, len(nodes))
	for i := range nodes {
		out = append(out, nodes[i].Page)
		out = append(out, flatten(nodes[i].Children)...)
	}
	return out
}

func TestEveryReadOfAPageCarriesTheCategoriesItIsFiledIn(t *testing.T) {
	t.Parallel()

	s := newShelf(t)

	cases := []struct {
		name      string
		read      func(t *testing.T) []pages.Page
		withTerms bool
	}{
		{
			name: "a page of the list", withTerms: true,
			read: func(t *testing.T) []pages.Page {
				listed, err := s.service.List(t.Context(), pages.ListRequest{SiteID: s.siteID})
				if err != nil {
					t.Fatalf("List: %v", err)
				}
				return listed.Items
			},
		},
		{
			name: "one page at a time", withTerms: true,
			read: func(t *testing.T) []pages.Page {
				out := make([]pages.Page, 0, len(s.pages))
				for _, page := range s.pages {
					got, err := s.service.Get(t.Context(), pages.GetRequest{ID: page.ID})
					if err != nil {
						t.Fatalf("Get %s: %v", page.Path, err)
					}
					out = append(out, got.Page)
				}
				return out
			},
		},
		{
			name: "the tree, a child under its parent", withTerms: true,
			read: func(t *testing.T) []pages.Page {
				tree, err := s.service.Tree(t.Context(), pages.TreeRequest{SiteID: s.siteID})
				if err != nil {
					t.Fatalf("Tree: %v", err)
				}
				if len(tree.Roots) != len(s.pages)-1 {
					t.Fatalf("the tree has %d roots, want %s under %s", len(tree.Roots), liquidPath, peptidesPath)
				}
				return flatten(tree.Roots)
			},
		},
		{
			name: "a service that reads no terms names the chain without term ids",
			read: func(t *testing.T) []pages.Page {
				listed, err := s.withoutTerms().List(t.Context(), pages.ListRequest{SiteID: s.siteID})
				if err != nil {
					t.Fatalf("List: %v", err)
				}
				return listed.Items
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want := s.wanted(tc.withTerms)
			got := tc.read(t)
			if len(got) != len(want) {
				t.Fatalf("read %d pages, want %d", len(got), len(want))
			}
			for i := range got {
				wantFiled(t, got[i], want[got[i].Path])
			}
		})
	}
}

func TestOnlyAWordPressPageNeedsThePluginForItsCategories(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		plugin site.PluginState
		needs  bool
	}{
		{name: "a site without the plugin", plugin: site.PluginState{Capabilities: []string{}}, needs: true},
		{
			name:   "a plugin from before page categories",
			plugin: site.PluginState{Installed: true, Version: "1.2.0", Capabilities: []string{"raw", "preview"}},
			needs:  true,
		},
		{
			name:   "a plugin that files pages in categories",
			plugin: site.PluginState{Installed: true, Version: "1.3.0", Capabilities: []string{"raw", "page_categories"}},
		},
		{
			name:   "a capability left behind by a plugin that is gone",
			plugin: site.PluginState{Version: "1.3.0", Capabilities: []string{"page_categories"}},
			needs:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newShelf(t)
			s.plugin(t, tc.plugin)
			listed, err := s.service.List(t.Context(), pages.ListRequest{SiteID: s.siteID})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			want := map[string]bool{peptidesPath: tc.needs, liquidPath: tc.needs}
			for _, page := range listed.Items {
				if page.CategoriesNeedPlugin != want[page.Path] {
					t.Errorf("%s categoriesNeedPlugin = %t, want %t", page.Path, page.CategoriesNeedPlugin, want[page.Path])
				}
			}
		})
	}
}

func TestEveryWriteAnswersWithTheCategoriesOfThePageItWrote(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		write func(t *testing.T, s shelf) pages.Page
		want  func(s shelf) filed
	}{
		{
			name: "a page created for an entity under two categories",
			write: func(t *testing.T, s shelf) pages.Page {
				nasal := s.categoryEntity(t, "Nasal Spray", false, &s.bpc)
				created, err := s.service.Create(t.Context(), pages.CreateRequest{
					SiteID: s.siteID, Path: "/peptides/nasal-spray/", Title: "Nasal Spray", EntityID: &nasal.ID,
				})
				if err != nil {
					t.Fatalf("Create: %v", err)
				}
				return created.Page
			},
			want: func(s shelf) filed {
				return filed{categories: []dto.Category{s.category(s.peptides, 5), s.category(s.bpc, 0)}, needsPlugin: true}
			},
		},
		{
			name: "a post retitled",
			write: func(t *testing.T, s shelf) pages.Page {
				updated, err := s.service.Update(t.Context(), pages.UpdateRequest{ID: s.pages[powderPath].ID, Title: new("Powder")})
				if err != nil {
					t.Fatalf("Update: %v", err)
				}
				return updated.Page
			},
			want: func(s shelf) filed {
				return filed{categories: []dto.Category{s.category(s.peptides, 5), s.category(s.bpc, 0)}}
			},
		},
		{
			name: "a page mapped to an entity under a category",
			write: func(t *testing.T, s shelf) pages.Page {
				nasal := s.categoryEntity(t, "Nasal Spray", false, &s.bpc)
				mapped, err := s.service.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: s.pages[aboutPath].ID, EntityID: nasal.ID})
				if err != nil {
					t.Fatalf("MapToEntity: %v", err)
				}
				return mapped.Page
			},
			want: func(s shelf) filed {
				return filed{categories: []dto.Category{s.category(s.peptides, 5), s.category(s.bpc, 0)}, needsPlugin: true}
			},
		},
		{
			name: "a page taken off its entity",
			write: func(t *testing.T, s shelf) pages.Page {
				unmapped, err := s.service.Unmap(t.Context(), pages.UnmapRequest{PageID: s.pages[liquidPath].ID})
				if err != nil {
					t.Fatalf("Unmap: %v", err)
				}
				return unmapped.Page
			},
			want: func(shelf) filed { return filed{categories: []dto.Category{}} },
		},
		{
			name: "a canonical page",
			write: func(t *testing.T, s shelf) pages.Page {
				canonical, err := s.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: s.liquid.ID, PageID: s.pages[liquidPath].ID})
				if err != nil {
					t.Fatalf("SetCanonical: %v", err)
				}
				return canonical.Page
			},
			want: func(s shelf) filed {
				return filed{categories: []dto.Category{s.category(s.peptides, 5), s.category(s.bpc, 0)}, needsPlugin: true}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newShelf(t)
			wantFiled(t, tc.write(t, s), tc.want(s))
		})
	}
}
