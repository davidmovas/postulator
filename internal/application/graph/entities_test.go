package graph_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type harness struct {
	service  *graph.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	clock    *clock.Fake
	siteID   string
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	recorder := &applicationtest.Recorder{}
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	return harness{
		service:  graph.New(sqlite.NewEntityRepo(store), sqlite.NewEdgeRepo(store), sqlite.NewSiteRepo(store), store, recorder, clk),
		store:    store,
		recorder: recorder,
		clock:    clk,
		siteID:   owner.ID,
	}
}

func ptr[T any](v T) *T {
	return &v
}

func (h harness) entity(t *testing.T, name, kind string) graph.Entity {
	t.Helper()
	created, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: name, Kind: kind, PrimaryKeyword: name})
	if err != nil {
		t.Fatalf("CreateEntity %s: %v", name, err)
	}
	return created.Entity
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
		switch payload := event.Payload.(type) {
		case events.GraphChangedPayload:
			if payload.SiteID != h.siteID {
				t.Errorf("event[%d] site = %s", i, payload.SiteID)
			}
		case events.PagesChangedPayload:
			if payload.SiteID != h.siteID {
				t.Errorf("event[%d] site = %s", i, payload.SiteID)
			}
		default:
			t.Errorf("event[%d] carries %T", i, event.Payload)
		}
	}
	h.recorder.Reset()
}

func TestCreateEntity(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: h.siteID, Name: " Running Shoes ", Kind: "hub", Intent: "commercial", PrimaryKeyword: "running shoes",
		SecondaryKeywords: []string{"trail shoes", "trail shoes"}, Anchors: []graph.Anchor{{Text: "running shoes", Source: "user", Weight: 1}},
	})
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	got := created.Entity
	if got.ID == "" || got.Name != "Running Shoes" || got.Kind != "hub" || got.Source != "user" || len(got.SecondaryKeywords) != 1 || len(got.Anchors) != 1 || got.Score != 0 {
		t.Errorf("CreateEntity = %+v", got)
	}
	h.wantEvents(t, events.GraphChanged)

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"siteId"`, `"primaryKeyword"`, `"secondaryKeywords":["trail shoes"]`, `"anchors":[{"text":"running shoes","source":"user","weight":1}]`, `"canonicalPageId":null`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}

	if _, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: "running shoes", Kind: "topic"}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate name code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: "missing", Name: "x", Kind: "topic"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: "x", Kind: "planet"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown kind code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{Name: "x", Kind: "topic"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("a failed mutation must not publish")
	}
}

func TestUpdateSetAnchorsAndGet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	entity := h.entity(t, "Shoes", "hub")
	h.recorder.Reset()
	h.clock.Advance(time.Minute)

	updated, err := h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: entity.ID, Name: ptr("Footwear"), Intent: ptr("informational"), SecondaryKeywords: ptr([]string{"boots"})})
	if err != nil {
		t.Fatalf("UpdateEntity: %v", err)
	}
	if updated.Entity.Name != "Footwear" || updated.Entity.Kind != "hub" || updated.Entity.PrimaryKeyword != "Shoes" || updated.Entity.Intent != "informational" || updated.Entity.UpdatedAt.String() != "2026-09-18T09:01:00Z" {
		t.Errorf("UpdateEntity = %+v", updated.Entity)
	}
	h.wantEvents(t, events.GraphChanged)

	anchored, err := h.service.SetAnchors(t.Context(), graph.SetAnchorsRequest{EntityID: entity.ID, Anchors: []graph.Anchor{{Text: "footwear", Source: "ai", Weight: 0.5}, {Text: "shoes", Source: "user", Weight: 1}}})
	if err != nil {
		t.Fatalf("SetAnchors: %v", err)
	}
	if len(anchored.Entity.Anchors) != 2 || anchored.Entity.Anchors[0].Text != "footwear" {
		t.Errorf("SetAnchors = %+v", anchored.Entity.Anchors)
	}
	h.wantEvents(t, events.GraphChanged)

	got, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: entity.ID})
	if err != nil || got.Entity.Name != "Footwear" || len(got.Entity.Anchors) != 2 {
		t.Errorf("GetEntity = %+v, %v", got.Entity, err)
	}

	if _, err = h.service.SetAnchors(t.Context(), graph.SetAnchorsRequest{EntityID: entity.ID, Anchors: []graph.Anchor{{Text: "a", Source: "user"}, {Text: "A", Source: "ai"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("repeated anchor code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: "missing", Name: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("missing entity code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: entity.ID, Kind: ptr("planet")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad kind code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("GetEntity missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestDeleteEntityPublishesGraphAndPages(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	entity := h.entity(t, "Shoes", "hub")
	page := sqlitetest.Page(t, h.store, h.siteID, "/shoes/")
	page.EntityID = &entity.ID
	if err := sqlite.NewPageRepo(h.store).Update(t.Context(), page); err != nil {
		t.Fatalf("map the page: %v", err)
	}
	h.recorder.Reset()

	if _, err := h.service.DeleteEntity(t.Context(), graph.DeleteEntityRequest{ID: entity.ID}); err != nil {
		t.Fatalf("DeleteEntity: %v", err)
	}
	h.wantEvents(t, events.GraphChanged, events.PagesChanged)

	unmapped, err := sqlite.NewPageRepo(h.store).Get(t.Context(), page.ID)
	if err != nil || unmapped.EntityID != nil {
		t.Errorf("page after entity delete = %+v, %v", unmapped, err)
	}
	if _, err = h.service.DeleteEntity(t.Context(), graph.DeleteEntityRequest{ID: entity.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestListEntities(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for _, name := range []string{"Boots", "Sandals", "Shoes"} {
		h.entity(t, name, "product")
		h.clock.Advance(time.Minute)
	}
	h.entity(t, "Footwear", "hub")

	page, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Limit: 2}, SiteID: h.siteID})
	if err != nil || len(page.Items) != 2 || page.Items[0].Name != "Boots" || !page.HasMore {
		t.Fatalf("ListEntities = %+v, %v", page, err)
	}
	rest, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Cursor: string(page.Next), Limit: 2}, SiteID: h.siteID})
	if err != nil || len(rest.Items) != 2 || rest.Items[0].Name != "Shoes" || rest.Items[1].Name != "Footwear" {
		t.Errorf("ListEntities after = %+v, %v", rest, err)
	}
	hubs, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: h.siteID, Kind: "hub"})
	if err != nil || len(hubs.Items) != 1 || hubs.Items[0].Name != "Footwear" {
		t.Errorf("hubs = %+v, %v", hubs, err)
	}
	prefixed, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "name"}}, SiteID: h.siteID, NamePrefix: "s"})
	if err != nil || len(prefixed.Items) != 2 || prefixed.Items[0].Name != "Sandals" {
		t.Errorf("prefixed = %+v, %v", prefixed, err)
	}
	mapped, err := h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: h.siteID, HasCanonicalPage: ptr(true)})
	if err != nil || len(mapped.Items) != 0 {
		t.Errorf("mapped = %+v, %v", mapped, err)
	}
	if _, err = h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{SiteID: h.siteID, Kind: "planet"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad kind code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "score"}}, SiteID: h.siteID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEntities(t.Context(), graph.ListEntitiesRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}
}
