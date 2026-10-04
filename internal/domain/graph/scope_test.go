package graph_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var edgeStamp = time.Date(2026, time.October, 2, 9, 0, 0, 0, time.UTC)

func named(id, name string, scope *string) graph.Entity {
	return graph.Entity{ID: id, SiteID: siteA, Name: name, Kind: graph.KindTopic, Source: graph.SourceImport, ScopeID: scope}
}

func parentEdge(id, from, to string, status graph.EdgeStatus, minute int) graph.Edge {
	return graph.Edge{
		ID: id, SiteID: siteA, FromEntityID: from, ToEntityID: to, Kind: graph.EdgeParent, Weight: 1,
		Source: graph.SourceUser, Status: status, CreatedAt: edgeStamp.Add(time.Duration(minute) * time.Minute),
	}
}

func settledScope(t *testing.T, entities []graph.Entity, edges []graph.Edge, id string) string {
	t.Helper()

	moved, err := graph.Settle(entities, edges)
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	var scope *string
	settled := slices.Concat(entities, moved)
	for i := range settled {
		if settled[i].ID == id {
			scope = settled[i].ScopeID
		}
	}
	if scope == nil {
		return ""
	}
	return *scope
}

func TestScopesFollowTheParentEdges(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		child graph.Entity
		edges []graph.Edge
		want  string
	}{
		{name: "an entity without a parent sits at the top", child: named(entC, "Liquid", nil), want: ""},
		{
			name:  "its approved parent",
			child: named(entC, "Liquid", nil),
			edges: []graph.Edge{parentEdge("g1", entC, entA, graph.StatusApproved, 0)},
			want:  entA,
		},
		{
			name:  "a parent edge that is only proposed or rejected does not count",
			child: named(entC, "Liquid", nil),
			edges: []graph.Edge{
				parentEdge("g1", entC, entA, graph.StatusProposed, 0), parentEdge("g2", entC, entB, graph.StatusRejected, 1),
			},
			want: "",
		},
		{
			name:  "the parent it already sits under is kept",
			child: named(entC, "Liquid", new(entB)),
			edges: []graph.Edge{
				parentEdge("g1", entC, entA, graph.StatusApproved, 0), parentEdge("g2", entC, entB, graph.StatusApproved, 1),
			},
			want: entB,
		},
		{
			name:  "of several new parents the first one taken",
			child: named(entC, "Liquid", nil),
			edges: []graph.Edge{
				parentEdge("g2", entC, entB, graph.StatusApproved, 5), parentEdge("g1", entC, entA, graph.StatusApproved, 1),
			},
			want: entA,
		},
		{
			name:  "a parent that is gone gives way to one that is left",
			child: named(entC, "Liquid", new(entD)),
			edges: []graph.Edge{parentEdge("g1", entC, entB, graph.StatusApproved, 0)},
			want:  entB,
		},
		{
			name:  "a parent that is gone with no other leaves it at the top",
			child: named(entC, "Liquid", new(entD)),
			want:  "",
		},
		{
			name:  "a related edge is no parent",
			child: named(entC, "Liquid", nil),
			edges: []graph.Edge{{
				ID: "g1", SiteID: siteA, FromEntityID: entA, ToEntityID: entC, Kind: graph.EdgeRelated, Weight: 0.5,
				Source: graph.SourceUser, Status: graph.StatusApproved, CreatedAt: edgeStamp,
			}},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entities := []graph.Entity{named(entA, "BPC-157", nil), named(entB, "TB-500", nil), named(entD, "Gone", nil), tc.child}
			if got := settledScope(t, entities, tc.edges, entC); got != tc.want {
				t.Fatalf("scope = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestScopeClashesAreTwoNamesUnderOneParent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		entities []graph.Entity
		clashes  [][]string
	}{
		{
			name: "the same name under two parents is two entities",
			entities: []graph.Entity{
				named(entA, "BPC-157", nil), named(entB, "TB-500", nil),
				named(entC, "Liquid", new(entA)), named(entD, "Liquid", new(entB)),
			},
		},
		{
			name: "the same name twice under one parent, whatever its case",
			entities: []graph.Entity{
				named(entA, "BPC-157", nil), named(entC, "Liquid", new(entA)), named(entD, "liquid", new(entA)),
			},
			clashes: [][]string{{entC, entD}},
		},
		{
			name:     "the same name twice at the top",
			entities: []graph.Entity{named(entA, "Shoes", nil), named(entB, "shoes", nil)},
			clashes:  [][]string{{entA, entB}},
		},
		{
			name: "names the store tells apart under one parent",
			entities: []graph.Entity{
				named(entA, "Coffee", nil), named(entC, "Café", new(entA)), named(entD, "CAFÉ", new(entA)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			found := graph.ScopeClashes(tc.entities)
			if len(found) != len(tc.clashes) {
				t.Fatalf("clashes = %+v, want %v", found, tc.clashes)
			}
			for i, ids := range tc.clashes {
				if !slices.Equal(found[i].EntityIDs, ids) {
					t.Fatalf("clash %d = %+v, want %v", i, found[i], ids)
				}
			}
		})
	}
}

func TestSettleNamesTheEntitiesWhoseScopeMoves(t *testing.T) {
	t.Parallel()

	entities := []graph.Entity{
		named(entA, "BPC-157", nil), named(entB, "TB-500", nil),
		named(entC, "Liquid", nil), named(entD, "Powder", new(entA)),
	}
	edges := []graph.Edge{
		parentEdge("g1", entC, entA, graph.StatusApproved, 0), parentEdge("g2", entD, entA, graph.StatusApproved, 0),
	}

	moved, err := graph.Settle(entities, edges)
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if len(moved) != 1 || moved[0].ID != entC || moved[0].ScopeID == nil || *moved[0].ScopeID != entA {
		t.Fatalf("moved = %+v, want Liquid alone, now under BPC-157", moved)
	}
}

func TestSettleRefusesTwoNamesUnderOneParent(t *testing.T) {
	t.Parallel()

	entities := []graph.Entity{
		named(entA, "BPC-157", nil), named(entC, "Liquid", new(entA)), named(entD, "liquid", nil),
	}
	edges := []graph.Edge{
		parentEdge("g1", entC, entA, graph.StatusApproved, 0), parentEdge("g2", entD, entA, graph.StatusApproved, 1),
	}

	_, err := graph.Settle(entities, edges)
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Settle = %v, want CONFLICT", err)
	}
	if want := "two entities named Liquid would sit under BPC-157; rename one of them or put it under another parent"; err.Error() != want {
		t.Fatalf("message = %q, want %q", err.Error(), want)
	}

	_, err = graph.Settle([]graph.Entity{named(entA, "Shoes", nil), named(entB, "shoes", nil)}, nil)
	if !errors.IsCode(err, errors.Conflict) || err.Error() != "two entities named Shoes would sit at the top of the graph; rename one of them or put it under a parent" {
		t.Fatalf("Settle at the top = %v", err)
	}
}

func TestALabelNamesTheParentOnlyWhenTheNameIsShared(t *testing.T) {
	t.Parallel()

	const (
		entF = "f6f6f6f6-f6f6-4f6f-8f6f-f6f6f6f6f6f6"
		entG = "a7a7a7a7-a7a7-4a7a-8a7a-a7a7a7a7a7a7"
		entH = "b8b8b8b8-b8b8-4b8b-8b8b-b8b8b8b8b8b8"
		entI = "c9c9c9c9-c9c9-4c9c-8c9c-c9c9c9c9c9c9"
	)

	cases := []struct {
		name     string
		entities []graph.Entity
		want     map[string]string
	}{
		{
			name: "a name used once is its own label",
			entities: []graph.Entity{
				named(entA, "BPC-157", nil), named(entE, "Powder", new(entA)),
			},
			want: map[string]string{entA: "BPC-157", entE: "Powder"},
		},
		{
			name: "a shared name carries its parent, whatever its case",
			entities: []graph.Entity{
				named(entA, "BPC-157", nil), named(entB, "TB-500", nil),
				named(entC, "Liquid", new(entA)), named(entD, "liquid", new(entB)), named(entE, "Powder", new(entA)),
			},
			want: map[string]string{entA: "BPC-157", entE: "Powder", entC: "BPC-157 Liquid", entD: "TB-500 liquid"},
		},
		{
			name: "a shared name at the top has no parent to carry",
			entities: []graph.Entity{
				named(entA, "Liquid", nil), named(entB, "TB-500", nil), named(entC, "Liquid", new(entB)),
			},
			want: map[string]string{entA: "Liquid", entC: "TB-500 Liquid"},
		},
		{
			name: "a parent whose name is shared too is labeled in turn",
			entities: []graph.Entity{
				named(entA, "BPC-157", nil), named(entB, "TB-500", nil),
				named(entC, "Generic", new(entA)), named(entD, "Generic", new(entB)),
				named(entE, "Liquid", new(entC)), named(entF, "Liquid", new(entD)),
			},
			want: map[string]string{entE: "BPC-157 Generic Liquid", entF: "TB-500 Generic Liquid"},
		},
		{
			name: "a parent that is gone leaves the name alone",
			entities: []graph.Entity{
				named(entG, "Liquid", new(entH)), named(entI, "Liquid", nil),
			},
			want: map[string]string{entG: "Liquid", entI: "Liquid"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			labels := graph.Labels(tc.entities)
			g, err := graph.New(tc.entities, nil)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			for id, want := range tc.want {
				if got := labels[id]; got != want {
					t.Errorf("Labels[%s] = %q, want %q", id, got, want)
				}
				if got := g.Label(id); got != want {
					t.Errorf("Label(%s) = %q, want %q", id, got, want)
				}
			}
			if got := g.Label("missing"); got != "" {
				t.Errorf("Label(missing) = %q, want nothing", got)
			}
		})
	}
}

func TestNewEntityRefusesAScopeItCannotHave(t *testing.T) {
	t.Parallel()

	for name, scope := range map[string]*string{"empty": new(""), "itself": new(entA)} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			entity := validEntity()
			entity.ScopeID = scope
			if _, err := graph.NewEntity(entity); err == nil || fieldOf(t, err) != "scopeId" {
				t.Fatalf("NewEntity = %v, want a refusal of the scope", err)
			}
		})
	}
}
