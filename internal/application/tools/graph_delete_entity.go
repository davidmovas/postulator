package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphDeleteEntityName = "graph_delete_entity"

func graphDeleteEntity(deps Deps) Tool {
	return NewTool(Def{
		Name:        graphDeleteEntityName,
		Description: "Delete an entity and the edges that touch it.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in graph.DeleteEntityRequest) (graph.DeleteEntityResponse, error) {
		return deps.Graph.DeleteEntity(ctx, in)
	})
}
