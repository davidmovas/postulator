package graph_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (h harness) parents(t *testing.T, entityID string) []string {
	t.Helper()

	listed, err := h.service.ListEdges(t.Context(), graph.ListEdgesRequest{
		SiteID: h.siteID, Kind: "parent", Status: "approved",
	})
	if err != nil {
		t.Fatalf("ListEdges: %v", err)
	}

	out := make([]string, 0, len(listed.Items))
	for i := range listed.Items {
		if listed.Items[i].FromEntityID == entityID {
			out = append(out, listed.Items[i].ToEntityID)
		}
	}
	slices.Sort(out)
	return out
}

func TestMoveEntityChangesTheParentInOneWrite(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	other := h.entity(t, "Other", "hub")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, leaf.ID, hub.ID, "parent", "approved")
	h.recorder.Reset()

	moved, err := h.service.MoveEntity(t.Context(), graph.MoveEntityRequest{
		EntityID: leaf.ID, NewParentID: other.ID,
	})
	if err != nil {
		t.Fatalf("MoveEntity: %v", err)
	}
	if moved.Edge.FromEntityID != leaf.ID || moved.Edge.ToEntityID != other.ID {
		t.Fatalf("the new edge = %+v", moved.Edge)
	}
	if moved.Edge.Kind != "parent" || moved.Edge.Status != "approved" || moved.Edge.Weight != 1 {
		t.Fatalf("the new edge = %+v", moved.Edge)
	}
	if len(moved.RemovedEdgeIDs) != 1 {
		t.Fatalf("removed = %v, want the one parent it replaced", moved.RemovedEdgeIDs)
	}
	if got := h.parents(t, leaf.ID); !slices.Equal(got, []string{other.ID}) {
		t.Fatalf("the leaf sits under %v, want only the new parent", got)
	}
	h.wantEvents(t, events.GraphChanged)
}

func TestMoveEntityKeepsBothParentsWhenAsked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	other := h.entity(t, "Other", "hub")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, leaf.ID, hub.ID, "parent", "approved")

	moved, err := h.service.MoveEntity(t.Context(), graph.MoveEntityRequest{
		EntityID: leaf.ID, NewParentID: other.ID, KeepBoth: true,
	})
	if err != nil {
		t.Fatalf("MoveEntity: %v", err)
	}
	if len(moved.RemovedEdgeIDs) != 0 {
		t.Fatalf("removed = %v, want nothing", moved.RemovedEdgeIDs)
	}

	want := []string{hub.ID, other.ID}
	slices.Sort(want)
	if got := h.parents(t, leaf.ID); !slices.Equal(got, want) {
		t.Fatalf("the leaf sits under %v, want %v", got, want)
	}
}

func TestMoveEntityLeavesTheGraphAloneWhenItWouldMakeACycle(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	mid := h.entity(t, "Mid", "topic")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, mid.ID, hub.ID, "parent", "approved")
	h.edge(t, leaf.ID, mid.ID, "parent", "approved")

	_, err := h.service.MoveEntity(t.Context(), graph.MoveEntityRequest{
		EntityID: hub.ID, NewParentID: leaf.ID,
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("a move that closes a loop = %v, want the refusal AddEdge gives", err)
	}
	if got := h.parents(t, mid.ID); !slices.Equal(got, []string{hub.ID}) {
		t.Fatalf("the refused move changed the graph: %v", got)
	}
	if got := h.parents(t, hub.ID); len(got) != 0 {
		t.Fatalf("the refused move left the hub under %v", got)
	}
}

func TestMoveEntityRefusesWhatItCannotDo(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	leaf := h.entity(t, "Leaf", "topic")

	cases := []struct {
		name    string
		request graph.MoveEntityRequest
		want    errors.Code
	}{
		{name: "no entity", request: graph.MoveEntityRequest{NewParentID: hub.ID}, want: errors.Invalid},
		{name: "no parent", request: graph.MoveEntityRequest{EntityID: leaf.ID}, want: errors.Invalid},
		{
			name:    "its own parent",
			request: graph.MoveEntityRequest{EntityID: leaf.ID, NewParentID: leaf.ID},
			want:    errors.Invalid,
		},
		{
			name:    "an entity that is gone",
			request: graph.MoveEntityRequest{EntityID: "00000000-0000-4000-8000-000000000000", NewParentID: hub.ID},
			want:    errors.NotFound,
		},
		{
			name:    "a parent that is gone",
			request: graph.MoveEntityRequest{EntityID: leaf.ID, NewParentID: "00000000-0000-4000-8000-000000000000"},
			want:    errors.NotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := h.service.MoveEntity(t.Context(), tc.request); !errors.IsCode(err, tc.want) {
				t.Fatalf("MoveEntity = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestMoveEntityRunAgainIsQuiet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	hub := h.entity(t, "Hub", "hub")
	other := h.entity(t, "Other", "hub")
	leaf := h.entity(t, "Leaf", "topic")
	h.edge(t, leaf.ID, hub.ID, "parent", "approved")

	first, err := h.service.MoveEntity(t.Context(), graph.MoveEntityRequest{
		EntityID: leaf.ID, NewParentID: other.ID,
	})
	if err != nil {
		t.Fatalf("MoveEntity: %v", err)
	}

	again, err := h.service.MoveEntity(t.Context(), graph.MoveEntityRequest{
		EntityID: leaf.ID, NewParentID: other.ID,
	})
	if err != nil {
		t.Fatalf("MoveEntity again: %v", err)
	}
	if again.Edge.ID != first.Edge.ID || len(again.RemovedEdgeIDs) != 0 {
		t.Fatalf("the second move = %+v, want the edge that is already there", again)
	}
	if got := h.parents(t, leaf.ID); !slices.Equal(got, []string{other.ID}) {
		t.Fatalf("the leaf sits under %v", got)
	}
}

func TestAnAnchorTakesItsSourceFromWhoAsked(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		actor kctx.Actor
		given string
		want  string
	}{
		{name: "a person adds one", actor: kctx.ActorUser, want: "user"},
		{name: "the model proposes one", actor: kctx.ActorAgent, want: "ai"},
		{name: "a schedule writes one", actor: kctx.ActorSchedule, want: "user"},
		{name: "the caller says which", actor: kctx.ActorAgent, given: "user", want: "user"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			leaf := h.entity(t, "Leaf", "topic")
			ctx := kctx.WithActor(t.Context(), tc.actor)

			set, err := h.service.SetAnchors(ctx, graph.SetAnchorsRequest{
				EntityID: leaf.ID,
				Anchors:  []graph.Anchor{{Text: "leaf", Source: tc.given, Weight: 0.5}},
			})
			if err != nil {
				t.Fatalf("SetAnchors: %v", err)
			}
			if len(set.Entity.Anchors) != 1 || set.Entity.Anchors[0].Source != tc.want {
				t.Fatalf("anchors = %+v, want the source %q", set.Entity.Anchors, tc.want)
			}
		})
	}
}
