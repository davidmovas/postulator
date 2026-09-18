package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphAddEdgeName = "graph_add_edge"

func graphAddEdge(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphAddEdgeName,
		Description: "Connect two entities with a parent or related edge.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.AddEdgeRequest) (graph.AddEdgeResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.AddEdge(ctx, in)
	})
}
