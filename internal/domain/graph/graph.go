package graph

import (
	"cmp"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type neighbor struct {
	id     string
	weight float64
}

type Neighbor struct {
	Entity Entity
	Weight float64
}

type Graph struct {
	entities map[string]Entity
	parents  map[string][]string
	children map[string][]string
	related  map[string][]neighbor
	ordered  []Entity
	edges    []Edge
}

func New(entities []Entity, edges []Edge) (Graph, error) {
	g := Graph{
		entities: make(map[string]Entity, len(entities)),
		parents:  make(map[string][]string),
		children: make(map[string][]string),
		related:  make(map[string][]neighbor),
		ordered:  make([]Entity, 0, len(entities)),
		edges:    make([]Edge, 0, len(edges)),
	}

	siteID := ""
	for i := range entities {
		e := &entities[i]
		if _, dup := g.entities[e.ID]; dup {
			return Graph{}, invalid("entity id is repeated", "id").WithDetail("entityId", e.ID)
		}
		if siteID == "" {
			siteID = e.SiteID
		} else if e.SiteID != siteID {
			return Graph{}, invalid("entities span more than one site", "siteId").WithDetail("entityId", e.ID)
		}
		g.entities[e.ID] = *e
		g.ordered = append(g.ordered, *e)
	}
	slices.SortFunc(g.ordered, byName)

	seen := make(map[string]struct{}, len(edges))
	for i := range edges {
		e, err := NewEdge(edges[i])
		if err != nil {
			return Graph{}, err
		}
		if _, dup := seen[e.ID]; dup {
			return Graph{}, invalid("edge id is repeated", "id").WithDetail("edgeId", e.ID)
		}
		if e.SiteID != siteID {
			return Graph{}, invalid("edge belongs to another site", "siteId").WithDetail("edgeId", e.ID)
		}
		if _, ok := g.entities[e.FromEntityID]; !ok {
			return Graph{}, invalid("edge references an unknown entity", "fromEntityId").WithDetail("edgeId", e.ID).WithDetail("entityId", e.FromEntityID)
		}
		if _, ok := g.entities[e.ToEntityID]; !ok {
			return Graph{}, invalid("edge references an unknown entity", "toEntityId").WithDetail("edgeId", e.ID).WithDetail("entityId", e.ToEntityID)
		}
		seen[e.ID] = struct{}{}
		g.edges = append(g.edges, e)
		if e.Status != StatusApproved {
			continue
		}
		switch e.Kind {
		case EdgeParent:
			g.parents[e.FromEntityID] = append(g.parents[e.FromEntityID], e.ToEntityID)
			g.children[e.ToEntityID] = append(g.children[e.ToEntityID], e.FromEntityID)
		case EdgeRelated:
			g.related[e.FromEntityID] = append(g.related[e.FromEntityID], neighbor{id: e.ToEntityID, weight: e.Weight})
			g.related[e.ToEntityID] = append(g.related[e.ToEntityID], neighbor{id: e.FromEntityID, weight: e.Weight})
		}
	}
	return g, nil
}

func byName(a, b Entity) int {
	if c := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}

func (g Graph) Entity(id string) (entity Entity, found bool) {
	entity, found = g.entities[id]
	return entity, found
}

func (g Graph) Entities() []Entity {
	return slices.Clone(g.ordered)
}

func (g Graph) Edges() []Edge {
	return slices.Clone(g.edges)
}

func (g Graph) collect(ids []string) []Entity {
	out := make([]Entity, 0, len(ids))
	for _, id := range ids {
		out = append(out, g.entities[id])
	}
	slices.SortFunc(out, byName)
	return out
}

func (g Graph) Parents(id string, depth int) []Entity {
	if _, ok := g.entities[id]; !ok || depth <= 0 {
		return nil
	}

	seen := map[string]struct{}{id: {}}
	frontier := []string{id}
	var out []Entity
	for level := 0; level < depth && len(frontier) > 0; level++ {
		next := make([]string, 0)
		for _, current := range frontier {
			for _, parentID := range g.parents[current] {
				if _, dup := seen[parentID]; dup {
					continue
				}
				seen[parentID] = struct{}{}
				next = append(next, parentID)
			}
		}
		out = append(out, g.collect(next)...)
		frontier = next
	}
	return out
}

func (g Graph) Children(id string) []Entity {
	return g.collect(g.children[id])
}

func (g Graph) Related(id string, minWeight float64) []Neighbor {
	out := make([]Neighbor, 0, len(g.related[id]))
	for _, n := range g.related[id] {
		if n.weight < minWeight {
			continue
		}
		out = append(out, Neighbor{Entity: g.entities[n.id], Weight: n.weight})
	}
	slices.SortFunc(out, func(a, b Neighbor) int {
		if a.Weight != b.Weight {
			return cmp.Compare(b.Weight, a.Weight)
		}
		return byName(a.Entity, b.Entity)
	})
	return out
}

func (g Graph) Roots() []Entity {
	out := make([]Entity, 0, len(g.ordered))
	for i := range g.ordered {
		if len(g.parents[g.ordered[i].ID]) == 0 {
			out = append(out, g.ordered[i])
		}
	}
	return out
}

func (g Graph) sortedParents(id string) []string {
	ids := slices.Clone(g.parents[id])
	slices.Sort(ids)
	return ids
}

func (g Graph) ValidateAcyclic() error {
	const (
		unvisited = iota
		active
		done
	)

	state := make(map[string]int, len(g.ordered))
	stack := make([]string, 0, len(g.ordered))

	var visit func(id string) []string
	visit = func(id string) []string {
		state[id] = active
		stack = append(stack, id)
		for _, parentID := range g.sortedParents(id) {
			switch state[parentID] {
			case active:
				start := slices.Index(stack, parentID)
				return append(slices.Clone(stack[start:]), parentID)
			case unvisited:
				if cycle := visit(parentID); cycle != nil {
					return cycle
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = done
		return nil
	}

	for i := range g.ordered {
		id := g.ordered[i].ID
		if state[id] != unvisited {
			continue
		}
		if cycle := visit(id); cycle != nil {
			return errors.New(errors.Invalid, "parent edges form a cycle").WithDetail("cycle", cycle)
		}
	}
	return nil
}
