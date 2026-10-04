package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphCreateEntitiesName = "graph_create_entities"

func graphCreateEntities(deps Deps) Tool {
	return newSiteTool(Def{
		Name: graphCreateEntitiesName,
		Description: "Add a tree of entities in one write, each naming its parent; " +
			"use it instead of graph_create_entity in a loop.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in graph.CreateEntitiesRequest) (graph.CreateEntitiesResponse, error) {
		in.SiteID = b.SiteID
		return deps.Graph.CreateEntities(ctx, in)
	})
}
