package graph

import (
	"context"

	"github.com/davidmovas/postulator/internal/application"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func entityViews(entities []graphdomain.Entity) []Entity {
	out := make([]Entity, 0, len(entities))
	for i := range entities {
		out = append(out, entityView(entities[i]))
	}
	return out
}

func edgeView(e graphdomain.Edge) Edge {
	return Edge{
		ID:           e.ID,
		SiteID:       e.SiteID,
		FromEntityID: e.FromEntityID,
		ToEntityID:   e.ToEntityID,
		Kind:         string(e.Kind),
		Weight:       e.Weight,
		Source:       string(e.Source),
		Status:       string(e.Status),
		Reason:       e.Reason,
		CreatedAt:    dto.NewTime(e.CreatedAt),
	}
}

func edgeViews(edges []graphdomain.Edge) []Edge {
	out := make([]Edge, 0, len(edges))
	for i := range edges {
		out = append(out, edgeView(edges[i]))
	}
	return out
}

func (s *Service) Load(ctx context.Context, siteID string) (graphdomain.Graph, error) {
	if err := requireSite(siteID); err != nil {
		return graphdomain.Graph{}, err
	}
	if _, err := s.sites.Get(ctx, siteID); err != nil {
		return graphdomain.Graph{}, err
	}
	return s.load(ctx, siteID)
}

func (s *Service) load(ctx context.Context, siteID string) (graphdomain.Graph, error) {
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return graphdomain.Graph{}, err
	}
	edges, err := s.edges.ListBySite(ctx, siteID)
	if err != nil {
		return graphdomain.Graph{}, err
	}
	return graphdomain.New(entities, edges)
}

func (s *Service) LoadGraph(ctx context.Context, req LoadGraphRequest) (LoadGraphResponse, error) {
	g, err := s.Load(ctx, req.SiteID)
	if err != nil {
		return LoadGraphResponse{}, err
	}
	return LoadGraphResponse{Entities: entityViews(g.Entities()), Edges: edgeViews(g.Edges())}, nil
}

func (s *Service) checkAcyclic(ctx context.Context, candidate graphdomain.Edge) error {
	entities, err := s.entities.ListBySite(ctx, candidate.SiteID)
	if err != nil {
		return err
	}
	edges, err := s.edges.ListBySite(ctx, candidate.SiteID)
	if err != nil {
		return err
	}

	candidate.Status = graphdomain.StatusApproved
	merged := make([]graphdomain.Edge, 0, len(edges)+1)
	for i := range edges {
		if edges[i].ID != candidate.ID {
			merged = append(merged, edges[i])
		}
	}
	merged = append(merged, candidate)

	g, err := graphdomain.New(entities, merged)
	if err != nil {
		return err
	}
	if candidate.Kind != graphdomain.EdgeParent {
		return nil
	}
	return g.ValidateAcyclic()
}

func (s *Service) AddEdge(ctx context.Context, req AddEdgeRequest) (AddEdgeResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return AddEdgeResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return AddEdgeResponse{}, err
	}

	source := graphdomain.Source(req.Source)
	if req.Source == "" {
		source = graphdomain.SourceUser
	}
	status := graphdomain.EdgeStatus(req.Status)
	if req.Status == "" {
		status = graphdomain.StatusApproved
	}
	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID:           id.New(),
		SiteID:       req.SiteID,
		FromEntityID: req.FromEntityID,
		ToEntityID:   req.ToEntityID,
		Kind:         graphdomain.EdgeKind(req.Kind),
		Weight:       req.Weight,
		Source:       source,
		Status:       status,
		Reason:       req.Reason,
		CreatedAt:    s.now(),
	})
	if err != nil {
		return AddEdgeResponse{}, err
	}

	doErr := s.uow.Do(ctx, func(c context.Context) error {
		if checkErr := s.checkAcyclic(c, edge); checkErr != nil {
			return checkErr
		}
		return s.edges.Insert(c, edge)
	})
	if doErr != nil {
		return AddEdgeResponse{}, doErr
	}
	if publishErr := s.changed(edge.SiteID); publishErr != nil {
		return AddEdgeResponse{}, publishErr
	}
	return AddEdgeResponse{Edge: edgeView(edge)}, nil
}

func (s *Service) setStatus(ctx context.Context, edgeID string, status graphdomain.EdgeStatus) (graphdomain.Edge, error) {
	var (
		edge    graphdomain.Edge
		changed bool
	)
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.edges.Get(c, edgeID)
		if getErr != nil {
			return getErr
		}
		edge = current
		if current.Status == status {
			return nil
		}
		if status == graphdomain.StatusApproved {
			if checkErr := s.checkAcyclic(c, current); checkErr != nil {
				return checkErr
			}
		}
		if setErr := s.edges.SetStatus(c, edgeID, status); setErr != nil {
			return setErr
		}
		edge.Status = status
		changed = true
		return nil
	})
	if err != nil {
		return graphdomain.Edge{}, err
	}
	if changed {
		if publishErr := s.changed(edge.SiteID); publishErr != nil {
			return graphdomain.Edge{}, publishErr
		}
	}
	return edge, nil
}

func (s *Service) ApproveEdge(ctx context.Context, req ApproveEdgeRequest) (ApproveEdgeResponse, error) {
	edge, err := s.setStatus(ctx, req.ID, graphdomain.StatusApproved)
	if err != nil {
		return ApproveEdgeResponse{}, err
	}
	return ApproveEdgeResponse{Edge: edgeView(edge)}, nil
}

func (s *Service) RejectEdge(ctx context.Context, req RejectEdgeRequest) (RejectEdgeResponse, error) {
	edge, err := s.setStatus(ctx, req.ID, graphdomain.StatusRejected)
	if err != nil {
		return RejectEdgeResponse{}, err
	}
	return RejectEdgeResponse{Edge: edgeView(edge)}, nil
}

func (s *Service) DeleteEdge(ctx context.Context, req DeleteEdgeRequest) (DeleteEdgeResponse, error) {
	var siteID string
	err := s.uow.Do(ctx, func(c context.Context) error {
		current, getErr := s.edges.Get(c, req.ID)
		if getErr != nil {
			return getErr
		}
		siteID = current.SiteID
		return s.edges.Delete(c, req.ID)
	})
	if err != nil {
		return DeleteEdgeResponse{}, err
	}
	if publishErr := s.changed(siteID); publishErr != nil {
		return DeleteEdgeResponse{}, publishErr
	}
	return DeleteEdgeResponse{}, nil
}

func (s *Service) ListEdges(ctx context.Context, req ListEdgesRequest) (paging.List[Edge], error) {
	if err := requireSite(req.SiteID); err != nil {
		return paging.List[Edge]{}, err
	}
	q := graphdomain.EdgeQuery{SiteID: req.SiteID, EntityID: req.EntityID}
	if req.Kind != "" {
		kind := graphdomain.EdgeKind(req.Kind)
		if !kind.Valid() {
			return paging.List[Edge]{}, errors.New(errors.Invalid, "edge kind is not recognized").WithDetail("field", "kind")
		}
		q.Kind = &kind
	}
	if req.Status != "" {
		status := graphdomain.EdgeStatus(req.Status)
		if !status.Valid() {
			return paging.List[Edge]{}, errors.New(errors.Invalid, "edge status is not recognized").WithDetail("field", "status")
		}
		q.Status = &status
	}
	if req.Sort != nil {
		if req.Sort.Field != string(graphdomain.EntitySortCreatedAt) {
			return paging.List[Edge]{}, errors.New(errors.Invalid, "edges are sorted by creation time only").WithDetail("field", "sort.field")
		}
		q.Desc = req.Sort.Desc
	}

	list, err := s.edges.List(ctx, q, application.PageRequest(req.ListRequest))
	if err != nil {
		return paging.List[Edge]{}, err
	}
	return application.MapList(list, edgeView), nil
}

func (s *Service) RecomputeScores(ctx context.Context, req RecomputeScoresRequest) (RecomputeScoresResponse, error) {
	if err := requireSite(req.SiteID); err != nil {
		return RecomputeScoresResponse{}, err
	}
	if _, err := s.sites.Get(ctx, req.SiteID); err != nil {
		return RecomputeScoresResponse{}, err
	}

	var scores map[string]float64
	err := s.uow.Do(ctx, func(c context.Context) error {
		g, loadErr := s.load(c, req.SiteID)
		if loadErr != nil {
			return loadErr
		}
		scores = g.Score()
		for entityID, score := range scores {
			if setErr := s.entities.SetScore(c, entityID, score); setErr != nil {
				return setErr
			}
		}
		return nil
	})
	if err != nil {
		return RecomputeScoresResponse{}, err
	}
	if publishErr := s.changed(req.SiteID); publishErr != nil {
		return RecomputeScoresResponse{}, publishErr
	}
	return RecomputeScoresResponse{Scores: scores}, nil
}
