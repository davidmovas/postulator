package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphUpdateEntityName = "graph_update_entity"

func graphUpdateEntity(deps Deps) Tool {
	return NewTool(Def{
		Name:        graphUpdateEntityName,
		Description: "Rename an entity or change its kind, intent or keywords.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in graph.UpdateEntityRequest) (graph.UpdateEntityResponse, error) {
		return deps.Graph.UpdateEntity(ctx, in)
	})
}
