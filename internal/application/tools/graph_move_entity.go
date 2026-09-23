package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphMoveEntityName = "graph_move_entity"

func graphMoveEntity(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphMoveEntityName,
		Description: "Move an entity under another parent in one step.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in graph.MoveEntityRequest) (graph.MoveEntityResponse, error) {
		return deps.Graph.MoveEntity(ctx, in)
	})
}
