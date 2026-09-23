package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphDeleteEdgeName = "graph_delete_edge"

func graphDeleteEdge(deps Deps) Tool {
	return NewTool(Def{
		Name:        graphDeleteEdgeName,
		Description: "Delete an edge from the graph.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in graph.DeleteEdgeRequest) (graph.DeleteEdgeResponse, error) {
		return deps.Graph.DeleteEdge(ctx, in)
	})
}
