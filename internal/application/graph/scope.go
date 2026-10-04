package graph

import (
	"context"

	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
)

func (s *Service) settleScopes(ctx context.Context, siteID string) error {
	return s.settle(ctx, siteID, "", "", "")
}

func (s *Service) settleScopesWithout(ctx context.Context, siteID, removedID string) error {
	return s.settle(ctx, siteID, removedID, "", "")
}

func (s *Service) settleScopesMovingTo(ctx context.Context, siteID, entityID, parentID string) error {
	return s.settle(ctx, siteID, "", entityID, parentID)
}

func (s *Service) settle(ctx context.Context, siteID, removedID, movedID, movedTo string) error {
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	edges, err := s.edges.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	kept := make([]graphdomain.Entity, 0, len(entities))
	stored := make(map[string]*string, len(entities))
	for i := range entities {
		if entities[i].ID == removedID {
			continue
		}
		stored[entities[i].ID] = entities[i].ScopeID
		if entities[i].ID == movedID {
			entities[i].ScopeID = &movedTo
		}
		kept = append(kept, entities[i])
	}
	held := make([]graphdomain.Edge, 0, len(edges))
	for i := range edges {
		if edges[i].FromEntityID != removedID && edges[i].ToEntityID != removedID {
			held = append(held, edges[i])
		}
	}

	settled, err := graphdomain.Settle(kept, held)
	if err != nil {
		return err
	}
	moved := make(map[string]*string, len(settled)+1)
	for i := range settled {
		moved[settled[i].ID] = settled[i].ScopeID
	}
	if movedID != "" {
		if _, again := moved[movedID]; !again {
			moved[movedID] = &movedTo
		}
	}

	now := s.now()
	for i := range kept {
		next, changes := moved[kept[i].ID]
		if !changes || sameScope(stored[kept[i].ID], next) {
			continue
		}
		if setErr := s.entities.SetScope(ctx, kept[i].ID, next, now); setErr != nil {
			return setErr
		}
	}
	return nil
}

func sameScope(a, b *string) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}
