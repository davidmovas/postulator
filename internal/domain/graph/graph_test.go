package graph_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func entity(id, name string) graph.Entity {
	return graph.Entity{ID: id, SiteID: siteA, Name: name, Kind: graph.KindTopic, Source: graph.SourceUser, CreatedAt: stamp, UpdatedAt: stamp}
}

func parent(id, child, parentID string, status graph.EdgeStatus) graph.Edge {
	return graph.Edge{ID: id, SiteID: siteA, FromEntityID: child, ToEntityID: parentID, Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: status, CreatedAt: stamp}
}

func related(id, a, b string, weight float64, status graph.EdgeStatus) graph.Edge {
	return graph.Edge{ID: id, SiteID: siteA, FromEntityID: a, ToEntityID: b, Kind: graph.EdgeRelated, Weight: weight, Source: graph.SourceAI, Status: status, CreatedAt: stamp}
}

func names(entities []graph.Entity) []string {
	out := make([]string, 0, len(entities))
	for i := range entities {
		out = append(out, entities[i].Name)
	}
	return out
}

func diamond(t *testing.T, extra ...graph.Edge) graph.Graph {
	t.Helper()
	edges := []graph.Edge{
		parent("e1", entD, entB, graph.StatusApproved),
		parent("e2", entD, entC, graph.StatusApproved),
		parent("e3", entB, entA, graph.StatusApproved),
		parent("e4", entC, entA, graph.StatusApproved),
		parent("e5", entD, entE, graph.StatusProposed),
	}
	g, err := graph.New([]graph.Entity{entity(entA, "Alpha"), entity(entB, "bravo"), entity(entC, "Charlie"), entity(entD, "delta"), entity(entE, "Echo")}, append(edges, extra...))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return g
}

func TestParentsIsBreadthFirstAndDeduplicated(t *testing.T) {
	t.Parallel()

	g := diamond(t)

	cases := []struct {
		name  string
		id    string
		depth int
		want  []string
	}{
		{name: "direct parents sorted by name", id: entD, depth: 1, want: []string{"bravo", "Charlie"}},
		{name: "grandparent appears once", id: entD, depth: 2, want: []string{"bravo", "Charlie", "Alpha"}},
		{name: "depth beyond the graph", id: entD, depth: 9, want: []string{"bravo", "Charlie", "Alpha"}},
		{name: "root has none", id: entA, depth: 3, want: nil},
		{name: "zero depth", id: entD, depth: 0, want: nil},
		{name: "unknown id", id: "nope", depth: 1, want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := names(g.Parents(tc.id, tc.depth)); !slices.Equal(got, tc.want) {
				t.Errorf("Parents = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParentsShortcutMovesTheAncestorToTheNearestLevel(t *testing.T) {
	t.Parallel()

	g := diamond(t, parent("e6", entD, entA, graph.StatusApproved))
	if got := names(g.Parents(entD, 2)); !slices.Equal(got, []string{"Alpha", "bravo", "Charlie"}) {
		t.Errorf("Parents = %v", got)
	}
}

func TestChildrenRootsAndRelated(t *testing.T) {
	t.Parallel()

	g := diamond(t, related("r1", entB, entC, 0.9, graph.StatusApproved), related("r2", entB, entE, 0.2, graph.StatusApproved), related("r3", entA, entB, 0.95, graph.StatusProposed))

	if got := names(g.Children(entA)); !slices.Equal(got, []string{"bravo", "Charlie"}) {
		t.Errorf("Children(A) = %v", got)
	}
	if got := names(g.Children(entD)); len(got) != 0 {
		t.Errorf("Children(D) = %v, want none", got)
	}
	if got := names(g.Roots()); !slices.Equal(got, []string{"Alpha", "Echo"}) {
		t.Errorf("Roots = %v", got)
	}

	neighbors := g.Related(entB, 0)
	if len(neighbors) != 2 || neighbors[0].Entity.Name != "Charlie" || neighbors[0].Weight != 0.9 || neighbors[1].Entity.Name != "Echo" {
		t.Errorf("Related(B, 0) = %+v", neighbors)
	}
	if got := g.Related(entB, 0.5); len(got) != 1 || got[0].Entity.Name != "Charlie" {
		t.Errorf("Related(B, 0.5) = %+v", got)
	}
	if got := g.Related(entC, 0); len(got) != 1 || got[0].Entity.Name != "bravo" {
		t.Errorf("related edges must be visible from both ends, got %+v", got)
	}
}

func TestAccessors(t *testing.T) {
	t.Parallel()

	g := diamond(t)
	if got := names(g.Entities()); !slices.Equal(got, []string{"Alpha", "bravo", "Charlie", "delta", "Echo"}) {
		t.Errorf("Entities = %v", got)
	}
	if len(g.Edges()) != 5 {
		t.Errorf("Edges = %d, want 5 including the proposed one", len(g.Edges()))
	}
	if e, found := g.Entity(entB); !found || e.Name != "bravo" {
		t.Errorf("Entity(B) = %+v, %v", e, found)
	}
	if _, found := g.Entity("nope"); found {
		t.Error("unknown id must not be found")
	}
}

func TestValidateAcyclic(t *testing.T) {
	t.Parallel()

	if err := diamond(t).ValidateAcyclic(); err != nil {
		t.Fatalf("a DAG must validate: %v", err)
	}

	cyclic, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b"), entity(entC, "c")},
		[]graph.Edge{parent("e1", entA, entB, graph.StatusApproved), parent("e2", entB, entC, graph.StatusApproved), parent("e3", entC, entA, graph.StatusApproved)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = cyclic.ValidateAcyclic()
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
	}
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatal("not a kernel error")
	}
	cycle, ok := kernel.Details["cycle"].([]string)
	if !ok || !slices.Equal(cycle, []string{entA, entB, entC, entA}) {
		t.Errorf("cycle = %v", kernel.Details["cycle"])
	}

	proposed, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b")},
		[]graph.Edge{parent("e1", entA, entB, graph.StatusApproved), parent("e2", entB, entA, graph.StatusProposed)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err = proposed.ValidateAcyclic(); err != nil {
		t.Errorf("a proposed edge must not count: %v", err)
	}
}

func TestNewRejectsInconsistentInput(t *testing.T) {
	t.Parallel()

	other := entity(entB, "b")
	other.SiteID = "another-site"

	cases := []struct {
		name     string
		entities []graph.Entity
		edges    []graph.Edge
		field    string
	}{
		{name: "duplicate entity", entities: []graph.Entity{entity(entA, "a"), entity(entA, "a")}, field: "id"},
		{name: "two sites", entities: []graph.Entity{entity(entA, "a"), other}, field: "siteId"},
		{name: "unknown endpoint", entities: []graph.Entity{entity(entA, "a")}, edges: []graph.Edge{parent("e1", entA, entB, graph.StatusApproved)}, field: "toEntityId"},
		{name: "duplicate edge", entities: []graph.Entity{entity(entA, "a"), entity(entB, "b")}, edges: []graph.Edge{parent("e1", entA, entB, graph.StatusApproved), parent("e1", entB, entA, graph.StatusApproved)}, field: "id"},
		{name: "edge from another site", entities: []graph.Entity{entity(entA, "a"), entity(entB, "b")}, edges: []graph.Edge{{ID: "e1", SiteID: "x", FromEntityID: entA, ToEntityID: entB, Kind: graph.EdgeParent, Weight: 1, Source: graph.SourceUser, Status: graph.StatusApproved}}, field: "siteId"},
		{name: "self edge", entities: []graph.Entity{entity(entA, "a")}, edges: []graph.Edge{parent("e1", entA, entA, graph.StatusApproved)}, field: "toEntityId"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := graph.New(tc.entities, tc.edges)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestEmptyGraph(t *testing.T) {
	t.Parallel()

	g, err := graph.New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(g.Entities()) != 0 || len(g.Roots()) != 0 || g.ValidateAcyclic() != nil {
		t.Error("an empty graph is valid and has nothing in it")
	}
}
