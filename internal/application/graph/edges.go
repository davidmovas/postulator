package graph

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/field"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
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
	states, err := s.pageStates(ctx, req.SiteID)
	if err != nil {
		return LoadGraphResponse{}, err
	}
	return LoadGraphResponse{
		Entities: entityViews(g.Entities()), Edges: edgeViews(g.Edges()), Pages: states,
	}, nil
}

func (s *Service) pageStates(ctx context.Context, siteID string) ([]EntityPage, error) {
	pages, err := s.pages.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	working, err := s.inFlight(ctx, siteID)
	if err != nil {
		return nil, err
	}
	identities, err := s.identities.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]pagemap.Identity, len(identities))
	for i := range identities {
		seen[identities[i].PageID] = identities[i]
	}
	fields, err := s.fields.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	states := make([]EntityPage, 0, len(pages))
	for i := range pages {
		if pages[i].EntityID == nil {
			continue
		}
		identity, observed := seen[pages[i].ID]
		states = append(states, EntityPage{
			EntityID:   *pages[i].EntityID,
			PageID:     pages[i].ID,
			Path:       pages[i].Path,
			Status:     string(pages[i].Status),
			PageTypeID: pages[i].PageTypeID,
			Work:       working[pages[i].ID],
			Mismatch:   observed && differsFromSite(pages[i], identity, fields[pages[i].ID]),
		})
	}
	return states, nil
}

func differsFromSite(page pagemap.Page, identity pagemap.Identity, values []field.Value) bool {
	asked := page.Status == pagemap.StatusPublished || page.Status == pagemap.StatusExists
	if asked && identity.Status != "" && pagemap.StatusFromWordPress(identity.Status) != page.Status {
		return true
	}
	if page.WPType.StoreAddressed() {
		return false
	}
	if path, err := pagemap.NormalizePath(identity.URL); err == nil && !strings.Contains(identity.URL, "?") && path != page.Path {
		return true
	}
	for i := range values {
		switch values[i].Key {
		case field.Title:
			if page.Title != "" && values[i].Current != "" && page.Title != values[i].Current {
				return true
			}
		case field.H1:
			if page.H1 != "" && values[i].Current != "" && page.H1 != values[i].Current {
				return true
			}
		}
	}
	return false
}

func (s *Service) inFlight(ctx context.Context, siteID string) (map[string]string, error) {
	if s.work == nil {
		return map[string]string{}, nil
	}
	items, err := s.work.ActiveBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	working := make(map[string]string, len(items))
	for i := range items {
		if _, seen := working[items[i].TargetID]; seen {
			continue
		}
		working[items[i].TargetID] = string(items[i].Status)
	}
	return working, nil
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
		if insertErr := s.edges.Insert(c, edge); insertErr != nil {
			return insertErr
		}
		return s.settleScopes(c, edge.SiteID)
	})
	if doErr != nil {
		return AddEdgeResponse{}, doErr
	}
	if movedErr := s.parentsMoved(ctx, edge.SiteID); movedErr != nil {
		return AddEdgeResponse{}, movedErr
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
		return s.settleScopes(c, edge.SiteID)
	})
	if err != nil {
		return graphdomain.Edge{}, err
	}
	if changed {
		if movedErr := s.parentsMoved(ctx, edge.SiteID); movedErr != nil {
			return graphdomain.Edge{}, movedErr
		}
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
		if deleteErr := s.edges.Delete(c, req.ID); deleteErr != nil {
			return deleteErr
		}
		return s.settleScopes(c, siteID)
	})
	if err != nil {
		return DeleteEdgeResponse{}, err
	}
	if movedErr := s.parentsMoved(ctx, siteID); movedErr != nil {
		return DeleteEdgeResponse{}, movedErr
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
