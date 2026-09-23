package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphApproveEdgeName = "graph_approve_edge"

func graphApproveEdge(deps Deps) Tool {
	return NewTool(Def{
		Name:        graphApproveEdgeName,
		Description: "Approve a proposed edge so the link rules may use it.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in graph.ApproveEdgeRequest) (graph.ApproveEdgeResponse, error) {
		return deps.Graph.ApproveEdge(ctx, in)
	})
}
