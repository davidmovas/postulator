package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphRejectEdgeName = "graph_reject_edge"

func graphRejectEdge(deps Deps) Tool {
	return NewTool(Def{
		Name:        graphRejectEdgeName,
		Description: "Reject a proposed edge.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in graph.RejectEdgeRequest) (graph.RejectEdgeResponse, error) {
		return deps.Graph.RejectEdge(ctx, in)
	})
}
