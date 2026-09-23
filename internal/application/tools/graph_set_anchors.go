package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphSetAnchorsName = "graph_set_anchors"

func graphSetAnchors(deps Deps) Tool {
	return NewTool(Def{
		Name:        graphSetAnchorsName,
		Description: "Replace the anchor texts an entity may be linked with.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in graph.SetAnchorsRequest) (graph.SetAnchorsResponse, error) {
		return deps.Graph.SetAnchors(ctx, in)
	})
}
