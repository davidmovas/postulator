package graph_test

import (
	"math"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

func TestScoreOrdersAHubAboveItsDescendants(t *testing.T) {
	t.Parallel()

	g, err := graph.New(
		[]graph.Entity{entity(entA, "Hub"), entity(entB, "Mid"), entity(entC, "Sibling"), entity(entD, "Leaf"), entity(entE, "Grandchild")},
		[]graph.Edge{
			parent("e1", entB, entA, graph.StatusApproved),
			parent("e2", entC, entA, graph.StatusApproved),
			parent("e3", entD, entA, graph.StatusApproved),
			parent("e4", entE, entB, graph.StatusApproved),
			related("r1", entB, entC, 0.8, graph.StatusApproved),
			parent("e5", entA, entD, graph.StatusRejected),
		},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	scores := g.Score()
	if len(scores) != 5 {
		t.Fatalf("Score has %d entries, want 5", len(scores))
	}
	if scores[entA] != 1 {
		t.Errorf("hub = %v, want exactly 1", scores[entA])
	}
	if !(scores[entA] > scores[entB] && scores[entB] > scores[entC] && scores[entC] > scores[entD]) {
		t.Errorf("order is wrong: hub %v mid %v sibling %v leaf %v", scores[entA], scores[entB], scores[entC], scores[entD])
	}
	if math.Abs(scores[entD]-scores[entE]) > 1e-12 {
		t.Errorf("nodes without incoming links must score alike: leaf %v grandchild %v", scores[entD], scores[entE])
	}
	for id, score := range scores {
		if score <= 0 || score > 1 {
			t.Errorf("%s = %v, want within (0, 1]", id, score)
		}
	}
}

func TestScoreTreatsRelatedEdgesSymmetrically(t *testing.T) {
	t.Parallel()

	g, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b")},
		[]graph.Edge{related("r1", entA, entB, 0.1, graph.StatusApproved)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	scores := g.Score()
	if scores[entA] != 1 || scores[entB] != 1 {
		t.Errorf("symmetric graph must score both ends 1, got %v", scores)
	}
}

func TestScoreIgnoresProposedEdgesAndHandlesEmptyGraphs(t *testing.T) {
	t.Parallel()

	g, err := graph.New(
		[]graph.Entity{entity(entA, "a"), entity(entB, "b")},
		[]graph.Edge{parent("e1", entA, entB, graph.StatusProposed)},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if scores := g.Score(); scores[entA] != 1 || scores[entB] != 1 {
		t.Errorf("without approved edges every node scores 1, got %v", scores)
	}

	empty, err := graph.New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if scores := empty.Score(); len(scores) != 0 {
		t.Errorf("empty graph scores = %v", scores)
	}
}
