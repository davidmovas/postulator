package graph

import (
	"context"
	"strings"

	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (s *Service) MoveEntity(ctx context.Context, req MoveEntityRequest) (MoveEntityResponse, error) {
	entityID := strings.TrimSpace(req.EntityID)
	parentID := strings.TrimSpace(req.NewParentID)
	switch {
	case entityID == "":
		return MoveEntityResponse{}, invalidField("a move needs the entity to move", "entityId")
	case parentID == "":
		return MoveEntityResponse{}, invalidField("a move needs the entity to move it under", "newParentId")
	case entityID == parentID:
		return MoveEntityResponse{}, invalidField("an entity cannot sit under itself", "newParentId")
	}

	var answer MoveEntityResponse
	err := s.uow.Do(ctx, func(c context.Context) error {
		moved, getErr := s.entities.Get(c, entityID)
		if getErr != nil {
			return getErr
		}
		parent, getErr := s.entities.Get(c, parentID)
		if getErr != nil {
			return getErr
		}
		if parent.SiteID != moved.SiteID {
			return invalidField("the new parent belongs to another site", "newParentId")
		}

		edges, listErr := s.edges.ListBySite(c, moved.SiteID)
		if listErr != nil {
			return listErr
		}

		held, replaced := parentEdges(edges, entityID, parentID)
		edge, addErr := s.parentEdge(c, held, moved.SiteID, entityID, parentID)
		if addErr != nil {
			return addErr
		}

		removed := make([]string, 0, len(replaced))
		if !req.KeepBoth {
			for i := range replaced {
				if deleteErr := s.edges.Delete(c, replaced[i].ID); deleteErr != nil {
					return deleteErr
				}
				removed = append(removed, replaced[i].ID)
			}
		}
		answer = MoveEntityResponse{Edge: edgeView(edge), RemovedEdgeIDs: removed}
		return nil
	})
	if err != nil {
		return MoveEntityResponse{}, err
	}
	if publishErr := s.changed(answer.Edge.SiteID); publishErr != nil {
		return MoveEntityResponse{}, publishErr
	}
	return answer, nil
}

func parentEdges(edges []graphdomain.Edge, entityID, parentID string) (held *graphdomain.Edge,
	replaced []graphdomain.Edge) {
	replaced = make([]graphdomain.Edge, 0, len(edges))
	for i := range edges {
		if edges[i].Kind != graphdomain.EdgeParent || edges[i].FromEntityID != entityID {
			continue
		}
		if edges[i].ToEntityID == parentID {
			if edges[i].Status == graphdomain.StatusApproved {
				held = &edges[i]
			}
			continue
		}
		if edges[i].Status == graphdomain.StatusApproved {
			replaced = append(replaced, edges[i])
		}
	}
	return held, replaced
}

func (s *Service) parentEdge(ctx context.Context, held *graphdomain.Edge,
	siteID, entityID, parentID string) (graphdomain.Edge, error) {
	if held != nil {
		return *held, nil
	}

	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID:           id.New(),
		SiteID:       siteID,
		FromEntityID: entityID,
		ToEntityID:   parentID,
		Kind:         graphdomain.EdgeParent,
		Weight:       1,
		Source:       sourceOfActor(ctx),
		Status:       graphdomain.StatusApproved,
		CreatedAt:    s.now(),
	})
	if err != nil {
		return graphdomain.Edge{}, err
	}
	if checkErr := s.checkAcyclic(ctx, edge); checkErr != nil {
		return graphdomain.Edge{}, checkErr
	}
	if insertErr := s.edges.Insert(ctx, edge); insertErr != nil {
		return graphdomain.Edge{}, insertErr
	}
	return edge, nil
}
