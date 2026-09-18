package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const graphListEntitiesName = "graph_list_entities"

func graphListEntities(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphListEntitiesName,
		Description: "List the entities of the site, newest first.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in graph.ListEntitiesRequest) (paging.List[graph.Entity], error) {
		in.SiteID = b.SiteID
		return deps.Graph.ListEntities(ctx, in)
	})
}
