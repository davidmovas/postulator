package graph

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Clash struct {
	ScopeID   *string
	Name      string
	EntityIDs []string
}

func Scopes(entities []Entity, edges []Edge) map[string]*string {
	parents := make(map[string][]Edge, len(entities))
	for i := range edges {
		if edges[i].Kind == EdgeParent && edges[i].Status == StatusApproved {
			parents[edges[i].FromEntityID] = append(parents[edges[i].FromEntityID], edges[i])
		}
	}

	scopes := make(map[string]*string, len(entities))
	for i := range entities {
		held := parents[entities[i].ID]
		slices.SortFunc(held, func(a, b Edge) int {
			if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
				return c
			}
			return strings.Compare(a.ID, b.ID)
		})
		scopes[entities[i].ID] = scopeAmong(entities[i].ScopeID, held)
	}
	return scopes
}

func scopeAmong(current *string, parents []Edge) *string {
	if current != nil {
		for i := range parents {
			if parents[i].ToEntityID == *current {
				return new(*current)
			}
		}
	}
	if len(parents) == 0 {
		return nil
	}
	return new(parents[0].ToEntityID)
}

func scopeKey(scope *string) string {
	if scope == nil {
		return ""
	}
	return *scope
}

func ScopeClashes(entities []Entity) []Clash {
	held := make(map[string]int, len(entities))
	clashes := make([]Clash, 0)
	for i := range entities {
		key := scopeKey(entities[i].ScopeID) + "\x00" + strings.ToLower(strings.TrimSpace(entities[i].Name))
		at, seen := held[key]
		if !seen {
			held[key] = -1 - i
			continue
		}
		if at < 0 {
			first := entities[-1-at]
			held[key] = len(clashes)
			clashes = append(clashes, Clash{ScopeID: entities[i].ScopeID, Name: first.Name, EntityIDs: []string{first.ID}})
			at = held[key]
		}
		clashes[at].EntityIDs = append(clashes[at].EntityIDs, entities[i].ID)
	}
	return clashes
}

func Settle(entities []Entity, edges []Edge) ([]Entity, error) {
	scopes := Scopes(entities, edges)
	next := make([]Entity, 0, len(entities))
	moved := make([]Entity, 0)
	byID := make(map[string]Entity, len(entities))
	for i := range entities {
		byID[entities[i].ID] = entities[i]
		entity := entities[i]
		entity.ScopeID = scopes[entity.ID]
		next = append(next, entity)
		if scopeKey(entity.ScopeID) != scopeKey(entities[i].ScopeID) {
			moved = append(moved, entity)
		}
	}

	clashes := ScopeClashes(next)
	if len(clashes) == 0 {
		return moved, nil
	}
	clash := clashes[0]
	if clash.ScopeID == nil {
		return nil, errors.New(errors.Conflict, "two entities named "+clash.Name+
			" would sit at the top of the graph; rename one of them or put it under a parent").
			WithDetail("name", clash.Name).WithDetail("entityIds", clash.EntityIDs)
	}
	return nil, errors.New(errors.Conflict, "two entities named "+clash.Name+" would sit under "+
		byID[*clash.ScopeID].Name+"; rename one of them or put it under another parent").
		WithDetail("name", clash.Name).WithDetail("entityIds", clash.EntityIDs).WithDetail("scopeId", *clash.ScopeID)
}

func (g Graph) Label(id string) string {
	entity, found := g.entities[id]
	if !found {
		return ""
	}
	if entity.ScopeID == nil || !g.shared(entity) {
		return entity.Name
	}
	parent, found := g.entities[*entity.ScopeID]
	if !found {
		return entity.Name
	}
	return parent.Name + " " + entity.Name
}

func (g Graph) shared(entity Entity) bool {
	for i := range g.ordered {
		if g.ordered[i].ID != entity.ID && strings.EqualFold(g.ordered[i].Name, entity.Name) {
			return true
		}
	}
	return false
}
