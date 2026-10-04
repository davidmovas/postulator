package graph_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (h harness) under(t *testing.T, name, parentID string) graph.Entity {
	t.Helper()
	created, err := h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: h.siteID, Name: name, Kind: "topic", ParentID: parentID,
	})
	if err != nil {
		t.Fatalf("CreateEntity %s under %s: %v", name, parentID, err)
	}
	return created.Entity
}

func (h harness) scopeOf(t *testing.T, id string) string {
	t.Helper()
	got, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: id})
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	if got.Entity.ScopeEntityID == nil {
		return ""
	}
	return *got.Entity.ScopeEntityID
}

func TestTheSameNameLivesUnderTwoParents(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	bpc := h.entity(t, "BPC-157", "product")
	tb := h.entity(t, "TB-500", "product")

	first := h.under(t, "Liquid", bpc.ID)
	second := h.under(t, "Liquid", tb.ID)
	if first.ScopeEntityID == nil || *first.ScopeEntityID != bpc.ID || second.ScopeEntityID == nil || *second.ScopeEntityID != tb.ID {
		t.Fatalf("scopes = %v and %v, want each under its product", first.ScopeEntityID, second.ScopeEntityID)
	}

	loaded, err := h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: h.siteID})
	if err != nil || len(loaded.Edges) != 2 {
		t.Fatalf("the graph holds %d edges, %v; want a parent edge for each", len(loaded.Edges), err)
	}

	_, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: "liquid", Kind: "topic", ParentID: bpc.ID})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("the same name twice under one parent = %v, want CONFLICT", err)
	}

	_, err = h.service.CreateEntity(t.Context(), graph.CreateEntityRequest{SiteID: h.siteID, Name: "x", Kind: "topic", ParentID: "missing"})
	if !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("a parent that does not exist = %v, want NOT_FOUND", err)
	}
}

func TestAScopeFollowsTheParentEdges(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	bpc := h.entity(t, "BPC-157", "product")
	tb := h.entity(t, "TB-500", "product")
	loose := h.entity(t, "Liquid", "topic")

	added, err := h.service.AddEdge(t.Context(), graph.AddEdgeRequest{
		SiteID: h.siteID, FromEntityID: loose.ID, ToEntityID: bpc.ID, Kind: "parent", Weight: 1,
	})
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if got := h.scopeOf(t, loose.ID); got != bpc.ID {
		t.Fatalf("after a parent edge the scope is %q, want BPC-157", got)
	}

	if _, err = h.service.MoveEntity(t.Context(), graph.MoveEntityRequest{EntityID: loose.ID, NewParentID: tb.ID}); err != nil {
		t.Fatalf("MoveEntity: %v", err)
	}
	if got := h.scopeOf(t, loose.ID); got != tb.ID {
		t.Fatalf("after a move the scope is %q, want TB-500", got)
	}

	loaded, err := h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	for _, edge := range loaded.Edges {
		if edge.ID != added.Edge.ID {
			if _, err = h.service.DeleteEdge(t.Context(), graph.DeleteEdgeRequest{ID: edge.ID}); err != nil {
				t.Fatalf("DeleteEdge: %v", err)
			}
		}
	}
	if got := h.scopeOf(t, loose.ID); got != "" {
		t.Fatalf("with no parent left the scope is %q, want the top", got)
	}
}

func TestAMoveThatWouldPutTwoNamesUnderOneParentIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	bpc := h.entity(t, "BPC-157", "product")
	tb := h.entity(t, "TB-500", "product")
	h.under(t, "Liquid", bpc.ID)
	other := h.under(t, "Liquid", tb.ID)

	_, err := h.service.MoveEntity(t.Context(), graph.MoveEntityRequest{EntityID: other.ID, NewParentID: bpc.ID})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("MoveEntity = %v, want CONFLICT", err)
	}
	if got := h.scopeOf(t, other.ID); got != tb.ID {
		t.Fatalf("a refused move left the scope at %q, want TB-500", got)
	}
}

func TestDeletingAParentThatWouldLeaveTwoNamesAtTheTopIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	bpc := h.entity(t, "BPC-157", "product")
	h.entity(t, "Liquid", "topic")
	child := h.under(t, "Liquid", bpc.ID)

	if _, err := h.service.DeleteEntity(t.Context(), graph.DeleteEntityRequest{ID: bpc.ID}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("DeleteEntity = %v, want CONFLICT", err)
	}
	if _, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: bpc.ID}); err != nil {
		t.Fatalf("a refused delete removed the entity: %v", err)
	}

	tb := h.entity(t, "TB-500", "product")
	if _, err := h.service.AddEdge(t.Context(), graph.AddEdgeRequest{
		SiteID: h.siteID, FromEntityID: child.ID, ToEntityID: tb.ID, Kind: "parent", Weight: 1,
	}); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if _, err := h.service.DeleteEntity(t.Context(), graph.DeleteEntityRequest{ID: bpc.ID}); err != nil {
		t.Fatalf("DeleteEntity once the child has another parent: %v", err)
	}
	if got := h.scopeOf(t, child.ID); got != tb.ID {
		t.Fatalf("the child of a deleted parent sits under %q, want its other parent", got)
	}
}

func TestABatchPutsTheSameNameUnderTwoParents(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
		SiteID: h.siteID,
		Entities: []graph.EntityInput{
			{Name: "BPC-157", Kind: "product"},
			{Name: "TB-500", Kind: "product"},
			{Name: "Liquid", Kind: "topic", ParentName: "BPC-157"},
			{Name: "Liquid", Kind: "topic", ParentName: "TB-500"},
		},
	})
	if err != nil {
		t.Fatalf("CreateEntities: %v", err)
	}
	if len(created.Entities) != 4 || len(created.Edges) != 2 {
		t.Fatalf("the batch wrote %d entities and %d edges", len(created.Entities), len(created.Edges))
	}

	cases := []struct {
		name     string
		entities []graph.EntityInput
	}{
		{
			name:     "a parent named by a name two entities share",
			entities: []graph.EntityInput{{Name: "Drops", Kind: "topic", ParentName: "Liquid"}},
		},
		{
			name:     "the same name twice under one parent",
			entities: []graph.EntityInput{{Name: "Powder", Kind: "topic", ParentName: "BPC-157"}, {Name: "powder", Kind: "topic", ParentName: "BPC-157"}},
		},
		{
			name:     "a name the parent already has",
			entities: []graph.EntityInput{{Name: "liquid", Kind: "topic", ParentName: "BPC-157"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, batchErr := h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{SiteID: h.siteID, Entities: tc.entities})
			if !errors.IsCode(batchErr, errors.Invalid) {
				t.Fatalf("CreateEntities = %v, want an invalid batch", batchErr)
			}
		})
	}
}

func TestABatchRefusesOnlyTheNamesTheStoreRefuses(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
		SiteID: h.siteID,
		Entities: []graph.EntityInput{
			{Name: "Coffee", Kind: "product"},
			{Name: "Café", Kind: "topic", ParentName: "coffee"},
			{Name: "CAFÉ", Kind: "topic", ParentName: "COFFEE"},
		},
	})
	if err != nil {
		t.Fatalf("CreateEntities of two names the store tells apart: %v", err)
	}
	if len(created.Entities) != 3 || len(created.Edges) != 2 {
		t.Fatalf("the batch wrote %d entities and %d edges, want 3 and 2", len(created.Entities), len(created.Edges))
	}

	_, err = h.service.CreateEntities(t.Context(), graph.CreateEntitiesRequest{
		SiteID: h.siteID, Entities: []graph.EntityInput{{Name: "café", Kind: "topic", ParentName: "Coffee"}},
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("a name the store holds under the parent = %v, want an invalid batch", err)
	}
}
