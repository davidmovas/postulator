package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphGetEntityName = "graph_get_entity"

func graphGetEntity(deps Deps) Tool {
	return NewTool(Def{
		Name:        graphGetEntityName,
		Description: "Read one entity with its keywords and anchors.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in graph.GetEntityRequest) (graph.GetEntityResponse, error) {
		return deps.Graph.GetEntity(ctx, in)
	})
}
