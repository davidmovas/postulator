package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type graphFake struct{ mode failure }

func (f graphFake) LoadGraph(context.Context, graph.LoadGraphRequest) (graph.LoadGraphResponse, error) {
	return answer[graph.LoadGraphResponse](f.mode)
}

func (f graphFake) CreateEntity(context.Context, graph.CreateEntityRequest) (graph.CreateEntityResponse, error) {
	return answer[graph.CreateEntityResponse](f.mode)
}

func (f graphFake) UpdateEntity(context.Context, graph.UpdateEntityRequest) (graph.UpdateEntityResponse, error) {
	return answer[graph.UpdateEntityResponse](f.mode)
}

func (f graphFake) DeleteEntity(context.Context, graph.DeleteEntityRequest) (graph.DeleteEntityResponse, error) {
	return answer[graph.DeleteEntityResponse](f.mode)
}

func (f graphFake) GetEntity(context.Context, graph.GetEntityRequest) (graph.GetEntityResponse, error) {
	return answer[graph.GetEntityResponse](f.mode)
}

func (f graphFake) ListEntities(context.Context, graph.ListEntitiesRequest) (paging.List[graph.Entity], error) {
	return answer[paging.List[graph.Entity]](f.mode)
}

func (f graphFake) SetAnchors(context.Context, graph.SetAnchorsRequest) (graph.SetAnchorsResponse, error) {
	return answer[graph.SetAnchorsResponse](f.mode)
}

func (f graphFake) AddEdge(context.Context, graph.AddEdgeRequest) (graph.AddEdgeResponse, error) {
	return answer[graph.AddEdgeResponse](f.mode)
}

func (f graphFake) ApproveEdge(context.Context, graph.ApproveEdgeRequest) (graph.ApproveEdgeResponse, error) {
	return answer[graph.ApproveEdgeResponse](f.mode)
}

func (f graphFake) RejectEdge(context.Context, graph.RejectEdgeRequest) (graph.RejectEdgeResponse, error) {
	return answer[graph.RejectEdgeResponse](f.mode)
}

func (f graphFake) DeleteEdge(context.Context, graph.DeleteEdgeRequest) (graph.DeleteEdgeResponse, error) {
	return answer[graph.DeleteEdgeResponse](f.mode)
}

func (f graphFake) MoveEntity(context.Context, graph.MoveEntityRequest) (graph.MoveEntityResponse, error) {
	return answer[graph.MoveEntityResponse](f.mode)
}

func (f graphFake) ListEdges(context.Context, graph.ListEdgesRequest) (paging.List[graph.Edge], error) {
	return answer[paging.List[graph.Edge]](f.mode)
}

func (f graphFake) RecomputeScores(context.Context, graph.RecomputeScoresRequest) (graph.RecomputeScoresResponse, error) {
	return answer[graph.RecomputeScoresResponse](f.mode)
}

func (f graphFake) ProposeFromPages(context.Context, graph.ProposeFromPagesRequest) (graph.ProposeFromPagesResponse, error) {
	return answer[graph.ProposeFromPagesResponse](f.mode)
}

func (f graphFake) ProposeRelated(context.Context, graph.ProposeRelatedRequest) (graph.ProposeRelatedResponse, error) {
	return answer[graph.ProposeRelatedResponse](f.mode)
}

func (f graphFake) PreviewFromPages(context.Context, graph.PreviewFromPagesRequest) (graph.PreviewFromPagesResponse, error) {
	return answer[graph.PreviewFromPagesResponse](f.mode)
}

func (f graphFake) ProposeFromKeywords(context.Context, graph.ProposeFromKeywordsRequest) (graph.ProposeFromKeywordsResponse, error) {
	return answer[graph.ProposeFromKeywordsResponse](f.mode)
}

func (f graphFake) ApplyProposals(context.Context, graph.ApplyProposalsRequest) (graph.ApplyProposalsResponse, error) {
	return answer[graph.ApplyProposalsResponse](f.mode)
}

func TestGraphServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewGraphService(zap.NewNop(), ready[wails.GraphUseCase](graphFake{})), []string{
		"AddEdge", "ApplyProposals", "ApproveEdge", "CreateEntity", "DeleteEdge", "DeleteEntity", "GetEntity",
		"ListEdges", "ListEntities", "LoadGraph", "MoveEntity", "PreviewFromPages", "ProposeFromKeywords",
		"ProposeFromPages", "ProposeRelated", "RecomputeScores", "RejectEdge", "SetAnchors", "UpdateEntity",
	})
	assertEveryMethodConverts(t, wails.NewGraphService(zap.NewNop(), ready[wails.GraphUseCase](graphFake{mode: missing})), missingBody)
	assertEveryMethodConverts(t, wails.NewGraphService(zap.NewNop(), ready[wails.GraphUseCase](graphFake{mode: panicking})), panicBody)
}
