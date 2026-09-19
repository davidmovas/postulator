package graph_test

import (
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func validEdge() graph.Edge {
	return graph.Edge{
		ID: "6f6f6f6f-6f6f-4f6f-8f6f-6f6f6f6f6f6f", SiteID: siteA, FromEntityID: entB, ToEntityID: entA,
		Kind: graph.EdgeParent, Weight: 0.3, Source: graph.SourceUser, Status: graph.StatusApproved,
		CreatedAt: time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC),
	}
}

func TestNewEdgeNormalises(t *testing.T) {
	t.Parallel()

	parent, err := graph.NewEdge(validEdge())
	if err != nil {
		t.Fatalf("NewEdge: %v", err)
	}
	if parent.Weight != 1 {
		t.Errorf("parent weight = %v, want 1", parent.Weight)
	}
	if parent.FromEntityID != entB || parent.ToEntityID != entA {
		t.Error("a parent edge keeps its direction")
	}

	related := validEdge()
	related.Kind = graph.EdgeRelated
	related.Weight = 0.7
	related.Reason = "  share one subject  "
	normalised, err := graph.NewEdge(related)
	if err != nil {
		t.Fatalf("NewEdge: %v", err)
	}
	if normalised.FromEntityID != entA || normalised.ToEntityID != entB {
		t.Errorf("related edge = %s -> %s, want the lower id first", normalised.FromEntityID, normalised.ToEntityID)
	}
	if normalised.Weight != 0.7 {
		t.Errorf("related weight = %v, want 0.7", normalised.Weight)
	}
	if normalised.Reason != "share one subject" {
		t.Errorf("reason = %q, want it trimmed", normalised.Reason)
	}

	long := validEdge()
	long.Reason = strings.Repeat("é", graph.EdgeReasonMax)
	kept, err := graph.NewEdge(long)
	if err != nil {
		t.Fatalf("NewEdge with a reason at the cap: %v", err)
	}
	if kept.Reason != long.Reason {
		t.Errorf("a reason at the cap must survive")
	}
}

func TestNewEdgeRejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*graph.Edge)
		field  string
	}{
		{name: "no id", mutate: func(e *graph.Edge) { e.ID = "" }, field: "id"},
		{name: "no site", mutate: func(e *graph.Edge) { e.SiteID = "" }, field: "siteId"},
		{name: "no from", mutate: func(e *graph.Edge) { e.FromEntityID = "" }, field: "fromEntityId"},
		{name: "self edge", mutate: func(e *graph.Edge) { e.ToEntityID = e.FromEntityID }, field: "toEntityId"},
		{name: "unknown kind", mutate: func(e *graph.Edge) { e.Kind = "cousin" }, field: "kind"},
		{name: "unknown source", mutate: func(e *graph.Edge) { e.Source = "wind" }, field: "source"},
		{name: "unknown status", mutate: func(e *graph.Edge) { e.Status = "maybe" }, field: "status"},
		{name: "weight above one", mutate: func(e *graph.Edge) { e.Kind = graph.EdgeRelated; e.Weight = 1.01 }, field: "weight"},
		{name: "weight below zero", mutate: func(e *graph.Edge) { e.Kind = graph.EdgeRelated; e.Weight = -0.1 }, field: "weight"},
		{name: "reason too long", mutate: func(e *graph.Edge) { e.Reason = strings.Repeat("x", graph.EdgeReasonMax+1) }, field: "reason"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			edge := validEdge()
			tc.mutate(&edge)
			_, err := graph.NewEdge(edge)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}
