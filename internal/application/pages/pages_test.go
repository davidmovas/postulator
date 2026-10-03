package pages_test

import (
	"encoding/json"
	stderrors "errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type harness struct {
	service  *pages.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	clock    *clock.Fake
	siteID   string
}

func newHarness(t *testing.T) harness {
	t.Helper()

	return newPreviewHarness(t, &recordingIssuer{})
}

func newPreviewHarness(t *testing.T, issuer *recordingIssuer) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	recorder := &applicationtest.Recorder{}
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	return harness{
		service: pages.New(pages.Deps{
			Pages: sqlite.NewPageRepo(store), Links: sqlite.NewPageLinkRepo(store), Entities: sqlite.NewEntityRepo(store),
			Edges: sqlite.NewEdgeRepo(store), Terms: sqlite.NewTermRepo(store), Sites: sqlite.NewSiteRepo(store),
			UnitOfWork: store, Publisher: recorder, Clock: clk, Preview: issuer,
		}),
		store:    store,
		recorder: recorder,
		clock:    clk,
		siteID:   owner.ID,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (h harness) page(t *testing.T, path string, entityID *string) pages.Page {
	t.Helper()
	created, err := h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: path, Title: path, EntityID: entityID})
	if err != nil {
		t.Fatalf("Create %s: %v", path, err)
	}
	return created.Page
}

func (h harness) entity(t *testing.T, name string) graph.Entity {
	t.Helper()
	return sqlitetest.Entity(t, h.store, h.siteID, name)
}

func (h harness) wantEvents(t *testing.T, want ...events.Type) {
	t.Helper()
	got := h.recorder.Events()
	if len(got) != len(want) {
		t.Fatalf("published %d events, want %d: %+v", len(got), len(want), got)
	}
	for i, event := range got {
		if event.Type != want[i] {
			t.Errorf("event[%d] = %s, want %s", i, event.Type, want[i])
		}
	}
	h.recorder.Reset()
}

func evidenceOf(t *testing.T, err error) []pages.Conflict {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	evidence, ok := kernel.Details["evidence"].([]pages.Conflict)
	if !ok {
		t.Fatalf("error %v carries no evidence", err)
	}
	return evidence
}

func TestAProductIsNeverPlacedUnderThePathAboveIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	product, err := h.service.Create(t.Context(), pages.CreateRequest{
		SiteID: h.siteID, Path: "/shop/liquid/", Title: "Liquid", WPType: string(pagemap.WPProduct),
	})
	if err != nil {
		t.Fatalf("Create the product: %v", err)
	}
	shop := h.page(t, "/shop/", nil)

	stored, err := h.service.Get(t.Context(), pages.GetRequest{ID: product.Page.ID})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Page.ParentPageID != nil {
		t.Errorf("the product was adopted by %s; the store decides where a product lives", *stored.Page.ParentPageID)
	}

	later, err := h.service.Create(t.Context(), pages.CreateRequest{
		SiteID: h.siteID, Path: "/shop/powder/", Title: "Powder", WPType: string(pagemap.WPProduct),
	})
	if err != nil {
		t.Fatalf("Create the second product: %v", err)
	}
	if later.Page.ParentPageID != nil {
		t.Errorf("a product created under %s got it as its parent", shop.Path)
	}
}

func TestCreateResolvesTheParentAndAdoptsChildren(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	child := h.page(t, "/Shop/Shoes/", nil)
	if child.Path != "/shop/shoes/" || child.Slug != "shoes" || child.ParentPageID != nil || child.Status != "planned" || child.WPType != "page" {
		t.Errorf("child = %+v", child)
	}
	h.wantEvents(t, events.PagesChanged)

	parent := h.page(t, "/shop/", nil)
	h.wantEvents(t, events.PagesChanged)
	adopted, err := h.service.Get(t.Context(), pages.GetRequest{ID: child.ID})
	if err != nil || adopted.Page.ParentPageID == nil || *adopted.Page.ParentPageID != parent.ID {
		t.Errorf("child after parent creation = %+v, %v", adopted.Page, err)
	}

	sibling := h.page(t, "/shop/bags/", nil)
	if sibling.ParentPageID == nil || *sibling.ParentPageID != parent.ID {
		t.Errorf("sibling parent = %v, want %s", sibling.ParentPageID, parent.ID)
	}

	encoded, err := json.Marshal(sibling)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"siteId"`, `"parentPageId":"` + parent.ID, `"wpType":"page"`, `"wpId":null`, `"entityId":null`, `"wpModifiedAt":null`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}

	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: "missing", Path: "/x/"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.Create(t.Context(), pages.CreateRequest{Path: "/x/"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/a b/"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad path code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/x/", Status: "lost"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/x/", EntityID: ptr("missing")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown entity code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if len(h.recorder.Events()) != 1 {
		t.Errorf("only the sibling creation should have published, got %+v", h.recorder.Events())
	}
}

func TestANewPageAdoptsOnlyTheOrphansRightUnderIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.page(t, "/shop/shoes/", nil)
	h.page(t, "/shop/bags/red/", nil)
	h.page(t, "/elsewhere/news/", nil)
	if _, err := h.service.Create(t.Context(), pages.CreateRequest{
		SiteID: h.siteID, Path: "/shop/liquid/", Title: "Liquid", WPType: string(pagemap.WPProduct),
	}); err != nil {
		t.Fatalf("Create the product: %v", err)
	}
	blog := h.page(t, "/blog/", nil)
	kept := pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: "/shop/kept/", Slug: "kept", WPType: pagemap.WPPage,
		Status: pagemap.StatusPlanned, ParentPageID: &blog.ID,
		CreatedAt: h.clock.Now(), UpdatedAt: h.clock.Now(),
	}
	if err := sqlite.NewPageRepo(h.store).Insert(t.Context(), kept); err != nil {
		t.Fatalf("insert a page that already sits under another: %v", err)
	}

	shop := h.page(t, "/shop/", nil)

	cases := []struct {
		name   string
		path   string
		parent string
	}{
		{name: "an orphan right under it", path: "/shop/shoes/", parent: shop.ID},
		{name: "an orphan two levels down", path: "/shop/bags/red/"},
		{name: "an orphan elsewhere", path: "/elsewhere/news/"},
		{name: "a product right under it", path: "/shop/liquid/"},
		{name: "a page that already sits under another", path: "/shop/kept/", parent: blog.ID},
	}

	listed, err := sqlite.NewPageRepo(h.store).ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	index := pagemap.NewIndex(listed)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stored, found := index.ByPath(tc.path)
			if !found {
				t.Fatalf("no page at %s", tc.path)
			}
			if tc.parent == "" && stored.ParentPageID != nil {
				t.Fatalf("%s sits under %s, want no parent", tc.path, *stored.ParentPageID)
			}
			if tc.parent != "" && (stored.ParentPageID == nil || *stored.ParentPageID != tc.parent) {
				t.Fatalf("%s sits under %v, want %s", tc.path, stored.ParentPageID, tc.parent)
			}
		})
	}
}

func TestCreateRefusesCannibalization(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	boots := h.entity(t, "Boots")
	canonical := h.page(t, "/shoes/", &shoes.ID)
	if _, err := h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: shoes.ID, PageID: canonical.ID}); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}
	h.recorder.Reset()

	_, err := h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/Shoes"})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("duplicate path code = %q, want CONFLICT", errors.CodeOf(err))
	}
	evidence := evidenceOf(t, err)
	if len(evidence) != 1 || evidence[0].Reason != "path_conflict" || evidence[0].PageID != canonical.ID || evidence[0].Path != "/shoes/" {
		t.Errorf("evidence = %+v", evidence)
	}

	_, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/trainers/", EntityID: &shoes.ID})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("second page for a canonical entity code = %q, want CONFLICT", errors.CodeOf(err))
	}
	evidence = evidenceOf(t, err)
	if len(evidence) != 1 || evidence[0].Reason != "same_entity_canonical" || evidence[0].EntityID != shoes.ID {
		t.Errorf("evidence = %+v", evidence)
	}

	rival := h.entity(t, "Rival")
	rival.Keywords = keyword.Of("SHOES")
	if err = sqlite.NewEntityRepo(h.store).Update(t.Context(), rival); err != nil {
		t.Fatalf("give the rival the same keyword: %v", err)
	}
	_, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/rival/", EntityID: &rival.ID})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("same keyword code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if evidence = evidenceOf(t, err); len(evidence) != 1 || evidence[0].Reason != "same_primary_keyword" || evidence[0].EntityID != shoes.ID {
		t.Errorf("evidence = %+v", evidence)
	}

	if _, err = h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/boots/", EntityID: &boots.ID}); err != nil {
		t.Errorf("a distinct entity with a free path is allowed: %v", err)
	}
	if len(h.recorder.Events()) != 1 {
		t.Errorf("refused creations must not publish, got %+v", h.recorder.Events())
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	parent := h.page(t, "/shop/", nil)
	child := h.page(t, "/shop/shoes/", nil)
	other := h.page(t, "/blog/", nil)
	h.recorder.Reset()
	h.clock.Advance(time.Minute)

	updated, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, Title: ptr("Shoes"), Status: ptr("exists"), TemplateID: ptr("")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Page.Title != "Shoes" || updated.Page.Status != "exists" || updated.Page.Path != "/shop/shoes/" || updated.Page.UpdatedAt.String() != "2026-09-18T09:01:00Z" {
		t.Errorf("Update = %+v", updated.Page)
	}
	h.wantEvents(t, events.PagesChanged)

	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: parent.ID, Path: ptr("/store/")}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("renaming a page with descendants code = %q, want CONFLICT", errors.CodeOf(err))
	}
	moved, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, Path: ptr("/blog/shoes/")})
	if err != nil {
		t.Fatalf("Update path: %v", err)
	}
	if moved.Page.Path != "/blog/shoes/" || moved.Page.Slug != "shoes" || moved.Page.ParentPageID == nil || *moved.Page.ParentPageID != other.ID {
		t.Errorf("moved = %+v", moved.Page)
	}
	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, Path: ptr("/blog/")}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("moving onto a taken path code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: child.ID, WPType: ptr("widget")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad wp type code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), pages.UpdateRequest{ID: "missing", Title: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("missing page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestAssignTemplateMovesTheNamedPagesOnly(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	picked := sqlitetest.Template(t, h.store, "Guide")
	first := h.page(t, "/shop/", nil)
	second := h.page(t, "/shop/mugs/", nil)
	untouched := h.page(t, "/about/", nil)
	elsewhere := sqlitetest.Page(t, h.store, sqlitetest.Site(t, h.store, "elsewhere").ID, "/away/")
	h.recorder.Reset()

	if _, err := h.service.AssignTemplate(t.Context(), pages.AssignTemplateRequest{
		SiteID: h.siteID, PageIDs: []string{first.ID, elsewhere.ID}, TemplateID: picked.ID,
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("a page of another site code = %q, want INVALID", errors.CodeOf(err))
	}
	if kept, err := h.service.Get(t.Context(), pages.GetRequest{ID: first.ID}); err != nil || kept.Page.TemplateID != nil {
		t.Fatalf("a refused assignment moved %+v, %v", kept.Page, err)
	}
	h.wantEvents(t)

	assigned, err := h.service.AssignTemplate(t.Context(), pages.AssignTemplateRequest{
		SiteID: h.siteID, PageIDs: []string{first.ID, second.ID}, TemplateID: picked.ID,
	})
	if err != nil || assigned.Changed != 2 {
		t.Fatalf("AssignTemplate = %+v, %v, want both pages moved", assigned, err)
	}
	h.wantEvents(t, events.PagesChanged)

	for _, pageID := range []string{first.ID, second.ID} {
		got, getErr := h.service.Get(t.Context(), pages.GetRequest{ID: pageID})
		if getErr != nil || got.Page.TemplateID == nil || *got.Page.TemplateID != picked.ID {
			t.Errorf("page %s = %+v, %v, want it on the picked template", pageID, got.Page, getErr)
		}
	}
	if got, getErr := h.service.Get(t.Context(), pages.GetRequest{ID: untouched.ID}); getErr != nil || got.Page.TemplateID != nil {
		t.Errorf("a page nobody named = %+v, %v, want it left alone", got.Page, getErr)
	}

	again, err := h.service.AssignTemplate(t.Context(), pages.AssignTemplateRequest{
		SiteID: h.siteID, PageIDs: []string{first.ID}, TemplateID: picked.ID,
	})
	if err != nil || again.Changed != 0 {
		t.Fatalf("AssignTemplate again = %+v, %v, want nothing to change", again, err)
	}
	h.wantEvents(t)
}

func TestMapUnmapAndCanonical(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	boots := h.entity(t, "Boots")
	page := h.page(t, "/shoes/", nil)
	entities := sqlite.NewEntityRepo(h.store)
	h.recorder.Reset()

	mapped, err := h.service.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: shoes.ID})
	if err != nil || mapped.Page.EntityID == nil || *mapped.Page.EntityID != shoes.ID {
		t.Fatalf("MapToEntity = %+v, %v", mapped.Page, err)
	}
	h.wantEvents(t, events.PagesChanged)

	if _, err = h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: boots.ID, PageID: page.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("canonical for a page mapped elsewhere code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: shoes.ID, PageID: page.ID}); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}
	h.wantEvents(t, events.GraphChanged, events.PagesChanged)
	stored, err := entities.Get(t.Context(), shoes.ID)
	if err != nil || stored.CanonicalPageID == nil || *stored.CanonicalPageID != page.ID {
		t.Errorf("canonical after SetCanonical = %v, %v", stored.CanonicalPageID, err)
	}

	second := h.page(t, "/boots/", nil)
	h.recorder.Reset()
	if _, err = h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: boots.ID, PageID: second.ID}); err != nil {
		t.Fatalf("SetCanonical on an unmapped page must map it: %v", err)
	}
	h.wantEvents(t, events.GraphChanged, events.PagesChanged)
	got, err := h.service.Get(t.Context(), pages.GetRequest{ID: second.ID})
	if err != nil || got.Page.EntityID == nil || *got.Page.EntityID != boots.ID {
		t.Errorf("auto-mapped page = %+v, %v", got.Page, err)
	}

	unmapped, err := h.service.Unmap(t.Context(), pages.UnmapRequest{PageID: page.ID})
	if err != nil || unmapped.Page.EntityID != nil {
		t.Fatalf("Unmap = %+v, %v", unmapped.Page, err)
	}
	h.wantEvents(t, events.PagesChanged, events.GraphChanged)
	stored, err = entities.Get(t.Context(), shoes.ID)
	if err != nil || stored.CanonicalPageID != nil {
		t.Errorf("canonical after Unmap = %v, %v", stored.CanonicalPageID, err)
	}
	if _, err = h.service.Unmap(t.Context(), pages.UnmapRequest{PageID: page.ID}); err != nil {
		t.Fatalf("Unmap twice: %v", err)
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("an idempotent unmap must not publish")
	}

	foreignSite := sqlitetest.Site(t, h.store, "blog")
	foreign := sqlitetest.Entity(t, h.store, foreignSite.ID, "Elsewhere")
	if _, err = h.service.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: foreign.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("mapping across sites code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown entity code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: foreign.ID, PageID: page.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("canonical across sites code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Unmap(t.Context(), pages.UnmapRequest{PageID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Unmap missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestListAndTree(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	h.page(t, "/shop/", nil)
	h.clock.Advance(time.Minute)
	h.page(t, "/shop/shoes/", &shoes.ID)
	h.clock.Advance(time.Minute)
	h.page(t, "/blog/", nil)

	byPath, err := h.service.List(t.Context(), pages.ListRequest{ListRequest: dto.ListRequest{Limit: 2, Sort: &dto.Sort{Field: "path"}}, SiteID: h.siteID})
	if err != nil || len(byPath.Items) != 2 || byPath.Items[0].Path != "/blog/" || !byPath.HasMore {
		t.Fatalf("List by path = %+v, %v", byPath, err)
	}
	rest, err := h.service.List(t.Context(), pages.ListRequest{ListRequest: dto.ListRequest{Cursor: string(byPath.Next), Limit: 2, Sort: &dto.Sort{Field: "path"}}, SiteID: h.siteID})
	if err != nil || len(rest.Items) != 1 || rest.Items[0].Path != "/shop/shoes/" {
		t.Errorf("List after = %+v, %v", rest, err)
	}
	unmapped, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, Unmapped: true})
	if err != nil || len(unmapped.Items) != 2 {
		t.Errorf("unmapped = %+v, %v", unmapped, err)
	}
	byEntity, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, EntityID: shoes.ID})
	if err != nil || len(byEntity.Items) != 1 {
		t.Errorf("by entity = %+v, %v", byEntity, err)
	}
	prefixed, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, PathPrefix: "/Shop/", Status: "planned"})
	if err != nil || len(prefixed.Items) != 2 {
		t.Errorf("prefixed = %+v, %v", prefixed, err)
	}
	if _, err = h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, Status: "lost"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.List(t.Context(), pages.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "title"}}, SiteID: h.siteID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.List(t.Context(), pages.ListRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}

	tree, err := h.service.Tree(t.Context(), pages.TreeRequest{SiteID: h.siteID})
	if err != nil || len(tree.Roots) != 2 || tree.Roots[1].Page.Path != "/shop/" || len(tree.Roots[1].Children) != 1 {
		t.Errorf("Tree = %+v, %v", tree, err)
	}
	if _, err = h.service.Tree(t.Context(), pages.TreeRequest{SiteID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.Tree(t.Context(), pages.TreeRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shoes := h.entity(t, "Shoes")
	page := h.page(t, "/shoes/", nil)
	if _, err := h.service.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: shoes.ID, PageID: page.ID}); err != nil {
		t.Fatalf("SetCanonical: %v", err)
	}
	h.recorder.Reset()

	if _, err := h.service.Delete(t.Context(), pages.DeleteRequest{ID: page.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	h.wantEvents(t, events.PagesChanged, events.GraphChanged)
	stored, err := sqlite.NewEntityRepo(h.store).Get(t.Context(), shoes.ID)
	if err != nil || stored.CanonicalPageID != nil {
		t.Errorf("canonical after page delete = %v, %v", stored.CanonicalPageID, err)
	}
	if _, err = h.service.Delete(t.Context(), pages.DeleteRequest{ID: page.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.Get(t.Context(), pages.GetRequest{ID: page.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestDeleteRemovesThePageFromTheSiteWhenAsked(t *testing.T) {
	t.Parallel()

	issuer := &recordingIssuer{}
	h := newPreviewHarness(t, issuer)
	published := h.placed(t, "/coffee/", pagemap.StatusPublished, 42)
	h.recorder.Reset()

	if _, err := h.service.Delete(t.Context(), pages.DeleteRequest{ID: published.ID, OnSite: true}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(issuer.trashed) != 1 || issuer.trashed[0].wpID != 42 || issuer.trashed[0].wpType != "page" {
		t.Fatalf("the site was asked %+v, want the page trashed", issuer.trashed)
	}
	if _, err := h.service.Get(t.Context(), pages.GetRequest{ID: published.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("the page survived the delete: %v", err)
	}
}

func TestDeleteLeavesTheSiteAloneByDefault(t *testing.T) {
	t.Parallel()

	issuer := &recordingIssuer{}
	h := newPreviewHarness(t, issuer)
	published := h.placed(t, "/coffee/", pagemap.StatusPublished, 42)

	if _, err := h.service.Delete(t.Context(), pages.DeleteRequest{ID: published.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(issuer.trashed) != 0 {
		t.Fatalf("the site was asked %+v, want it left alone", issuer.trashed)
	}
}

func TestDeleteKeepsThePageWhenTheSiteRefuses(t *testing.T) {
	t.Parallel()

	issuer := &recordingIssuer{trashErr: errors.New(errors.External, "the site is down")}
	h := newPreviewHarness(t, issuer)
	published := h.placed(t, "/coffee/", pagemap.StatusPublished, 42)

	if _, err := h.service.Delete(t.Context(), pages.DeleteRequest{ID: published.ID, OnSite: true}); !errors.IsCode(err, errors.External) {
		t.Fatalf("Delete = %v, want the site's refusal", err)
	}
	if _, err := h.service.Get(t.Context(), pages.GetRequest{ID: published.ID}); err != nil {
		t.Fatalf("the page was dropped although the site refused: %v", err)
	}
}

func TestDeleteOnSiteLeavesWhatTheStoreKeepsToTheStore(t *testing.T) {
	t.Parallel()

	for _, wpType := range []pagemap.WPType{pagemap.WPProduct, pagemap.WPProductCategory} {
		t.Run(string(wpType), func(t *testing.T) {
			t.Parallel()

			issuer := &recordingIssuer{}
			h := newPreviewHarness(t, issuer)
			placed := h.placed(t, "/product/espresso-machine/", pagemap.StatusPublished, 42)
			repo := sqlite.NewPageRepo(h.store)
			stored, err := repo.Get(t.Context(), placed.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			stored.WPType = wpType
			if err = repo.Update(t.Context(), stored); err != nil {
				t.Fatalf("Update: %v", err)
			}

			_, err = h.service.Delete(t.Context(), pages.DeleteRequest{ID: placed.ID, OnSite: true})
			if !errors.IsCode(err, errors.Invalid) || fieldOf(err) != "onSite" || !strings.Contains(err.Error(), "WooCommerce") {
				t.Fatalf("Delete = %v, want a refusal that sends the delete to WooCommerce", err)
			}
			if len(issuer.trashed) != 0 {
				t.Errorf("the site was asked %+v", issuer.trashed)
			}
			if _, err = h.service.Get(t.Context(), pages.GetRequest{ID: placed.ID}); err != nil {
				t.Errorf("the row was dropped although the delete was refused: %v", err)
			}
		})
	}
}

func TestDeleteOnSiteNeedsAPageThatIsOnTheSite(t *testing.T) {
	t.Parallel()

	issuer := &recordingIssuer{}
	h := newPreviewHarness(t, issuer)
	planned := h.placed(t, "/planned/", pagemap.StatusPlanned, 0)

	_, err := h.service.Delete(t.Context(), pages.DeleteRequest{ID: planned.ID, OnSite: true})
	if !errors.IsCode(err, errors.Invalid) || fieldOf(err) != "wpId" {
		t.Fatalf("Delete = %v, want an invalid error naming wpId", err)
	}
}

func TestCreateAndUpdateCarryTheKeywordsOfThePage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), pages.CreateRequest{
		SiteID: h.siteID, Path: "/shoes/", Title: "Shoes",
		Keywords: []dto.Keyword{{Text: "trail shoes"}, {Text: " running shoes ", Volume: new(9000)}, {Text: "Trail Shoes"}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := keywordTexts(created.Page.Keywords); !slices.Equal(got, []string{"running shoes", "trail shoes"}) {
		t.Fatalf("Create answered %v, want the list trimmed, deduplicated and ordered by volume", got)
	}
	if volume := created.Page.Keywords[0].Volume; volume == nil || *volume != 9000 {
		t.Fatalf("Create answered %+v, want the volume kept", created.Page.Keywords[0])
	}

	renamed, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: created.Page.ID, Title: new("Running shoes")})
	if err != nil {
		t.Fatalf("Update the title: %v", err)
	}
	if got := keywordTexts(renamed.Page.Keywords); !slices.Equal(got, []string{"running shoes", "trail shoes"}) {
		t.Fatalf("an update that leaves the keywords out answered %v, want them kept", got)
	}

	updated, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: created.Page.ID, Keywords: &[]dto.Keyword{{Text: "road shoes"}}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := keywordTexts(updated.Page.Keywords); !slices.Equal(got, []string{"road shoes"}) {
		t.Fatalf("Update answered %v, want the whole list replaced", got)
	}

	cleared, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: created.Page.ID, Keywords: &[]dto.Keyword{}})
	if err != nil {
		t.Fatalf("Update to no keywords: %v", err)
	}
	if cleared.Page.Keywords == nil || len(cleared.Page.Keywords) != 0 {
		t.Fatalf("an empty list answered %#v, want the page to carry none", cleared.Page.Keywords)
	}

	bare, err := h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/socks/"})
	if err != nil {
		t.Fatalf("Create without keywords: %v", err)
	}
	if bare.Page.Keywords == nil {
		t.Fatal("a page without keywords answers null instead of an empty list")
	}
}

func keywordTexts(keywords []dto.Keyword) []string {
	out := make([]string, 0, len(keywords))
	for _, item := range keywords {
		out = append(out, item.Text)
	}
	return out
}

func TestListKeepsThePagesOfAnEntityAndEverythingUnderIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	peptides, bpc, liquid, other := h.entity(t, "Peptides"), h.entity(t, "BPC-157"), h.entity(t, "Liquid"), h.entity(t, "Other")
	edges := sqlite.NewEdgeRepo(h.store)
	for _, pair := range [][2]string{{bpc.ID, peptides.ID}, {liquid.ID, bpc.ID}} {
		if err := edges.Insert(t.Context(), graph.Edge{
			ID: id.New(), SiteID: h.siteID, FromEntityID: pair[0], ToEntityID: pair[1], Kind: graph.EdgeParent, Weight: 1,
			Source: graph.SourceUser, Status: graph.StatusApproved, CreatedAt: sqlitetest.Stamp,
		}); err != nil {
			t.Fatalf("insert the edge: %v", err)
		}
	}
	h.page(t, "/peptides/", &peptides.ID)
	h.page(t, "/peptides/bpc-157/", &bpc.ID)
	h.page(t, "/peptides/bpc-157/liquid/", &liquid.ID)
	h.page(t, "/other/", &other.ID)

	cases := []struct {
		name   string
		entity string
		under  bool
		want   []string
	}{
		{name: "the entity alone", entity: peptides.ID, want: []string{"/peptides/"}},
		{name: "the entity and everything under it", entity: peptides.ID, under: true,
			want: []string{"/peptides/", "/peptides/bpc-157/", "/peptides/bpc-157/liquid/"}},
		{name: "a leaf and nothing more", entity: liquid.ID, under: true, want: []string{"/peptides/bpc-157/liquid/"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listed, err := h.service.List(t.Context(), pages.ListRequest{
				SiteID: h.siteID, EntityID: tc.entity, IncludeDescendants: tc.under,
				ListRequest: dto.ListRequest{Limit: 20, Sort: &dto.Sort{Field: "path"}},
			})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			got := make([]string, 0, len(listed.Items))
			for _, item := range listed.Items {
				got = append(got, item.Path)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("paths = %v, want %v", got, tc.want)
			}
		})
	}

	_, err := h.service.List(t.Context(), pages.ListRequest{SiteID: h.siteID, IncludeDescendants: true})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("List without an entity = %v, want it refused", err)
	}
}

func TestAPageShowsTheNotesItCarries(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	noted, bare := h.page(t, "/bpc-157/", nil), h.page(t, "/tb-500/", nil)
	repo := sqlite.NewPageRepo(h.store)
	stored, err := repo.Get(t.Context(), noted.ID)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	stored.Notes = []pagemap.Note{{Label: "Intent Owner", Text: "Commercial"}}
	if err = repo.Update(t.Context(), stored); err != nil {
		t.Fatalf("store the notes: %v", err)
	}

	cases := []struct {
		name   string
		pageID string
		want   []pages.Note
	}{
		{name: "a page with notes", pageID: noted.ID, want: []pages.Note{{Label: "Intent Owner", Text: "Commercial"}}},
		{name: "a page without any", pageID: bare.ID, want: []pages.Note{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, getErr := h.service.Get(t.Context(), pages.GetRequest{ID: tc.pageID})
			if getErr != nil {
				t.Fatalf("Get: %v", getErr)
			}
			if !slices.Equal(got.Page.Notes, tc.want) || got.Page.Notes == nil {
				t.Fatalf("notes = %#v, want %#v", got.Page.Notes, tc.want)
			}
		})
	}
}
