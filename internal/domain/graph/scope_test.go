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

func scopeOf(t *testing.T, scopes map[string]*string, id string) string {
	t.Helper()
	scope, held := scopes[id]
	if !held {
		t.Fatalf("Scopes answered nothing for %s", id)
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
			if got := scopeOf(t, graph.Scopes(entities, tc.edges), entC); got != tc.want {
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

	entities := []graph.Entity{
		named(entA, "BPC-157", nil), named(entB, "TB-500", nil),
		named(entC, "Liquid", new(entA)), named(entD, "liquid", new(entB)), named(entE, "Powder", new(entA)),
	}
	g, err := graph.New(entities, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := map[string]string{
		entA: "BPC-157", entE: "Powder", entC: "BPC-157 Liquid", entD: "TB-500 liquid", "missing": "",
	}
	for id, want := range cases {
		if got := g.Label(id); got != want {
			t.Errorf("Label(%s) = %q, want %q", id, got, want)
		}
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
