package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type graphUseCase interface {
	LoadGraph(ctx context.Context, req graph.LoadGraphRequest) (graph.LoadGraphResponse, error)
	CreateEntity(ctx context.Context, req graph.CreateEntityRequest) (graph.CreateEntityResponse, error)
	UpdateEntity(ctx context.Context, req graph.UpdateEntityRequest) (graph.UpdateEntityResponse, error)
	DeleteEntity(ctx context.Context, req graph.DeleteEntityRequest) (graph.DeleteEntityResponse, error)
	GetEntity(ctx context.Context, req graph.GetEntityRequest) (graph.GetEntityResponse, error)
	ListEntities(ctx context.Context, req graph.ListEntitiesRequest) (paging.List[graph.Entity], error)
	SetAnchors(ctx context.Context, req graph.SetAnchorsRequest) (graph.SetAnchorsResponse, error)
	AddEdge(ctx context.Context, req graph.AddEdgeRequest) (graph.AddEdgeResponse, error)
	ApproveEdge(ctx context.Context, req graph.ApproveEdgeRequest) (graph.ApproveEdgeResponse, error)
	RejectEdge(ctx context.Context, req graph.RejectEdgeRequest) (graph.RejectEdgeResponse, error)
	DeleteEdge(ctx context.Context, req graph.DeleteEdgeRequest) (graph.DeleteEdgeResponse, error)
	ListEdges(ctx context.Context, req graph.ListEdgesRequest) (paging.List[graph.Edge], error)
	RecomputeScores(ctx context.Context, req graph.RecomputeScoresRequest) (graph.RecomputeScoresResponse, error)
	ProposeFromPages(ctx context.Context, req graph.ProposeFromPagesRequest) (graph.ProposeFromPagesResponse, error)
	ProposeRelated(ctx context.Context, req graph.ProposeRelatedRequest) (graph.ProposeRelatedResponse, error)
}

type GraphService struct {
	loadGraph        middleware.Handler[graph.LoadGraphRequest, graph.LoadGraphResponse]
	createEntity     middleware.Handler[graph.CreateEntityRequest, graph.CreateEntityResponse]
	updateEntity     middleware.Handler[graph.UpdateEntityRequest, graph.UpdateEntityResponse]
	deleteEntity     middleware.Handler[graph.DeleteEntityRequest, graph.DeleteEntityResponse]
	getEntity        middleware.Handler[graph.GetEntityRequest, graph.GetEntityResponse]
	listEntities     middleware.Handler[graph.ListEntitiesRequest, paging.List[graph.Entity]]
	setAnchors       middleware.Handler[graph.SetAnchorsRequest, graph.SetAnchorsResponse]
	addEdge          middleware.Handler[graph.AddEdgeRequest, graph.AddEdgeResponse]
	approveEdge      middleware.Handler[graph.ApproveEdgeRequest, graph.ApproveEdgeResponse]
	rejectEdge       middleware.Handler[graph.RejectEdgeRequest, graph.RejectEdgeResponse]
	deleteEdge       middleware.Handler[graph.DeleteEdgeRequest, graph.DeleteEdgeResponse]
	listEdges        middleware.Handler[graph.ListEdgesRequest, paging.List[graph.Edge]]
	recomputeScores  middleware.Handler[graph.RecomputeScoresRequest, graph.RecomputeScoresResponse]
	proposeFromPages middleware.Handler[graph.ProposeFromPagesRequest, graph.ProposeFromPagesResponse]
	proposeRelated   middleware.Handler[graph.ProposeRelatedRequest, graph.ProposeRelatedResponse]
}

func NewGraphService(logger *zap.Logger, useCase graphUseCase) *GraphService {
	return &GraphService{
		loadGraph:        Wrap(logger, "graph.loadGraph", useCase.LoadGraph),
		createEntity:     Wrap(logger, "graph.createEntity", useCase.CreateEntity),
		updateEntity:     Wrap(logger, "graph.updateEntity", useCase.UpdateEntity),
		deleteEntity:     Wrap(logger, "graph.deleteEntity", useCase.DeleteEntity),
		getEntity:        Wrap(logger, "graph.getEntity", useCase.GetEntity),
		listEntities:     Wrap(logger, "graph.listEntities", useCase.ListEntities),
		setAnchors:       Wrap(logger, "graph.setAnchors", useCase.SetAnchors),
		addEdge:          Wrap(logger, "graph.addEdge", useCase.AddEdge),
		approveEdge:      Wrap(logger, "graph.approveEdge", useCase.ApproveEdge),
		rejectEdge:       Wrap(logger, "graph.rejectEdge", useCase.RejectEdge),
		deleteEdge:       Wrap(logger, "graph.deleteEdge", useCase.DeleteEdge),
		listEdges:        Wrap(logger, "graph.listEdges", useCase.ListEdges),
		recomputeScores:  Wrap(logger, "graph.recomputeScores", useCase.RecomputeScores),
		proposeFromPages: Wrap(logger, "graph.proposeFromPages", useCase.ProposeFromPages),
		proposeRelated:   Wrap(logger, "graph.proposeRelated", useCase.ProposeRelated),
	}
}

func (s *GraphService) LoadGraph(c context.Context, req graph.LoadGraphRequest) (graph.LoadGraphResponse, error) {
	return s.loadGraph(c, req)
}

func (s *GraphService) CreateEntity(c context.Context, req graph.CreateEntityRequest) (graph.CreateEntityResponse, error) {
	return s.createEntity(c, req)
}

func (s *GraphService) UpdateEntity(c context.Context, req graph.UpdateEntityRequest) (graph.UpdateEntityResponse, error) {
	return s.updateEntity(c, req)
}

func (s *GraphService) DeleteEntity(c context.Context, req graph.DeleteEntityRequest) (graph.DeleteEntityResponse, error) {
	return s.deleteEntity(c, req)
}

func (s *GraphService) GetEntity(c context.Context, req graph.GetEntityRequest) (graph.GetEntityResponse, error) {
	return s.getEntity(c, req)
}

func (s *GraphService) ListEntities(c context.Context, req graph.ListEntitiesRequest) (paging.List[graph.Entity], error) {
	return s.listEntities(c, req)
}

func (s *GraphService) SetAnchors(c context.Context, req graph.SetAnchorsRequest) (graph.SetAnchorsResponse, error) {
	return s.setAnchors(c, req)
}

func (s *GraphService) AddEdge(c context.Context, req graph.AddEdgeRequest) (graph.AddEdgeResponse, error) {
	return s.addEdge(c, req)
}

func (s *GraphService) ApproveEdge(c context.Context, req graph.ApproveEdgeRequest) (graph.ApproveEdgeResponse, error) {
	return s.approveEdge(c, req)
}

func (s *GraphService) RejectEdge(c context.Context, req graph.RejectEdgeRequest) (graph.RejectEdgeResponse, error) {
	return s.rejectEdge(c, req)
}

func (s *GraphService) DeleteEdge(c context.Context, req graph.DeleteEdgeRequest) (graph.DeleteEdgeResponse, error) {
	return s.deleteEdge(c, req)
}

func (s *GraphService) ListEdges(c context.Context, req graph.ListEdgesRequest) (paging.List[graph.Edge], error) {
	return s.listEdges(c, req)
}

func (s *GraphService) RecomputeScores(c context.Context, req graph.RecomputeScoresRequest) (graph.RecomputeScoresResponse, error) {
	return s.recomputeScores(c, req)
}

func (s *GraphService) ProposeFromPages(c context.Context, req graph.ProposeFromPagesRequest) (graph.ProposeFromPagesResponse, error) {
	return s.proposeFromPages(c, req)
}

func (s *GraphService) ProposeRelated(c context.Context, req graph.ProposeRelatedRequest) (graph.ProposeRelatedResponse, error) {
	return s.proposeRelated(c, req)
}
