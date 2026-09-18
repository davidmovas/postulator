package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphCreateEntityName = "graph_create_entity"

func graphCreateEntity(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphCreateEntityName,
		Description: "Add an entity to the graph of the site.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.CreateEntityRequest) (graph.CreateEntityResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.CreateEntity(ctx, in)
	})
}
