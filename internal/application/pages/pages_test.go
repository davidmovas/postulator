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
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
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
		service: pages.New(sqlite.NewPageRepo(store), sqlite.NewPageLinkRepo(store), sqlite.NewEntityRepo(store),
			sqlite.NewSiteRepo(store), store, recorder, clk, issuer),
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
	rival.PrimaryKeyword = "SHOES"
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
		SiteID: h.siteID, Path: "/shoes/", Title: "Shoes", PrimaryKeyword: " running shoes ", Keywords: []string{"trail shoes", "Trail Shoes", ""},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Page.PrimaryKeyword != "running shoes" || !slices.Equal(created.Page.Keywords, []string{"trail shoes"}) {
		t.Fatalf("Create answered %q %v", created.Page.PrimaryKeyword, created.Page.Keywords)
	}

	updated, err := h.service.Update(t.Context(), pages.UpdateRequest{ID: created.Page.ID, Keywords: []string{"road shoes"}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Page.PrimaryKeyword != "running shoes" || !slices.Equal(updated.Page.Keywords, []string{"road shoes"}) {
		t.Fatalf("Update answered %q %v, want the primary kept and the list replaced", updated.Page.PrimaryKeyword, updated.Page.Keywords)
	}

	bare, err := h.service.Create(t.Context(), pages.CreateRequest{SiteID: h.siteID, Path: "/socks/"})
	if err != nil {
		t.Fatalf("Create without keywords: %v", err)
	}
	if bare.Page.Keywords == nil {
		t.Fatal("a page without keywords answers null instead of an empty list")
	}
}
