package graph_test

import (
	stderrors "errors"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (h harness) edge(t *testing.T, from, to, kind, status string) graph.Edge {
	t.Helper()
	added, err := h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: 0.7, Status: status})
	if err != nil {
		t.Fatalf("AddEdge %s -> %s: %v", from, to, err)
	}
	return added.Edge
}

func TestAddEdgeAndCycles(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	mid := h.entity(t, "Mid", "topic")
	leaf := h.entity(t, "Leaf", "topic")
	h.recorder.Reset()

	first := h.edge(t, mid.ID, hub.ID, "parent", "")
	if first.Status != "approved" || first.Source != "user" || first.Weight != 1 {
		t.Errorf("AddEdge = %+v", first)
	}
	h.wantEvents(t, events.GraphChanged)
	h.edge(t, leaf.ID, mid.ID, "parent", "approved")
	h.recorder.Reset()

	_, err := h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: hub.ID, ToEntityID: leaf.ID, Kind: "parent"})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("cycle code = %q, want INVALID", errors.CodeOf(err))
	}
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatal("not a kernel error")
	}
	cycle, ok := kernel.Details["cycle"].([]string)
	if !ok || !slices.Equal(cycle, []string{hub.ID, leaf.ID, mid.ID, hub.ID}) {
		t.Errorf("cycle = %v", kernel.Details["cycle"])
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("a refused edge must not publish")
	}

	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: hub.ID, ToEntityID: leaf.ID, Kind: "parent", Status: "proposed"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a proposal closing a cycle is refused at once, got %v", err)
	}
	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: mid.ID, ToEntityID: hub.ID, Kind: "parent"}); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("duplicate edge code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: h.siteID, FromEntityID: mid.ID, ToEntityID: "missing", Kind: "parent"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown endpoint code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{SiteID: "missing", FromEntityID: mid.ID, ToEntityID: hub.ID, Kind: "parent"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.AddEdge(t.Context(), graph.AddEdgeRequest{FromEntityID: mid.ID, ToEntityID: hub.ID, Kind: "parent"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}

	related := h.edge(t, mid.ID, leaf.ID, "related", "proposed")
	if related.Kind != "related" || related.Weight != 0.7 || related.Status != "proposed" || related.FromEntityID > related.ToEntityID {
		t.Errorf("related edge = %+v", related)
	}
}

func TestApproveRejectDeleteAndList(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	mid := h.entity(t, "Mid", "topic")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, mid.ID, hub.ID, "parent", "approved")
	leafMid := h.edge(t, leaf.ID, mid.ID, "parent", "proposed")
	closing := h.edge(t, hub.ID, leaf.ID, "parent", "proposed")
	h.recorder.Reset()

	pending, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Status: "proposed"})
	if err != nil || len(pending.Items) != 2 {
		t.Fatalf("ListEdges proposed = %+v, %v", pending, err)
	}
	approved, err := h.service.ApproveEdge(t.Context(), graph.ApproveEdgeRequest{ID: leafMid.ID})
	if err != nil || approved.Edge.Status != "approved" {
		t.Fatalf("ApproveEdge = %+v, %v", approved, err)
	}
	h.wantEvents(t, events.GraphChanged)

	if _, err = h.service.ApproveEdge(t.Context(), graph.ApproveEdgeRequest{ID: closing.ID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("approving the closing edge must be refused, got %v", err)
	}
	rejected, err := h.service.RejectEdge(t.Context(), graph.RejectEdgeRequest{ID: closing.ID})
	if err != nil || rejected.Edge.Status != "rejected" {
		t.Fatalf("RejectEdge = %+v, %v", rejected, err)
	}
	h.wantEvents(t, events.GraphChanged)
	if _, err = h.service.RejectEdge(t.Context(), graph.RejectEdgeRequest{ID: closing.ID}); err != nil {
		t.Fatalf("RejectEdge twice: %v", err)
	}
	if len(h.recorder.Events()) != 0 {
		t.Error("an unchanged status must not publish")
	}

	touching, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{ListRequest: dto.ListRequest{Limit: 1}, SiteID: h.siteID, EntityID: leaf.ID})
	if err != nil || len(touching.Items) != 1 || !touching.HasMore {
		t.Errorf("ListEdges touching = %+v, %v", touching, err)
	}
	parents, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Kind: "parent"})
	if err != nil || len(parents.Items) != 3 {
		t.Errorf("ListEdges parents = %+v, %v", parents, err)
	}
	newest, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "createdAt", Desc: true}, Limit: 1}, SiteID: h.siteID})
	if err != nil || len(newest.Items) != 1 || newest.Items[0].ID != closing.ID {
		t.Errorf("ListEdges newest = %+v, %v", newest, err)
	}
	if _, err = h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Kind: "cousin"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad kind code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEdges(t.Context(), graph.ListEdgesRequest{SiteID: h.siteID, Status: "maybe"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEdges(t.Context(), graph.ListEdgesRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "weight"}}, SiteID: h.siteID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad sort code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.ListEdges(t.Context(), graph.ListEdgesRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}

	if _, err = h.service.DeleteEdge(t.Context(), graph.DeleteEdgeRequest{ID: closing.ID}); err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}
	h.wantEvents(t, events.GraphChanged)
	if _, err = h.service.DeleteEdge(t.Context(), graph.DeleteEdgeRequest{ID: closing.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DeleteEdge twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.ApproveEdge(t.Context(), graph.ApproveEdgeRequest{ID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("ApproveEdge missing code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestLoadGraphAndRecomputeScores(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	mid := h.entity(t, "Mid", "topic")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, mid.ID, hub.ID, "parent", "approved")
	h.edge(t, leaf.ID, mid.ID, "parent", "approved")
	h.recorder.Reset()

	loaded, err := h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: h.siteID})
	if err != nil || len(loaded.Entities) != 3 || len(loaded.Edges) != 2 || loaded.Entities[0].Name != "Hub" {
		t.Fatalf("LoadGraph = %+v, %v", loaded, err)
	}
	domainGraph, err := h.service.Load(t.Context(), h.siteID)
	if err != nil || len(domainGraph.Roots()) != 1 {
		t.Errorf("Load = %d roots, %v", len(domainGraph.Roots()), err)
	}
	if _, err = h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{SiteID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.LoadGraph(t.Context(), graph.LoadGraphRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}

	scored, err := h.service.RecomputeScores(t.Context(), graph.RecomputeScoresRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("RecomputeScores: %v", err)
	}
	if scored.Scores[hub.ID] != 1 || scored.Scores[mid.ID] >= 1 || scored.Scores[leaf.ID] >= scored.Scores[mid.ID] {
		t.Errorf("scores = %v", scored.Scores)
	}
	h.wantEvents(t, events.GraphChanged)

	got, err := h.service.GetEntity(t.Context(), graph.GetEntityRequest{ID: hub.ID})
	if err != nil || got.Entity.Score != 1 {
		t.Errorf("stored hub score = %v, %v", got.Entity.Score, err)
	}
	if _, err = h.service.RecomputeScores(t.Context(), graph.RecomputeScoresRequest{SiteID: "missing"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("unknown site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.RecomputeScores(t.Context(), graph.RecomputeScoresRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("missing site code = %q, want INVALID", errors.CodeOf(err))
	}
}
